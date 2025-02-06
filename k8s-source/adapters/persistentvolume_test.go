package adapters

import (
	"testing"
)

var persistentVolumeYAML = `
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-test-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /tmp/pv-test-pv
`

func TestPersistentVolumeAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "",
	}

	adapter := newPersistentVolumeAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "pv-test-pv",
		GetScope:      sd.String(),
		SetupYAML:     persistentVolumeYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)
}
