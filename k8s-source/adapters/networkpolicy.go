package adapters

import (
	v1 "k8s.io/api/networking/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func NetworkPolicyExtractor(resource *v1.NetworkPolicy, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "Pod",
			Method: sdp.QueryMethod_SEARCH,
			Query:  LabelSelectorToQuery(&resource.Spec.PodSelector),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Changes to pods won't affect the network policy or anything else
			// that shares it
			In: false,
			// Changes to the network policy will affect pods
			Out: true,
		},
	})

	var peers []v1.NetworkPolicyPeer

	for _, ig := range resource.Spec.Ingress {
		peers = append(peers, ig.From...)
	}

	for _, eg := range resource.Spec.Egress {
		peers = append(peers, eg.To...)
	}

	// Link all peers
	for _, peer := range peers {
		if ps := peer.PodSelector; ps != nil {
			// TODO: Link to namespaces that are allowed to ingress e.g.
			// - namespaceSelector:
			// matchLabels:
			//   project: something

			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  scope,
					Method: sdp.QueryMethod_GET,
					Query:  LabelSelectorToQuery(ps),
					Type:   "Pod",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to pods won't affect the network policy or anything else
					// that shares it
					In: false,
					// Changes to the network policy will affect pods
					Out: true,
				},
			})
		}
	}

	return queries, nil
}

func newNetworkPolicyAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.NetworkPolicy, *v1.NetworkPolicyList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "NetworkPolicy",
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.NetworkPolicy, *v1.NetworkPolicyList] {
			return cs.NetworkingV1().NetworkPolicies(namespace)
		},
		ListExtractor: func(list *v1.NetworkPolicyList) ([]*v1.NetworkPolicy, error) {
			extracted := make([]*v1.NetworkPolicy, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: NetworkPolicyExtractor,
		AdapterMetadata:          networkPolicyAdapterMetadata,
	}
}

var networkPolicyAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "NetworkPolicy",
	DescriptiveName: "Network Policy",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
	PotentialLinks:  []string{"Pod"},
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_network_policy.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_network_policy_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newNetworkPolicyAdapter)
}
