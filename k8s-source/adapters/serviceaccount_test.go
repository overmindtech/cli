package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var serviceAccountYAML = `
apiVersion: v1
kind: Secret
metadata:
  name: service-account-secret
type: Opaque
data:
  username: Zm9vCg==
  password: Zm9vCg==
---
apiVersion: v1
kind: Secret
metadata:
  name: service-account-secret-pull
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: eyJhdXRocyI6eyJnaGNyLmlvIjp7InVzZXJuYW1lIjoiaHVudGVyIiwicGFzc3dvcmQiOiJodW50ZXIyIiwiZW1haWwiOiJmb29AYmFyLmNvbSIsImF1dGgiOiJhSFZ1ZEdWeU9taDFiblJsY2pJPSJ9fX0=
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-service-account
secrets:
- name: service-account-secret
imagePullSecrets:
- name: service-account-secret-pull
`

func TestServiceAccountAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newServiceAccountAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "test-service-account",
		GetScope:  sd.String(),
		SetupYAML: serviceAccountYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "Secret",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "service-account-secret",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "Secret",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "service-account-secret-pull",
				ExpectedScope:  sd.String(),
			},
		},
	}

	st.Execute(t)
}
