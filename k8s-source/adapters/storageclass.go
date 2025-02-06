package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/storage/v1"

	"k8s.io/client-go/kubernetes"
)

func newStorageClassAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.StorageClass, *v1.StorageClassList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "StorageClass",
		ClusterInterfaceBuilder: func() ItemInterface[*v1.StorageClass, *v1.StorageClassList] {
			return cs.StorageV1().StorageClasses()
		},
		ListExtractor: func(list *v1.StorageClassList) ([]*v1.StorageClass, error) {
			extracted := make([]*v1.StorageClass, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		AdapterMetadata: storageClassAdapterMetadata,
	}
}

var storageClassAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "StorageClass",
	DescriptiveName:       "Storage Class",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
	SupportedQueryMethods: DefaultSupportedQueryMethods("Storage Class"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_storage_class.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_storage_class_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newStorageClassAdapter)
}
