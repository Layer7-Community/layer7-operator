# Layer7 Gateway Operator
The Layer7 Gateway Operator, built using the [Operator SDK](https://github.com/operator-framework/operator-sdk) covers all aspects of deploying, maintaining and upgrading API Gateways in Kubernetes.

## About
The Operator is currently in an Alpha state and therefore does not currently carry a support/maintenance statement. Please check out the [Gateway Helm Chart](https://github.com/CAAPIM/apim-charts/tree/stable/charts/gateway) for supported container Gateway Deployment options

The initial release gives basic coverage to instantiating one or more Operator managed Gateway Deployments with or without an external MySQL Database.

The Layer7 Operator is restricted to manage the namespace it's deployed in by default. There is also a cluster-wide option available where the operator can watch all or multiple namespaces.

### Deployment Types

#### Application Level
- Database Backed Gateway Clusters
- Embedded Derby Database (Ephemeral) Gateways

#### Features
- Gateway Helm Chart feature parity (no sample mysql/hazelcast/influxdb/grafana deployments)
- Basic Git Integration for Gateway bundles.
- Dynamic Volumes for existing Kubernetes Configmaps/Secrets.
- Dynamic Volumes for CSI Secret Volumes.
- Dedicated Service for access to Policy Manager/Gateway management services.

#### Coming
- Restman/GMU integration
- Graphman integration

### Under consideration
- OTK support (operator managed)
- Additional Custom Resources


## Prerequisites
- Kubernetes v1.23
- Gateway v10.x License
- Ingress Controller (You can also expose Gateway Services as L4 LoadBalancers)

## Installation
There are currently two ways to deploy the Layer7 Gateway Operator. A Helm Chart will be available in the future.

### Installing with kubectl
Clone this repository to get started
```
$ git clone https://github.com/Layer7-Community/layer7-operator.git
$ cd layer7-operator
```
bundle.yaml contains all of the manifests that the Layer7 Operator requires.

#### OwnNamespace
By default the Operator manages the namespace that it is deployed into and does not create any cluster roles/role bindings. 

```
$ kubectl apply -f deploy/bundle.yaml
```

#### All/Multiple Namespaces
You can also configure the Operator to watch all or multiple namespaces. This will create a namespace called <i>layer7-operator-system</i>. The default is all namespaces, you can update this by changing the following in deploy/cw-bundle.yaml

default (watches all namespaces)
```
env:
- name: WATCH_NAMESPACE
  value: ""
```
limit to specific namespaces
```
env:
- name: WATCH_NAMESPACE
  value: "ns1,ns2,ns3"
```

Once you have updated deploy/cw-bundle.yaml run the following command to install the Operator

```
$ kubectl apply -f deploy/cw-bundle.yaml
```

### Installing on OpenShift
The Layer7 Operator <b>has not been published</b> to any Operator Catalogs, you can still deploy it using the operator-sdk cli. The only supported install mode in OpenShift is OwnNamespace.

```
operator-sdk run bundle docker.io/layer7api/layer7-operator-bundle:v0.0.1 --install-mode OwnNamespace
```


### Creating a Gateway Resource
Deploying a Gateway resource requires additional files like a gateway license, secret. The example folder provides a quickstart to give you an idea of what can be configured.
1. Place a Gateway license as license.xml into the example folder
2. If you would like to create a TLS secret then add tls.crt and tls.key, then uncomment lines 15-19 in example/kustomization.yaml.
3. Update example/security_v1_gateway.yaml with any changes you would like to make (eg. ingress configuration)

The default external traffic exposure method for Operator Managed Gateways is via Kubernetes Ingress Controller. This can be disabled in example/security_v1_gateway.yaml if you'd like to use a L4 Loadbalancer.

The default mode also deploys a management service of type ClusterIP that selects a single Gateway Pod for Policy Manager Connections on 9443.

```
$ kubectl apply -k example/
serviceaccount/ssg-serviceaccount created
secret/gateway-license created
secret/gateway-secret created
gateway.security.brcmlabs.com/ssg created
```

### Uninstall


#### Remove the Gateway Exampole 
```
$ kubectl delete -k example/
serviceaccount/ssg-serviceaccount deleted
secret "gateway-license" deleted
secret "gateway-secret" deleted
gateway.security.brcmlabs.com "ssg" deleted
```


#### Remove the Operator
if you installed the operator using kubectl
```
$ kubectl delete -k deploy/bundle.yaml|cw-bundle.yaml
```

if you installed the operator in Openshift

``` 
$ operator-sdk cleanup <operatorPackageName> [flags]
```


