package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var horizontalPodAutoscalerYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hpa-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hpa-app
  template:
    metadata:
      labels:
        app: hpa-app
    spec:
      containers:
      - name: hpa-container
        image: nginx:latest
        ports:
        - containerPort: 80
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: hpa-deployment
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 50
`

func TestHorizontalPodAutoscalerAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newHorizontalPodAutoscalerAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "my-hpa",
		GetScope:  sd.String(),
		SetupYAML: horizontalPodAutoscalerYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "Deployment",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedScope:  sd.String(),
				ExpectedQuery:  "hpa-deployment",
			},
		},
	}

	st.Execute(t)
}
