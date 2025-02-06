package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var roleBindingYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rb-test-service-account
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: rb-test-role
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-test-role-binding
subjects:
- kind: ServiceAccount
  name: rb-test-service-account
  namespace: default
roleRef:
  kind: Role
  name: rb-test-role
  apiGroup: rbac.authorization.k8s.io
---
`

var roleBindingYAML2 = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rb-test-service-account2
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: rb-test-cluster-role
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "watch", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: rb-test-role-binding-cluster
  namespace: default
roleRef:
  kind: ClusterRole
  name: rb-test-cluster-role
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: ServiceAccount
  name: rb-test-service-account2
  namespace: default
`

func TestRoleBindingAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newRoleBindingAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	t.Run("With a Role", func(t *testing.T) {
		st := AdapterTests{
			Adapter:   adapter,
			GetQuery:  "rb-test-role-binding",
			GetScope:  sd.String(),
			SetupYAML: roleBindingYAML,
			GetQueryTests: QueryTests{
				{
					ExpectedType:   "Role",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "rb-test-role",
					ExpectedScope:  sd.String(),
				},
				{
					ExpectedType:   "ServiceAccount",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "rb-test-service-account",
					ExpectedScope:  sd.String(),
				},
			},
		}

		st.Execute(t)
	})

	t.Run("With a ClusterRole", func(t *testing.T) {
		st := AdapterTests{
			Adapter:   adapter,
			GetQuery:  "rb-test-role-binding-cluster",
			GetScope:  sd.String(),
			SetupYAML: roleBindingYAML2,
			GetQueryTests: QueryTests{
				{
					ExpectedType:   "ClusterRole",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "rb-test-cluster-role",
					ExpectedScope:  sd.ClusterName,
				},
				{
					ExpectedType:   "ServiceAccount",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "rb-test-service-account2",
					ExpectedScope:  sd.String(),
				},
			},
		}

		st.Execute(t)
	})

}
