package adapters

import (
	"time"

	v1 "k8s.io/api/discovery/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func endpointSliceExtractor(resource *v1.EndpointSlice, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	sd, err := ParseScope(scope, true)

	if err != nil {
		return nil, err
	}

	for _, endpoint := range resource.Endpoints {
		if endpoint.Hostname != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  *endpoint.Hostname,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Always propagate over DNS
					In:  true,
					Out: true,
				},
			})
		}

		if endpoint.NodeName != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "Node",
					Method: sdp.QueryMethod_GET,
					Query:  *endpoint.NodeName,
					Scope:  sd.ClusterName,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the node can affect the endpoint
					In: true,
					// Changes to the endpoint cannot affect the node
					Out: false,
				},
			})
		}

		if endpoint.TargetRef != nil {
			queries = append(queries, ObjectReferenceToQuery(endpoint.TargetRef, sd, &sdp.BlastPropagation{
				// Changes to the pod could affect the endpoint and vice versa
				In:  true,
				Out: true,
			}))
		}

		for _, address := range endpoint.Addresses {
			switch resource.AddressType {
			case v1.AddressTypeIPv4, v1.AddressTypeIPv6:
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  address,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Always propagate over IP
						In:  true,
						Out: true,
					},
				})
			case v1.AddressTypeFQDN:
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  address,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Always propagate over DNS
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	return queries, nil
}

func newEndpointSliceAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.EndpointSlice, *v1.EndpointSliceList]{
		ClusterName:   cluster,
		Namespaces:    namespaces,
		TypeName:      "EndpointSlice",
		CacheDuration: 1 * time.Minute, // very low since this changes a lot
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.EndpointSlice, *v1.EndpointSliceList] {
			return cs.DiscoveryV1().EndpointSlices(namespace)
		},
		ListExtractor: func(list *v1.EndpointSliceList) ([]*v1.EndpointSlice, error) {
			extracted := make([]*v1.EndpointSlice, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: endpointSliceExtractor,
		AdapterMetadata:          endpointSliceAdapterMetadata,
	}
}

var endpointSliceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "EndpointSlice",
	DescriptiveName:       "Endpoint Slice",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks:        []string{"Node", "Pod", "dns", "ip"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("EndpointSlice"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_endpoints_slice_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_endpoints_slice.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newEndpointSliceAdapter)
}
