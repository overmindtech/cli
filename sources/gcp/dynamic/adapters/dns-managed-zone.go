package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// DNS Managed Zone adapter for Cloud DNS managed zones
var _ = registerableAdapter{
	sdpType: gcpshared.DNSManagedZone,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/get
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones/{managedZone}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://dns.googleapis.com/dns/v1/projects/%s/managedZones/%s"),
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/list
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://dns.googleapis.com/dns/v1/projects/%s/managedZones"),
		UniqueAttributeKeys: []string{"managedZones"},
		IAMPermissions:      []string{"dns.managedZones.get", "dns.managedZones.list"},
		PredefinedRole:      "roles/dns.reader",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"dnsName": {
			ToSDPItemType:    stdlib.NetworkDNS,
			Description:      "Tightly coupled with the DNS Managed Zone.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"privateVisibilityConfig.networks.networkUrl": gcpshared.ComputeNetworkImpactInOnly,
		// The resource name of the cluster to bind this ManagedZone to. This should be specified in the format like: projects/*/locations/*/clusters/*.
		// This is referenced from GKE projects.locations.clusters.get
		// API: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters/get
		"privateVisibilityConfig.gkeClusters.gkeClusterName": {
			ToSDPItemType: gcpshared.ContainerCluster,
		},
		"forwardingConfig.targetNameServers.ipv4Address": gcpshared.IPImpactBothWays,
		"forwardingConfig.targetNameServers.ipv6Address": gcpshared.IPImpactBothWays,
		// The presence of this field indicates that DNS Peering is enabled for this zone. The value of this field contains the network to peer with.
		"peeringConfig.targetNetwork.networkUrl": gcpshared.ComputeNetworkImpactInOnly,
		// This field links to the associated service directory namespace.
		// The fully qualified URL of the namespace associated with the zone.
		// Format must be https://servicedirectory.googleapis.com/v1/projects/{project}/locations/{location}/namespaces/{namespace}
		"serviceDirectoryConfig.namespace.namespaceUrl": {
			ToSDPItemType:    gcpshared.ServiceDirectoryNamespace,
			Description:      "If the Service Directory Namespace is deleted or updated: The DNS Managed Zone may lose its association or fail to resolve names. If the DNS Managed Zone is updated: The namespace remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone#name",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dns_managed_zone.name",
			},
		},
	},
}.Register()
