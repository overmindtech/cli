package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/rbac/v1"

	"k8s.io/client-go/kubernetes"
)

func newClusterRoleAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.ClusterRole, *v1.ClusterRoleList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "ClusterRole",
		ClusterInterfaceBuilder: func() ItemInterface[*v1.ClusterRole, *v1.ClusterRoleList] {
			return cs.RbacV1().ClusterRoles()
		},
		ListExtractor: func(list *v1.ClusterRoleList) ([]*v1.ClusterRole, error) {
			bindings := make([]*v1.ClusterRole, len(list.Items))

			for i := range list.Items {
				bindings[i] = &list.Items[i]
			}

			return bindings, nil
		},
		AdapterMetadata: clusterRoleAdapterMetadata,
	}
}

var clusterRoleAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "ClusterRole",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	DescriptiveName:       "Cluster Role",
	SupportedQueryMethods: DefaultSupportedQueryMethods("Cluster Role"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_cluster_role_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newClusterRoleAdapter)
}
