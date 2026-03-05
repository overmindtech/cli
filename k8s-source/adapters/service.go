package adapters

import (
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
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
		})
	}

	// Services generate an Endpoints object with the same name (older K8s API)
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "Endpoints",
			Method: sdp.QueryMethod_GET,
			Query:  resource.Name,
			Scope:  scope,
		},
	})

	// Modern K8s clusters also create EndpointSlices labelled with the service name
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "EndpointSlice",
			Method: sdp.QueryMethod_SEARCH,
			Query: ListOptionsToQuery(&metaV1.ListOptions{
				LabelSelector: "kubernetes.io/service-name=" + resource.Name,
			}),
			Scope: scope,
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
			})
		}
	}

	return queries, nil
}

func newServiceAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string, cache sdpcache.Cache) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Service, *v1.ServiceList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Service",
		cache:       cache,
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
	PotentialLinks:        []string{"Pod", "ip", "dns", "Endpoints", "EndpointSlice"},
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
