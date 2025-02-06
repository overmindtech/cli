package adapters

import (
	"errors"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func PersistentVolumeClaimExtractor(resource *v1.PersistentVolumeClaim, scope string) ([]*sdp.LinkedItemQuery, error) {
	if resource == nil {
		return nil, errors.New("resource is nil")
	}

	links := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.VolumeName != "" {
		links = append(links, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "PersistentVolume",
				Method: sdp.QueryMethod_GET,
				Query:  resource.Spec.VolumeName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the volume could affect the claim
				In: true,
				// Changes to the claim could affect the volume if there are
				// other claims
				Out: true,
			},
		})
	}

	return links, nil
}

func newPersistentVolumeClaimAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.PersistentVolumeClaim, *v1.PersistentVolumeClaimList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "PersistentVolumeClaim",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.PersistentVolumeClaim, *v1.PersistentVolumeClaimList] {
			return cs.CoreV1().PersistentVolumeClaims(namespace)
		},
		ListExtractor: func(list *v1.PersistentVolumeClaimList) ([]*v1.PersistentVolumeClaim, error) {
			extracted := make([]*v1.PersistentVolumeClaim, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: PersistentVolumeClaimExtractor,
		AdapterMetadata:          persistentVolumeClaimAdapterMetadata,
	}
}

var persistentVolumeClaimAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "PersistentVolumeClaim",
	DescriptiveName:       "Persistent Volume Claim",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
	PotentialLinks:        []string{"PersistentVolume"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("PersistentVolumeClaim"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_persistent_volume_claim.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_persistent_volume_claim_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newPersistentVolumeClaimAdapter)
}
