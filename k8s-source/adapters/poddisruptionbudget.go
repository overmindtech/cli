package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/policy/v1"
	"k8s.io/client-go/kubernetes"
)

func podDisruptionBudgetExtractor(resource *v1.PodDisruptionBudget, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.Selector != nil {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "Pod",
				Method: sdp.QueryMethod_SEARCH,
				Query:  LabelSelectorToQuery(resource.Spec.Selector),
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to pods won't affect the disruption budget
				In: false,
				// Changes to the disruption budget will affect pods
				Out: true,
			},
		})
	}

	return queries, nil
}

func newPodDisruptionBudgetAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.PodDisruptionBudget, *v1.PodDisruptionBudgetList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "PodDisruptionBudget",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.PodDisruptionBudget, *v1.PodDisruptionBudgetList] {
			return cs.PolicyV1().PodDisruptionBudgets(namespace)
		},
		ListExtractor: func(list *v1.PodDisruptionBudgetList) ([]*v1.PodDisruptionBudget, error) {
			extracted := make([]*v1.PodDisruptionBudget, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: podDisruptionBudgetExtractor,
		AdapterMetadata:          podDisruptionBudgetAdapterMetadata,
	}
}

var podDisruptionBudgetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "PodDisruptionBudget",
	DescriptiveName:       "Pod Disruption Budget",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	PotentialLinks:        []string{"Pod"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("PodDisruptionBudget"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_pod_disruption_budget_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newPodDisruptionBudgetAdapter)
}
