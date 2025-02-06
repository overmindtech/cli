package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var statefulSetYAML = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: stateful-set-test
spec:
  serviceName: nginx
  replicas: 1
  selector:
    matchLabels:
      app: stateful-set-test
  template:
    metadata:
      labels:
        app: stateful-set-test
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        volumeMounts:
        - name: stateful-set-test-storage
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: stateful-set-test-storage
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
      storageClassName: standard
`

func TestStatefulSetAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newStatefulSetAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "stateful-set-test",
		GetScope:  sd.String(),
		SetupYAML: statefulSetYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedQueryMatches: regexp.MustCompile(`app=stateful-set-test`),
				ExpectedScope:        sd.String(),
			},
			{
				ExpectedType:         "PersistentVolumeClaim",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedQueryMatches: regexp.MustCompile(`app=stateful-set-test`),
				ExpectedScope:        sd.String(),
			},
		},
	}

	st.Execute(t)
}
