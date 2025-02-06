package adapters

import (
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var persistentVolumeClaimYAML = `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-test-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pvc-test-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: /tmp/pvc-test-pv
---
apiVersion: v1
kind: Pod
metadata:
  name: pvc-test-pod
spec:
  containers:
  - name: pvc-test-container
    image: nginx
    volumeMounts:
    - name: pvc-test-volume
      mountPath: /data
  volumes:
  - name: pvc-test-volume
    persistentVolumeClaim:
      claimName: pvc-test-pvc
`

func TestPersistentVolumeClaimAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newPersistentVolumeClaimAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "pvc-test-pvc",
		GetScope:  sd.String(),
		SetupYAML: persistentVolumeClaimYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:         "PersistentVolume",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedQueryMatches: regexp.MustCompile("pvc-"),
				ExpectedScope:        sd.String(),
			},
		},
		Wait: func(item *sdp.Item) bool {
			phase, _ := item.GetAttributes().Get("status.phase")

			return phase != "Pending"
		},
	}

	st.Execute(t)
}
