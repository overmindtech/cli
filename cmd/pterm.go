package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/overmindtech/pterm"
	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/tracing"
	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

func PTermSetup() {
	pterm.Success.Prefix.Text = OkSymbol()
	pterm.Warning.Prefix.Text = UnknownSymbol()
	pterm.Error.Prefix.Text = ErrSymbol()

	pterm.DefaultMultiPrinter.UpdateDelay = 80 * time.Millisecond

	pterm.DefaultSpinner.Sequence = []string{" ⠋ ", " ⠙ ", " ⠹ ", " ⠸ ", " ⠼ ", " ⠴ ", " ⠦ ", " ⠧ ", " ⠇ ", " ⠏ "}
	pterm.DefaultSpinner.Delay = 80 * time.Millisecond

	// ensure that only error messages are printed to the console,
	// disrupting bubbletea rendering (and potentially getting overwritten).
	// Otherwise, when TEABUG is set, log to a file.
	if len(os.Getenv("TEABUG")) > 0 {
		f, err := os.OpenFile("teabug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600) //nolint:gomnd
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		// leave the log file open until the very last moment, so we capture everything
		// defer f.Close()
		log.SetOutput(f)
		formatter := new(log.TextFormatter)
		formatter.DisableTimestamp = false
		log.SetFormatter(formatter)
		viper.Set("log", "trace")
		log.SetLevel(log.TraceLevel)
	} else {
		// avoid log messages from sources and others to interrupt bubbletea rendering
		viper.Set("log", "fatal")
		log.SetLevel(log.FatalLevel)
	}
}

func StartSources(ctx context.Context, cmd *cobra.Command, args []string) (context.Context, sdp.OvermindInstance, *oauth2.Token, func(), error) {
	multi := pterm.DefaultMultiPrinter
	_, _ = multi.Start()
	defer func() {
		_, _ = multi.Stop()
	}()

	ctx, oi, token, err := login(ctx, cmd, []string{"explore:read", "changes:write", "config:write", "request:receive", "api:read", "sources:read"}, multi.NewWriter())
	if err != nil {
		return ctx, sdp.OvermindInstance{}, nil, nil, err
	}

	// use only-use-managed-sources flag to determine if we should start local sources
	if viper.GetBool("only-use-managed-sources") {
		return ctx, oi, token, nil, nil
	}
	cleanup, err := StartLocalSources(ctx, oi, token, args, false)
	if err != nil {
		return ctx, sdp.OvermindInstance{}, nil, nil, err
	}

	return ctx, oi, token, cleanup, nil
}

// start revlink warmup in the background
func RunRevlinkWarmup(ctx context.Context, oi sdp.OvermindInstance, postPlanPrinter *atomic.Pointer[pterm.MultiPrinter], args []string) *pool.ErrorPool {
	p := pool.New().WithErrors()
	p.Go(func() error {
		ctx, span := tracing.Tracer().Start(ctx, "revlink warmup")
		defer span.End()

		client := AuthenticatedManagementClient(ctx, oi)
		stream, err := client.RevlinkWarmup(ctx, &connect.Request[sdp.RevlinkWarmupRequest]{
			Msg: &sdp.RevlinkWarmupRequest{},
		})
		if err != nil {
			return fmt.Errorf("error warming up revlink: %w", err)
		}

		// this will get set once the terminal is available
		var spinner *pterm.SpinnerPrinter
		for stream.Receive() {
			msg := stream.Msg()

			if spinner == nil {
				multi := postPlanPrinter.Load()
				if multi != nil {
					// start the spinner in the background, now that a multi
					// printer is available
					spinner, _ = pterm.DefaultSpinner.WithWriter(multi.NewWriter()).Start("Discovering and linking all resources")
				}
			}

			// only update the spinner if we have access to the terminal
			if spinner != nil {
				items := msg.GetItems()
				edges := msg.GetEdges()
				if items+edges > 0 {
					spinner.UpdateText(fmt.Sprintf("Discovering and linking all resources: %v (%v items, %v edges)", msg.GetStatus(), items, edges))
				} else {
					spinner.UpdateText(fmt.Sprintf("Discovering and linking all resources: %v", msg.GetStatus()))
				}
			}
		}

		err = stream.Err()
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			if spinner != nil {
				spinner.Fail(fmt.Sprintf("Error warming up revlink: %v", err))
			}
			return fmt.Errorf("error warming up revlink: %w", err)
		}

		if spinner != nil {
			spinner.Success("Discovered and linked all resources")
		} else {
			// if we didn't have a spinner, print a success message
			// this can happen if the terminal is not available, or if the revlink warmup is very fast
			pterm.Success.Println("Discovered and linked all resources")
		}

		return nil
	})

	return p
}

func RunPlan(ctx context.Context, args []string) error {
	c := exec.CommandContext(ctx, "terraform", args...)

	// remove go's default process cancel behaviour, so that terraform has a
	// chance to gracefully shutdown when ^C is pressed. Otherwise the
	// process would get killed immediately and leave locks lingering behind
	c.Cancel = func() error {
		return nil
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	_, span := tracing.Tracer().Start(ctx, "terraform plan")
	defer span.End()

	log.WithField("args", c.Args).Debug("running terraform plan")

	pterm.Println("Running terraform plan: " + strings.Join(c.Args, " "))

	err := c.Run()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to run terraform plan: %w", err)
	}

	return nil
}

func RunApply(ctx context.Context, args []string) error {
	c := exec.CommandContext(ctx, "terraform", args...)

	// remove go's default process cancel behaviour, so that terraform has a
	// chance to gracefully shutdown when ^C is pressed. Otherwise the
	// process would get killed immediately and leave locks lingering behind
	c.Cancel = func() error {
		return nil
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	_, span := tracing.Tracer().Start(ctx, "terraform apply")
	defer span.End()

	log.WithField("args", c.Args).Debug("running terraform apply")

	pterm.Println("Running terraform apply: " + strings.Join(c.Args, " "))

	err := c.Run()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to run terraform apply: %w", err)
	}

	return nil
}

func snapshotDetail(state string, items, edges uint32) string {
	itemStr := ""
	switch items {
	case 0:
		itemStr = "0 items"
	case 1:
		itemStr = "1 item"
	default:
		itemStr = fmt.Sprintf("%d items", items)
	}

	edgeStr := ""
	switch edges {
	case 0:
		edgeStr = "0 edges"
	case 1:
		edgeStr = "1 edge"
	default:
		edgeStr = fmt.Sprintf("%d edges", edges)
	}

	detailStr := state
	if itemStr != "" || edgeStr != "" {
		detailStr = fmt.Sprintf("%s (%s, %s)", state, itemStr, edgeStr)
	}
	return detailStr
}

func natsOptions(ctx context.Context, oi sdp.OvermindInstance, token *oauth2.Token) auth.NATSOptions {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	natsNamePrefix := "overmind-cli"

	openapiUrl := *oi.ApiUrl
	openapiUrl.Path = "/api"
	tokenClient := auth.NewOAuthTokenClientWithContext(
		ctx,
		openapiUrl.String(),
		"",
		oauth2.StaticTokenSource(token),
	)

	return auth.NATSOptions{
		NumRetries:        3,
		RetryDelay:        1 * time.Second,
		Servers:           []string{oi.NatsUrl.String()},
		ConnectionName:    fmt.Sprintf("%v.%v", natsNamePrefix, hostname),
		ConnectionTimeout: (10 * time.Second), // TODO: Make configurable
		MaxReconnects:     -1,
		ReconnectWait:     1 * time.Second,
		ReconnectJitter:   1 * time.Second,
		TokenClient:       tokenClient,
	}
}

func heartbeatOptions(oi sdp.OvermindInstance, token *oauth2.Token) *discovery.HeartbeatOptions {
	tokenSource := oauth2.StaticTokenSource(token)

	transport := oauth2.Transport{
		Source: tokenSource,
		Base:   http.DefaultTransport,
	}
	authenticatedClient := http.Client{
		Transport: otelhttp.NewTransport(&transport),
	}

	return &discovery.HeartbeatOptions{
		ManagementClient: sdpconnect.NewManagementServiceClient(
			&authenticatedClient,
			oi.ApiUrl.String(),
		),
		Frequency: time.Second * 10,
	}
}

func HasScopesFlexible(token *oauth2.Token, requiredScopes []string) (bool, string, error) {
	if token == nil {
		return false, "", errors.New("HasScopesFlexible: token is nil")
	}

	claims, err := extractClaims(token.AccessToken)
	if err != nil {
		return false, "", fmt.Errorf("error extracting claims from token: %w", err)
	}

	for _, scope := range requiredScopes {
		if !claims.HasScope(scope) {
			// If they don't have the *exact* scope, check to see if they have
			// write access to the same service
			sections := strings.Split(scope, ":")
			var hasWriteInstead bool

			if len(sections) == 2 {
				service, action := sections[0], sections[1]

				if action == "read" {
					hasWriteInstead = claims.HasScope(fmt.Sprintf("%v:write", service))
				}
			}

			if !hasWriteInstead {
				return false, scope, nil
			}
		}
	}

	return true, "", nil
}

// extracts custom claims from a JWT token. Note that this does not verify the
// signature of the token, it just extracts the claims from the payload
func extractClaims(token string) (*auth.CustomClaims, error) {
	// We aren't interested in checking the signature of the token since
	// the server will do that. All we need to do is make sure it
	// contains the right scopes. Therefore we just parse the payload
	// directly
	sections := strings.Split(token, ".")
	if len(sections) != 3 {
		return nil, errors.New("token is not a JWT")
	}

	// Decode the payload
	decodedPayload, err := base64.RawURLEncoding.DecodeString(sections[1])
	if err != nil {
		return nil, fmt.Errorf("error decoding token payload: %w", err)
	}

	// Parse the payload
	claims := new(auth.CustomClaims)
	err = json.Unmarshal(decodedPayload, claims)
	if err != nil {
		return nil, fmt.Errorf("error parsing token payload: %w", err)
	}

	return claims, nil
}
