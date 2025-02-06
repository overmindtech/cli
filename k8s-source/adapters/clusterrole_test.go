package adapters

import (
	"testing"
)

var clusterRoleYAML = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: read-only
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["get", "list", "watch"]

`

func TestClusterRoleAdapter(t *testing.T) {
	adapter := newClusterRoleAdapter(CurrentCluster.ClientSet, CurrentCluster.Name, []string{})

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "read-only",
		GetScope:      CurrentCluster.Name,
		SetupYAML:     clusterRoleYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)
}
