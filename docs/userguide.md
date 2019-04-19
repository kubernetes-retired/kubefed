<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [User Guide](#user-guide)
  - [Prerequisites](#prerequisites)
    - [Binaries](#binaries)
    - [Deployment Image](#deployment-image)
    - [Create Clusters](#create-clusters)
  - [Helm Chart Deployment](#helm-chart-deployment)
  - [Operations](#operations)
    - [Join Clusters](#join-clusters)
    - [Check Status of Joined Clusters](#check-status-of-joined-clusters)
    - [Unjoin Clusters](#unjoin-clusters)
  - [Enabling federation of an API type](#enabling-federation-of-an-api-type)
  - [Disabling federation of an API type](#disabling-federation-of-an-api-type)
  - [Deletion policy](#deletion-policy)
  - [Example](#example)
    - [Create the Test Namespace](#create-the-test-namespace)
    - [Create Test Resources](#create-test-resources)
    - [Check Status of Resources](#check-status-of-resources)
    - [Update FederatedNamespace Placement](#update-federatednamespace-placement)
      - [Using Cluster Selector](#using-cluster-selector)
        - [Neither `spec.placement.clusterNames` nor `spec.placement.clusterSelector` is provided](#neither-specplacementclusternames-nor-specplacementclusterselector-is-provided)
        - [Both `spec.placement.clusterNames` and `spec.placement.clusterSelector` are provided](#both-specplacementclusternames-and-specplacementclusterselector-are-provided)
        - [`spec.placement.clusterNames` is not provided, `spec.placement.clusterSelector` is provided but empty](#specplacementclusternames-is-not-provided-specplacementclusterselector-is-provided-but-empty)
        - [`spec.placement.clusterNames` is not provided, `spec.placement.clusterSelector` is provided and not empty](#specplacementclusternames-is-not-provided-specplacementclusterselector-is-provided-and-not-empty)
    - [Example Cleanup](#example-cleanup)
    - [Troubleshooting](#troubleshooting)
  - [Namespaced Federation](#namespaced-federation)
    - [Helm Configuration](#helm-configuration)
    - [Joining Clusters](#joining-clusters)
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

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# User Guide

If you are looking to use federation v2, you've come to the right place. Below
is a walkthrough tutorial for how to deploy the federation v2 control plane.

Please refer to [Federation V2 Concepts](./concepts.md) first before you go through this user guide.

## Prerequisites

The federation v2 deployment requires kubernetes version >= 1.11. The following
is a detailed list of binaries required.

### Binaries

The federation deployment depends on `kubebuilder`, `etcd`, `kubectl`, and
`kube-apiserver` >= v1.13 being installed in the path. The `kubebuilder`
([v1.0.8](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v1.0.8)
as of this writing) release packages all of these dependencies together.

These binaries can be installed via the `download-binaries.sh` script, which
downloads them to `./bin`:

```bash
./scripts/download-binaries.sh
export PATH=$(pwd)/bin:${PATH}
```

Or you can install them manually yourself using the guidelines provided below.

### Deployment Image

If you follow this user guide without any changes you will be using the latest
stable released version of the federation-v2 image tagged as `latest`.
Alternatively, we support the ability to deploy the [latest master image tagged
as `canary`](development.md#test-latest-master-changes-canary) or [your own
custom image](development.md#test-your-changes).

### Create Clusters

The federation v2 control plane can run on any v1.13 or greater Kubernetes clusters. The following is a list of
Kubernetes environments that have been tested and are supported by the Federation v2 community:

- [kind](./environments/kind.md)

- [Minikube](./environments/minikube.md)

- [Google Kubernetes Engine (GKE)](./environments/gke.md)

- [IBM Cloud Private](./environments/icp.md)

After completing the steps in one of the above guides, return here to continue the Federation v2 deployment.

**NOTE:** You must set the correct context using the command below as this guide depends on it.

```bash
kubectl config use-context cluster1
```

## Helm Chart Deployment

You can refer to [helm chart installation guide](https://github.com/kubernetes-sigs/federation-v2/blob/master/charts/federation-v2/README.md)
to install and uninstall a federation-v2 control plane.

## Operations

### Join Clusters

Next, you'll want to use the `kubefed2` tool to join all your
clusters that you want to test against.

```bash
./bin/kubefed2 join cluster1 --cluster-context cluster1 \
    --host-cluster-context cluster1 --add-to-registry --v=2
./bin/kubefed2 join cluster2 --cluster-context cluster2 \
    --host-cluster-context cluster1 --add-to-registry --v=2
```

You can repeat these steps to join any additional clusters.

**NOTE:** `cluster-context` will default to use the joining cluster name if not
specified.

### Check Status of Joined Clusters

Check the status of the joined clusters until you verify they are ready:

```bash
kubectl -n federation-system describe federatedclusters
```

The output of `Status.Conditions` field should include cluster ready info as follows:

```yaml
Status:
  Conditions:
    ...
    Reason:                ClusterReady
    Status:                True
    Type:                  Ready
```
### Unjoin Clusters

If required, federation allows you to unjoin clusters using `kubefed2` tool.

```bash
./bin/kubefed2 unjoin cluster2 --cluster-context cluster2 --host-cluster-context cluster1 --remove-from-registry --v=2
```
You can repeat these steps to unjoin any additional clusters.

## Enabling federation of an API type

It is possible to enable federation of any Kubernetes API type (including CRDs) using the
`kubefed2` command:

```bash
kubefed2 enable <target kubernetes API type>
```

The `<target kubernetes API type>` above can be the Kind (e.g. `Deployment`), plural name
(e.g. `deployments`), group-qualified plural name (e.g `deployment.apps`), or short name
(e.g. `deploy`) of the intended target API type.

The command will create a CRD for the federated type named `Federated<Kind>`.  The command will also
create a `FederatedTypeConfig` in the federation system namespace with the group-qualified plural name
of the target type.  A `FederatedTypeConfig` associates the federated type CRD with the target
kubernetes type, enabling propagation of federated resources of the given type to the member clusters.
The format used to name the `FederatedTypeConfig` is `<target kubernetes API type name>.<group name>`
except kubernetes `core` group types where the name format used is `<target kubernetes API type name>`.

It is also possible to output the yaml to `stdout` instead of applying it to the API Server:

```bash
kubefed2 enable <target API type> --output=yaml
```

**NOTE:** Federation of a CRD requires that the CRD be installed on all member clusters.  If
the CRD is not installed on a member cluster, propagation to that cluster will fail.

## Disabling federation of an API type

It is possible to disable propagation of a type that is configured for propagation using the
`kubefed2` command:

```bash
kubefed2 disable <FederatedTypeConfig Name>
```

This command will set the `propagationEnabled` field in the `FederatedTypeConfig`
associated with this target API type to `false`, which will prompt the sync
controller for the target API type to be stopped.

If the goal is to permanently disable federation of the target API type, passing the
`--delete-from-api` flag will remove the `FederatedTypeConfig` and federated type CRD created by
`enable`:

```bash
kubefed2 disable <FederatedTypeConfig Name> --delete-from-api
```

**WARNING: All custom resources for the type will be removed by this command.**

## Deletion policy

All federated resources reconciled by the sync controller have a
finalizer (`federation.k8s.io/sync-controller`) added to their
metadata. This finalizer will prevent deletion of a federated resource
until the sync controller has a chance to perform pre-deletion
cleanup.

Pre-deletion cleanup of a federated resource includes removal of
resources managed by the federated resource from member clusters. To
ensure retention of managed resources, add `federation.k8s.io/orphan:
true` as an annotation to the federated resource prior to deletion:

```bash
kubectl patch <federated type> <name> \
    --type=merge -p '{"metadata": {"annotations": {"federation.k8s.io/orphan": "true"}}}'
```

In the event that a sync controller for a given federated type is not
able to reconcile a federated resource slated for deletion - due to
propagation being disabled for a given type or the federated control
plane not running - a federated resource that still has the federation
finalizer will linger rather than being garbage collected. If
necessary, the federation finalizer can be manually removed to ensure
garbage collection.

## Example

Follow these instructions for running an example to verify your deployment is
working. The example will create a test namespace with a `federatednamespace`
resource as well as a federated resource for the following k8s resources:
`configmap`, `secret`, `deployment`, `service` and `serviceaccount`. It will
then show how to update the `federatednamespace` resource to move resources.

### Create the Test Namespace

First create the `test-namespace` for the test resources:

```bash
kubectl apply -f example/sample1/namespace.yaml \
    -f example/sample1/federatednamespace.yaml
```

### Create Test Resources

Create all the test resources by running:

```bash
kubectl apply -R -f example/sample1
```

**NOTE:** If you get the following error while creating a test resource i.e.

```
unable to recognize "example/sample1/federated<type>.yaml": no matches for kind "Federated<type>" in version "types.federation.k8s.io/v1alpha1",

```

then it indicates that a given type may need to be enabled with `kubefed2 enable <type>`

### Check Status of Resources

Check the status of all the resources in each cluster by running:

```bash
for r in configmaps secrets service deployment serviceaccount job; do
    for c in cluster1 cluster2; do
        echo; echo ------------ ${c} resource: ${r} ------------; echo
        kubectl --context=${c} -n test-namespace get ${r}
        echo; echo
    done
done
```

Now make sure `nginx` is running properly in each cluster:

```bash
for c in cluster1 cluster2; do
    NODE_PORT=$(kubectl --context=${c} -n test-namespace get service \
        test-service -o jsonpath='{.spec.ports[0].nodePort}')
    echo; echo ------------ ${c} ------------; echo
    curl $(echo -n $(minikube ip -p ${c})):${NODE_PORT}
    echo; echo
done
```

### Update FederatedNamespace Placement

Remove `cluster2` via a patch command or manually:

```bash
kubectl -n test-namespace patch federatednamespace test-namespace \
    --type=merge -p '{"spec": {"placement": {"clusterNames": ["cluster1"]}}}'

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

We can quickly add back all the resources by simply updating the
`FederatedNamespace` to add `cluster2` again via a patch command or
manually:

```bash
kubectl -n test-namespace patch federatednamespace test-namespace \
    --type=merge -p '{"spec": {"placement": {"clusterNames": ["cluster1", "cluster2"]}}}'

kubectl -n test-namespace edit federatednamespace test-namespace
```

Then wait and verify all resources are added back to `cluster2`:

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
successfully verified a working federation-v2 deployment.

#### Using Cluster Selector

In addition to specifying an explicit list of clusters that a resource should be propagated
to via the `spec.placement.clusterNames` field of a federated resource, it is possible to
use the `spec.placement.clusterSelector` field to provide a label selector that determines
that list of clusters at runtime.

If the goal is to select a subset of member clusters, make sure that the `FederatedCluster`
resources that are intended to be selected have the appropriate labels applied.

The following command is an example to label a `FederatedCluster`:

```bash
kubectl label federatedclusters -n federation-system cluster1 foo=bar
```

Please refer to [Kubernetes label command](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands#label)
to get more detail for how `kubectl label` works.

The following sections detail how `spec.placement.clusterNames` and
`spec.placement.clusterSelector` are used in determining the clusters that a federated
resource should be propagated to.

##### Neither `spec.placement.clusterNames` nor `spec.placement.clusterSelector` is provided

```yaml
spec:
  placement: {}
```

In this case, you can either set `spec: {}` as above or remove `spec` field from your
placement policy. The resource will not be propagated to member clusters.

##### Both `spec.placement.clusterNames` and `spec.placement.clusterSelector` are provided

```yaml
spec:
  placement:
    clusterNames:
      - cluster2
      - cluster1
    clusterSelector:
      matchLabels:
        foo: bar
```

For this case, `spec.placement.clusterSelector` will be ignored as
`spec.placement.clusterNames` is provided. This ensures that the results of runtime
scheduling have priority over manual definition of a cluster selector.

##### `spec.placement.clusterNames` is not provided, `spec.placement.clusterSelector` is provided but empty

```yaml
spec:
  placement:
    clusterSelector: {}
```

In this case, the resource will be propagated to all member clusters.

##### `spec.placement.clusterNames` is not provided, `spec.placement.clusterSelector` is provided and not empty

```yaml
spec:
  placement:
    clusterSelector:
      matchLabels:
        foo: bar
```

In this case, the resource will only be propagated to member clusters that are labeled
with `foo: bar`.

### Example Cleanup

To cleanup the example simply delete the namespace:

```bash
kubectl delete ns test-namespace
```

### Troubleshooting

If federated resources are not propagated as expected to the member clusters, you can
use the following command to view `Events` which may aid in diagnosing the problem.

```bash
kubectl describe <federated CRD> <CR name> -n test-namespace
```

An example for CRD of `federatedserviceaccounts` is as follows:

```bash
kubectl describe federatedserviceaccounts test-serviceaccount -n test-namespace
```

It may also be useful to inspect the federation controller log as follows:

```bash
kubectl logs -f federation-controller-manager-0 -n federation-system
```

## Namespaced Federation

All prior instructions referred to the deployment and use of a
cluster-scoped federation control plane. It is also possible to
deploy a namespace-scoped control plane. In this mode of operation,
federation controllers will target resources in a single namespace on
both host and member clusters. This may be desirable when
experimenting with federation on a production cluster.

### Helm Configuration

To deploy a federation in a namespaced configuration, set
`global.limitedScope` to `true` as per the Helm chart [install
instructions](https://github.com/kubernetes-sigs/federation-v2/blob/master/charts/federation-v2/README.md#configuration).


### Joining Clusters

Joining additional clusters to a namespaced federation requires
providing additional arguments to `kubefed2 join`:

- `--federation-namespace=<namespace>` to ensure the cluster is joined
  to the federation running in the specified namespace

To join `mycluster` when `FEDERATION_NAMESPACE=test-namespace` was used for deployment:

```bash
./bin/kubefed2 join mycluster --cluster-context mycluster \
    --host-cluster-context mycluster --add-to-registry --v=2 \
    --federation-namespace=test-namespace
```

## Local Value Retention

In most cases, the federation sync controller will overwrite any
changes made to resources it manages in member clusters.  The
exceptions appear in the following table.  Where retention is
conditional, an explanation will be provided in a subsequent section.

| Resource Type  | Fields                    | Retention   | Requirement                                                                  |
|----------------|---------------------------|-------------|------------------------------------------------------------------------------|
| All            | metadata.resourceVersion  | Always      | Updates require the most recent resourceVersion for concurrency control.     |
| Scalable       | spec.replicas             | Conditional | The HPA controller may be managing the replica count of a scalable resource. |
| Service        | spec.clusterIP,spec.ports | Always      | A controller may be managing these fields.                                   |
| ServiceAccount | secrets                   | Conditional | A controller may be managing this field.                                     |

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
federation will be retained if the managing federated resource does
not specify a value for the field.  This avoids the possibility of the
sync controller attempting to repeatedly clear the field while a local
serviceaccounts controller attempts to repeatedly set it to a
generated value.

## Higher order behaviour

The architecture of federation v2 API allows higher level APIs to be constructed using the
mechanics provided by the standard form of the federated API types (containing fields for
`template`, `placement` and `override`) and associated controllers for a given resource.
Further sections describe few of higher level APIs implemented as part of Federation V2.

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
to actually propagate the k8s resource to federated clusters. Its noteworthy that
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
apiVersion: scheduling.federation.k8s.io/v1alpha1
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
apiVersion: scheduling.federation.k8s.io/v1alpha1
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
apiVersion: scheduling.federation.k8s.io/v1alpha1
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
apiVersion: scheduling.federation.k8s.io/v1alpha1
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
apiVersion: scheduling.federation.k8s.io/v1alpha1
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
