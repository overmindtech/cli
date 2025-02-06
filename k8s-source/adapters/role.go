package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/rbac/v1"

	"k8s.io/client-go/kubernetes"
)

func newRoleAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Role, *v1.RoleList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Role",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Role, *v1.RoleList] {
			return cs.RbacV1().Roles(namespace)
		},
		ListExtractor: func(list *v1.RoleList) ([]*v1.Role, error) {
			extracted := make([]*v1.Role, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		AdapterMetadata: roleAdapterMetadata,
	}
}

var roleAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Role",
	DescriptiveName:       "Role",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Role"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_role_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_role.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newRoleAdapter)
}
