package gateway

import (
	securityv1 "github.com/Layer7-Community/layer7-operator/api/v1"

	"github.com/Layer7-Community/layer7-operator/pkg/gateway/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDeployment(gw *securityv1.Gateway) *appsv1.Deployment {
	var image string = gw.Spec.App.Image

	ports := []corev1.ContainerPort{}

	for p := range gw.Spec.App.Service.Ports {
		ports = append(ports, corev1.ContainerPort{
			Name:          gw.Spec.App.Service.Ports[p].Name,
			ContainerPort: gw.Spec.App.Service.Ports[p].Port,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	if gw.Spec.App.Management.Service.Enabled {
		for p := range gw.Spec.App.Management.Service.Ports {
			ports = append(ports, corev1.ContainerPort{
				Name:          gw.Spec.App.Management.Service.Ports[p].Name,
				ContainerPort: gw.Spec.App.Management.Service.Ports[p].Port,
				Protocol:      corev1.ProtocolTCP,
			})
		}
	}

	secretName := gw.Name
	if gw.Spec.App.Management.SecretName != "" {
		secretName = gw.Spec.App.Management.SecretName
	}
	defaultMode := int32(420)
	optional := false
	terminationGracePeriodSeconds := int64(30)
	volumes := []corev1.Volume{{
		Name: "gateway-license",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "gateway-license",
				Items: []corev1.KeyToPath{{
					Path: "license.xml",
					Key:  "license.xml"},
				},
				DefaultMode: &defaultMode,
				Optional:    &optional,
			},
		},
	}, {
		Name: "system-properties",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name + "-system"},
				Items: []corev1.KeyToPath{{
					Path: "system.properties",
					Key:  "system.properties"},
				},
				DefaultMode: &defaultMode,
				Optional:    &optional,
			},
		},
	}}

	volumeMounts := []corev1.VolumeMount{{
		Name:      "gateway-license",
		MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/license/license.xml",
		SubPath:   "license.xml",
	}, {
		Name:      "system-properties",
		MountPath: "/opt/SecureSpan/Gateway/node/default/etc/conf/system.properties",
		SubPath:   "system.properties",
	}}

	if gw.Spec.App.ClusterProperties.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      gw.Name + "-cwp-bundle",
			MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/bundle/" + gw.Name + "-cwp-bundle",
		})

		vs := corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name + "-cwp-bundle"},
			DefaultMode:          &defaultMode,
			Optional:             &optional,
		}}

		vs = corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name + "-cwp-bundle"},
			DefaultMode:          &defaultMode,
			Optional:             &optional,
		}}

		volumes = append(volumes, corev1.Volume{
			Name:         gw.Name + "-cwp-bundle",
			VolumeSource: vs,
		})
	}

	if gw.Spec.App.ListenPorts.Harden {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      gw.Name + "-listen-port-bundle",
			MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/bundle/" + gw.Name + "-listen-port-bundle",
		})

		defaultMode := int32(420)
		optional := false

		vs := corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name + "-listen-port-bundle"},
			DefaultMode:          &defaultMode,
			Optional:             &optional,
		}}

		vs = corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name + "-listen-port-bundle"},
			DefaultMode:          &defaultMode,
			Optional:             &optional,
		}}

		volumes = append(volumes, corev1.Volume{
			Name:         gw.Name + "-listen-port-bundle",
			VolumeSource: vs,
		})
	}

	if gw.Spec.App.Management.Restman.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "restman",
			MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/services/restman",
		})

		volumes = append(volumes, corev1.Volume{
			Name:         "restman",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
	}

	if gw.Spec.App.Management.Graphman.Enabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "graphman",
			MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/services/graphman",
		})
		volumes = append(volumes, corev1.Volume{
			Name:         "graphman",
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		})
	}

	if gw.Spec.App.Hazelcast.External {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "hazelcast-client",
			MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/assertions/ExternalHazelcastSharedStateProviderAssertion/hazelcast-client.xml",
			SubPath:   "hazelcast-client.xml",
		})
		volumes = append(volumes, corev1.Volume{
			Name: "hazelcast-client",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name},
					Items: []corev1.KeyToPath{{
						Path: "hazelcast-client.xml",
						Key:  "hazelcast-client.xml"},
					},
				},
			},
		})
	}

	for v := range gw.Spec.App.Bundle {
		switch gw.Spec.App.Bundle[v].Type {
		case "configMap":
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      gw.Spec.App.Bundle[v].Name,
				MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/bundle/" + gw.Spec.App.Bundle[v].Name,
			})

			defaultMode := int32(420)
			optional := false

			vs := corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: gw.Spec.App.Bundle[v].Name},
				DefaultMode:          &defaultMode,
				Optional:             &optional,
			}}

			if gw.Spec.App.Bundle[v].ConfigMap != (securityv1.ConfigMap{}) {
				vs = corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: gw.Spec.App.Bundle[v].Name},
					DefaultMode:          gw.Spec.App.Bundle[v].ConfigMap.DefaultMode,
					Optional:             &gw.Spec.App.Bundle[v].ConfigMap.Optional,
				}}
			}

			volumes = append(volumes, corev1.Volume{
				Name:         gw.Spec.App.Bundle[v].Name,
				VolumeSource: vs,
			})
		case "secret":
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      gw.Spec.App.Bundle[v].Name,
				MountPath: "/opt/SecureSpan/Gateway/node/default/etc/bootstrap/bundle/" + gw.Spec.App.Bundle[v].Name,
			})

			if gw.Spec.App.Bundle[v].CSI == (securityv1.CSI{}) {
				volumes = append(volumes, corev1.Volume{
					Name: gw.Spec.App.Bundle[v].Name,
					VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{
						SecretName: gw.Spec.App.Bundle[v].Name,
					}},
				})
			} else {
				vs := corev1.CSIVolumeSource{
					Driver:           gw.Spec.App.Bundle[v].CSI.Driver,
					ReadOnly:         &gw.Spec.App.Bundle[v].CSI.ReadOnly,
					VolumeAttributes: map[string]string{"secretProviderClass": gw.Spec.App.Bundle[v].CSI.VolumeAttributes.SecretProviderClass},
				}
				volumes = append(volumes, corev1.Volume{
					Name:         gw.Spec.App.Bundle[v].Name,
					VolumeSource: corev1.VolumeSource{CSI: &vs},
				})
			}
		}
	}

	for vm := range gw.Spec.App.InitContainers {
		volumeMounts = append(volumeMounts, gw.Spec.App.InitContainers[vm].VolumeMounts...)
		for v := range gw.Spec.App.InitContainers[vm].VolumeMounts {
			volumes = append(volumes, corev1.Volume{
				Name: gw.Spec.App.InitContainers[vm].VolumeMounts[v].Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
		}
	}

	for vm := range gw.Spec.App.Sidecars {
		volumeMounts = append(volumeMounts, gw.Spec.App.Sidecars[vm].VolumeMounts...)
		for v := range gw.Spec.App.Sidecars[vm].VolumeMounts {
			volumes = append(volumes, corev1.Volume{
				Name: gw.Spec.App.Sidecars[vm].VolumeMounts[v].Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
		}
	}

	strategy := appsv1.DeploymentStrategy{}

	if gw.Spec.App.UpdateStrategy != (securityv1.UpdateStrategy{}) {
		switch gw.Spec.App.UpdateStrategy.Type {
		case "rollingUpdate":
			strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
			strategy.RollingUpdate = &gw.Spec.App.UpdateStrategy.RollingUpdate
		case "recreate":
			strategy.Type = appsv1.RecreateDeploymentStrategyType
		}

	}

	containers := []corev1.Container{}
	initContainers := gw.Spec.App.InitContainers

	resources := corev1.ResourceRequirements{
		Requests: gw.Spec.App.Resources.Requests,
		Limits:   gw.Spec.App.Resources.Limits,
	}

	if gw.Spec.App.Repository.Enabled && gw.Spec.App.Repository.Method == "init" {
		init := gw.Spec.App.Repository.Init
		env := []corev1.EnvVar{{Name: "GIT_REPO_URL", Value: gw.Spec.App.Repository.URL}, {Name: "BUNDLE_DIR", Value: gw.Spec.App.Repository.BundleDirectory}}
		init.Env = append(init.Env, env...)
		initContainers = append(initContainers, init)
		volumeMounts = append(volumeMounts, init.VolumeMounts...)
		for v := range init.VolumeMounts {
			volumes = append(volumes, corev1.Volume{Name: init.VolumeMounts[v].Name, VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
		}
	}

	gateway := corev1.Container{
		Image:                    image,
		ImagePullPolicy:          corev1.PullPolicy(gw.Spec.App.ImagePullPolicy),
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		Name:                     "gateway",
		EnvFrom: []corev1.EnvFromSource{
			{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: gw.Name},
				}},

			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				}},
		},
		Ports:        ports,
		VolumeMounts: volumeMounts,
		LivenessProbe: &corev1.Probe{

			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/bash", "/opt/docker/rc.d/diagnostic/health_check.sh"},
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      1,
			PeriodSeconds:       10,
			FailureThreshold:    10,
			SuccessThreshold:    1,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/bash", "/opt/docker/rc.d/diagnostic/health_check.sh"},
				},
			},
			InitialDelaySeconds: 30,
			TimeoutSeconds:      1,
			PeriodSeconds:       10,
			FailureThreshold:    10,
			SuccessThreshold:    1,
		},
		Resources: resources,
	}

	containers = append(containers, gateway)
	containers = append(containers, gw.Spec.App.Sidecars...)

	ls := util.DefaultLabels(gw)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gw.Name,
			Namespace: gw.Namespace,
			Labels:    ls,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName:            gw.Spec.App.ServiceAccountName,
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					SecurityContext:               &corev1.PodSecurityContext{},
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					InitContainers:                initContainers,
					Containers:                    containers,
					Volumes:                       volumes,
				},
			},
		},
	}

	dep.Spec.Template.Spec.ImagePullSecrets = append(dep.Spec.Template.Spec.ImagePullSecrets, gw.Spec.App.ImagePullSecrets...)
	dep.Spec.Template.Labels = ls

	if gw.Spec.App.Repository.Enabled {
		dep.Spec.Template.Annotations = map[string]string{"commitId": gw.Status.CommitID}
	}

	if !gw.Spec.App.Autoscaling.Enabled {
		dep.Spec.Replicas = &gw.Spec.App.Replicas
	}

	return dep
}
