package cmd

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/k8s-source/adapters"
	"github.com/overmindtech/cli/logging"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otellogrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-source",
	Short: "Kubernetes source",
	Long: `Gathers details from existing kubernetes clusters
`,
	Run: func(cmd *cobra.Command, args []string) {
		exitcode := run(cmd, args)
		os.Exit(exitcode)
	},
}

func run(_ *cobra.Command, _ []string) int {
	kubeconfig := viper.GetString("kubeconfig")
	// get engine config
	engineConfig, err := discovery.EngineConfigFromViper("k8s", tracing.Version())
	if err != nil {
		log.WithError(err).Fatal("Could not get engine config from viper")
	}

	log.WithFields(log.Fields{
		"kubeconfig": kubeconfig,
	}).Info("Got config")

	var clientSet *kubernetes.Clientset
	var restConfig *rest.Config

	if kubeconfig == "" {
		log.Info("Using in-cluster config")

		restConfig, err = rest.InClusterConfig()
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not load in-cluster config")

			return 1
		}
	} else {
		// Load kubernetes config from a file
		restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			sentry.CaptureException(err)
			log.WithError(err).Error("Could not load kubernetes config")

			return 1
		}
	}

	restConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper { return otelhttp.NewTransport(rt) })
	// Set up rate limiting
	restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(
		float32(viper.GetFloat64("rate-limit-qps")),
		viper.GetInt("rate-limit-burst"),
	)
	// Create clientSet
	clientSet, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Error("Could not create kubernetes client")

		return 1
	}
	//
	// Discover info
	//
	// Now that we have a connection to the kubernetes cluster we need to go
	// about generating some adapters.
	var k8sURL *url.URL

	k8sURL, err = url.Parse(restConfig.Host)
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Errorf("Could not parse kubernetes url: %v", restConfig.Host)

		return 1
	}

	// Calculate the SHA-1 hash of the config to use as the queue name. This
	// means that adapters with the same config will be in the same queue.
	// Note that the config object implements redaction in the String()
	// method so we don't have to worry about leaking secrets
	configHash := fmt.Sprintf("%x", sha256.Sum256([]byte(restConfig.String())))
	engineConfig.NATSQueueName = fmt.Sprintf("k8s-source-%v", configHash)

	// If there is no port then set one
	if k8sURL.Port() == "" {
		switch k8sURL.Scheme {
		case "http":
			k8sURL.Host = k8sURL.Host + ":80"
		case "https":
			k8sURL.Host = k8sURL.Host + ":443"
		}
	}

	err = engineConfig.CreateClients()
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Fatal("could not create auth clients")
	}

	// Work out the cluster name
	clusterName := viper.GetString("cluster-name")
	if clusterName == "" {
		clusterName = k8sURL.Host
	}
	if engineConfig.HeartbeatOptions == nil {
		engineConfig.HeartbeatOptions = &discovery.HeartbeatOptions{}
	}

	e, err := discovery.NewEngine(engineConfig)
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Error("Error initializing Engine")

		return 1
	}

	// ReadinessCheck verifies adapters are healthy by using a Node adapter
	// Timeout is handled by SendHeartbeat, HTTP handlers rely on request context
	e.SetReadinessCheck(func(ctx context.Context) error {
		// Find a Node adapter to verify adapter health
		adapters := e.AdaptersByType("Node")
		if len(adapters) == 0 {
			return fmt.Errorf("readiness check failed: no Node adapters available")
		}
		// Use first adapter and try to list from first scope
		adapter := adapters[0]
		scopes := adapter.Scopes()
		if len(scopes) == 0 {
			return fmt.Errorf("readiness check failed: no scopes available for Node adapter")
		}
		listableAdapter, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			return fmt.Errorf("readiness check failed: Node adapter is not listable")
		}
		_, err := listableAdapter.List(ctx, scopes[0], true)
		if err != nil {
			return fmt.Errorf("readiness check (listing nodes) failed: %w", err)
		}
		return nil
	})

	// Start HTTP server for health checks
	healthCheckPort := viper.GetInt("health-check-port")
	e.ServeHealthProbes(healthCheckPort)

	// Create channels for interrupts
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	restart := make(chan watch.Event, 1024)

	// Get the initial starting point
	list, listErr := clientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if listErr != nil {
		// Don't exit - store error and continue with degraded state
		initErr := fmt.Errorf("could not list namespaces: %w", listErr)
		log.WithError(initErr).Error("K8s source initialization failed - pod will stay running with error status")
		e.SetInitError(initErr)
		sentry.CaptureException(initErr)
	}

	var wi watch.Interface
	// Only try to watch if listing succeeded
	if listErr == nil {
		var watchErr error
		wi, watchErr = clientSet.CoreV1().Namespaces().Watch(context.Background(), metav1.ListOptions{
			ResourceVersion: list.ResourceVersion,
		})
		if watchErr != nil {
			// Don't exit - store error and continue with degraded state
			initErr := fmt.Errorf("could not start watching namespaces: %w", watchErr)
			log.WithError(initErr).Error("K8s source initialization failed - pod will stay running with error status")
			e.SetInitError(initErr)
			sentry.CaptureException(initErr)
			// Ensure watch interface is nil for safety
			wi = nil
		}
	}

	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	// Only start watch goroutine if we have a valid watcher
	if wi != nil {
		go func() {
			defer tracing.LogRecoverToReturn(watchCtx, "Namespace watch")

			attempts := 0
			sleep := 1 * time.Second

			for {
				select {
				case event, ok := <-wi.ResultChan():
					if !ok {
						// When the channel is closed then we need to restart the
						// watch. This happens regularly on EKS.
						log.Debug("Namespace watch channel closed, re-subscribing")

						wi, err = watchNamespaces(watchCtx, clientSet)
						// Check for transient network errors
						if err != nil {
							var netErr *net.OpError
							if errors.As(err, &netErr) {
								// Mark a failure
								attempts++

								// If we have had less than 3 failures then retry
								if attempts < 4 {
									// The watch interface will be nil if we
									// couldn't connect, so create a fake watcher
									// that is closed so that we end up in this loop
									// again
									wi = watch.NewFake()
									wi.Stop()

									jitter := time.Duration(rand.Int63n(int64(sleep))) //nolint:gosec // we don't need cryptographically secure randomness here
									sleep = sleep + jitter/2

									log.WithError(err).Errorf("Transient network error, retrying in %v seconds", sleep.String())
									time.Sleep(sleep)
									continue
								}
							}

							sentry.CaptureException(err)
							log.WithError(err).Error("could not list namespaces")

							// Send a fatal event that will kill the main goroutine
							restart <- watch.Event{
								Type: watch.EventType("FATAL"),
							}

							return
						}

						// If it's worked, reset the failure counter
						attempts = 0
					} else {
						// If a watch event is received then we need to restart the
						// engine
						restart <- event
					}
				case <-watchCtx.Done():
					return
				}
			}
		}()
	}

	start := func() error {
		ctx := context.Background()

		// Query all namespaces
		log.Info("Listing namespaces")
		list, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return err
		}

		namespaces := make([]string, len(list.Items))

		for i := range list.Items {
			namespaces[i] = list.Items[i].Name
		}

		log.Infof("got %v namespaces", len(namespaces))

		// Create a shared cache for all adapters in this source
		sharedCache := sdpcache.NewCache(ctx)

		// Create the adapter list
		adapterList := adapters.LoadAllAdapters(clientSet, clusterName, namespaces, sharedCache)

		// Add adapters to the engine
		err = e.AddAdapters(adapterList...)
		if err != nil {
			return err
		}

		// Start the engine
		err = e.Start(ctx)

		return err
	}

	stop := func() error {
		// Stop the engine
		err = e.Stop()
		if err != nil {
			return err
		}

		// Clear the adapters
		e.ClearAdapters()

		return nil
	}

	// Start the service initially (only if no init error from namespace operations)
	if e.GetInitError() == nil {
		startErr := start()
		if startErr != nil {
			// Don't exit - store error and continue with degraded state
			initErr := fmt.Errorf("could not start engine: %w", startErr)
			log.WithError(initErr).Error("K8s source initialization failed - pod will stay running with error status")
			e.SetInitError(initErr)
			sentry.CaptureException(initErr)
		}
	}

	defer func() {
		err := stop()
		if err != nil {
			err = fmt.Errorf("Could not stop engine: %w", err)
			sentry.CaptureException(err)
			log.WithError(err)
		}
	}()

	for {
		select {
		case <-quit:
			log.Info("Stopping engine")

			// Stopping will be handled by deferred stop()

			return 0
		case event := <-restart:
			switch event.Type { //nolint:exhaustive // we on purpose fall through to default
			case "":
				// Discard empty events. After a certain period kubernetes
				// starts sending occasional empty events, I can't work out why,
				// maybe it's to keep the connection open. Either way they don't
				// represent anything and should be discarded
				log.Debug("Discarding empty event")
			case "FATAL":
				// This is a custom event type from permanent watch failures
				// Don't exit - store error and continue in degraded state
				fatalErr := fmt.Errorf("permanent failure in namespace watch after retries")
				log.WithError(fatalErr).Error("K8s namespace watch failed permanently - pod will stay running with error status")
				e.SetInitError(fatalErr)
				sentry.CaptureException(fatalErr)
				// Continue running without watch functionality
			case "MODIFIED":
				log.Debug("Namespace modified, ignoring")
			default:
				stopErr := stop()
				if stopErr != nil {
					sentry.CaptureException(stopErr)
					log.WithError(stopErr).Error("Could not stop engine")
					// Don't exit - continue running in degraded state
					e.SetInitError(fmt.Errorf("could not stop engine: %w", stopErr))
				} else {
					startErr := start()
					if startErr != nil {
						// Don't exit - store error and continue with degraded state
						initErr := fmt.Errorf("could not restart engine: %w", startErr)
						log.WithError(initErr).Error("K8s source restart failed - pod will stay running with error status")
						e.SetInitError(initErr)
						sentry.CaptureException(initErr)
					} else {
						// Restart succeeded, clear any previous init error
						e.SetInitError(nil)
						log.Info("K8s source restarted successfully")
					}
				}
			}
		}
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Watches k8s namespaces from the current state, sending new events for each change
func watchNamespaces(ctx context.Context, clientSet *kubernetes.Clientset) (watch.Interface, error) {
	// Get the initial starting point
	list, err := clientSet.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Watch namespaces from here
	wi, err := clientSet.CoreV1().Namespaces().Watch(ctx, metav1.ListOptions{
		ResourceVersion: list.ResourceVersion,
	})
	if err != nil {
		return nil, err
	}

	return wi, nil
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	var logLevel string

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/srcman/config/k8s-source.yaml", "config file path")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info", "Set the log level. Valid values: panic, fatal, error, warn, info, debug, trace")
	rootCmd.PersistentFlags().Int("health-check-port", 8080, "The port on which to serve health check endpoints (/healthz/alive, /healthz/ready, /healthz)")

	// engine flags
	discovery.AddEngineFlags(rootCmd)

	// source-specific flags
	rootCmd.PersistentFlags().String("kubeconfig", "", "Path to the kubeconfig file containing cluster details. If this is blank, the in-cluster config will be used")
	rootCmd.PersistentFlags().Float32("rate-limit-qps", 10.0, "The maximum sustained queries per second from this source to the kubernetes API")
	rootCmd.PersistentFlags().Int("rate-limit-burst", 30, "The maximum burst of queries from this source to the kubernetes API")
	rootCmd.PersistentFlags().String("cluster-name", "", "The descriptive name of the cluster this source is running on. If this is blank, the hostname will be used from the Kube config")

	// tracing
	rootCmd.PersistentFlags().String("honeycomb-api-key", "", "If specified, configures opentelemetry libraries to submit traces to honeycomb")
	rootCmd.PersistentFlags().String("sentry-dsn", "", "If specified, configures sentry libraries to capture errors")
	rootCmd.PersistentFlags().String("run-mode", "release", "Set the run mode for this service, 'release', 'debug' or 'test'. Defaults to 'release'.")
	rootCmd.PersistentFlags().Bool("json-log", true, "Set to false to emit logs as text for easier reading in development.")
	cobra.CheckErr(viper.BindEnv("json-log", "K8S_SOURCE_JSON_LOG", "JSON_LOG")) // fallback to global config

	// Bind these to viper
	cobra.CheckErr(viper.BindPFlags(rootCmd.PersistentFlags()))

	// Run this before we do anything to set up the loglevel
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if lvl, err := log.ParseLevel(logLevel); err == nil {
			log.SetLevel(lvl)
		} else {
			log.SetLevel(log.InfoLevel)
		}

		log.AddHook(TerminationLogHook{})
		log.AddHook(otellogrus.NewHook(otellogrus.WithLevels(
			log.AllLevels[:log.GetLevel()+1]...,
		)))

		// Bind flags that haven't been set to the values from viper of we have them
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			// Bind the flag to viper only if it has a non-empty default
			if f.DefValue != "" || f.Changed {
				err := viper.BindPFlag(f.Name, f)
				if err != nil {
					log.WithError(err).Errorf("Could not bind flag %s to viper", f.Name)
				}
			}
		})

		if viper.GetBool("json-log") {
			logging.ConfigureLogrusJSON(log.StandardLogger())
		}

		if err := tracing.InitTracerWithUpstreams("k8s-source", viper.GetString("honeycomb-api-key"), viper.GetString("sentry-dsn")); err != nil {
			log.WithError(err).Error("could not init tracer")
			return fmt.Errorf("could not init tracer: %w", err)
		}
		return nil
	}

	// shut down tracing at the end of the process
	rootCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		tracing.ShutdownTracer(context.Background())
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigFile(cfgFile)

	replacer := strings.NewReplacer("-", "_")

	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Infof("Using config file: %v", viper.ConfigFileUsed())
	}
}

// TerminationLogHook A hook that logs fatal errors to the termination log
type TerminationLogHook struct{}

func (t TerminationLogHook) Levels() []log.Level {
	return []log.Level{log.FatalLevel}
}

func (t TerminationLogHook) Fire(e *log.Entry) error {
	tLog, err := os.OpenFile("/dev/termination-log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	var message string

	message = e.Message

	for k, v := range e.Data {
		message = fmt.Sprintf("%v %v=%v", message, k, v)
	}

	_, err = tLog.WriteString(message)

	return err
}
