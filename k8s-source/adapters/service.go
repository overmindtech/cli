package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func serviceExtractor(resource *v1.Service, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.Selector != nil {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "Pod",
				Method: sdp.QueryMethod_SEARCH,
				Query: LabelSelectorToQuery(&metaV1.LabelSelector{
					MatchLabels: resource.Spec.Selector,
				}),
				Scope: scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Bidirectional propagation since we control the pods, and the
				// pods host the service
				In:  true,
				Out: true,
			},
		})
	}

	ips := make([]string, 0)

	if len(resource.Spec.ClusterIPs) > 0 {
		ips = append(ips, resource.Spec.ClusterIPs...)
	} else if resource.Spec.ClusterIP != "" {
		ips = append(ips, resource.Spec.ClusterIP)
	}

	ips = append(ips, resource.Spec.ExternalIPs...)
	ips = append(ips, resource.Spec.LoadBalancerIP)

	for _, ip := range ips {
		if ip != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  ip,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs are always bidirectional
					In:  true,
					Out: true,
				},
			})
		}
	}

	if resource.Spec.ExternalName != "" {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  resource.Spec.ExternalName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS is always bidirectional
				In:  true,
				Out: true,
			},
		})
	}

	// Services also generate an endpoint with the same name
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "Endpoint",
			Method: sdp.QueryMethod_GET,
			Query:  resource.Name,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// The service causes the endpoint to be created, so changes to the
			// service can affect the endpoint and vice versa
			In:  true,
			Out: true,
		},
	})

	for _, ingress := range resource.Status.LoadBalancer.Ingress {
		if ingress.IP != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  ingress.IP,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs are always bidirectional
					In:  true,
					Out: true,
				},
			})
		}

		if ingress.Hostname != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  ingress.Hostname,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS is always bidirectional
					In:  true,
					Out: true,
				},
			})
		}
	}

	return queries, nil
}

func newServiceAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Service, *v1.ServiceList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Service",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Service, *v1.ServiceList] {
			return cs.CoreV1().Services(namespace)
		},
		ListExtractor: func(list *v1.ServiceList) ([]*v1.Service, error) {
			extracted := make([]*v1.Service, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: serviceExtractor,
		AdapterMetadata:          serviceAdapterMetadata,
	}
}

var serviceAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Service",
	DescriptiveName:       "Service",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks:        []string{"Pod", "ip", "dns", "Endpoint"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Service"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_service.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_service_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newServiceAdapter)
}
