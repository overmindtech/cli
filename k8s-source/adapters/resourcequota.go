package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func newResourceQuotaAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.ResourceQuota, *v1.ResourceQuotaList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "ResourceQuota",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.ResourceQuota, *v1.ResourceQuotaList] {
			return cs.CoreV1().ResourceQuotas(namespace)
		},
		ListExtractor: func(list *v1.ResourceQuotaList) ([]*v1.ResourceQuota, error) {
			extracted := make([]*v1.ResourceQuota, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		AdapterMetadata: resourceQuotaAdapterMetadata,
	}
}

var resourceQuotaAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "ResourceQuota",
	DescriptiveName:       "Resource Quota",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Resource Quota"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_resource_quota_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_resource_quota.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newResourceQuotaAdapter)
}
