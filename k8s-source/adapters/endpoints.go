// This adapter uses the deprecated core/v1.Endpoints API intentionally.
//
// We use the latest K8s SDK version but balance that against supporting as many
// Kubernetes versions as possible. Older clusters may not have the
// discoveryv1.EndpointSlice API, so we retain this adapter for backward
// compatibility. The staticcheck lint exceptions below are therefore expected
// and acceptable. When the SDK eventually drops support for v1.Endpoints we
// will need to split out version-specific builds of the k8s-source.

package adapters

import (
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func EndpointsExtractor(resource *v1.Endpoints, scope string) ([]*sdp.LinkedItemQuery, error) { //nolint:staticcheck,nolintlint // SA1019: v1.Endpoints deprecated; see note at top of file
	queries := make([]*sdp.LinkedItemQuery, 0)

	sd, err := ParseScope(scope, true)

	if err != nil {
		return nil, err
	}

	for _, subset := range resource.Subsets {
		for _, address := range subset.Addresses {
			if address.Hostname != "" {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Scope:  "global",
						Method: sdp.QueryMethod_GET,
						Query:  address.Hostname,
						Type:   "dns",
					},
				})
			}

			if address.NodeName != nil {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "Node",
						Scope:  sd.ClusterName,
						Method: sdp.QueryMethod_GET,
						Query:  *address.NodeName,
					},
				})
			}

			if address.IP != "" {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  address.IP,
						Scope:  "global",
					},
				})
			}

			if address.TargetRef != nil {
				queries = append(queries, ObjectReferenceToQuery(address.TargetRef, sd))
			}
		}
	}

	return queries, nil
}

func newEndpointsAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string, cache sdpcache.Cache) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Endpoints, *v1.EndpointsList]{ //nolint:staticcheck,nolintlint // SA1019: v1.Endpoints deprecated; see note at top of file
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Endpoints",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Endpoints, *v1.EndpointsList] { //nolint:staticcheck,nolintlint // SA1019
			return cs.CoreV1().Endpoints(namespace)
		},
		ListExtractor: func(list *v1.EndpointsList) ([]*v1.Endpoints, error) { //nolint:staticcheck,nolintlint // SA1019
			extracted := make([]*v1.Endpoints, len(list.Items)) //nolint:staticcheck,nolintlint // SA1019

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: EndpointsExtractor,
		AdapterMetadata:          endpointsAdapterMetadata,
		cache:                    cache,
	}
}

var endpointsAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName:       "Endpoints",
	Type:                  "Endpoints",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Endpoints"),
	PotentialLinks:        []string{"Node", "ip", "Pod", "ExternalName", "DNS"},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_endpoints.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_endpoints_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newEndpointsAdapter)
}
