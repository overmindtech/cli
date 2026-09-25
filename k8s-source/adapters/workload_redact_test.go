package adapters

import (
	"encoding/json"
	"testing"

	"github.com/overmindtech/cli/go/sdpcache"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	workloadEnvSentinel     = "ovm-k8s-workload-env-sentinel-4e1d"
	workloadInitEnvSentinel = "ovm-k8s-workload-init-env-sentinel-7c2b"
	workloadEphEnvSentinel  = "ovm-k8s-workload-eph-env-sentinel-9f3a"
	workloadEnvName         = "DATABASE_PASSWORD"
	workloadInitEnvName     = "INIT_TOKEN"
	workloadEphEnvName      = "DEBUG_TOKEN"
	workloadSecretEnvName   = "API_KEY"
	workloadSecretRefName   = "api-secret"
)

func testPodSpecWithEnvSecrets() corev1.PodSpec {
	return corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "app",
				Image: "example.invalid/app:latest",
				Env: []corev1.EnvVar{
					{
						Name:  workloadEnvName,
						Value: workloadEnvSentinel,
					},
					{
						Name: workloadSecretEnvName,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								Name: workloadSecretRefName,
								Key:  "token",
							},
						},
					},
				},
			},
		},
		InitContainers: []corev1.Container{
			{
				Name: "init",
				Env: []corev1.EnvVar{
					{
						Name:  workloadInitEnvName,
						Value: workloadInitEnvSentinel,
					},
				},
			},
		},
		EphemeralContainers: []corev1.EphemeralContainer{
			{
				Name: "debug",
				Env: []corev1.EnvVar{
					{
						Name:  workloadEphEnvName,
						Value: workloadEphEnvSentinel,
					},
				},
			},
		},
	}
}

func TestRedactWorkloadEnvValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource any
	}{
		{
			name: "Pod",
			resource: &corev1.Pod{
				Name: "pod",
				Spec: testPodSpecWithEnvSecrets(),
			},
		},
		{
			name: "Deployment",
			resource: &appsv1.Deployment{
				Name: "deploy",
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
		{
			name: "ReplicaSet",
			resource: &appsv1.ReplicaSet{
				Name: "rs",
				Spec: appsv1.ReplicaSetSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
		{
			name: "StatefulSet",
			resource: &appsv1.StatefulSet{
				Name: "sts",
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
		{
			name: "DaemonSet",
			resource: &appsv1.DaemonSet{
				Name: "ds",
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
		{
			name: "Job",
			resource: &batchv1.Job{
				Name: "job",
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
		{
			name: "CronJob",
			resource: &batchv1.CronJob{
				Name: "cron",
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
						},
					},
				},
			},
		},
		{
			name: "ReplicationController",
			resource: &corev1.ReplicationController{
				Name: "rc",
				Spec: corev1.ReplicationControllerSpec{
					Template: &corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			redactWorkloadEnvValues(tt.resource)

			b, err := json.Marshal(tt.resource)
			if err != nil {
				t.Fatalf("marshal resource: %v", err)
			}
			attrsJSON := string(b)

			for _, sentinel := range []string{workloadEnvSentinel, workloadInitEnvSentinel, workloadEphEnvSentinel} {
				if containsJSONStringValue(attrsJSON, sentinel) {
					t.Errorf("plaintext env value %q leaked in %s attributes: %s", sentinel, tt.name, attrsJSON)
				}
			}
			for _, name := range []string{workloadEnvName, workloadInitEnvName, workloadEphEnvName, workloadSecretEnvName, workloadSecretRefName} {
				if !containsJSONStringValue(attrsJSON, name) {
					t.Errorf("expected %q to remain in %s attributes: %s", name, tt.name, attrsJSON)
				}
			}
		})
	}
}

func TestRedactWorkloadEnvValues_nilReplicationControllerTemplate(t *testing.T) {
	t.Parallel()

	rc := &corev1.ReplicationController{
		Name: "rc",
	}
	redactWorkloadEnvValues(rc)
	if rc.Name != "rc" {
		t.Errorf("expected name to remain %q, got %q", "rc", rc.Name)
	}
}

func TestRedactWorkloadEnvValues_unknownType(t *testing.T) {
	t.Parallel()

	secret := &corev1.Secret{
		Name: "s",
		Data: map[string][]byte{"password": []byte("keep-me")},
	}
	redactWorkloadEnvValues(secret)
	if string(secret.Data["password"]) != "keep-me" {
		t.Errorf("expected unknown types to be left unchanged, got %q", secret.Data["password"])
	}
}

func TestResourceToItemRedactsCronJobEnv(t *testing.T) {
	t.Parallel()

	adapter := &KubeTypeAdapter[*batchv1.CronJob, *batchv1.CronJobList]{
		ClusterName: "minikube",
		TypeName:    "CronJob",
		cache:       sdpcache.NewNoOpCache(),
		ListExtractor: func(list *batchv1.CronJobList) ([]*batchv1.CronJob, error) {
			return nil, nil
		},
	}

	item, err := adapter.resourceToItem(&batchv1.CronJob{
		Name: "cron", Namespace: "default",
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{Spec: testPodSpecWithEnvSecrets()},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("resourceToItem: %v", err)
	}

	attrMap := item.GetAttributes().GetAttrStruct().AsMap()
	attrsJSON, marshalErr := json.Marshal(attrMap)
	if marshalErr != nil {
		t.Fatalf("marshal attributes: %v", marshalErr)
	}
	for _, sentinel := range []string{workloadEnvSentinel, workloadInitEnvSentinel, workloadEphEnvSentinel} {
		if containsJSONStringValue(string(attrsJSON), sentinel) {
			t.Errorf("plaintext env value %q leaked in CronJob attributes: %s", sentinel, attrsJSON)
		}
	}
	for _, name := range []string{workloadEnvName, workloadSecretEnvName, workloadSecretRefName} {
		if !containsJSONStringValue(string(attrsJSON), name) {
			t.Errorf("expected %q to remain in CronJob attributes: %s", name, attrsJSON)
		}
	}
}

func containsJSONStringValue(attrsJSON, value string) bool {
	var decoded any
	if err := json.Unmarshal([]byte(attrsJSON), &decoded); err != nil {
		return false
	}
	return anyContainsStringValue(decoded, value)
}

func anyContainsStringValue(v any, value string) bool {
	switch x := v.(type) {
	case string:
		return x == value
	case map[string]any:
		for _, child := range x {
			if anyContainsStringValue(child, value) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if anyContainsStringValue(child, value) {
				return true
			}
		}
	}
	return false
}
