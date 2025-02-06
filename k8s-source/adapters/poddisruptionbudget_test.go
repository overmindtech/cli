package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var PodDisruptionBudgetYAML = `
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: example-pdb
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: example-app
`

func TestPodDisruptionBudgetAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newPodDisruptionBudgetAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "example-pdb",
		GetScope:  sd.String(),
		SetupYAML: PodDisruptionBudgetYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile("app=example-app"),
				ExpectedType:         "Pod",
				ExpectedMethod:       sdp.QueryMethod_SEARCH,
				ExpectedScope:        sd.String(),
			},
		},
	}

	st.Execute(t)
}
