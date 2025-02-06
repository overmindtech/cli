package adapters

import (
	"testing"
)

var storageClassYAML = `
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: storage-class-test
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2

`

func TestStorageClassAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newStorageClassAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "storage-class-test",
		GetScope:  sd.String(),
		SetupYAML: storageClassYAML,
	}

	st.Execute(t)
}
