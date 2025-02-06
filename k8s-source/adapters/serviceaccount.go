package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func serviceAccountExtractor(resource *v1.ServiceAccount, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	for _, secret := range resource.Secrets {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Scope:  scope,
				Method: sdp.QueryMethod_GET,
				Query:  secret.Name,
				Type:   "Secret",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the secret will affect the service account and the
				// things that use it
				In: true,
				// The service account cannot affect the secret
				Out: false,
			},
		})
	}

	for _, ipSecret := range resource.ImagePullSecrets {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Scope:  scope,
				Method: sdp.QueryMethod_GET,
				Query:  ipSecret.Name,
				Type:   "Secret",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the secret will affect the service account and the
				// things that use it
				In: true,
				// The service account cannot affect the secret
				Out: false,
			},
		})
	}

	return queries, nil
}

func newServiceAccountAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.ServiceAccount, *v1.ServiceAccountList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "ServiceAccount",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.ServiceAccount, *v1.ServiceAccountList] {
			return cs.CoreV1().ServiceAccounts(namespace)
		},
		ListExtractor: func(list *v1.ServiceAccountList) ([]*v1.ServiceAccount, error) {
			extracted := make([]*v1.ServiceAccount, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: serviceAccountExtractor,
		AdapterMetadata:          serviceAccountAdapterMetadata,
	}
}

var serviceAccountAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "ServiceAccount",
	DescriptiveName:       "Service Account",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	PotentialLinks:        []string{"Secret"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("ServiceAccount"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_service_account.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_service_account_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newServiceAccountAdapter)
}
