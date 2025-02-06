package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/batch/v1"

	"k8s.io/client-go/kubernetes"
)

func newCronJobAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.CronJob, *v1.CronJobList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "CronJob",
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.CronJob, *v1.CronJobList] {
			return cs.BatchV1().CronJobs(namespace)
		},
		ListExtractor: func(list *v1.CronJobList) ([]*v1.CronJob, error) {
			bindings := make([]*v1.CronJob, len(list.Items))

			for i := range list.Items {
				bindings[i] = &list.Items[i]
			}

			return bindings, nil
		},
		// Cronjobs don't need linked items as the jobs they produce are linked
		// automatically
		AdapterMetadata: cronJobAdapterMetadata,
	}
}

var cronJobAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:                  "CronJob",
	Category:              sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	DescriptiveName:       "Cron Job",
	SupportedQueryMethods: DefaultSupportedQueryMethods("Cron Job"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_cron_job_v1.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_cron_job.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newCronJobAdapter)
}
