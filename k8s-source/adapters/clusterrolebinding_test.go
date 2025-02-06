package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var clusterRoleBindingYAML = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admin-binding
subjects:
- kind: Group
  name: system:serviceaccounts:default
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: admin
  apiGroup: rbac.authorization.k8s.io
`

func TestClusterRoleBindingAdapter(t *testing.T) {
	adapter := newClusterRoleBindingAdapter(CurrentCluster.ClientSet, CurrentCluster.Name, []string{})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "admin-binding",
		GetScope:  CurrentCluster.Name,
		SetupYAML: clusterRoleBindingYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "ClusterRole",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "admin",
				ExpectedScope:  CurrentCluster.Name,
			},
			{
				ExpectedType:   "Group",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "system:serviceaccounts:default",
				ExpectedScope:  CurrentCluster.Name,
			},
		},
	}

	st.Execute(t)
}
