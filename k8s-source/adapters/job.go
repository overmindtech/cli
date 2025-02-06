package adapters

import (
	v1 "k8s.io/api/batch/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"k8s.io/client-go/kubernetes"
)

func jobExtractor(resource *v1.Job, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	if resource.Spec.Selector != nil {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Scope:  scope,
				Method: sdp.QueryMethod_SEARCH,
				Query:  LabelSelectorToQuery(resource.Spec.Selector),
				Type:   "Pod",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to a job will replace the pods, changes to the pods
				// could break the job
				In:  true,
				Out: true,
			},
		})
	}

	return queries, nil
}

func newJobAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Job, *v1.JobList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "Job",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Job, *v1.JobList] {
			return cs.BatchV1().Jobs(namespace)
		},
		ListExtractor: func(list *v1.JobList) ([]*v1.Job, error) {
			extracted := make([]*v1.Job, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: jobExtractor,
		AdapterMetadata:          jobAdapterMetadata,
	}
}

var jobAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "Job",
	DescriptiveName:       "Job",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	PotentialLinks:        []string{"Pod"},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Job"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_job.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_job_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newJobAdapter)
}
