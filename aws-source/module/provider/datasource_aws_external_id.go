package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/go/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

var _ datasource.DataSource = (*awsExternalIdDataSource)(nil)

type awsExternalIdDataSource struct {
	mgmt sdpconnect.ManagementServiceClient
}

type awsExternalIdDataSourceModel struct {
	ExternalID types.String `tfsdk:"external_id"`
}

func NewAWSExternalIdDataSource() datasource.DataSource {
	return &awsExternalIdDataSource{}
}

func (d *awsExternalIdDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_external_id"
}

func (d *awsExternalIdDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Retrieves the stable AWS STS external ID for the current Overmind account. " +
			"Use this to configure the trust policy on an IAM role before creating the source.",
		Attributes: map[string]dsschema.Attribute{
			"external_id": dsschema.StringAttribute{
				Description: "AWS STS external ID, stable per Overmind account.",
				Computed:    true,
			},
		},
	}
}

func (d *awsExternalIdDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	mgmt, ok := req.ProviderData.(sdpconnect.ManagementServiceClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected sdpconnect.ManagementServiceClient, got %T", req.ProviderData))
		return
	}
	d.mgmt = mgmt
}

func (d *awsExternalIdDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSExternalId Read")
	defer span.End()

	extIDResp, err := d.mgmt.GetOrCreateAWSExternalId(ctx,
		connect.NewRequest(&sdp.GetOrCreateAWSExternalIdRequest{}))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS external ID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "GetOrCreateAWSExternalId failed")
		return
	}

	externalID := extIDResp.Msg.GetAwsExternalId()
	span.SetAttributes(attribute.String("ovm.externalId", externalID))

	resp.Diagnostics.Append(resp.State.Set(ctx, &awsExternalIdDataSourceModel{
		ExternalID: types.StringValue(externalID),
	})...)
}
