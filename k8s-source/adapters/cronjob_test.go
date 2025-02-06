package adapters

import (
	"context"
	"testing"
	"time"
)

var cronJobYAML = `
apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: my-container
            image: alpine
            command: ["/bin/sh", "-c"]
            args:
            - sleep 10; echo "Hello, world!"
          restartPolicy: OnFailure
`

func TestCronJobAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newCronJobAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "my-cronjob",
		GetScope:      sd.String(),
		SetupYAML:     cronJobYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)

	// Additionally, make sure that the job has a link back to the cronjob that
	// created it
	jobAdapter := newJobAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	// Wait for the job to be created
	err := WaitFor(60*time.Second, func() bool {
		jobs, err := jobAdapter.List(context.Background(), sd.String(), false)

		if err != nil {
			t.Fatal(err)
			return false
		}

		// Ensure that the job has a link back to the cronjob
		for _, job := range jobs {
			for _, q := range job.GetLinkedItemQueries() {
				if q.GetQuery() != nil {
					if q.GetQuery().GetQuery() == "my-cronjob" {
						return true
					}
				}
			}

		}

		return false
	})

	if err != nil {
		t.Fatal(err)
	}
}
