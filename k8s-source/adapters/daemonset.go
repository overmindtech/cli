package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/apps/v1"

	"k8s.io/client-go/kubernetes"
)

func newDaemonSetAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.DaemonSet, *v1.DaemonSetList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "DaemonSet",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.DaemonSet, *v1.DaemonSetList] {
			return cs.AppsV1().DaemonSets(namespace)
		},
		ListExtractor: func(list *v1.DaemonSetList) ([]*v1.DaemonSet, error) {
			extracted := make([]*v1.DaemonSet, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		// Pods are linked automatically
		AdapterMetadata: daemonSetAdapterMetadata,
	}
}

var daemonSetAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "DaemonSet",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	DescriptiveName:       "Daemon Set",
	SupportedQueryMethods: DefaultSupportedQueryMethods("Daemon Set"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_daemon_set_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_daemonset.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newDaemonSetAdapter)
}
