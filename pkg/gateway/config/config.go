package config

import (
	"strconv"
	"strings"

	securityv1 "github.com/Layer7-Community/layer7-operator/api/v1"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewConfigMap
func NewConfigMap(gw *securityv1.Gateway, name string) *corev1.ConfigMap {
	javaArgs := strings.Join(gw.Spec.App.Java.ExtraArgs, " ")
	data := make(map[string]string)
	jvmHeap := setJVMHeapSize(gw)
	switch name {
	case gw.Name + "-system":
		data["system.properties"] = gw.Spec.App.System.Properties
	case gw.Name:
		data["ACCEPT_LICENSE"] = gw.Spec.License.Accept
		data["SSG_CLUSTER_HOST"] = gw.Spec.App.Management.Cluster.Hostname
		data["SSG_JVM_HEAP"] = jvmHeap
		data["EXTRA_JAVA_ARGS"] = javaArgs

		if gw.Spec.App.Management.Database.Enabled {
			data["SSG_DATABASE_JDBC_URL"] = gw.Spec.App.Management.Database.JDBCUrl
		}

		if gw.Spec.App.Hazelcast.External {
			/// external hazelcast Dcom - update
			data["hazelcast-client.xml"] = `<hazelcast-client xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.hazelcast.com/schema/client-config http://www.hazelcast.com/schema/client-config/hazelcast-client-config-3.10.xsd" xmlns="http://www.hazelcast.com/schema/client-config"><instance-name>` + gw.Name + `-hazelcast-client</instance-name><network><cluster-members><address>` + gw.Spec.App.Hazelcast.Endpoint + `</address></cluster-members><connection-attempt-limit>10</connection-attempt-limit><redo-operation>true</redo-operation></network><connection-strategy async-start="false" reconnect-mode="ON" /></hazelcast-client>`
			data["EXTRA_JAVA_ARGS"] = javaArgs + " -Dcogw.l7tech.server.extension.sharedCounterProvider=externalhazelcast -Dcogw.l7tech.server.extension.sharedKeyValueStoreProvider=externalhazelcast -Dcogw.l7tech.server.extension.sharedClusterInfoProvider=externalhazelcast"
		}
	case gw.Name + "-cwp-bundle":
		props := map[string]string{}

		for _, p := range gw.Spec.App.ClusterProperties.Properties {
			props[p.Name] = p.Value
		}
		bundle, _ := util.BuildCWPBundle(props)
		data["cwp.bundle"] = string(bundle)
	case gw.Name + "-listen-port-bundle":
		bundle, _ := util.BuildListenPortBundle(gw.Spec.App.ListenPorts.CipherSuites, gw.Spec.App.ListenPorts.TlsVersions)
		data["listen-ports.bundle"] = string(bundle)
	}

	cmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gw.Namespace,
			Labels:    util.DefaultLabels(gw),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		Data: data,
	}
	return cmap
}

// func NewBundleConfigMap(gw *securityv1.Gateway, name string) *corev1.ConfigMap {

// }

func setJVMHeapSize(gw *securityv1.Gateway) string {
	var jvmHeap string
	memLimit := gw.Spec.App.Resources.Limits.Memory()

	if gw.Spec.App.Java.JVMHeap.Calculate && memLimit.IsZero() && gw.Spec.App.Java.JVMHeap.Default != "" {
		jvmHeap = gw.Spec.App.Java.JVMHeap.Default
	}

	if gw.Spec.App.Java.JVMHeap.Calculate && !memLimit.IsZero() {
		memMB := float64(memLimit.Value()) * 0.00000095367432 //binary conversion
		heapPercntg := float64(gw.Spec.App.Java.JVMHeap.Percentage) / 100.0
		heapMb := strconv.FormatInt(int64(memMB*heapPercntg), 10)
		jvmHeap = heapMb + "m"
	}

	if jvmHeap == "" {
		jvmHeap = "2g"
	}

	return jvmHeap
}
