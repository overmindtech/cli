package adapters

import (
	"regexp"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/apps/v1"

	"k8s.io/client-go/kubernetes"
)

var replicaSetProgressedRegex = regexp.MustCompile(`ReplicaSet "([^"]+)" has successfully progressed`)

func newDeploymentAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Deployment, *v1.DeploymentList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "Deployment",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Deployment, *v1.DeploymentList] {
			return cs.AppsV1().Deployments(namespace)
		},
		ListExtractor: func(list *v1.DeploymentList) ([]*v1.Deployment, error) {
			extracted := make([]*v1.Deployment, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: func(deployment *v1.Deployment, scope string) ([]*sdp.LinkedItemQuery, error) {
			queries := make([]*sdp.LinkedItemQuery, 0)

			for _, condition := range deployment.Status.Conditions {
				// Parse out conditions that mention replica sets e.g.
				//
				// - lastTransitionTime: "2023-06-16T14:23:33Z"
				//   lastUpdateTime: "2023-09-15T13:07:07Z"
				//   message: ReplicaSet "gateway-5cf5578d94" has successfully progressed.
				//   reason: NewReplicaSetAvailable
				//   status: "True"
				//   type: Progressing
				if condition.Type == v1.DeploymentProgressing && condition.Reason == "NewReplicaSetAvailable" {
					matches := replicaSetProgressedRegex.FindStringSubmatch(condition.Message)

					if len(matches) > 1 {
						queries = append(queries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "ReplicaSet",
								Method: sdp.QueryMethod_GET,
								Query:  matches[1],
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								// These are tightly bound
								In:  true,
								Out: true,
							},
						})
					}
				}
			}

			return queries, nil
		},
		HealthExtractor: func(deployment *v1.Deployment) *sdp.Health {
			conditions := map[v1.DeploymentConditionType]bool{
				v1.DeploymentAvailable:      false,
				v1.DeploymentProgressing:    false,
				v1.DeploymentReplicaFailure: false,
			}

			for _, condition := range deployment.Status.Conditions {
				// Extract the condition
				conditions[condition.Type] = condition.Status == "True"
			}

			// If there is a replica failure, the deployment is unhealthy
			if conditions[v1.DeploymentReplicaFailure] {
				return sdp.Health_HEALTH_ERROR.Enum()
			}

			// If the deployment is available then it's healthy
			if conditions[v1.DeploymentAvailable] {
				return sdp.Health_HEALTH_OK.Enum()
			}

			// If the deployment is progressing (but not healthy) then it's
			// pending
			if conditions[v1.DeploymentProgressing] {
				return sdp.Health_HEALTH_PENDING.Enum()
			}

			// We should never reach here
			return sdp.Health_HEALTH_UNKNOWN.Enum()
		},
		AdapterMetadata: deploymentAdapterMetadata,
	}
}

var deploymentAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Deployment",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	PotentialLinks:        []string{"ReplicaSet"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Deployment"),
	DescriptiveName:       "Deployment",
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_deployment_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_deployment.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newDeploymentAdapter)
}
