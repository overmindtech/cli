package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var endpointsYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: endpoint-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: endpoint-test
  template:
    metadata:
      labels:
        app: endpoint-test
    spec:
      containers:
        - name: endpoint-test
          image: nginx:latest
          ports:
            - containerPort: 80

---
apiVersion: v1
kind: Service
metadata:
  name: endpoint-service
spec:
  selector:
    app: endpoint-test
  ports:
    - name: http
      port: 80
      targetPort: 80
  type: ClusterIP

`

func TestEndpointsAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newEndpointsAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "endpoint-service",
		GetScope:  sd.String(),
		SetupYAML: endpointsYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile(`^10\.`),
				ExpectedType:         "ip",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedScope:        "global",
			},
			{
				ExpectedType:   "Node",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "local-tests-control-plane",
				ExpectedScope:  CurrentCluster.Name,
			},
			{
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedQueryMatches: regexp.MustCompile("endpoint-deployment"),
				ExpectedScope:        sd.String(),
			},
		},
		Wait: func(item *sdp.Item) bool {
			return len(item.GetLinkedItemQueries()) > 0
		},
	}

	st.Execute(t)
}
