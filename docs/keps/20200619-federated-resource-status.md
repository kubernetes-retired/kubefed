---
kep-number: 20200619
short-desc: Kubefed -- Federated Resources Status
title: Kubefed -- Federated Resources Status
authors:
  - "@hectorj2f"
reviewers:
  - "@irfan"
  - "@hectorj2f"
  - "@jimmidyson"
  - "@pmorie"
approvers:
- "@irfan"
- "@jimmidyson"
- "@pmorie"
editor: TBD
creation-date: 2020-06-19
last-updated: 2020-06-19
status: provisional
---

# Kubefed -- Federated Resources Status

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
<!-- /toc -->

## Summary

Kubefed needs to improve its definition of status for the federated resources.
Users lack of proper visibility over the status of the federated
resources. For instance, if you federated a deployment the federated status should
report if the deployment failed or error at any time.

A federated resource only reflects the status of propagation actions, but it doesn't
reflect the status if whether the resource is running or failed.

## Motivation

Nowadays users have to connect to the kubefed clusters to be aware if the federated
resources are healthy or not across clusters.

### Goals

* Quickly identify unhealthy federated resources by relying on the status of the federated resources.
* Improve the troubleshooting of failures when propagating resources.


## Proposal

Kubefed reports a limited set of states for the federation of resources.


```go
CreationTimedOut     PropagationStatus = "CreationTimedOut"
UpdateTimedOut       PropagationStatus = "UpdateTimedOut"
DeletionTimedOut     PropagationStatus = "DeletionTimedOut"
LabelRemovalTimedOut PropagationStatus = "LabelRemovalTimedOut"

AggregateSuccess       AggregateReason = ""
ComputePlacementFailed AggregateReason = "ComputePlacementFailed"
NamespaceNotFederated  AggregateReason = "NamespaceNotFederated"

PropagationConditionType ConditionType = "Propagation"
```

However the current federated resource properties don't help to track the status of the deployed resources in the kubefed clusters.

The idea is to extend current GenericFederatedStatus with the Status of the resources:

```go

type GenericFederatedStatus struct {
  	ObservedGeneration  int64                  `json:"observedGeneration,omitempty"`
  	Conditions          []*GenericCondition    `json:"conditions,omitempty"`
  	Clusters            []GenericClusterStatus `json:"clusters,omitempty"`
}

type GenericFederatedResource struct {
	metav1.TypeMeta                `json:",inline"`
	metav1.ObjectMeta              `json:"metadata,omitempty"`

	Status *GenericFederatedStatus `json:"status,omitempty"`
}

```

Nowadays `Conditions` hold the status of the federated actions (aka propagation status).
In other words, it defines the conditions of the propagation status for a resource.

```yaml
- apiVersion: types.kubefed.io/v1beta1
  kind: FederatedDeployment
  metadata:
    finalizers:
    - kubefed.io/sync-controller
    generation: 1
    name: asystem
    namespace: asystem
    resourceVersion: "70174497"
  spec:
    placement:
      clusters:
      - name: cluster3
      - name: cluster2
      - name: cluster1
    template:
      metadata:
        labels:
          app: nginx
      spec:
        replicas: 3
        selector:
          matchLabels:
            app: nginx
        template:
          metadata:
            labels:
              app: nginx
          spec:
            containers:
            - image: nginx
              name: nginx
  status:
    conditions:
    - lastTransitionTime: "2020-05-25T19:47:59Z"
      lastUpdateTime: "2020-05-25T19:47:59Z"
      status: "True"
      type: Propagation
```

`status.conditions` reports the latest status which defines the state of the propagation.
Obviously it is not necessary to report all the clusters for which the propagation when
successful.

The intention in this proposal is to extend the current available `Conditions` to
hold the status of the federated resources, e.g Ready, NotReady.

The status of the federated resources should determine whether the resources satisfy
a `Ready` condition, and otherwise report their error status.
To do so, this property reports the status of the federated resources in their
target clusters whenever a `ReadyCondition` is not satisfied.
This condition would need to be identified per type or by the usage of an interface
`IsReady` that determines this value per type of resource.
By doing so, we ensure the `Conditions` property shows the status of only unhealthy
resources.

If we re-use the example from above and imagine a scenario where this `FederatedDeployment` resource remained `Ready=True` in two clusters, but crashed in `cluster3`.
The value of `status.conditions` should reflect the new state for that specific cluster.

```yaml
- apiVersion: types.kubefed.io/v1beta1
  kind: FederatedDeployment
  metadata:
    finalizers:
    - kubefed.io/sync-controller
    generation: 1
    name: asystem
    namespace: asystem
    resourceVersion: "70174497"
  spec:
    placement:
      clusters:
      - name: cluster2
      - name: cluster1
    template:
      metadata:
        labels:
          app: nginx
      spec:
        replicas: 3
        selector:
          matchLabels:
            app: nginx
        template:
          metadata:
            labels:
              app: nginx
          spec:
            containers:
            - image: nginx
              name: nginx
  status:
    clusters:
    - name: "cluster3"
      status: "ReplicaFailure"
    conditions:
    - lastTransitionTime: "2020-05-25T20:23:59Z"
      lastUpdateTime: "2020-05-25T20:23:59Z"
      status: "True"
      type: "NotReady"
```

The value of `spec.conditions` contains a `NotReady` condition type, that is the
result of checking the status of that remote `Deployment` in the target clusters.
The value of `status.clusters.status` can be extracted from the `"ReplicaFailure"`
status of the `Deployment` and so reused for visibility.

Likewise, the status of the `FederatedDeployment` remains `Ready=True` in the rest
of clusters: `cluster1` and `cluster2`.

If a federated resource does not have a status field, a successful creation/update would
reflect its readiness. Then the `ReadyCondition` would be satisfied by its creation.
For these resources the value of `Conditions` would rely on the value of the propagation state of that resource.
An example could be a `ClusterRole` resource that doesn't have a status property, but
kubefed nowadays reports if the propagation of that resource worked.

```yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedClusterRole
metadata:
  name: test-clusterrole
spec:
  template:
    rules:
    - apiGroups:
      - '*'
      resources:
      - '*'
      verbs:
      - '*'
  placement:
    clusters:
    - name: cluster2
    - name: cluster1
status:
  conditions:
  - lastTransitionTime: "2020-05-25T19:47:59Z"
    lastUpdateTime: "2020-05-25T19:47:59Z"
    status: "True"
    type: Propagation
```

However, there is a problem with this approach, the status schema varies based on the custom
resource. Unfortunately that brings a problem when determining if a federated resource
is ready or not.

In the following, there is a list of Status objects of different custom resource definitions:

```go
type AddonStatus struct {
	Ready bool          `json:"ready" yaml:"ready"`
	Stage status.Status `json:"stage,omitempty" yaml:"stage,omitempty"`
}

type PodStatus struct {
  Phase PodPhase
 ...
 }

 type ServiceStatus struct {
  LoadBalancer LoadBalancerStatus
 }

// KonvoyClusterStatus defines the observed state of KonvoyCluster
type KonvoyClusterStatus struct {
	// Phase represents the current phase of Konvoy cluster actuation.
	// E.g. Pending, Provisioning, Provisioned, Deleting, Failed, etc.
	// +optional
	Phase KonvoyClusterPhase `json:"phase,omitempty"`

  ...
}
```

As mentioned, their Status schema is quite different from one to another.

Consequently, the intention is to **enforce** in the federated resources this approved [recommendation](https://github.com/kubernetes/enhancements/pull/1624/files).
to expect a common schema for `.status.conditions` and share golang logic for common Get, Set, Is for `.status.conditions`.

1. For all new APIs, have a common type for `.status.conditions`.
2. Provide common utility methods for `HasCondition`, `IsConditionTrue`, `SetCondition`, etc.
3. Provide recommended defaulting functions that set required fields and can be embedded into conversion/default functions.

By following this approach, kubefed would be able to properly consume and report the status
of any federated resource by checking the `status.conditions` (e.g `Ready=True`) fields.


### User Stories

#### Story 1

Users create federated resources and want to be aware of their status without having
to access to the remote clusters.

In the following example, we have a `FederatedAddon`, named `reloader`, deployed across ten `kubefedclusters`.
An `Addon` is a custom resource definition that abstract the creation of apps composed of
one or multiple Helm charts.

```yaml
---
apiVersion: types.kubefed.io/v1beta1
kind: FederatedAddon
metadata:
  name: reloader
  namespace: kubeaddons
spec:
  placement:
    clusters:
    - name: cluster10
    - name: cluster9
    - name: cluster8
    - name: cluster7
    - name: cluster6
    - name: cluster5
    - name: cluster4
    - name: cluster3
    - name: cluster2
    - name: cluster1
  template:
    metadata:
      labels:
        kubeaddons.mesosphere.io/name: reloader
    spec:
      chartReference:
        chart: reloader
        repo: https://stakater.github.io/stakater-charts
        values: |
          ---
          reloader:
            deployment:
              resources:
                limits:
                  cpu: "100m"
                  memory: "512Mi"
                requests:
                  cpu: "100m"
                  memory: "128Mi"
        version: v0.0.49
      kubernetes:
        minSupportedVersion: v1.15.6
status:
  conditions:
  - lastTransitionTime: "2020-05-25T19:47:59Z"
    lastUpdateTime: "2020-05-25T19:47:59Z"
    status: "True"
    type: Propagation        
```

At any specific time, this `FederatedAddon` crashed on three clusters.
As a consequence, the value of its status should look similar to this:

```yaml
---
apiVersion: types.kubefed.io/v1beta1
kind: FederatedAddon
metadata:
  name: reloader
  namespace: kubeaddons
spec:
  placement:
    clusters:
    - name: cluster10
    - name: cluster9
    - name: cluster8
    - name: cluster7
    - name: cluster6
    - name: cluster5
    - name: cluster4
    - name: cluster3
    - name: cluster2
    - name: cluster1
  template:
    metadata:
      labels:
        kubeaddons.mesosphere.io/name: reloader
    spec:
      chartReference:
        chart: reloader
        repo: https://stakater.github.io/stakater-charts
        values: |
          ---
          reloader:
            deployment:
              resources:
                limits:
                  cpu: "100m"
                  memory: "512Mi"
                requests:
                  cpu: "100m"
                  memory: "128Mi"
        version: v0.0.49
      kubernetes:
        minSupportedVersion: v1.15.6
status:
  status:
    clusters:
    - name: "cluster1"
      status: "Failed"
    - name: "cluster2"
      status: "Failed"               
    - name: "cluster3"
      status: "Failed"  
  conditions:
  - lastTransitionTime: "2020-05-25T19:47:59Z"
    lastUpdateTime: "2020-05-25T19:47:59Z"
    status: "True"
    type: NotReady
```

`Failed` could be extracted from the status of an addon whose `status.stage` is `Failed` and
`status.ready` value is `false`.
