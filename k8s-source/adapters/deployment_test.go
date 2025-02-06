package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var deploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-deployment
  template:
    metadata:
      labels:
        app: my-deployment
    spec:
      containers:
      - name: my-container
        image: nginx:latest
        ports:
        - containerPort: 80
`

func TestDeploymentSource(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newDeploymentAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "my-deployment",
		GetScope:  sd.String(),
		SetupYAML: deploymentYAML,
		Wait: func(item *sdp.Item) bool {
			return item.GetHealth() == sdp.Health_HEALTH_OK
		},
		GetQueryTests: QueryTests{
			{
				ExpectedType:         "ReplicaSet",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedScope:        "local-tests.default",
				ExpectedQueryMatches: regexp.MustCompile("my-deployment"),
			},
		},
	}

	st.Execute(t)
}
