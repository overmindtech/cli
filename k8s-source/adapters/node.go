package adapters

import (
	"strings"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes"
)

func linkedItemExtractor(resource *v1.Node, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	for _, addr := range resource.Status.Addresses {
		switch addr.Type {
		case v1.NodeExternalDNS, v1.NodeInternalDNS, v1.NodeHostName:
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  addr.Address,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Always propagate over DNS
					In:  true,
					Out: true,
				},
			})

		case v1.NodeExternalIP, v1.NodeInternalIP:
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  addr.Address,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Always propagate over IP
					In:  true,
					Out: true,
				},
			})
		}
	}

	for _, vol := range resource.Status.VolumesAttached {
		// Look for EBS volumes since they follow the format:
		// kubernetes.io/csi/ebs.csi.aws.com^vol-043e04d9cc6d72183
		if strings.HasPrefix(string(vol.Name), "kubernetes.io/csi/ebs.csi.aws.com") {
			sections := strings.Split(string(vol.Name), "^")

			if len(sections) == 2 {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ec2-volume",
						Method: sdp.QueryMethod_GET,
						Query:  sections[1],
						Scope:  "*",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changes to the volume can affect the node
						In: true,
						// Changes to the node cannot affect the volume
						Out: true,
					},
				})
			}
		}
	}

	return queries, nil
}

func newNodeAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Node, *v1.NodeList]{
		ClusterName: cluster,
		Namespaces:  namespaces,
		TypeName:    "Node",
		ClusterInterfaceBuilder: func() ItemInterface[*v1.Node, *v1.NodeList] {
			return cs.CoreV1().Nodes()
		},
		ListExtractor: func(list *v1.NodeList) ([]*v1.Node, error) {
			extracted := make([]*v1.Node, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: linkedItemExtractor,
		AdapterMetadata:          nodeAdapterMetadata,
	}
}

var nodeAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Node",
	DescriptiveName:       "Node",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	PotentialLinks:        []string{"dns", "ip", "ec2-volume"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Node"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_node_taint.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newNodeAdapter)
}
