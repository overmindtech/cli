package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func newConfigMapAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.ConfigMap, *v1.ConfigMapList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "ConfigMap",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.ConfigMap, *v1.ConfigMapList] {
			return cs.CoreV1().ConfigMaps(namespace)
		},
		ListExtractor: func(list *v1.ConfigMapList) ([]*v1.ConfigMap, error) {
			bindings := make([]*v1.ConfigMap, len(list.Items))

			for i := range list.Items {
				bindings[i] = &list.Items[i]
			}

			return bindings, nil
		},
		AdapterMetadata: configMapAdapterMetadata,
	}
}

var configMapAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "ConfigMap",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	DescriptiveName:       "Config Map",
	SupportedQueryMethods: DefaultSupportedQueryMethods("Config Map"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_config_map_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_config_map.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newConfigMapAdapter)
}
