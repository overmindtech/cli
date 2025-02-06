package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var jobYAML = `
apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
spec:
  template:
    spec:
      containers:
      - name: my-container
        image: nginx
        command: ["/bin/sh", "-c"]
        args:
        - echo "Hello, world!"; sleep 5
      restartPolicy: OnFailure
  backoffLimit: 4
---
apiVersion: batch/v1
kind: Job
metadata:
  name: my-job2
spec:
  template:
    spec:
      containers:
      - name: my-container
        image: nginx
        command: ["/bin/sh", "-c"]
        args:
        - echo "Hello, world!"; sleep 5
      restartPolicy: OnFailure
  backoffLimit: 4
`

func TestJobAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newJobAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "my-job",
		GetScope:  sd.String(),
		SetupYAML: jobYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile("controller-uid="),
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedScope:        sd.String(),
			},
		},
	}

	st.Execute(t)
}
