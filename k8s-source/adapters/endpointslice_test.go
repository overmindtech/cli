package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var endpointSliceYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: endpointslice-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: endpointslice-test
  template:
    metadata:
      labels:
        app: endpointslice-test
    spec:
      containers:
        - name: endpointslice-test
          image: nginx:latest
          ports:
            - containerPort: 80

---
apiVersion: v1
kind: Service
metadata:
  name: endpointslice-service
spec:
  selector:
    app: endpointslice-test
  ports:
    - name: http
      port: 80
      targetPort: 80
  type: ClusterIP

`

func TestEndpointSliceAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newEndpointSliceAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:        adapter,
		GetQueryRegexp: regexp.MustCompile("endpoint-service"),
		GetScope:       sd.String(),
		SetupYAML:      endpointSliceYAML,
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
	}

	st.Execute(t)
}
