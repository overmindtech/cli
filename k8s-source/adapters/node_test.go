package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

func TestNodeAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newNodeAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:  adapter,
		GetQuery: "local-tests-control-plane",
		GetScope: sd.String(),
		GetQueryTests: QueryTests{
			{
				ExpectedType:         "ip",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedScope:        "global",
				ExpectedQueryMatches: regexp.MustCompile(`172\.`),
			},
		},
	}

	st.Execute(t)
}
