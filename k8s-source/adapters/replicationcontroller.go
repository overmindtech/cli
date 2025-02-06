package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func replicationControllerExtractor(resource *v1.ReplicationController, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.Selector != nil {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Scope:  scope,
				Method: sdp.QueryMethod_SEARCH,
				Query: LabelSelectorToQuery(&metaV1.LabelSelector{
					MatchLabels: resource.Spec.Selector,
				}),
				Type: "Pod",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Bidirectional propagation since we control the pods, and the
				// pods host the service
				In:  true,
				Out: true,
			},
		})
	}

	return queries, nil
}

func newReplicationControllerAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.ReplicationController, *v1.ReplicationControllerList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "ReplicationController",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.ReplicationController, *v1.ReplicationControllerList] {
			return cs.CoreV1().ReplicationControllers(namespace)
		},
		ListExtractor: func(list *v1.ReplicationControllerList) ([]*v1.ReplicationController, error) {
			extracted := make([]*v1.ReplicationController, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: replicationControllerExtractor,
		AdapterMetadata:          replicationControllerAdapterMetadata,
	}
}

var replicationControllerAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "ReplicationController",
	DescriptiveName:       "Replication Controller",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	PotentialLinks:        []string{"Pod"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("ReplicationController"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_replication_controller.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_replication_controller_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newReplicationControllerAdapter)
}
