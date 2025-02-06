package adapters

import (
	v1 "k8s.io/api/rbac/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func roleBindingExtractor(resource *v1.RoleBinding, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	sd, err := ParseScope(scope, true)

	if err != nil {
		return nil, err
	}

	for _, subject := range resource.Subjects {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Method: sdp.QueryMethod_GET,
				Query:  subject.Name,
				Type:   subject.Kind,
				Scope: ScopeDetails{
					ClusterName: sd.ClusterName,
					Namespace:   subject.Namespace,
				}.String(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the subject (the group we're applying permissions
				// to) won't affect the role or the binding
				In: false,
				// Changes to the binding will affect the subject
				Out: true,
			},
		})
	}

	refSD := ScopeDetails{
		ClusterName: sd.ClusterName,
	}

	switch resource.RoleRef.Kind {
	case "Role":
		// If this binding is linked to a role then it's in the same namespace
		refSD.Namespace = sd.Namespace
	case "ClusterRole":
		// If this is linked to a ClusterRole (which is not namespaced) we need
		// to make sure that we are querying the root scope i.e. the
		// non-namespaced scope
		refSD.Namespace = ""
	}

	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Scope:  refSD.String(),
			Method: sdp.QueryMethod_GET,
			Query:  resource.RoleRef.Name,
			Type:   resource.RoleRef.Kind,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Changes to the role will affect the things bound to it since the
			// role contains the permissions
			In: true,
			// Changes to the binding won't affect the role
			Out: false,
		},
	})

	return queries, nil
}

func newRoleBindingAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.RoleBinding, *v1.RoleBindingList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "RoleBinding",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.RoleBinding, *v1.RoleBindingList] {
			return cs.RbacV1().RoleBindings(namespace)
		},
		ListExtractor: func(list *v1.RoleBindingList) ([]*v1.RoleBinding, error) {
			extracted := make([]*v1.RoleBinding, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: roleBindingExtractor,
		AdapterMetadata:          roleBindingAdapterMetadata,
	}
}

var roleBindingAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "RoleBinding",
	DescriptiveName:       "Role Binding",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	PotentialLinks:        []string{"Role", "ClusterRole", "ServiceAccount", "User", "Group"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("RoleBinding"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_role_binding.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_role_binding_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newRoleBindingAdapter)
}
