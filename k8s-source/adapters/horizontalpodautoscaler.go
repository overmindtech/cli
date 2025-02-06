package adapters

import (
	v2 "k8s.io/api/autoscaling/v2"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func horizontalPodAutoscalerExtractor(resource *v2.HorizontalPodAutoscaler, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   resource.Spec.ScaleTargetRef.Kind,
			Method: sdp.QueryMethod_GET,
			Query:  resource.Spec.ScaleTargetRef.Name,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Changes to the target won't affect the hpa
			In: false,
			// Changes to the hpa can affect the target i.e. by scaling the pods
			Out: true,
		},
	})

	return queries, nil
}

func newHorizontalPodAutoscalerAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v2.HorizontalPodAutoscaler, *v2.HorizontalPodAutoscalerList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "HorizontalPodAutoscaler",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v2.HorizontalPodAutoscaler, *v2.HorizontalPodAutoscalerList] {
			return cs.AutoscalingV2().HorizontalPodAutoscalers(namespace)
		},
		ListExtractor: func(list *v2.HorizontalPodAutoscalerList) ([]*v2.HorizontalPodAutoscaler, error) {
			extracted := make([]*v2.HorizontalPodAutoscaler, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: horizontalPodAutoscalerExtractor,
		AdapterMetadata:          horizontalPodAutoscalerAdapterMetadata,
	}
}

var horizontalPodAutoscalerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "HorizontalPodAutoscaler",
	DescriptiveName:       "Horizontal Pod Autoscaler",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Horizontal Pod Autoscaler"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_horizontal_pod_autoscaler_v2.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newHorizontalPodAutoscalerAdapter)
}
