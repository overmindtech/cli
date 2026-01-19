package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Service Usage Service adapter for enabled GCP services
var _ = registerableAdapter{
	sdpType: gcpshared.ServiceUsageService,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/service-usage/docs/reference/rest/v1/services/get
		// GET https://serviceusage.googleapis.com/v1/{name=*/*/services/*}
		// An example name would be: projects/123/services/service
		// where 123 is the project number TODO: make sure that this is working with project ID as well
		// IAM Perm: serviceusage.services.get
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://serviceusage.googleapis.com/v1/projects/%s/services/%s"),
		// Reference: https://cloud.google.com/service-usage/docs/reference/rest/v1/services/list
		// GET https://serviceusage.googleapis.com/v1/{parent=*/*}/services
		/*
			List all services available to the specified project, and the current state of those services with respect to the project.
			The list includes all public services, all services for which the calling user has the `servicemanagement.services.bind` permission,
			and all services that have already been enabled on the project.
			The list can be filtered to only include services in a specific state, for example to only include services enabled on the project.
		*/
		// Let's use the filter to only list enabled services.
		// IAM Perm: serviceusage.services.list
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://serviceusage.googleapis.com/v1/projects/%s/services?filter=state:ENABLED"),
		UniqueAttributeKeys: []string{"services"},
		// HEALTH: https://cloud.google.com/service-usage/docs/reference/rest/v1/services#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"serviceusage.services.get", "serviceusage.services.list"},
		PredefinedRole: "roles/serviceusage.serviceUsageViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"parent": {
			ToSDPItemType:    gcpshared.CloudResourceManagerProject,
			Description:      "If the Project is deleted or updated: The Service Usage Service may become invalid or inaccessible. If the Service Usage Service is updated: The project remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"config.name": {
			ToSDPItemType:    stdlib.NetworkDNS,
			Description:      "The DNS address at which this service is available. They are tightly coupled with the Service Usage Service.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"config.usage.producerNotificationChannel": {
			// Google Service Management currently only supports Google Cloud Pub/Sub as a notification channel.
			// To use Google Cloud Pub/Sub as the channel, this must be the name of a Cloud Pub/Sub topic
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: The Service Usage Service may fail to send notifications. If the Service Usage Service is updated: The topic remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"config.endpoints.name": {
			ToSDPItemType:    stdlib.NetworkDNS,
			Description:      "The canonical DNS name of the endpoint. DNS names and endpoints are tightly coupled - if DNS resolution fails, the endpoint becomes inaccessible.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"config.endpoints.target": {
			// The target field can contain either an IP address or FQDN.
			// The linker automatically detects which type the value is and creates the appropriate link.
			ToSDPItemType:    stdlib.NetworkIP,
			Description:      "The address of the API frontend (IP address or FQDN). Network connectivity to this address is required for the endpoint to function. The linker automatically detects whether the value is an IP address or DNS name.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"config.endpoints.aliases": {
			// Note: This field is deprecated but may still be present in existing configurations.
			// The linker will process each alias in the array.
			ToSDPItemType:    stdlib.NetworkDNS,
			Description:      "Additional DNS names/aliases for the endpoint. DNS names and endpoints are tightly coupled - if DNS resolution fails, the endpoint becomes inaccessible.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"config.documentation.documentationRootUrl": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "The HTTP/HTTPS URL to the root of the service documentation. HTTP connectivity to this URL is required to access the documentation.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"config.documentation.serviceRootUrl": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "The HTTP/HTTPS service root URL. HTTP connectivity to this URL may be required for service operations.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
