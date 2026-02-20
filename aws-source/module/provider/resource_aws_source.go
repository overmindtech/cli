package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"github.com/overmindtech/cli/go/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	_ resource.Resource                = (*awsSourceResource)(nil)
	_ resource.ResourceWithImportState = (*awsSourceResource)(nil)
)

type awsSourceResource struct {
	mgmt sdpconnect.ManagementServiceClient
}

type awsSourceResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	AWSRoleARN types.String `tfsdk:"aws_role_arn"`
	AWSRegions types.List   `tfsdk:"aws_regions"`
	ExternalID types.String `tfsdk:"external_id"`
}

func NewAWSSourceResource() resource.Resource {
	return &awsSourceResource{}
}

func (r *awsSourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_aws_source"
}

func (r *awsSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Overmind AWS infrastructure source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Source UUID assigned by the Overmind API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for this source.",
				Required:    true,
			},
			"aws_role_arn": schema.StringAttribute{
				Description: "ARN of the IAM role to assume in the customer's AWS account.",
				Required:    true,
			},
			"aws_regions": schema.ListAttribute{
				Description: "AWS regions this source should discover resources in.",
				Required:    true,
				ElementType: types.StringType,
			},
			"external_id": schema.StringAttribute{
				Description: "AWS STS external ID for the IAM trust policy, stable per Overmind account.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *awsSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	mgmt, ok := req.ProviderData.(sdpconnect.ManagementServiceClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected sdpconnect.ManagementServiceClient, got %T", req.ProviderData))
		return
	}
	r.mgmt = mgmt
}

func (r *awsSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSSource Create")
	defer span.End()

	var plan awsSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	span.SetAttributes(
		attribute.String("ovm.source.name", plan.Name.ValueString()),
		attribute.String("ovm.source.roleArn", plan.AWSRoleARN.ValueString()),
	)

	extIDResp, err := r.mgmt.GetOrCreateAWSExternalId(ctx,
		connect.NewRequest(&sdp.GetOrCreateAWSExternalIdRequest{}))
	if err != nil {
		resp.Diagnostics.AddError("Failed to get AWS external ID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "GetOrCreateAWSExternalId failed")
		return
	}
	externalID := extIDResp.Msg.GetAwsExternalId()

	regions, diags := regionsFromList(ctx, plan.AWSRegions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sourceConfig, err := structpb.NewStruct(map[string]any{
		"aws-access-strategy": "external-id",
		"aws-external-id":     externalID,
		"aws-target-role-arn": plan.AWSRoleARN.ValueString(),
		"aws-regions":         strings.Join(regions, ","),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to build source config", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "config build failed")
		return
	}

	createResp, err := r.mgmt.CreateSource(ctx, connect.NewRequest(&sdp.CreateSourceRequest{
		Properties: &sdp.SourceProperties{
			DescriptiveName: plan.Name.ValueString(),
			Type:            "aws",
			Config:          sourceConfig,
		},
	}))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create source", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "CreateSource failed")
		return
	}

	source := createResp.Msg.GetSource()
	sourceUUID, err := uuid.FromBytes(source.GetMetadata().GetUUID())
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse source UUID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "UUID parse failed")
		return
	}

	plan.ID = types.StringValue(sourceUUID.String())
	plan.ExternalID = types.StringValue(externalID)

	span.SetAttributes(attribute.String("ovm.source.id", sourceUUID.String()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *awsSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSSource Read")
	defer span.End()

	var state awsSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	span.SetAttributes(attribute.String("ovm.source.id", state.ID.ValueString()))

	uuidBytes, err := uuidToBytes(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid source ID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid UUID")
		return
	}

	getResp, err := r.mgmt.GetSource(ctx, connect.NewRequest(&sdp.GetSourceRequest{
		UUID: uuidBytes,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			span.SetAttributes(attribute.Bool("ovm.source.removed", true))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read source", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "GetSource failed")
		return
	}

	source := getResp.Msg.GetSource()
	props := source.GetProperties()

	state.Name = types.StringValue(props.GetDescriptiveName())

	if cfg := props.GetConfig(); cfg != nil {
		fields := cfg.GetFields()
		if v, ok := fields["aws-target-role-arn"]; ok {
			state.AWSRoleARN = types.StringValue(v.GetStringValue())
		}
		if v, ok := fields["aws-regions"]; ok {
			regionStr := v.GetStringValue()
			regionVals := splitNonEmpty(regionStr, ",")
			listVal, diags := types.ListValueFrom(ctx, types.StringType, regionVals)
			resp.Diagnostics.Append(diags...)
			state.AWSRegions = listVal
		}
		if v, ok := fields["aws-external-id"]; ok {
			state.ExternalID = types.StringValue(v.GetStringValue())
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *awsSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSSource Update")
	defer span.End()

	var plan awsSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state awsSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	span.SetAttributes(
		attribute.String("ovm.source.id", state.ID.ValueString()),
		attribute.String("ovm.source.name", plan.Name.ValueString()),
	)

	uuidBytes, err := uuidToBytes(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid source ID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid UUID")
		return
	}

	regions, diags := regionsFromList(ctx, plan.AWSRegions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	externalID := state.ExternalID.ValueString()

	sourceConfig, err := structpb.NewStruct(map[string]any{
		"aws-access-strategy": "external-id",
		"aws-external-id":     externalID,
		"aws-target-role-arn": plan.AWSRoleARN.ValueString(),
		"aws-regions":         strings.Join(regions, ","),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to build source config", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "config build failed")
		return
	}

	_, err = r.mgmt.UpdateSource(ctx, connect.NewRequest(&sdp.UpdateSourceRequest{
		UUID: uuidBytes,
		Properties: &sdp.SourceProperties{
			DescriptiveName: plan.Name.ValueString(),
			Type:            "aws",
			Config:          sourceConfig,
		},
	}))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update source", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "UpdateSource failed")
		return
	}

	plan.ID = state.ID
	plan.ExternalID = state.ExternalID

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *awsSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSSource Delete")
	defer span.End()

	var state awsSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	span.SetAttributes(attribute.String("ovm.source.id", state.ID.ValueString()))

	uuidBytes, err := uuidToBytes(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid source ID", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid UUID")
		return
	}

	_, err = r.mgmt.DeleteSource(ctx, connect.NewRequest(&sdp.DeleteSourceRequest{
		UUID: uuidBytes,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			span.SetAttributes(attribute.Bool("ovm.source.alreadyGone", true))
			return
		}
		resp.Diagnostics.AddError("Failed to delete source", err.Error())
		span.RecordError(err)
		span.SetStatus(codes.Error, "DeleteSource failed")
	}
}

func (r *awsSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ctx, span := tracing.Tracer().Start(ctx, "AWSSource Import")
	defer span.End()

	span.SetAttributes(attribute.String("ovm.source.id", req.ID))

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- helpers ---

func uuidToBytes(s string) ([]byte, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("parsing UUID %q: %w", s, err)
	}
	b := parsed[:]
	return b, nil
}

func regionsFromList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var regions []string
	diags := list.ElementsAs(ctx, &regions, false)
	return regions, diags
}

func splitNonEmpty(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
