package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdpcache"
)

var RoleYAML = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: role-test-role
rules:
  - apiGroups:
      - ""
      - "apps"
      - "batch"
      - "extensions"
    resources:
      - pods
      - deployments
      - jobs
      - cronjobs
      - configmaps
      - secrets
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - delete
`

func TestRoleAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newRoleAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace}, sdpcache.NewNoOpCache())

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "role-test-role",
		GetScope:  sd.String(),
		SetupYAML: RoleYAML,
	}

	st.Execute(t)
}
