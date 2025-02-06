package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var ingressYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ingress-app
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ingress-app
  template:
    metadata:
      labels:
        app: ingress-app
    spec:
      containers:
      - name: ingress-app
        image: nginx
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: ingress-app
spec:
  selector:
    app: ingress-app
  ports:
  - name: http
    port: 80
    targetPort: 80
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ingress-app
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /ingress-app
        pathType: Prefix
        backend:
          service:
            name: ingress-app
            port:
              name: http

`

func TestIngressAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newIngressAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "ingress-app",
		GetScope:  sd.String(),
		SetupYAML: ingressYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "dns",
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "example.com",
				ExpectedScope:  "global",
			},
			{
				ExpectedType:   "Service",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "ingress-app",
				ExpectedScope:  sd.String(),
			},
		},
	}

	st.Execute(t)
}
