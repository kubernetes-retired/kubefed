# Federate a CRD in the target cluster

Handling arbitrary resources is one of the essential use-cases for federation. This example will demonstrate how to 
configure federation to handle a previously unknown, arbitrary Custom Resource.

**Note:**
If your Custom resource works with a controller, the controller should be handle separately.  

### Prerequisites

The federation v2 suppports installation for both cluster-scoped and namespace-scoped. In [user guide](./userguide.md), we are following cluster-scoped federation and deploying federation controller in the namespace `federation-system`. Also, you will be creating the two clusters i.e. `cluster1` and `cluster2` and federation v2 will be installed on `cluster1`.

### Example CRD to federate

Let's say you want to federate a CRD of the type `Bar` then use the following [bar_crd.yaml](../example/crd/bar_crd.yaml).

```
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: bars.example.io
spec:
  group: example.io
  version: v1
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          data:
            type: string
        status:
          type: object
  names:
    plural: bars
    singular: bar
    kind: Bar
```

### Install the target CRD in all clusters

Make sure to install target CRDs on all member clusters otherwise this example will not work as expected.

```shell
$ kubectl --validate=false apply -f ./example/crd/bar_crd.yaml --context=cluster1
customresourcedefinition.apiextensions.k8s.io/bars.example.io created

$ kubectl --validate=false apply -f ./example/crd/bar_crd.yaml --context=cluster2
customresourcedefinition.apiextensions.k8s.io/bars.example.io created
```

### Create the Federation API for your CRD

Now that we've created the CRD in all the clusters we want to federate it to, let's create the federation API for that CRD. The federation API for your CRD is distinct from the CRD itself and is the API surface that declares what the state that should be spread to different clusters is.

There are three pieces of the federation API for a type.
* The **template type** describes the base definition of a resource that federation should propagate.
* The **placement type** holds information about which clusters a federated resource should be spread to.
* The **overrides type** optionally defines how a particular resource should be varied in certain federated clusters:

We stored all these definition in the [federatedbar_crd.yaml](../example/crd/federatedbar_crd.yaml) file.

```shell 
#template type
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: federatedbars.federation.example.io
spec:
  group: federation.example.io
  names:
    kind: FederatedBar
    plural: federatedbars
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          data:
            type: string
        status:
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
---
# placement type
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: federatedbarplacements.federation.example.io
spec:
  group: federation.example.io
  names:
    kind: FederatedBarPlacement
    plural: federatedbarplacements
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            clusterNames:
              items:
                type: string
              type: array
          type: object
        status:
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
---
# override type
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: federatedbaroverrides.federation.example.io
spec:
  group: federation.example.io
  names:
    kind: FederatedBarOverride
    plural: federatedbaroverrides
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            overrides:
              items:
                properties:
                  clusterName:
                    type: string
                  data:
                    type: string
                type: object
              type: array
          type: object
        status:
          type: object
  version: v1alpha1
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
```

The federation APIs must be created in the cluster that hosts federation.

```shell
$ kubectl --validate=false apply -f ./example/crd/federatedbar_crd.yaml --context=cluster1
customresourcedefinition.apiextensions.k8s.io/federatedbars.federation.example.io created
customresourcedefinition.apiextensions.k8s.io/federatedbarplacements.federation.example.io created
customresourcedefinition.apiextensions.k8s.io/federatedbaroverrides.federation.example.io created

```

**Note:**
Only federated resources are propagated. Hence it's important to create a `federatedbar` to have an instance of the 
target type created in member clusters.

### Enable propagation of your federated CRD

It's time to work towards enabling the push configuration for those CRDs by creating a `FederatedTypeConfig` for `Bar`. 
See example in the [federatedBar.yaml](../example/crd/federatedBar.yaml) file.

```
apiVersion: core.federation.k8s.io/v1alpha1
kind: FederatedTypeConfig
metadata:
  name: bars.example.io
  namespace: federation-system
spec:
  target:
    version: v1
    kind: Bar
  namespaced: true
  comparisonField: Generation
  propagationEnabled: true
  template:
    group: federation.example.io
    version: v1alpha1
    kind: FederatedBar
  placement:
    kind: FederatedBarPlacement
  override:
    kind: FederatedBarOverride
  overridePath:
   - spec
   - data
```

Deploy it to the federated host cluster

```shell
$ kubectl apply  --validate=false   -f ./example/crd/federatedBar.yaml --context=cluster1
federatedtypeconfig.core.federation.k8s.io/bars.example.io created
``` 

### Create federated API resources and see them propagate

Use  [federatedbar_test.yaml](../example/crd/federatedbar_test.yaml)  file to verify if you can federate a CRD of the type `Bar` in the target clusters.

```
apiVersion: federation.example.io/v1alpha1
kind: FederatedBar
metadata:
  name: test-crd
  namespace: test-namespace
spec:
  template:
    metadata:
      labels:
        app: fed-crd
    spec:
      data: "Hello"
---
apiVersion: federation.example.io/v1alpha1
kind: FederatedBarOverride
metadata:
  name: test-crd
  namespace: test-namespace
spec:
  overrides:
  - clusterName: cluster2
     data: "World"    
---
apiVersion: federation.example.io/v1alpha1
kind: FederatedBarPlacement
metadata:
  name: test-crd
  namespace: test-namespace
spec:
  clusterNames:
  - cluster2
  - cluster1

```
Deploy it to the cluster that hosts federation
```shell
kubectl --validate=false apply -f example/crd/federatedbar_test.yaml  --context=cluster1
federatedbar.federation.example.io/test-crd created
federatedbaroverride.federation.example.io/test-crd created
federatedbarplacement.federation.example.io/test-crd created
``` 

Now we can check instances of our CRD

```
$ kubectl get bars -n test-namespace --context=cluster1
NAME       AGE
test-crd   30m

$ kubectl get bars test-crd -n test-namespace --context=cluster1 -oyaml
apiVersion: example.io/v1
kind: Bar
metadata:
  creationTimestamp: 2018-10-19T07:12:07Z
  generation: 1
  labels:
    app: fed-crd
  name: test-crd
  namespace: test-namespace
  resourceVersion: "14445996"
  selfLink: /apis/example.io/v1/namespaces/test-namespace/bars/test-crd
  uid: 49d31994-d36e-11e8-a4f3-fa163e77d0a4
spec:
  data: Hello


$ kubectl get bars -n test-namespace --context=cluster2
NAME       AGE
test-crd   30m

$ kubectl get bars test-crd -n test-namespace --context=cluster2 -oyaml

apiVersion: example.io/v1
kind: Bar
metadata:
  creationTimestamp: 2018-10-19T07:12:06Z
  generation: 1
  labels:
    app: fed-crd
  name: test-crd
  namespace: test-namespace
  resourceVersion: "7018452"
  selfLink: /apis/example.io/v1/namespaces/test-namespace/bars/test-crd
  uid: 495ebd3a-d36e-11e8-837a-06f4106cee1c
spec:
  data: World

```
