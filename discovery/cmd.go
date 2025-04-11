package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpconnect"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

const defaultApp = "https://app.overmind.tech"

func AddEngineFlags(command *cobra.Command) {
	command.PersistentFlags().String("source-name", "", "The name of the source")
	cobra.CheckErr(viper.BindEnv("source-name", "SOURCE_NAME"))
	command.PersistentFlags().String("source-uuid", "", "The UUID of the source, is this is blank it will be auto-generated. This is used in heartbeats and shouldn't be supplied usually")
	cobra.CheckErr(viper.BindEnv("source-uuid", "SOURCE_UUID"))
	command.PersistentFlags().String("source-access-token", "", "The access token to use to authenticate the source for managed sources")
	cobra.CheckErr(viper.BindEnv("source-access-token", "SOURCE_ACCESS_TOKEN"))
	command.PersistentFlags().String("source-access-token-type", "", "The type of token to use to authenticate the source for managed sources")
	cobra.CheckErr(viper.BindEnv("source-access-token-type", "SOURCE_ACCESS_TOKEN_TYPE"))

	command.PersistentFlags().String("api-server-service-host", "", "The host of the API server service, only if the source is managed by Overmind")
	cobra.CheckErr(viper.BindEnv("api-server-service-host", "API_SERVER_SERVICE_HOST"))
	command.PersistentFlags().String("api-server-service-port", "", "The port of the API server service, only if the source is managed by Overmind")
	cobra.CheckErr(viper.BindEnv("api-server-service-port", "API_SERVER_SERVICE_PORT"))
	command.PersistentFlags().String("nats-service-host", "", "The host of the NATS service, only if the source is managed by Overmind")
	cobra.CheckErr(viper.BindEnv("nats-service-host", "NATS_SERVICE_HOST"))
	command.PersistentFlags().String("nats-service-port", "", "The port of the NATS service, only if the source is managed by Overmind")
	cobra.CheckErr(viper.BindEnv("nats-service-port", "NATS_SERVICE_PORT"))

	command.PersistentFlags().Bool("overmind-managed-source", false, "If you are running the source yourself or if it is managed by Overmind")
	_ = command.Flags().MarkHidden("overmind-managed-source")
	cobra.CheckErr(viper.BindEnv("overmind-managed-source", "OVERMIND_MANAGED_SOURCE"))

	command.PersistentFlags().String("app", defaultApp, "The URL of the Overmind app to use")
	cobra.CheckErr(viper.BindEnv("app", "APP"))
	command.PersistentFlags().String("api-key", "", "The API key to use to authenticate to the Overmind API")
	cobra.CheckErr(viper.BindEnv("api-key", "OVM_API_KEY", "API_KEY"))

	command.PersistentFlags().String("nats-connection-name", "", "The name that the source should use to connect to NATS")
	cobra.CheckErr(viper.BindEnv("nats-connection-name", "NATS_CONNECTION_NAME"))
	command.PersistentFlags().Int("nats-connection-timeout", 10, "The timeout for connecting to NATS")
	cobra.CheckErr(viper.BindEnv("nats-connection-timeout", "NATS_CONNECTION_TIMEOUT"))

	command.PersistentFlags().Int("max-parallel", 0, "The maximum number of parallel executions")
	cobra.CheckErr(viper.BindEnv("max-parallel", "MAX_PARALLEL"))
}

func EngineConfigFromViper(engineType, version string) (*EngineConfig, error) {
	var sourceName string
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error getting hostname: %w", err)
	}

	if viper.GetString("source-name") == "" {
		sourceName = fmt.Sprintf("%s-%s", engineType, hostname)
	} else {
		sourceName = viper.GetString("source-name")
	}

	sourceUUIDString := viper.GetString("source-uuid")
	var sourceUUID uuid.UUID
	if sourceUUIDString == "" {
		sourceUUID = uuid.New()
	} else {
		var err error
		sourceUUID, err = uuid.Parse(sourceUUIDString)
		if err != nil {
			return nil, fmt.Errorf("error parsing source-uuid: %w", err)
		}
	}

	var managedSource sdp.SourceManaged
	if viper.GetBool("overmind-managed-source") {
		managedSource = sdp.SourceManaged_MANAGED
	} else {
		managedSource = sdp.SourceManaged_LOCAL
	}

	var apiServerURL string
	var natsServerURL string
	appURL := viper.GetString("app")
	if managedSource == sdp.SourceManaged_MANAGED {
		apiServerHost := viper.GetString("api-server-service-host")
		apiServerPort := viper.GetString("api-server-service-port")
		if apiServerHost == "" || apiServerPort == "" {
			return nil, errors.New("API_SERVER_SERVICE_HOST and API_SERVER_SERVICE_PORT (provided by k8s) must be set for managed sources")
		}
		apiServerURL = net.JoinHostPort(apiServerHost, apiServerPort)
		if apiServerPort == "443" {
			apiServerURL = "https://" + apiServerURL
		} else {
			apiServerURL = "http://" + apiServerURL
		}

		natsServerHost := viper.GetString("nats-service-host")
		natsServerPort := viper.GetString("nats-service-port")
		if natsServerHost == "" || natsServerPort == "" {
			return nil, errors.New("NATS_SERVICE_HOST and NATS_SERVICE_PORT (provided by k8s) must be set for managed sources")
		}
		natsServerURL = net.JoinHostPort(natsServerHost, natsServerPort)
		natsServerURL = "nats://" + natsServerURL
	} else {
		// look up the api server url from the app url
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		oi, err := sdp.NewOvermindInstance(ctx, appURL)
		if err != nil {
			err = fmt.Errorf("Could not determine Overmind instance URLs from app URL %s: %w", appURL, err)
			return nil, err
		}
		apiServerURL = oi.ApiUrl.String()
		natsServerURL = oi.NatsUrl.String()
	}

	// setup natsOptions
	var natsConnectionName string
	if viper.GetString("nats-connection-name") == "" {
		natsConnectionName = hostname
	}
	natsOptions := auth.NATSOptions{
		NumRetries:        -1,
		RetryDelay:        5 * time.Second,
		Servers:           []string{natsServerURL},
		ConnectionName:    natsConnectionName,
		ConnectionTimeout: time.Duration(viper.GetInt("nats-connection-timeout")) * time.Second,
		MaxReconnects:     -1,
		ReconnectWait:     1 * time.Second,
		ReconnectJitter:   1 * time.Second,
	}

	allow := os.Getenv("ALLOW_UNAUTHENTICATED")
	allowUnauthenticated := allow == "true"

	// order of precedence is:
	// unauthenticated overrides everything  # used for local development
	// if managed source, we expect a token
	// if local source, we expect an api key

	if allowUnauthenticated {
		log.Warn("Using unauthenticated mode as ALLOW_UNAUTHENTICATED is set")
	} else {
		if viper.GetBool("overmind-managed-source") {
			log.Info("Running source in managed mode")
			// If managed source, we expect a token
			if viper.GetString("source-access-token") == "" {
				return nil, errors.New("source-access-token must be set for managed sources")
			}
		} else if viper.GetString("api-key") == "" {
			return nil, errors.New("api-key must be set for local sources")
		}
	}

	maxParallelExecutions := viper.GetInt("max-parallel")
	if maxParallelExecutions == 0 {
		maxParallelExecutions = runtime.NumCPU()
	}

	return &EngineConfig{
		EngineType:            engineType,
		Version:               version,
		SourceName:            sourceName,
		SourceUUID:            sourceUUID,
		OvermindManagedSource: managedSource,
		SourceAccessToken:     viper.GetString("source-access-token"),
		SourceAccessTokenType: viper.GetString("source-access-token-type"),
		App:                   appURL,
		APIServerURL:          apiServerURL,
		ApiKey:                viper.GetString("api-key"),
		NATSOptions:           &natsOptions,
		Unauthenticated:       allowUnauthenticated,
		MaxParallelExecutions: maxParallelExecutions,
	}, nil
}

// MapFromEngineConfig Returns the config as a map
func MapFromEngineConfig(ec *EngineConfig) map[string]any {
	var apiKeyClientSecret string
	if ec.ApiKey != "" {
		apiKeyClientSecret = "[REDACTED]"
	}
	var sourceAccessToken string
	if ec.SourceAccessToken != "" {
		sourceAccessToken = "[REDACTED]"
	}

	return map[string]interface{}{
		"engine-type":              ec.EngineType,
		"version":                  ec.Version,
		"source-name":              ec.SourceName,
		"source-uuid":              ec.SourceUUID,
		"source-access-token":      sourceAccessToken,
		"source-access-token-type": ec.SourceAccessTokenType,
		"managed-source":           ec.OvermindManagedSource,
		"app":                      ec.App,
		"api-key":                  apiKeyClientSecret,
		"api-server-url":           ec.APIServerURL,
		"max-parallel-executions":  ec.MaxParallelExecutions,
		"nats-servers":             ec.NATSOptions.Servers,
		"nats-connection-name":     ec.NATSOptions.ConnectionName,
		"nats-connection-timeout":  ec.NATSConnectionTimeout,
		"nats-queue-name":          ec.NATSQueueName,
		"unauthenticated":          ec.Unauthenticated,
	}
}

// CreateClients we need to have some checks, as it is called by the cli tool
func (ec *EngineConfig) CreateClients() error {
	// If we are running in unauthenticated mode then do nothing here
	if ec.Unauthenticated {
		log.Warn("Using unauthenticated NATS as ALLOW_UNAUTHENTICATED is set")
		log.WithFields(MapFromEngineConfig(ec)).Info("Engine config")
		return nil
	}

	switch ec.OvermindManagedSource {
	case sdp.SourceManaged_LOCAL:
		log.Info("Using API Key for authentication, heartbeats will be sent")
		tokenClient, err := auth.NewAPIKeyClient(ec.APIServerURL, ec.ApiKey)
		if err != nil {
			err = fmt.Errorf("error creating API key client %w", err)
			return err
		}
		tokenSource := auth.NewAPIKeyTokenSource(ec.ApiKey, ec.APIServerURL)
		transport := oauth2.Transport{
			Source: tokenSource,
			Base:   http.DefaultTransport,
		}
		authenticatedClient := http.Client{
			Transport: otelhttp.NewTransport(&transport),
		}
		heartbeatOptions := HeartbeatOptions{
			ManagementClient: sdpconnect.NewManagementServiceClient(
				&authenticatedClient,
				ec.APIServerURL,
			),
			Frequency: time.Second * 30,
		}
		ec.HeartbeatOptions = &heartbeatOptions
		ec.NATSOptions.TokenClient = tokenClient
		// lets print out the config
		log.WithFields(MapFromEngineConfig(ec)).Info("Engine config")
		return nil
	case sdp.SourceManaged_MANAGED:
		log.Info("Using static token for authentication, heartbeats will be sent")
		tokenClient, err := auth.NewStaticTokenClient(ec.APIServerURL, ec.SourceAccessToken, ec.SourceAccessTokenType)
		if err != nil {
			err = fmt.Errorf("error creating static token client %w", err)
			sentry.CaptureException(err)
			return err
		}
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: ec.SourceAccessToken,
			TokenType:   ec.SourceAccessTokenType,
		})
		transport := oauth2.Transport{
			Source: tokenSource,
			Base:   http.DefaultTransport,
		}
		authenticatedClient := http.Client{
			Transport: otelhttp.NewTransport(&transport),
		}
		heartbeatOptions := HeartbeatOptions{
			ManagementClient: sdpconnect.NewManagementServiceClient(
				&authenticatedClient,
				ec.APIServerURL,
			),
			Frequency: time.Second * 30,
		}
		ec.NATSOptions.TokenClient = tokenClient
		ec.HeartbeatOptions = &heartbeatOptions
		// lets print out the config
		log.WithFields(MapFromEngineConfig(ec)).Info("Engine config")
		return nil
	}

	err := fmt.Errorf("unable to setup authentication. Please check your configuration %v", ec)
	return err
}
