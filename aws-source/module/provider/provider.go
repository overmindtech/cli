package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/overmindtech/cli/go/auth"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/go/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/oauth2"
)

var _ provider.Provider = (*overmindProvider)(nil)

type overmindProvider struct {
	version string
}

type overmindProviderModel struct {
	AppURL types.String `tfsdk:"app_url"`
	APIKey types.String `tfsdk:"api_key"`
}

func NewProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &overmindProvider{version: version}
	}
}

func (p *overmindProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "overmind"
	resp.Version = p.version
}

func (p *overmindProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Overmind provider manages infrastructure sources via the Overmind API. " +
			"Configuration is read from the OVERMIND_API_KEY and OVERMIND_APP_URL environment variables by default.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Overmind API key. Can also be set via the OVERMIND_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"app_url": schema.StringAttribute{
				Description: "Overmind application URL (e.g. https://app.overmind.tech). " +
					"Can also be set via the OVERMIND_APP_URL environment variable.",
				Optional: true,
			},
		},
	}
}

func (p *overmindProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "Provider Configure")
	defer span.End()

	var config overmindProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("OVERMIND_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	appURL := os.Getenv("OVERMIND_APP_URL")
	if !config.AppURL.IsNull() {
		appURL = config.AppURL.ValueString()
	}
	if appURL == "" {
		appURL = "https://app.overmind.tech"
	}

	span.SetAttributes(attribute.String("ovm.provider.appUrl", appURL))

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"An Overmind API key must be provided via the api_key provider attribute or the OVERMIND_API_KEY environment variable.",
		)
		span.SetStatus(codes.Error, "missing API key")
		return
	}

	oi, err := sdp.NewOvermindInstance(ctx, appURL)
	if err != nil {
		resp.Diagnostics.AddError("Failed to resolve Overmind instance",
			fmt.Sprintf("Could not resolve instance data from %s: %s", appURL, err))
		span.RecordError(err)
		span.SetStatus(codes.Error, "instance resolution failed")
		return
	}

	apiURL := oi.ApiUrl.String()
	span.SetAttributes(attribute.String("ovm.provider.apiUrl", apiURL))

	tokenSource := auth.NewAPIKeyTokenSource(apiKey, apiURL)
	httpClient := tracing.HTTPClient()
	httpClient.Transport = &oauth2.Transport{
		Source: tokenSource,
		Base:   httpClient.Transport,
	}

	mgmtClient := sdpconnect.NewManagementServiceClient(httpClient, apiURL)

	resp.DataSourceData = mgmtClient
	resp.ResourceData = mgmtClient
}

func (p *overmindProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAWSSourceResource,
	}
}

func (p *overmindProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAWSExternalIdDataSource,
	}
}
