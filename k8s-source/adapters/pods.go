package adapters

import (
	"net"
	"slices"
	"time"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func PodExtractor(resource *v1.Pod, scope string) ([]*sdp.LinkedItemQuery, error) {
	queries := make([]*sdp.LinkedItemQuery, 0)

	sd, err := ParseScope(scope, true)

	if err != nil {
		return nil, err
	}

	// Link service accounts
	if resource.Spec.ServiceAccountName != "" {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ServiceAccount",
				Scope:  scope,
				Method: sdp.QueryMethod_GET,
				Query:  resource.Spec.ServiceAccountName,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changes to the service account can affect the pod
				In: true,
				// Changes to the pod cannot affect the service account
				Out: false,
			},
		})
	}

	// Link items from volumes
	for _, vol := range resource.Spec.Volumes {
		// Link PVCs
		if vol.PersistentVolumeClaim != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  scope,
					Method: sdp.QueryMethod_GET,
					Query:  vol.PersistentVolumeClaim.ClaimName,
					Type:   "PersistentVolumeClaim",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the PVC will affect the pod
					In: true,
					// The pod can definitely affect the PVC, by filling it up
					// for example
					Out: true,
				},
			})
		}

		// Link to EBS volumes
		if vol.AWSElasticBlockStore != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  "*",
					Method: sdp.QueryMethod_GET,
					Query:  vol.AWSElasticBlockStore.VolumeID,
					Type:   "ec2-volume",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changes to the volume will affect the pod
					In: true,
					// The pod can definitely affect the volume
					Out: true,
				},
			})
		}

		// Link secrets
		if vol.Secret != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  scope,
					Method: sdp.QueryMethod_GET,
					Query:  vol.Secret.SecretName,
					Type:   "Secret",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the secret could easily break the pod
					In: true,
					// The pod however isn't going to affect a secret
					Out: false,
				},
			})
		}

		if vol.NFS != nil {
			// This is either the hostname or IP of the NFS server so we can
			// link to that. We'll try to parse the IP and if not fall back to
			// DNS for the hostname
			if net.ParseIP(vol.NFS.Server) != nil {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Scope:  "global",
						Method: sdp.QueryMethod_GET,
						Query:  vol.NFS.Server,
						Type:   "ip",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// NFS server can affect the pod
						In: true,
						// Pod can't affect the NFS server
						Out: false,
					},
				})
			} else {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Scope:  "global",
						Method: sdp.QueryMethod_SEARCH,
						Type:   "dns",
						Query:  vol.NFS.Server,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// NFS server can affect the pod
						In: true,
						// Pod can't affect the NFS server
						Out: false,
					},
				})
			}
		}

		// Link config map volumes
		if vol.ConfigMap != nil {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  scope,
					Method: sdp.QueryMethod_GET,
					Query:  vol.ConfigMap.Name,
					Type:   "ConfigMap",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the config map could easily break the pod
					In: true,
					// The pod however isn't going to affect a config map
					Out: false,
				},
			})
		}

		// Link projected volumes
		if vol.Projected != nil {
			for _, source := range vol.Projected.Sources {
				if source.ConfigMap != nil {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Scope:  scope,
							Method: sdp.QueryMethod_GET,
							Query:  source.ConfigMap.Name,
							Type:   "ConfigMap",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the config map could easily break the pod
							In: true,
							// The pod however isn't going to affect a config map
							Out: false,
						},
					})
				}

				if source.Secret != nil {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Scope:  scope,
							Method: sdp.QueryMethod_GET,
							Query:  source.Secret.Name,
							Type:   "Secret",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the secret could easily break the pod
							In: true,
							// The pod however isn't going to affect a secret
							Out: false,
						},
					})
				}
			}
		}
	}

	// Link items from containers
	for _, container := range resource.Spec.Containers {
		// Loop over environment variables
		for _, env := range container.Env {
			if env.ValueFrom != nil {
				if env.ValueFrom.SecretKeyRef != nil {
					// Add linked item from spec.containers[].env[].valueFrom.secretKeyRef
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Scope:  scope,
							Method: sdp.QueryMethod_GET,
							Query:  env.ValueFrom.SecretKeyRef.Name,
							Type:   "Secret",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the secret could easily break the pod
							In: true,
							// The pod however isn't going to affect a secret
							Out: false,
						},
					})
				}

				if env.ValueFrom.ConfigMapKeyRef != nil {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Scope:  scope,
							Method: sdp.QueryMethod_GET,
							Query:  env.ValueFrom.ConfigMapKeyRef.Name,
							Type:   "ConfigMap",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Changing the config map could easily break the pod
							In: true,
							// The pod however isn't going to affect a config map
							Out: false,
						},
					})
				}
			}
		}

		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil {
				// Add linked item from spec.containers[].EnvFrom[].secretKeyRef
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Scope:  scope,
						Method: sdp.QueryMethod_GET,
						Query:  envFrom.SecretRef.Name,
						Type:   "Secret",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the secret could easily break the pod
						In: true,
						// The pod however isn't going to affect a secret
						Out: false,
					},
				})
			}

			if envFrom.ConfigMapRef != nil {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Scope:  scope,
						Method: sdp.QueryMethod_GET,
						Query:  envFrom.ConfigMapRef.Name,
						Type:   "ConfigMap",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Changing the config map could easily break the pod
						In: true,
						// The pod however isn't going to affect a config map
						Out: false,
					},
				})
			}
		}
	}

	if resource.Spec.PriorityClassName != "" {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Scope:  sd.ClusterName,
				Method: sdp.QueryMethod_GET,
				Query:  resource.Spec.PriorityClassName,
				Type:   "PriorityClass",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Changing the priority class could break a pod by meaning that
				// it would now be scheduled with a lower priority and could
				// therefore end up pending for ages
				In: true,
				// The pod however isn't going to affect a priority class
				Out: false,
			},
		})
	}

	if len(resource.Status.PodIPs) > 0 {
		for _, ip := range resource.Status.PodIPs {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Scope:  "global",
					Method: sdp.QueryMethod_GET,
					Query:  ip.IP,
					Type:   "ip",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// IPs go in both directions
					In:  true,
					Out: true,
				},
			})
		}
	} else if resource.Status.PodIP != "" {
		queries = append(queries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ip",
				Method: sdp.QueryMethod_GET,
				Query:  resource.Status.PodIP,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// IPs go in both directions
				In:  true,
				Out: true,
			},
		})
	}

	return queries, nil
}

func newPodAdapter(cs *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter {
	return &KubeTypeAdapter[*v1.Pod, *v1.PodList]{
		ClusterName:      cluster,
		Namespaces:       namespaces,
		TypeName:         "Pod",
		CacheDuration:    10 * time.Minute, // somewhat low since pods are replaced a lot
		AutoQueryExtract: true,
		NamespacedInterfaceBuilder: func(namespace string) ItemInterface[*v1.Pod, *v1.PodList] {
			return cs.CoreV1().Pods(namespace)
		},
		ListExtractor: func(list *v1.PodList) ([]*v1.Pod, error) {
			extracted := make([]*v1.Pod, len(list.Items))

			for i := range list.Items {
				extracted[i] = &list.Items[i]
			}

			return extracted, nil
		},
		LinkedItemQueryExtractor: PodExtractor,
		HealthExtractor: func(resource *v1.Pod) *sdp.Health {
			switch resource.Status.Phase {
			case v1.PodPending:
				//  a special case were the pod has never actually started
				if hasWaitingContainerErrors(resource.Status.ContainerStatuses) {
					return sdp.Health_HEALTH_ERROR.Enum()
				}
				return sdp.Health_HEALTH_PENDING.Enum()
			case v1.PodRunning, v1.PodSucceeded:
				// a special case were the pod has started but it was modified
				if hasWaitingContainerErrors(resource.Status.ContainerStatuses) {
					return sdp.Health_HEALTH_ERROR.Enum()
				}
				return sdp.Health_HEALTH_OK.Enum()
			case v1.PodFailed:
				return sdp.Health_HEALTH_ERROR.Enum()
			case v1.PodUnknown:
				return sdp.Health_HEALTH_UNKNOWN.Enum()
			}

			return nil
		},
		AdapterMetadata: podAdapterMetadata,
	}
}

// a pod's status phase can be ok, but the container may not be ok
// this is a check for the container statuses
// hasWaitingContainerErrors returns true if any of the container statuses are in a waiting state with an error reason
func hasWaitingContainerErrors(containerStatuses []v1.ContainerStatus) bool {
	for _, c := range containerStatuses {
		if c.State.Waiting != nil {
			// list of image errors from https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/images/types.go#L27-L42
			if slices.Contains([]string{"CrashLoopBackOff", "ImagePullBackOff", "ImageInspectError", "ErrImagePull", "ErrImageNeverPull", "InvalidImageName"}, c.State.Waiting.Reason) {
				return true
			}
		}
	}
	return false
}

var podAdapterMetadata = Metadata.Register(&sdp.AdapterMetadata{
	Type:            "Pod",
	DescriptiveName: "Pod",
	Category:        sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
	PotentialLinks: []string{
		"ConfigMap",
		"ec2-volume",
		"dns",
		"ip",
		"PersistentVolumeClaim",
		"PriorityClass",
		"Secret",
		"ServiceAccount",
	},
	SupportedQueryMethods: DefaultSupportedQueryMethods("Pod"),
	TerraformMappings: []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_pod.metadata[0].name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "kubernetes_pod_v1.metadata[0].name",
		},
	},
})

func init() {
	registerAdapterLoader(newPodAdapter)
}
