package adapters

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
)

var PodYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: pod-test-serviceaccount
---
apiVersion: v1
kind: Secret
metadata:
  name: pod-test-secret
type: Opaque
data:
  username: dXNlcm5hbWU=
  password: cGFzc3dvcmQ=
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: pod-test-configmap
data:
  config.ini: |
    [database]
    host=example.com
    port=5432
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: pod-test-configmap-cert
data:
  ca.pem: |
    wow such cert
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pod-test-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: pod-test-pod
spec:
  serviceAccountName: pod-test-serviceaccount
  volumes:
  - name: pod-test-pvc-volume
    persistentVolumeClaim:
      claimName: pod-test-pvc
  - name: database-config
    configMap:
      name: pod-test-configmap
  - name: projected-config
    projected:
      sources:
        - configMap:
            name: pod-test-configmap-cert
            items:
              - key: ca.pem
                path: ca.pem
  containers:
  - name: pod-test-container
    image: nginx
    volumeMounts:
    - name: pod-test-pvc-volume
      mountPath: /mnt/data
    - name: database-config
      mountPath: /etc/database
    - name: projected-config
      mountPath: /etc/projected
    envFrom:
    - secretRef:
        name: pod-test-secret
---
apiVersion: v1
kind: Pod
metadata:
  name: pod-bad-pod
spec:
  serviceAccountName: pod-test-serviceaccount
  volumes:
  - name: pod-test-pvc-volume
    persistentVolumeClaim:
      claimName: pod-test-pvc
  - name: database-config
    configMap:
      name: pod-test-configmap
  - name: projected-config
    projected:
      sources:
        - configMap:
            name: pod-test-configmap-cert
            items:
              - key: ca.pem
                path: ca.pem
  containers:
  - name: pod-test-container
    image: nginx:this-tag-does-not-exist
    volumeMounts:
    - name: pod-test-pvc-volume
      mountPath: /mnt/data
    - name: database-config
      mountPath: /etc/database
    - name: projected-config
      mountPath: /etc/projected
    envFrom:
    - secretRef:
        name: pod-test-secret
`

func TestPodAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newPodAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace})

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "pod-test-pod",
		GetScope:  sd.String(),
		SetupYAML: PodYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedQueryMatches: regexp.MustCompile(`10\.`),
				ExpectedType:         "ip",
				ExpectedMethod:       sdp.QueryMethod_GET,
				ExpectedScope:        "global",
			},
			{
				ExpectedType:   "ServiceAccount",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "pod-test-serviceaccount",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "Secret",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "pod-test-secret",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "ConfigMap",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "pod-test-configmap",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "PersistentVolumeClaim",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "pod-test-pvc",
				ExpectedScope:  sd.String(),
			},
			{
				ExpectedType:   "ConfigMap",
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "pod-test-configmap-cert",
				ExpectedScope:  sd.String(),
			},
		},
		Wait: func(item *sdp.Item) bool {
			return len(item.GetLinkedItemQueries()) >= 9
		},
	}

	st.Execute(t)
	// the pods are still running let check their health

	// get the bad pod
	item, err := adapter.Get(context.Background(), sd.String(), "pod-bad-pod", true)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get pod: %w", err))
	}
	if item.GetHealth() != sdp.Health_HEALTH_ERROR {
		t.Errorf("expected status to be unhealthy, got %s", item.GetHealth())
	}
	// get the healthy pod
	item, err = adapter.Get(context.Background(), sd.String(), "pod-test-pod", true)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to get pod: %w", err))
	}
	if item.GetHealth() != sdp.Health_HEALTH_OK {
		t.Errorf("expected status to be healthy, got %s", item.GetHealth())
	}
}

func TestHasWaitingContainerErrors(t *testing.T) {
	tests := []struct {
		name              string
		containerStatuses []v1.ContainerStatus
		expectedResult    bool
	}{
		{
			name: "No waiting containers",
			containerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Waiting container with non-error reason",
			containerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "Waiting container with error reason",
			containerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Multiple containers with one error",
			containerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ImagePullBackOff",
						},
					},
				},
			},
			expectedResult: true,
		},
		{
			name: "Multiple containers with no errors",
			containerStatuses: []v1.ContainerStatus{
				{
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
				},
				{
					State: v1.ContainerState{
						Waiting: &v1.ContainerStateWaiting{
							Reason: "ContainerCreating",
						},
					},
				},
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasWaitingContainerErrors(tt.containerStatuses)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}
