<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [User Guide](#user-guide)
  - [kubefedctl CLI](#kubefedctl-cli)
  - [Deployment Image](#deployment-image)
  - [Create Clusters](#create-clusters)
  - [Helm Chart Deployment](#helm-chart-deployment)
  - [Cluster Registration](#cluster-registration)
  - [Federated API types](#federated-api-types)
    - [Enabling federation of an API type](#enabling-federation-of-an-api-type)
    - [Verifying API type is installed on all member clusters](#verifying-api-type-is-installed-on-all-member-clusters)
    - [Enabling an API type with a non-default API group](#enabling-an-api-type-with-a-non-default-api-group)
    - [Disabling propagation of an API type](#disabling-propagation-of-an-api-type)
  - [Federating a target resource](#federating-a-target-resource)
    - [Federate a namespace with contents](#federate-a-namespace-with-contents)
    - [Optionally enable type while federating a resource](#optionally-enable-type-while-federating-a-resource)
    - [Federate resources from input file and stdin](#federate-resources-from-input-file-and-stdin)
  - [Propagation status](#propagation-status)
    - [Troubleshooting condition status](#troubleshooting-condition-status)
      - [Troubleshooting CheckClusters](#troubleshooting-checkclusters)
  - [Deletion policy](#deletion-policy)
  - [Verify your deployment is working](#verify-your-deployment-is-working)
    - [Creating the test namespace](#creating-the-test-namespace)
    - [Creating test resources](#creating-test-resources)
    - [Checking resources status](#checking-resources-status)
    - [Updating FederatedNamespace placement](#updating-federatednamespace-placement)
    - [Cleaning up](#cleaning-up)
  - [Overrides](#overrides)
    - [Overriding retained fields](#overriding-retained-fields)
  - [Using Cluster Selector](#using-cluster-selector)
    - [Neither `spec.placement.clusters` nor `spec.placement.clusterSelector` is provided](#neither-specplacementclusters-nor-specplacementclusterselector-is-provided)
    - [Both `spec.placement.clusters` and `spec.placement.clusterSelector` are provided](#both-specplacementclusters-and-specplacementclusterselector-are-provided)
    - [`spec.placement.clusters` is not provided, `spec.placement.clusterSelector` is provided but empty](#specplacementclusters-is-not-provided-specplacementclusterselector-is-provided-but-empty)
    - [`spec.placement.clusters` is not provided, `spec.placement.clusterSelector` is provided and not empty](#specplacementclusters-is-not-provided-specplacementclusterselector-is-provided-and-not-empty)
  - [Troubleshooting](#troubleshooting)
  - [Cleanup](#cleanup)
    - [Deployment Cleanup](#deployment-cleanup)
  - [Namespace-scoped control plane](#namespace-scoped-control-plane)
    - [Helm Configuration](#helm-configuration)
    - [Cluster Registration](#cluster-registration-1)
  - [Local Value Retention](#local-value-retention)
    - [Scalable](#scalable)
    - [ServiceAccount](#serviceaccount)
  - [Higher order behaviour](#higher-order-behaviour)
    - [Multi-Cluster Ingress DNS](#multi-cluster-ingress-dns)
    - [Multi-Cluster Service DNS](#multi-cluster-service-dns)
    - [ReplicaSchedulingPreference](#replicaschedulingpreference)
      - [Distribute total replicas evenly in all available clusters](#distribute-total-replicas-evenly-in-all-available-clusters)
      - [Distribute total replicas in weighted proportions](#distribute-total-replicas-in-weighted-proportions)
      - [Distribute replicas in weighted proportions, also enforcing replica limits per cluster](#distribute-replicas-in-weighted-proportions-also-enforcing-replica-limits-per-cluster)
      - [Distribute replicas evenly in all clusters, however not more than 20 in C](#distribute-replicas-evenly-in-all-clusters-however-not-more-than-20-in-c)
  - [Controller-Manager Leader Election](#controller-manager-leader-election)
  - [Limitations](#limitations)
    - [Immutable Fields](#immutable-fields)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# User Guide

Please refer to [KubeFed Concepts](./concepts.md) first before you go through this user guide.

This user guide contains concepts and procedures to help you get started with KubeFed.

For information about installing KubeFed, see the [installation documentation](./installation.md).

## kubefedctl CLI

`kubefedctl` is the KubeFed command line utility. You can download
the latest binary from the [release page](https://github.com/kubernetes-sigs/kubefed/releases).

```bash
VERSION=<latest-version, e.g. 0.1.0-rc3>
OS=<darwin/linux>
ARCH=amd64
curl -LO https://github.com/kubernetes-sigs/kubefed/releases/download/v${VERSION}/kubefedctl-${VERSION}-${OS}-${ARCH}.tgz
tar -zxvf kubefedctl-*.tgz
chmod u+x kubefedctl
sudo mv kubefedctl /usr/local/bin/ # make sure the location is in the PATH
```

**NOTE:** `kubefedctl` is built for Linux and OSX only in the release package.

## Deployment Image

If you follow this user guide without any changes you will be using the latest
stable released version of the KubeFed image tagged as `latest`.
Alternatively, we support the ability to deploy the [latest master image tagged
as `canary`](development.md#test-latest-master-changes-canary) or [your own
custom image](development.md#test-your-changes).

## Create Clusters

The KubeFed control plane can run on any v1.13 or greater Kubernetes clusters. The following is a list of
Kubernetes environments that have been tested and are supported by the KubeFed community:

- [kind](./environments/kind.md)

- [Minikube](./environments/minikube.md)

- [Google Kubernetes Engine (GKE)](./environments/gke.md)

- [IBM Cloud Private](./environments/icp.md)

After completing the steps in one of the above guides, return here to continue the KubeFed deployment.

**NOTE:** You must set the correct context using the command below as this guide depends on it.

```bash
kubectl config use-context cluster1
```

## Helm Chart Deployment

You can refer to [helm chart installation guide](https://github.com/kubernetes-sigs/kubefed/blob/master/charts/kubefed/README.md)
to install and uninstall a KubeFed control plane.

## Cluster Registration

You can join, unjoin and check the status of clusters using the `kubefedctl` command.
See the [Cluster Registration documentation](./cluster-registration.md) for more information.

## Federated API types

### Enabling federation of an API type

You can enable federation of any Kubernetes API type (including CRDs) by using the
`kubefedctl` command as follows.

**NOTE:** Federation of a CRD requires that the CRD be installed on all member clusters.  If
the CRD is not installed on a member cluster, propagation to that cluster will fail.

```bash
kubefedctl enable <target kubernetes API type>
```

The `<target kubernetes API type>` can be any of the following

- the Kind (e.g. `Deployment`)
- plural name (e.g. `deployments`)
- group-qualified plural name (e.g `deployment.apps`), or
- short name (e.g. `deploy`)

for the intended target API type.

The `kubefedctl` command will create
 - a CRD for the federated type named `Federated<Kind>`
 - a `FederatedTypeConfig` in the KubeFed system namespace with the group-qualified plural name of the target type.

A `FederatedTypeConfig` associates the federated type CRD with the target
kubernetes type, enabling propagation of federated resources of the given type to the member clusters.

The format used to name the `FederatedTypeConfig` is `<target kubernetes API type name>.<group name>`
except kubernetes `core` group types where the name format used is `<target kubernetes API type name>`.

You can also output the yaml to `stdout` instead of applying it to the API Server, using the following command.

```bash
kubefedctl enable <target API type> --output=yaml
```

**NOTE:** Federation of an API type requires that the API type be installed on
all member clusters. If the API type is not installed on a member cluster,
propagation to that cluster will fail. See issue
[314](https://github.com/kubernetes-sigs/kubefed/issues/314) for more
details.

### Verifying API type is installed on all member clusters

If the API type is not installed on one of your member clusters, you will see a
repeated `controller-manager` log error similar to the one reported in issue
[314](https://github.com/kubernetes-sigs/kubefed/issues/314). At this
time, you must manually verify that the API type is installed on each of your
clusters as the `controller-manager` log error is the only indication.

For an example API type `bars.example.com`, you can verify that the API type is
installed on each of your clusters by running:

```bash

CLUSTER_CONTEXTS="cluster1 cluster2"
for c in ${CLUSTER_CONTEXTS}; do
    echo ----- ${c} -----
    kubectl --context=${c} api-resources --api-group=example.com
done
```

The output should look like the following:

```bash
----- cluster1 -----
NAME   SHORTNAMES   APIGROUP      NAMESPACED   KIND
bars                example.com   true         Bar
----- cluster2 -----
NAME   SHORTNAMES   APIGROUP      NAMESPACED   KIND
bars                example.com   true         Bar
```

The output shown below is an example if you do *not* have the API type
installed on `cluster2`. Note that `cluster2` did not return any resources:

```bash
----- cluster1 -----
NAME   SHORTNAMES   APIGROUP      NAMESPACED   KIND
bars                example.com   true         Bar
----- cluster2 -----
NAME   SHORTNAMES   APIGROUP   NAMESPACED   KIND
```

Verifying the API type exists on all member clusters will ensure successful
propagation to that cluster.

### Enabling an API type with a non-default API group

When `kubefedctl enable` is used to enable types whose plural names (e.g. **deployments**.example.com
and **deployments**.apps) match, the crd name of the generated federated type would also match (e.g.
**deployments**.types.kubefed.io).

`kubefedctl enable --federated-group string` specifies the name of the API
group to use for the generated federated type. It is `types.kubefed.io` by
default. If a non-default group is used to enable federation of a type, the
RBAC permissions for the KubeFed controller manager will need to be updated to
include permissions for the new group.

For example, as part of deployment of a KubeFed control plane,
`deployments.apps` is enabled by default. To enable `deployments.example.com`,
you should:

```bash
kubefedctl enable deployments.example.com --federated-group kubefed.example.com
kubectl patch clusterrole kubefed-role --type='json' -p='[{"op": "add", "path": "/rules/1", "value": {
            "apiGroups": [
                "kubefed.example.com"
            ],
            "resources": [
                "*"
            ],
            "verbs": [
                "get",
                "watch",
                "list",
                "update"
            ]
        }
}]'
```

This example is for a cluster-scoped KubeFed control plane. For a namespaced
KubeFed control plane, patch role `kubefed-role` in the KubeFed system namespace
instead.

### Disabling propagation of an API type

You can disable propagation of an API type by editing its `FederatedTypeConfig`
resource:

```bash
kubectl patch --namespace <KUBEFED_SYSTEM_NAMESPACE> federatedtypeconfigs <NAME> \
    --type=merge -p '{"spec": {"propagation": "Disabled"}}'
```

This patch command sets the `propagation` field in the `FederatedTypeConfig`
associated with this target API type to `Disabled`, which will prompt the sync
controller for the target API type to be stopped.

If you want to permanently disable federation of the target API type, use:

```bash
kubefedctl disable <FederatedTypeConfig Name>
```

This will remove the `FederatedTypeConfig` that configures federation of the
type. If supplied with the optional `--delete-crd` flag, the command will also
remove the federated type CRD if none of its instances exist.

## Federating a target resource
Apart from `enabling` and `disabling` a `type` for `propagation` as specified in the previous
section, `kubefedctl` can also be used to `federate` a target resource of an API type.
We define the term `federate` [here](concepts.md#kubefed-concepts) and use the command keyword
`federate` in `kudefedctl` with similar meaning.

`kubefedctl federate` creates a federated resource from a kubernetes resource. The federated
resource will embed the kubernetes resource as its template and its placement will select all
clusters.

***Syntax***
```bash
kubefedctl federate <target kubernetes API type> <target resource> [flags]
```

If the flag `--namespace` is additionally not specified, the `<target resource>` will be
searched for in the namespace according to the client kubeconfig context. Please take note that `--namespace` flag is of no
meaning when federating a `namespace` itself and is discarded even if specified.
Please check the next section for more details about [federating a namespace](#federate-a-namespace).

***Example:***
Federate a resource named "my-configmap" in namespace "my-namespace" of kubernetes type "configmaps"
```bash
kubefedctl federate configmaps my-configmap -n my-namespace
```

By default, `kubefedctl federate` creates a federated resource in the same namespace as the
target resource.  This requires that the target type already be enabled for federation
(i.e. via `kubefedctl enable`).

If `--output=yaml` is specified, and the target type is not yet enabled for federation,
`kubefedctl federate` will assume the default form of the federated type in generating the
federated resource.  This may not be compatible with a kubefed control plane that has enabled
a federated type in a non-default way (e.g. the group of the federated type has been set to
something other than `types.kubefed.io`).

### Federate a namespace with contents
`kubefedctl federate` can also be used to federate a target namespace and its contained resources
with a single invocation. This can be achieved using the flag `--contents` which is valid only when
the `<target kubernetes API type>` is a `namespace`. `kubefedctl federate` with `--contents` looks
up all the existing resources in the target namespace and federates them one by one. It will skip the
resources created by controllers (e.g. `endpoints` and `events`).
It is also possible to explicitly skip resource types with the `--skip-api-resources` argument.

***Example:***
Federate a namespace named "my-namespace" skipping API Resource "configmaps" and API Resource group "apps"
```bash
kubefedctl federate namespace my-namespace --contents --skip-api-resources "configmaps,apps"
```

### Optionally enable type while federating a resource
`kubefedctl federate` allows optionally enabling the given `<target kubernetes API type>` before
federating the resource by supplying the `--enable-type flag`. This will enable federation of the
target type if it is not already enabled. It's recommended to use
[`kubefedctl enable`](#enabling-federation-of-an-api-type) beforehand if the intention is to
specify non default type configuration values.

***Example:***
Federate a configmap named "my-configmap" while also enabling type `configmaps` for propagation
```bash
kubefedctl federate configmap my-configmap --enable-type
```

### Federate resources from input file and stdin
In addition to supporting conversion of resources in a Kubernetes API, `kubefedctl federeate`
supports converting resources to `stdout` from resources read from a local file. API resources can
be read in yaml format via the `--filename` argument. The command currently does not look up
for an already enabled type to use the type configuration values while translating yaml resources
and uses default values for the same.
The command in this mode can also take input from `stdin` in place of an actual file.
The output could be piped to `kubectl create -f -` if the intention is to create the federate
resource in the federated API surface.
No other arguments or flag options are needed in this mode.

***Example:***
Get federated resources for the target resources listed in a yaml file "my-file"
```bash
kubefedctl federate --filename ./my-file
```

## Propagation status

When the sync controller reconciles a federated resource with member
clusters, propagation status will be written to the resource as per
the following example:

```yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedNamespace
metadata:
  name: myns
  namespace: myns
spec:
  placement:
    clusterSelector: {}
status:
  # The status True of the condition of type Propagation
  # indicates that the state of all member clusters is as
  # intended as of the last probe time.
  conditions:
  - type: Propagation
    status: True
    lastProbeTime: "2019-05-08T01:23:20Z"
    lastTransitionTime: "2019-05-08T01:23:20Z"
  # The namespace 'myns' has been verified to exist in the
  # following clusters as of the lastProbeTime recorded
  # in the 'Propagation' condition.
  clusters:
  - name: cluster1
  - name: cluster2
```

### Troubleshooting condition status

If the sync controller encounters an error in creating, updating or
deleting managed resources in member clusters, the `Propagation`
condition will have a status of `False` and the reason field will be
one of the following values:

| Reason                 | Description                                      |
|------------------------|--------------------------------------------------|
| CheckClusters          | One or more clusters is not in the desired state. |
| ClusterRetrievalFailed | An error prevented retrieval of member clusters. |
| ComputePlacementFailed | An error prevented computation of placement. |
| NamespaceNotFederated  | The containing namespace is not federated. |

For reasons other than `CheckClusters`, an event will be logged with
the same reason and can be examined for more detail:

```bash
kubectl describe federatednamespace myns -n myns | grep ComputePlacementFailed

Warning  ComputePlacementFailed  5m   federatednamespace-controller  Invalid selector <nil>
```

If the reason is `NamespaceNotFederated`, the containing namespace can be
federated by invoking `kubefedctl federate namespace <namespace name>`.

#### Troubleshooting CheckClusters

If the `Propagation` condition has status `False` and reason
`CheckClusters`, the cluster status can be examined to determine the
clusters for which reconciliation was not successful. In the following
example, namespace `myns` has been verified to exist in `cluster1`.
The namespace should not exist in `cluster2`, but deletion has failed.

```yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedNamespace
metadata:
  name: myns
  namespace: myns
spec:
  placement:
    clusters:
    - name: cluster1
status:
  conditions:
  - type: Propagation
    status: False
    reason: CheckClusters
    lastProbeTime: "2019-05-08T01:23:20Z"
    lastTransitionTime: "2019-05-08T01:23:20Z"
  clusters:
  - name: cluster1
  - name: cluster2
    status: DeletionFailed
```

When a cluster has a populated status, as in the example above, the
sync controller will have written an event with a matching `Reason`
that may provide more detail as to the nature of the problem.

```bash
kubectl describe federatednamespace myns -n myns | grep cluster2 | grep DeletionFailed

Warning  DeletionFailed  5m   federatednamespace-controller  Failed to delete Namespace "myns" in cluster "cluster2"...
```

The following table enumerates the possible values for cluster status:

| Status                 | Description                  |
|------------------------|------------------------------|
| AlreadyExists          | The target resource already exists in the cluster, and cannot be adopted due to `adoptResources` being disabled. |
| ApplyOverridesFailed   | An error occurred while attempting to apply overrides to the computed form of the target resource. |
| CachedRetrievalFailed  | An error occurred when retrieving the cached target resource. |
| ClientRetrievalFailed  | An error occurred while attempting to create an API client for the member cluster. |
| ClusterNotReady        | The latest health check for the cluster did not succeed. |
| ComputeResourceFailed  | An error occurred when determining the form of the target resource that should exist in the cluster. |
| CreationFailed         | Creation of the target resource failed. |
| CreationTimedOut       | Creation of the target resource timed out. |
| DeletionFailed         | Deletion of the target resource failed. |
| DeletionTimedOut       | Deletion of the target resource timed out. |
| FieldRetentionFailed   | An error occurred while attempting to retain the value of one or more fields in the target resource (e.g. `clusterIP` for a service) |
| LabelRemovalFailed     | Removal of the KubeFed label from the target resource failed. |
| LabelRemovalTimedOut   | Removal of the KubeFed label from the target resource timed out. |
| RetrievalFailed        | Retrievel of the target resource from the cluster failed. |
| UpdateFailed           | Update of the target resource failed. |
| UpdateTimedOut         | Update of the target resource timed out. |
| VersionRetrievalFailed | An error occurred while attempting to retrieve the last recorded version of the target resource. |
| WaitingForRemoval      | The target resource has been marked for deletion and is awaiting garbage collection. |

## Deletion policy

All federated resources reconciled by the sync controller have a finalizer (`kubefed.io/sync-controller`) added to their
metadata. This finalizer will prevent deletion of a federated resource
until the sync controller has a chance to perform pre-deletion
cleanup.

Pre-deletion cleanup of a federated resource includes removal of
resources managed by the federated resource from member clusters. To
ensure retention of managed resources, add `kubefed.io/orphan:
true` as an annotation to the federated resource prior to deletion:

Pre-deletion cleanup includes removal of
resources managed by the federated resource from member clusters.

To prevent removal of these managed resources, add `kubefed.io/orphan:
true` as an annotation to the federated resource prior to deletion, as follows.

You can do it by
```bash 
kubefedctl orphaning-deletion enable <federated type> <name>
```
You can also check the current `orphaning-deletion` status by:
 ```bash 
 kubefedctl orphaning-deletion status <federated type> <name>
 ```
And finally, if you want to return to the default deletion behavior, you can disable 
the `orphaning-deletion` by:
```bash 
 kubefedctl orphaning-deletion disable <federated type> <name>
 ```

If the flag `--namespace` is additionally not specified, the federated resource will
be searched for in the namespace according to the client kubeconfig context.
If the sync controller for a given federated type is not able to reconcile a
federated resource slated for deletion, a federated resource that still has the
KubeFed finalizer will linger rather than being garbage collected. If
necessary, the KubeFed finalizer can be manually removed to ensure garbage
collection.

## Verify your deployment is working

You can verify that your deployment is working properly by completing the following example.

The example creates a test namespace with a `federatednamespace`
resource, as well as a federated resource for the following k8s resources.

- `configmap`
- `secret`
- `deployment`
- `service`, and
- `serviceaccount`

It will then show how to update the `federatednamespace` resource to move resources.

### Creating the test namespace

Create the `test-namespace` for the test resources.

```bash
kubectl apply -f example/sample1/namespace.yaml \
    -f example/sample1/federatednamespace.yaml
```

### Creating test resources

Create test resources.

```bash
kubectl apply -R -f example/sample1
```
 **NOTE:** If you get the following error while creating a test resource
```
unable to recognize "example/sample1/federated<type>.yaml": no matches for kind "Federated<type>" in version "types.kubefed.io/v1beta1",

```
then it indicates that a given type may need to be enabled with `kubefedctl enable <type>`

### Checking resources status

Check the status of all the resources in each cluster.

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

The [status of propagation](#propagation-status) is also recorded on each federated resource:

```bash
for r in federatedconfigmaps federatedsecrets federatedservice federateddeployment federatedserviceaccount federatedjob; do
    echo; echo ------------ resource: ${r} ------------; echo
    kubectl -n test-namespace get ${r} -o yaml
    echo; echo
done
```
Ensure `nginx` is running properly in each cluster:

```bash
for c in cluster1 cluster2; do
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo ------------ ${c} ------------; echo
    NODE_IP=$(kubectl get node --context=${c} \
        -o jsonpath='{.items[].status.addresses[*].address}'|sed 's/\S*cluster1\S*//'|tr -d " ")
    curl ${NODE_IP}:${NODE_PORT}
    echo; echo
done
```
### Updating FederatedNamespace placement

Remove `cluster2` via a patch command or manually.

```bash
kubectl -n test-namespace patch federatednamespace test-namespace \
    --type=merge -p '{"spec": {"placement": {"clusters": [{"name": "cluster1"}]}}}'

kubectl -n test-namespace edit federatednamespace test-namespace
```
Then wait to verify all resources are removed from `cluster2`:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

You can quickly add back all the resources by simply updating the
`FederatedNamespace` to add `cluster2` again via a patch command or
manually:

```bash
kubectl -n test-namespace patch federatednamespace test-namespace \
    --type=merge -p '{"spec": {"placement": {"clusters": [{"name": "cluster1"}, {"name": "cluster2"}]}}}'

kubectl -n test-namespace edit federatednamespace test-namespace
```
Wait and verify all resources are added back to `cluster2`:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```
Lastly, make sure `nginx` is running properly in each cluster:

```bash
for c in cluster1 cluster2; do
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo ------------ ${c} ------------; echo
    curl $(echo -n $(minikube ip -p ${c})):${NODE_PORT}
    echo; echo
done
```

If you were able to verify the resources removed and added back then you have
successfully verified a working KubeFed deployment.

### Cleaning up

To cleanup the example simply delete the namespace:

```bash
kubectl delete ns test-namespace
```
> **NOTE:** Deleting the test namespace requires that the KubeFed controllers first perform the removal of managed resources from member clusters. This may take a few moments.

## Overrides

Overrides can be specified for any federated resource and allow varying
resource content from the template on a per-cluster basis. Overrides are
implemented via a subset of [jsonpatch](http://jsonpatch.com/), as follows:

 - `op` defines the operation to perform (`add`, `remove` or `replace` are supported)
   - `replace` replaces a value
     - if not specified, `op` will default to `replace`
   - `add` adds a value to an object or array
   - `remove` removes a value from an object or array
 - `path` specifies a valid location in the managed resource to target for modification
   - `path` must start with a leading `/` and entries must be separated by `/`
     - e.g. `/spec/replicas`
   - indexed paths start at zero
     - e.g. `/spec/template/spec/containers/0/image`
 - `value` specifies the value to `add` or `replace`.
   - `value` is ignored for `remove`

For example:

```yaml
kind: FederatedDeployment
...
spec:
  ...
  overrides:
  # Apply overrides to cluster1
    - clusterName: cluster1
      clusterOverrides:
        # Set the replicas field to 5
        - path: "/spec/replicas"
          value: 5
        # Set the image of the first container
        - path: "/spec/template/spec/containers/0/image"
          value: "nginx:1.17.0-alpine"
        # Ensure the annotation "foo: bar" exists
        - path: "/metadata/annotations"
          op: "add"
          value:
            foo: bar
        # Ensure an annotation with key "foo" does not exist
        - path: "/metadata/annotations/foo"
          op: "remove"
```

### Overriding retained fields

When computing the form of a managed resource that should appear in a cluster
registered with KubeFed, the following operations are executed in sequence:

 - A new resource is computed from the template of the federated resource
 - If an existing resource is present, the contents of fields subject to retention are preserved
 - Overrides are applied
 - The managed label is set

This order of operations ensures that [fields subject to
retention](#local-value-retention) can still be overridden (e.g. adding an
entry to `metadata.annotations`). Care should be taken in applying overrides
to retained fields that may be modified by controllers in member clusters or
a managed resource may end up being continuously updated first by the
controller in the member cluster and then by KubeFed.

## Using Cluster Selector

In addition to specifying an explicit list of clusters that a resource should be propagated
to via the `spec.placement.clusters` field of a federated resource, it is possible to
use the `spec.placement.clusterSelector` field to provide a label selector that determines
a list of clusters at runtime.

If the goal is to select a subset of member clusters, make sure that the `KubeFedCluster` binaries from pre-reqs [now covered by Helm installation]
resources that are intended to be selected have the appropriate labels applied.

The following command is an example to label a `KubeFedCluster`:

```bash
kubectl label kubefedclusters -n kube-federation-system cluster1 foo=bar
```

Please refer to [Kubernetes label command](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#label)
for more information on how `kubectl label` works.

The following sections detail how `spec.placement.clusters` and
`spec.placement.clusterSelector` are used in determining the clusters that a federated
resource should be propagated to.

### Neither `spec.placement.clusters` nor `spec.placement.clusterSelector` is provided

```yaml
spec:
  placement: {}
```

In this case, you can either set `spec: {}` as above or remove `spec` field from your
placement policy. The resource will not be propagated to member clusters.

### Both `spec.placement.clusters` and `spec.placement.clusterSelector` are provided

```yaml
spec:
  placement:
    clusters:
      - name: cluster2
      - name: cluster1
    clusterSelector:
      matchLabels:
        foo: bar
```

For this case, `spec.placement.clusterSelector` will be ignored as
`spec.placement.clusters` is provided. This ensures that the results of runtime
scheduling have priority over manual definition of a cluster selector.

### `spec.placement.clusters` is not provided, `spec.placement.clusterSelector` is provided but empty

In this case, `spec.placement.clusterSelector` will be ignored, since
`spec.placement.clusters` is provided. This ensures that the results of runtime
scheduling have priority over manual definition of a cluster selector.

```yaml
spec:
  placement:
    clusterSelector: {}
```

In this case, the resource will be propagated to all member clusters.

### `spec.placement.clusters` is not provided, `spec.placement.clusterSelector` is provided and not empty

```yaml
spec:
  placement:
    clusterSelector:
      matchLabels:
        foo: bar
```

In this case, the resource will only be propagated to member clusters that are labeled
with `foo: bar`.

## Troubleshooting

If federated resources are not propagated as expected to the member clusters, you can
use the following command to view `Events`, which may help you to diagnose the problem.

```bash
kubectl describe <federated CRD> <CR name> -n test-namespace
```
An example for CRD of `federatedserviceaccounts` is as follows:

```bash
kubectl describe federatedserviceaccounts test-serviceaccount -n test-namespace
```

It may also be useful to inspect the KubeFed controller log as follows:

```bash
kubectl logs deployment/kubefed-controller-manager -n kube-federation-system
```

## Cleanup

### Deployment Cleanup

Resources such as `namespaces` associated with a `FederatedNamespace` or `FederatedClusterRoles`
should be deleted before cleaning up the deployment, otherwise, the process will fail.

Run the following command to perform a cleanup of the cluster registry and
KubeFed deployments:

```bash
./scripts/delete-kubefed.sh
```

The above script unjoins the all of the clusters from the KubeFed control plane it deploys,
by default.

On successful completion of the script, both `cluster1` and
`cluster2` will be unjoined from the deployed KubeFed control plane.

## Namespace-scoped control plane

All prior instructions referred to the deployment and use of a
cluster-scoped KubeFed control plane. It is also possible to
deploy a namespace-scoped control plane. In this mode of operation,
KubeFed controllers will target resources in a single namespace on
both host and member clusters. This may be desirable when
experimenting with KubeFed on a production cluster.

### Helm Configuration

To deploy KubeFed in a namespaced configuration, set
`global.scope` to `Namespaced` as per the Helm chart [install
instructions](https://github.com/kubernetes-sigs/kubefed/blob/master/charts/kubefed/README.md#configuration).

### Cluster Registration

You can join, unjoin and check the status of clusters using the `kubefedctl` command.
See the [Cluster Registration documentation](./cluster-registration.md) for more information.

## Local Value Retention

In most cases, the KubeFed sync controller will overwrite any
changes made to resources it manages in member clusters.  The
exceptions appear in the following table.  Where retention is
conditional, an explanation will be provided in a subsequent section.

| Resource Type  | Fields                    | Retention   | Requirement                                                                        |
|----------------|---------------------------|-------------|------------------------------------------------------------------------------------|
| All            | metadata.annotations      | Always      | The annotations field is intended to be managed by controllers in member clusters. |
| All            | metadata.finalizers       | Always      | The finalizers field is intended to be managed by controllers in member clusters.  |
| All            | metadata.resourceVersion  | Always      | Updates require the most recent resourceVersion for concurrency control.           |
| Scalable       | spec.replicas             | Conditional | The HPA controller may be managing the replica count of a scalable resource.       |
| Service        | spec.clusterIP,spec.ports | Always      | A controller may be managing these fields.                                         |
| ServiceAccount | secrets                   | Conditional | A controller may be managing this field.                                           |

### Scalable

For scalable resources (those that have a scale subtype
e.g. `ReplicaSet` and `Deployment`), retention of the `spec.replicas`
field is controlled by the `retainReplicas` boolean field of the
federated resource.  `retainReplicas` defaults to `false`, and should
be set to `true` only if the resource will be managed by HPA in member
clusters.

Retention of the replicas field is possible for all
clusters or no clusters.  If a resource will be managed by HPA in some
clusters but not others, it will be necessary to create a separate
federated resource for each retention strategy (i.e. one with
`retainReplicas: true` and one with `retainReplicas: false`).

### ServiceAccount

A populated `secrets` field of a `ServiceAccount` resource managed by
KubeFed will be retained if the managing federated resource does
not specify a value for the field.  This avoids the possibility of the
sync controller attempting to repeatedly clear the field while a local
serviceaccounts controller attempts to repeatedly set it to a
generated value.

## Higher order behaviour

The architecture of KubeFed API allows higher level APIs to be constructed using the
mechanics provided by the standard form of the federated API types (containing fields for
`template`, `placement` and `override`) and associated controllers for a given resource.
Further sections describe few of higher level APIs implemented as part of KubeFed.

### Multi-Cluster Ingress DNS

Multi-Cluster Ingress DNS provides the ability to programmatically manage DNS resource records of Ingress objects
through [ExternalDNS](https://github.com/kubernetes-incubator/external-dns) integration. Review the guides below for
different DNS provider to learn more.
- [Multi-Cluster Ingress DNS with ExternalDNS Guide for Google Cloud DNS](./ingressdns-with-externaldns.md)
- [Multi-Cluster Ingress DNS with ExternalDNS Guide for CoreDNS in minikube](./ingress-service-dns-with-coredns.md)

### Multi-Cluster Service DNS

Multi-Cluster Service DNS provides the ability to programmatically manage DNS resource records of Service objects
through [ExternalDNS](https://github.com/kubernetes-incubator/external-dns) integration. Review the guides below for
different DNS provider to learn more.
- [Multi-Cluster Service DNS with ExternalDNS Guide for Google Cloud DNS](./servicedns-with-externaldns.md)
- [Multi-Cluster Service DNS with ExternalDNS Guide for CoreDNS in minikube](./ingress-service-dns-with-coredns.md)

### ReplicaSchedulingPreference

ReplicaSchedulingPreference provides an automated mechanism of distributing
and maintaining total number of replicas for `deployment` or `replicaset` based
federated workloads into federated clusters. This is based on high level
user preferences given by the user. These preferences include the semantics
of weighted distribution and limits (min and max) for distributing the replicas.
These also include semantics to allow redistribution of replicas dynamically
in case some replica pods remain unscheduled in some clusters, for example
due to insufficient resources in that cluster.

RSP is used in place of ReplicaSchedulingPreference for brevity in text further on.

The RSP controller works in a sync loop observing the RSP resource and the
matching `namespace/name` pair `FederatedDeployment` or `FederatedReplicaset`
resource.

If it finds that both RSP and its associated federated resource, the type of which
is specified using `spec.targetKind`, exists, it goes ahead to list currently
healthy clusters and distributes the `spec.totalReplicas` using the associated
per cluster user preferences. If the per cluster preferences are absent, it
distributes the `spec.totalReplicas` evenly among all clusters. It updates (or
creates if missing) the same `namespace/name` for the
`targetKind` with the replica values calculated, leveraging the sync controller
to actually propagate the k8s resource to federated clusters. It's noteworthy that
if an RSP is present, the `spec.replicas` from the federated resource are unused.
RSP also provides a further more useful feature using `spec.rebalance`. If this is
set to `true`, the RSP controller monitors the replica pods for target replica
workload from each federated cluster and if it finds that some clusters are not
able to schedule those pods for long, it moves (rebalances) the replicas to
clusters where all the pods are running and healthy. This in other words helps
moving the replica workloads to those clusters where there is enough capacity
and away from those clusters which are currently running out of capacity. The
`rebalance` feature might cause initial shuffle of replicas to reach an eventually
balanced state of distribution. The controller might further keep trying to move
few replicas back into the cluster(s) which ran out of capacity, to check if it can
be scheduled again to reach the normalised state (even distribution or the state
desired by user preferences), which apparently is the only mechanism to check if
this cluster has capacity now. The `spec.rebalance` should not be used if this
behaviour is unacceptable.

The RSP can be considered as more user friendly mechanism to distribute the
replicas, where the inputs needed from the user at federated control plane are
reduced. The user only needs to create the RSP resource and associated federated
resource (with only spec.template populated) to distribute the replicas. It can
also be considered as a more automated approach at distribution and further
reconciliation of the workload replicas.

The usage of the RSP semantics is illustrated using some examples below. The
examples considers 3 federated clusters `A`, `B` and `C`.

#### Distribute total replicas evenly in all available clusters

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
```

or

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    "*":
      weight: 1
```

A, B and C get 3 replicas each.

#### Distribute total replicas in weighted proportions

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    A:
      weight: 1
    B:
      weight: 2
```

A gets 3 and B gets 6 replicas in the proportion of 1:2. C does not get
any replica as missing weight preference is considered as weight=0.

#### Distribute replicas in weighted proportions, also enforcing replica limits per cluster

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 9
  clusters:
    A:
      minReplicas: 4
      maxReplicas: 6
      weight: 1
    B:
      minReplicas: 4
      maxReplicas: 8
      weight: 2
```

A gets 4 and B get 5 as weighted distribution is capped by cluster A minReplicas=4.

#### Distribute replicas evenly in all clusters, however not more than 20 in C

```yaml
apiVersion: scheduling.kubefed.io/v1alpha1
kind: ReplicaSchedulingPreference
metadata:
  name: test-deployment
  namespace: test-ns
spec:
  targetKind: FederatedDeployment
  totalReplicas: 50
  clusters:
    "*":
      weight: 1
    "C":
      maxReplicas: 20
      weight: 1
```

Possible scenarios

All have capacity.

```
Replica layout: A=16 B=17 C=17.
```

B is offline/has no capacity

```
Replica layout: A=30 B=0 C=20
```

A and B are offline:

```
Replica layout: C=20
```

## Controller-Manager Leader Election

The KubeFed controller manager is always deployed with leader election feature
to ensure high availability of the control plane. Leader election module ensures
there is always a leader elected among multiple instances which takes care of
running the controllers. In case the active instance goes down, one of the standby instances
gets elected as leader to ensure minimum downtime. Leader election ensures that
only one instance is responsible for reconciliation. You can refer to the
[helm chart configuration](https://github.com/kubernetes-sigs/kubefed/tree/master/charts/kubefed#configuration)
to configure parameters for leader election to tune for your environment
(the defaults should be sane for most environments).

## Limitations
### Immutable Fields
KubeFed API does not implement immutable fields in the federated resource yet.

A kubernetes resource field can be modified at runtime to change the resource
specification. An immutable field cannot be modified after the resource is created.

For a federated resource, `spec.template` defines the resource specification common
to all clusters. Though it is possible to modify any template field of a federated
resource (or set an override for the field), changing the value of an immutable field
will prevent all subsequent updates from completing successfully. This will be
indicated by a propagation status of `UpdateFailed` for affected clusters. These
errors can only be resolved by reverting the template field back to the value set at
creation.

For example, `spec.completions` is an immutable field of a job resource. You cannot
change it after a job has been created. Changing `spec.template.spec.completions`
of the federated job resource will prevent all subsequent updates to jobs managed by
the federated job. The changed value does not propagate to member clusters.

Support for validation of immutable fields in federated resources is intended to be
implemented before KubeFed is GA.
