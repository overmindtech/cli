package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Project adapter for Compute Engine project metadata
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeProject,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/projects/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}
		/*
			https://cloud.google.com/compute/docs/reference/rest/v1/projects/get
			To decrease latency for this method, you can optionally omit any unneeded information from the response by using a field mask.
			This practice is especially recommended for unused quota information (the quotas field).
			To exclude one or more fields, set your request's fields query parameter to only include the fields you need.
			For example, to only include the id and selfLink fields, add the query parameter ?fields=id,selfLink to your request.
		*/
		// We only need the name field for this adapter
		// This resource won't carry any attributes to link it to other resources.
		// It will always be a linked item from the other resources by its name.
		// Note: This adapter uses the query as the project ID, and validates it
		// against the adapter's configured project via location.ProjectID.
		GetEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			if query == "" {
				return ""
			}
			if query != location.ProjectID {
				return ""
			}
			return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s?fields=name", query)
		},
		UniqueAttributeKeys: []string{"projects"},
		IAMPermissions:      []string{"compute.projects.get"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"defaultServiceAccount": {
			Description:   "If the IAM Service Account is deleted: Project resources may fail to work as before. If the project is deleted: service account is deleted.",
			ToSDPItemType: gcpshared.IAMServiceAccount,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"usageExportLocation.bucketName": {
			Description:   "If the Compute Bucket is deleted: Project usage export may fail. If the project is deleted: bucket is deleted.",
			ToSDPItemType: gcpshared.StorageBucket,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
