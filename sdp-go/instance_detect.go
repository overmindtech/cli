package sdp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Information about a particular instance of Overmind. This is used to
// determine where to send requests, how to authenticate etc.
type OvermindInstance struct {
	FrontendUrl *url.URL
	ApiUrl      *url.URL
	NatsUrl     *url.URL
	Audience    string
	Auth0Domain string
	CLIClientID string
}

// GatewayUrl returns the URL for the gateway for this instance.
func (oi OvermindInstance) GatewayUrl() string {
	return fmt.Sprintf("%v/api/gateway", oi.ApiUrl.String())
}

func (oi OvermindInstance) String() string {
	return fmt.Sprintf("Frontend: %v, API: %v, Nats: %v, Audience: %v", oi.FrontendUrl, oi.ApiUrl, oi.NatsUrl, oi.Audience)
}

type instanceData struct {
	Api         string `json:"api_url"`
	Nats        string `json:"nats_url"`
	Aud         string `json:"aud"`
	Auth0Domain string `json:"auth0_domain"`
	CLIClientID string `json:"auth0_cli_client_id"`
}

// NewOvermindInstance creates a new OvermindInstance from the given app URL
// with all URLs filled in, or an error. The app URL should be the URL of the
// frontend of the Overmind instance. e.g. https://app.overmind.tech
func NewOvermindInstance(ctx context.Context, app string) (OvermindInstance, error) {
	var instance OvermindInstance
	var err error

	instance.FrontendUrl, err = url.Parse(app)
	if err != nil {
		return instance, fmt.Errorf("invalid app value '%v', error: %w", app, err)
	}

	// Get the instance data
	instanceDataUrl := fmt.Sprintf("%v/api/public/instance-data", instance.FrontendUrl)
	req, err := http.NewRequest(http.MethodGet, instanceDataUrl, nil)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("could not initialize instance-data fetch: %w", err)
	}

	req = req.WithContext(ctx)
	res, err := otelhttp.DefaultClient.Do(req)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("could not fetch instance-data: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return OvermindInstance{}, fmt.Errorf("instance-data fetch returned non-200 status: %v", res.StatusCode)
	}

	defer res.Body.Close()
	data := instanceData{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("could not parse instance-data: %w", err)
	}

	instance.ApiUrl, err = url.Parse(data.Api)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("invalid api_url value '%v' in instance-data, error: %w", data.Api, err)
	}
	instance.NatsUrl, err = url.Parse(data.Nats)
	if err != nil {
		return OvermindInstance{}, fmt.Errorf("invalid nats_url value '%v' in instance-data, error: %w", data.Nats, err)
	}

	instance.Audience = data.Aud
	instance.CLIClientID = data.CLIClientID
	instance.Auth0Domain = data.Auth0Domain

	return instance, nil
}
