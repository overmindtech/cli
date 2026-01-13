package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdpcache"
)

var secretYAML = `
apiVersion: v1
kind: Secret
metadata:
  name: secret-test-secret
type: Opaque
data:
  username: dXNlcm5hbWUx   # base64-encoded "username1"
  password: cGFzc3dvcmQx   # base64-encoded "password1"

`

func TestSecretAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newSecretAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace}, sdpcache.NewNoOpCache())

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "secret-test-secret",
		GetScope:  sd.String(),
		SetupYAML: secretYAML,
	}

	st.Execute(t)
}
