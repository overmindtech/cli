package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var NetworkPolicyYAML = `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-nginx
spec:
  podSelector:
    matchLabels:
      app: nginx
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: frontend
    ports:
    - protocol: TCP
      port: 80
`

func TestNetworkPolicyAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newNetworkPolicyAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "allow-nginx",
		GetScope:  sd.String(),
		SetupYAML: NetworkPolicyYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile("nginx"),
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedScope:        sd.String(),
			},
		},
	}

	st.Execute(t)
}
