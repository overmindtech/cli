package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func newLimitRangeAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.LimitRange, *v1.LimitRangeList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "LimitRange",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.LimitRange, *v1.LimitRangeList] {
			return cs.CoreV1().LimitRanges(namespace)
		},
		ListExtractor: func(list *v1.LimitRangeList) ([]*v1.LimitRange, error) {
			extracted := make([]*v1.LimitRange, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		AdapterMetadata: limitRangeAdapterMetadata,
	}
}

var limitRangeAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "LimitRange",
	DescriptiveName:       "Limit Range",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Limit Range"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_limit_range_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_limit_range.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newLimitRangeAdapter)
}
