package adapters

import (
	v1 "k8s.io/api/networking/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func ingressExtractor(resource *v1.Ingress, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.IngressClassName != nil {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "IngressClass",
				Method: sdp.QueryMethod_GET,
				Query:  *resource.Spec.IngressClassName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the ingress (e.g. nginx) class can affect the
				// ingresses that use it
				In: true,
				// Changes to an ingress wont' affect the class
				Out: false,
			},
		})
	}

	if resource.Spec.DefaultBackend != nil {
		if resource.Spec.DefaultBackend.Service != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "Service",
					Method: sdp.QueryMethod_GET,
					Query:  resource.Spec.DefaultBackend.Service.Name,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the service affects the ingress' endpoints
					In: true,
					// Changing an ingress does not affect the service
					Out: false,
				},
			})
		}

		if linkRes := resource.Spec.DefaultBackend.Resource; linkRes != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   linkRes.Kind,
					Method: sdp.QueryMethod_GET,
					Query:  linkRes.Name,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the default backend won't affect the ingress
					// itself
					In: false,
					// Changes to the ingress could affect the default backend
					Out: true,
				},
			})
		}
	}

	for _, rule := range resource.Spec.Rules {
		if rule.Host != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  rule.Host,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Always propagate through rules
					In:  true,
					Out: true,
				},
			})
		}

		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "Service",
							Method: sdp.QueryMethod_GET,
							Query:  path.Backend.Service.Name,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes to the service affects the ingress' endpoints
							In: true,
							// Changing an ingress does not affect the service
							Out: false,
						},
					})
				}

				if path.Backend.Resource != nil {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   path.Backend.Resource.Kind,
							Method: sdp.QueryMethod_GET,
							Query:  path.Backend.Resource.Name,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changes can go in both directions here. An
							// backend change can affect the ingress by rendering
							// backend change can affect the ingress by rending
							// it broken
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	return queries, nil
}

func newIngressAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Ingress, *v1.IngressList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Ingress",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Ingress, *v1.IngressList] {
			return cs.NetworkingV1().Ingresses(namespace)
		},
		ListExtractor: func(list *v1.IngressList) ([]*v1.Ingress, error) {
			extracted := make([]*v1.Ingress, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: ingressExtractor,
		AdapterMetadata:          ingressAdapterMetadata,
	}
}

var ingressAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Ingress",
	DescriptiveName:       "Ingress",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks:        []string{"Service", "IngressClass", "dns"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Ingress"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_ingress_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newIngressAdapter)
}
