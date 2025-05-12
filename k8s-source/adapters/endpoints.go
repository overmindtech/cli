package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func EndpointsExtractor(resource *v1.Endpoints, scope string) ([]*sdp.LinkedItemQuery, error) { //nolint:staticcheck
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
					BlastPropagation: &sdp.BlastPropagation{
						// Always propagate over DNS
						In:  true,
						Out: true,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the node can affect the endpoint
						In: true,
						// Changes to the endpoint cannot affect the node
						Out: false,
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
					BlastPropagation: &sdp.BlastPropagation{
						// Always propagate over IP
						In:  true,
						Out: true,
					},
				})
			}

			if address.TargetRef != nil {
				queries = append(queries, ObjectReferenceToQuery(address.TargetRef, sd, &sdp.BlastPropagation{
					// These are tightly coupled
					In:  true,
					Out: true,
				}))
			}
		}
	}

	return queries, nil
}

func newEndpointsAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Endpoints, *v1.EndpointsList]{ //nolint:staticcheck
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Endpoints",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Endpoints, *v1.EndpointsList] { //nolint:staticcheck
			return cs.CoreV1().Endpoints(namespace)
		},
		ListExtractor: func(list *v1.EndpointsList) ([]*v1.Endpoints, error) { //nolint:staticcheck
			extracted := make([]*v1.Endpoints, len(list.Items)) //nolint:staticcheck

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: EndpointsExtractor,
		AdapterMetadata:          endpointsAdapterMetadata,
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
