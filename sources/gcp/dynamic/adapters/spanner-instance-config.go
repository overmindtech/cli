package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Spanner Instance Config adapter for Cloud Spanner instance configurations
var _ = registerableAdapter{
	sdpType: gcpshared.SpannerInstanceConfig,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		InDevelopment:      true,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instanceConfigs/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instanceConfigs/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instanceConfigs/%s"),
		// https://// https://spanner.googleapis.com/v1/projects/*/instanceConfigs
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://spanner.googleapis.com/v1/projects/%s/instanceConfigs"),
		UniqueAttributeKeys: []string{"instanceConfigs"},
		IAMPermissions:      []string{"spanner.instanceConfigs.get", "spanner.instanceConfigs.list"},
		PredefinedRole:      "roles/spanner.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
