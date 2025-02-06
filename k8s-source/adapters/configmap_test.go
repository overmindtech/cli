package adapters

import "testing"

var configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
data:
  DATABASE_URL: "postgres://myuser:mypassword@mydbhost:5432/mydatabase"
  APP_SECRET_KEY: "mysecretkey123"
`

func TestConfigMapAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newConfigMapAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "my-configmap",
		GetScope:      sd.String(),
		SetupYAML:     configMapYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)
}
