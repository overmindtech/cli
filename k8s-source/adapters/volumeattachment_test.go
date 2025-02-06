package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

var volumeAttachmentYAML = `
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: volume-attachment-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: standard
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: volume-attachment-pv
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /data
---
apiVersion: v1
kind: Pod
metadata:
  name: volume-attachment-pod
spec:
  containers:
  - name: volume-attachment-container
    image: nginx
    volumeMounts:
    - name: volume-attachment-volume
      mountPath: /data
  volumes:
  - name: volume-attachment-volume
    persistentVolumeClaim:
      claimName: volume-attachment-pvc
---
apiVersion: storage.k8s.io/v1
kind: VolumeAttachment
metadata:
  name: volume-attachment-attachment
spec:
  nodeName: local-tests-control-plane
  attacher: kubernetes.io
  source:
    persistentVolumeName: volume-attachment-pv

`

func TestVolumeAttachmentAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
	}

	adapter := newVolumeAttachmentAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "volume-attachment-attachment",
		GetScope:  sd.String(),
		SetupYAML: volumeAttachmentYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "PersistentVolume",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "volume-attachment-pv",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "Node",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "local-tests-control-plane",
				ExpectedScope:  sd.String(),
			},
		},
	}

	st.Execute(t)
}
