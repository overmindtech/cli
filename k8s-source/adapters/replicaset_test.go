package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var replicaSetYAML = `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: replica-set-test
spec:
  replicas: 1
  selector:
    matchLabels:
      app: replica-set-test
  template:
    metadata:
      labels:
        app: replica-set-test
    spec:
      containers:
        - name: replica-set-test
          image: nginx:latest
          ports:
            - containerPort: 80

`

func TestReplicaSetAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newReplicaSetAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "replica-set-test",
		GetScope:  sd.String(),
		SetupYAML: replicaSetYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile("app=replica-set-test"),
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedScope:        sd.String(),
			},
		},
	}

	st.Execute(t)
}
