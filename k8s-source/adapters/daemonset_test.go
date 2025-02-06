package adapters

import (
	"testing"
)

var daemonSetYAML = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: my-daemonset
spec:
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-container
        image: nginx:latest
        ports:
        - containerPort: 80

`

func TestDaemonSetSource(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newDaemonSetAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "my-daemonset",
		GetScope:  sd.String(),
		SetupYAML: daemonSetYAML,
	}

	st.Execute(t)
}
