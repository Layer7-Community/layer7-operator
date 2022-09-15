/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GatewaySpec defines the desired state of Gateway
type GatewaySpec struct {
	License License `json:"license,omitempty"`
	App     App     `json:"app,omitempty"`
	Version string  `json:"version,omitempty"`
}

// GatewayStatus defines the observed state of Gateway
type GatewayStatus struct {
	Host               string                       `json:"host,omitempty"`
	Conditions         []appsv1.DeploymentCondition `json:"conditions,omitempty"`
	Phase              corev1.PodPhase              `json:"phase,omitempty"`
	Gateway            []GatewayState               `json:"gateway,omitempty"`
	ObservedGeneration int64                        `json:"observedGeneration,omitempty"`
	CommitID           string                       `json:"commitId,omitempty"`
	Ready              int32                        `json:"ready,omitempty"`
	State              string                       `json:"state,omitempty"`
	Replicas           int32                        `json:"replicas,omitempty"`
	Version            string                       `json:"version,omitempty"`
	Image              string                       `json:"image,omitempty"`
	LabelSelectorPath  string                       `json:"labelSelectorPath,omitempty"`
	ManagementPod      string                       `json:"managementPod,omitempty"`
}

type GatewayContainerState struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Gateway is the Schema for the gateways API
type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewaySpec   `json:"spec,omitempty"`
	Status GatewayStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GatewayList contains a list of Gateway
type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

type GatewayState struct {
	Name         string          `json:"name,omitempty"`
	Phase        corev1.PodPhase `json:"phase,omitempty"`
	ResponseTime string          `json:"responseTime,omitempty"`
	Ready        bool            `json:"ready"`
	StartTime    string          `json:"startTime,omitempty"`
}

type Management struct {
	SecretName string   `json:"secretName,omitempty"`
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	Cluster    Cluster  `json:"cluster,omitempty"`
	Database   Database `json:"database,omitempty"`
	Restman    Restman  `json:"restman,omitempty"`
	Graphman   Graphman `json:"graphman,omitempty"`
	Service    Service  `json:"service,omitempty"`
}

type Restman struct {
	Enabled bool `json:"enabled,omitempty"`
}

type Graphman struct {
	Enabled bool `json:"enabled,omitempty"`
}

type Bundle struct {
	Type      string    `json:"type,omitempty"`
	Name      string    `json:"name,omitempty"`
	ConfigMap ConfigMap `json:"configMap,omitempty"`
	CSI       CSI       `json:"csi,omitempty"`
}

type ConfigMap struct {
	DefaultMode *int32 `json:"defaultMode,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Name        string `json:"name,omitempty"`
}

type CSI struct {
	Driver           string `json:"driver,omitempty"`
	ReadOnly         bool   `json:"readOnly,omitempty"`
	VolumeAttributes `json:"volumeAttributes,omitempty"`
}

type VolumeAttributes struct {
	SecretProviderClass string `json:"secretProviderClass,omitempty"`
}

type License struct {
	Accept     string `json:"accept,omitempty"`
	SecretName string `json:"secretName,omitempty"`
}

type Database struct {
	Enabled  bool   `json:"enabled"`
	JDBCUrl  string `json:"jdbcUrl,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Image struct {
	Registry   string `json:"registry"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

type App struct {
	Annotations        map[string]string             `json:"annotations,omitempty"`
	ClusterProperties  ClusterProperties             `json:"cwp,omitempty"`
	Java               Java                          `json:"java,omitempty"`
	Management         Management                    `json:"management,omitempty"`
	System             System                        `json:"system,omitempty"`
	UpdateStrategy     UpdateStrategy                `json:"updateStrategy,omitempty"`
	Image              string                        `json:"image,omitempty"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	ImagePullPolicy    corev1.PullPolicy             `json:"imagePullPolicy,omitempty"`
	ListenPorts        ListenPorts                   `json:"listenPorts,omitempty"`
	Replicas           int32                         `json:"replicas,omitempty"`
	Service            Service                       `json:"service,omitempty"`
	Bundle             []Bundle                      `json:"bundle,omitempty"`
	Repository         Repository                    `json:"repository,omitempty"`
	Ingress            Ingress                       `json:"ingress,omitempty"`
	Sidecars           []corev1.Container            `json:"sidecars,omitempty"`
	InitContainers     []corev1.Container            `json:"initContainers,omitempty"`
	Resources          PodResources                  `json:"resources,omitempty"`
	Autoscaling        Autoscaling                   `json:"autoscaling,omitempty"`
	ServiceAccountName string                        `json:"serviceAccountName,omitempty"`
	Hazelcast          Hazelcast                     `json:"hazelcast,omitempty"`
}

type ClusterProperties struct {
	Enabled    bool              `json:"enabled,omitempty"`
	Properties []ClusterProperty `json:"properties,omitempty"`
}

type ClusterProperty struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// Layer7 Gateway instantiates the following HTTP(s) ports by default
// Harden applies the following changes. These values are currently hardcoded and will be more configurable in the future
// - 8080 (HTTP)
//   - Disable
//   - Allow Published Service Message input only
// - 8443 (HTTPS)
//   - Remove Management Features (no Policy Manager Access)
//   - Enables TLSv1.2,TLS1.3 only
//   - Disables insecure Cipher Suites
// - 9443 (HTTPS)
//   - Enables TLSv1.2,TLS1.3 only
//   - Disables insecure Cipher Suites
// - 2124 (Internode communication)
//   - No changes
type ListenPorts struct {
	Harden       bool     `json:"harden,omitempty"`
	CipherSuites []string `json:"cipherSuites,omitempty"`
	TlsVersions  []string `json:"tlsVersions,omitempty"`
}

type Hazelcast struct {
	External bool   `json:"external,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
}

type UpdateStrategy struct {
	Type          string                         `json:"type,omitempty"`
	RollingUpdate appsv1.RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

type Autoscaling struct {
	Enabled bool `json:"enabled,omitempty"`
	HPA     HPA  `json:"hpa,omitempty"`
}

type HPA struct {
	MinReplicas *int32                                        `json:"minReplicas,omitempty"`
	MaxReplicas int32                                         `json:"maxReplicas,omitempty"`
	Behavior    autoscalingv2.HorizontalPodAutoscalerBehavior `json:"behavior,omitempty"`
	Metrics     []autoscalingv2.MetricSpec                    `json:"metrics,omitempty"`
}

type System struct {
	Properties string `json:"properties,omitempty"`
}

type Repository struct {
	Enabled         bool             ` json:"enabled,omitempty"`
	Name            string           `json:"name,omitempty"`
	URL             string           `json:"url,omitempty"`
	Method          string           `json:"method,omitempty"`
	Init            corev1.Container `json:"init,omitempty"`
	SecretName      string           `json:"secretName,omitempty"`
	BundleDirectory string           `json:"bundleDirectory,omitempty"`
}

type PodDisruptionBudgetSpec struct {
	MinAvailable   *intstr.IntOrString `json:"minAvailable,omitempty"`
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`
}

type PodAffinity struct {
	TopologyKey *string          `json:"antiAffinityTopologyKey,omitempty"`
	Advanced    *corev1.Affinity `json:"advanced,omitempty"`
}

type PodResources struct {
	Requests corev1.ResourceList `json:"requests,omitempty"`
	Limits   corev1.ResourceList `json:"limits,omitempty"`
}

type ResourceList struct {
	Memory           resource.Quantity `json:"memory,omitempty"`
	CPU              resource.Quantity `json:"cpu,omitempty"`
	EphemeralStorage resource.Quantity `json:"ephemeral-storage,omitempty"`
}

type VolumeSpec struct {
	EmptyDir              *corev1.EmptyDirVolumeSource      `json:"emptyDir,omitempty"`
	HostPath              *corev1.HostPathVolumeSource      `json:"hostPath,omitempty"`
	PersistentVolumeClaim *corev1.PersistentVolumeClaimSpec `json:"persistentVolumeClaim,omitempty"`
}

type Cluster struct {
	Password string `json:"password,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type Service struct {
	Enabled     bool               ` json:"enabled,omitempty"`
	Annotations map[string]string  `json:"annotations,omitempty"`
	Type        corev1.ServiceType `json:"type,omitempty"`
	Ports       []Ports            `json:"ports,omitempty"`
}

type Ingress struct {
	Enabled          bool                       `json:"enabled,omitempty"`
	Annotations      map[string]string          `json:"annotations,omitempty"`
	IngressClassName string                     `json:"ingressClassName,omitempty"`
	TLS              []networkingv1.IngressTLS  `json:"tls,omitempty"`
	Rules            []networkingv1.IngressRule `json:"rules,omitempty"`
}

type Ports struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port,omitempty"`
	TargetPort int32  `json:"targetPort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type Java struct {
	JVMHeap   JVMHeap  `json:"jvmHeap,omitempty"`
	ExtraArgs []string `json:"extraArgs,omitempty"`
}

type JVMHeap struct {
	Calculate  bool   `json:"calculate,omitempty"`
	Percentage int    `json:"percentage,omitempty"`
	Default    string `json:"default,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Gateway{}, &GatewayList{})
}
