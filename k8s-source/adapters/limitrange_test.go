package adapters

import (
	"testing"
)

var limitRangeYAML = `
apiVersion: v1
kind: LimitRange
metadata:
  name: example-limit-range
spec:
  limits:
  - type: Pod
    max:
      memory: 200Mi
    min:
      cpu: 50m
  - type: Container
    max:
      memory: 150Mi
      cpu: 100m
    min:
      memory: 50Mi
      cpu: 50m
    default:
      memory: 100Mi
      cpu: 50m
    defaultRequest:
      memory: 80Mi
      cpu: 50m
`

func TestLimitRangeAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newLimitRangeAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "example-limit-range",
		GetScope:      sd.String(),
		SetupYAML:     limitRangeYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)
}
