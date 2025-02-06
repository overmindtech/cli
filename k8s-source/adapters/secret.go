package adapters

import (
	"crypto/sha512"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func newSecretAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Secret, *v1.SecretList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Secret",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Secret, *v1.SecretList] {
			return cs.CoreV1().Secrets(namespace)
		},
		ListExtractor: func(list *v1.SecretList) ([]*v1.Secret, error) {
			extracted := make([]*v1.Secret, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		Redact: func(resource *v1.Secret) *v1.Secret {
			// We want to redact the data from a secret, but we also went to
			// show people when it has changed, to that end we will hash all of
			// the data in the secret and return the hash
			hash := sha512.New()

			for k, v := range resource.Data {
				// Write the data into the hash
				hash.Write([]byte(k))
				hash.Write(v)
			}

			resource.Data = map[string][]byte{
				"data-redacted": hash.Sum(nil),
			}

			return resource
		},
		AdapterMetadata: secretAdapterMetadata,
	}
}

var secretAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Secret",
	DescriptiveName:       "Secret",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Secret"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_secret_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_secret.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newSecretAdapter)
}
