package adapters

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

// redactWorkloadEnvValues clears plaintext container environment values on
// Kubernetes workload objects before they are serialized to SDP attributes.
// Variable names and valueFrom secret/configmap references are kept so linked
// item queries still work. Unknown types are a no-op.
func redactWorkloadEnvValues(resource any) {
	switch r := resource.(type) {
	case *corev1.Pod:
		redactPodSpecEnvValues(&r.Spec)
	case *appsv1.Deployment:
		redactPodSpecEnvValues(&r.Spec.Template.Spec)
	case *appsv1.ReplicaSet:
		redactPodSpecEnvValues(&r.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		redactPodSpecEnvValues(&r.Spec.Template.Spec)
	case *appsv1.DaemonSet:
		redactPodSpecEnvValues(&r.Spec.Template.Spec)
	case *batchv1.Job:
		redactPodSpecEnvValues(&r.Spec.Template.Spec)
	case *batchv1.CronJob:
		redactPodSpecEnvValues(&r.Spec.JobTemplate.Spec.Template.Spec)
	case *corev1.ReplicationController:
		redactPodTemplateEnvValues(r.Spec.Template)
	}
}

func redactPodTemplateEnvValues(template *corev1.PodTemplateSpec) {
	if template == nil {
		return
	}
	redactPodSpecEnvValues(&template.Spec)
}

func redactPodSpecEnvValues(spec *corev1.PodSpec) {
	if spec == nil {
		return
	}
	redactContainerEnv(spec.Containers)
	redactContainerEnv(spec.InitContainers)
	for i := range spec.EphemeralContainers {
		redactEnvVars(spec.EphemeralContainers[i].Env)
	}
}

func redactContainerEnv(containers []corev1.Container) {
	for i := range containers {
		redactEnvVars(containers[i].Env)
	}
}

func redactEnvVars(env []corev1.EnvVar) {
	for i := range env {
		env[i].Value = ""
	}
}
