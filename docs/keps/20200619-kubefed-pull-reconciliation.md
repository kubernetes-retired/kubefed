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

# Kubefed v2 -- Architectural changes

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Summary

Design the new kubefed architecture to improve scalability and performance of
kubefed on large scale scenarios.

## Motivation

All Kubefed logic is computed in the control-plane, that can represents a bottleneck
whenever the amount of clusters and/or federated resources increases.
The community has shared their intention to transition from a push-reconciler
to a pull-reconciler approach.
Scalability is important but also performance to avoid high response times when
managing the lifecycle of certain federated resources.
Additionally, users prefer to avoid giving write permissions of their clusters to
store them in the control-plane cluster.

### Goals

* Design a pull-based reconciliation.
* Scale kubefed to manage thousands of resources without major penalties.
* Design a new architecture that could easily be extended to add new functionalities in the future.
* Promote towards a more decentralized computation model.
* The management of what is federated and where must remain centralized to the control-plane cluster.

## Non Goals

* Migrate all the controllers to use the controller-runtime. This intention exists
but is not the ultimate goal during this refactoring.
* Improve the status property of federated resources to reflect the current state of
the resources and not the propagation status only.

## Proposal

This approach aims to split the current kubefed architecture into two parts: agents (aka kubefed daemons) and a control-plane.

The kubefed agents are daemons running in the registered/joined kubefed clusters.
The control-plane logic changes to use a pull-based reconciliation and rely on
the kubefed daemons running on the target clusters to perform most of the computation.
The federated resource management controllers remain in the cluster where the control-plane is deployed.

<img src="./images/kubefedArch.jpg">

### Kubefed Daemon

On each registered kubefed cluster, a daemon should be deployed to watch the state
of the cluster and federated resources. Likewise this daemon should periodically
reconcile the desired state to create/delete/update the federated resources.

As done in the past, the `kubefedctl join` command creates a kubefed cluster in
the control-plane cluster, and now deploys the kubefed daemon in the target cluster.
This operation also exchanges the required permissions (tokens, kubeconfigs) and url to enable
the communication between the control-plane and the daemons, and viceversa.

In the following we explain how this bi-directional communication occurs:

* `control-plane --> kubefed daemon`: the control-plane reads the status of the cluster
and federated resources in the kubefed clusters.
This operation is important to determine the status of the propagation of resources.

* `kubefed daemon --> control-plane`: the daemon needs to reconcile with the control-plane
to be synced with the federated resources to create/update/delete.
This is crucial to keep the kubefed clusters in sync with the desired state
defined in the control-plane.

#### Kubefed Daemon Handler

This new component, named `kubefed daemon`, exposes certain endpoints to report
the cluster health and the federated resources.

Initially the kubefed daemon exposes the following endpoints:

* `/healthz`: This endpoint returns information about the health of the kubefed cluster.
This health is defined by the healthz of the kubefed daemon and cluster Kubernetes healthz
API endpoints.

* `/federatedresources/notready`: This endpoint returns all the federated resources
that crashed or have an unknown status.
By default, the federated resource without status, are considered as `ready` whenever their creation/update succeeded, otherwise they are not ready.
The intention of this endpoint is to filter the chunk of data periodically exchanged with the
control-plane to report the status of the resources.
The data returned is compared with the expected resources that should be ready.
To determine the `Readiness` of a resource, kubefed daemon will perform a `best-effort` approach that varies based on the
status schema of each resource. To reduce the complexity, resources with a custom
schema in their status are considered `ready` if none error is detected.

* `/federatedresources/ready`: (optional...) This endpoint returns all the federated
`ready` resources.
This endpoint might not be useful due to the amount of data exchange with the control-plane
it can be unmanageable for all the clusters.

* `/federatedresources/{resource}/{namespace}/{name}`: This endpoint returns a specific
federated namespaced resource.

* `/federatedresources/{resource}/{name}`: This endpoint returns a specific
federated non-namespaced resource.

* `/federatedresources`: This endpoint returns the list of federated resources that
are currently allocated in this kubefed cluster.
The daemon creates a `Lister` per `FederatedTypeConfig`, and in particular per `TargetType`, to track the federated resources.
The `Lister` filters all the resources in the cluster from those with a label selector `kubefed.io/managed=true`, so the `Lister` only contains federated resources.

A cache should be used to reduce the response times of these operations.

The daemon creates two servers one to serve the mentioned endpoints and another server
to expose custom metrics.

The `KubefedAgentHandlerConfig` represents a draft of a handler with the different
functions used to expose the aforementioned routes.

```go
type KubefedAgentHandlerConfig struct {
	GetHealthz   HealthzHandlerFunc
  GetFederatedResource GetFederatedResourceHandlerFunc
  GetFederatedResources GetFederatedResourcesHandlerFunc
  // GetNotReadyFederatedResources is meant to enumerate the non-ready federated resources
  GetNotReadyFederatedResources NotReadyFederatedResourcesListerFunc
  GetReadyFederatedResources ReadyFederatedResourcesListerFunc
  StreamIdleTimeout     time.Duration
  StreamCreationTimeout time.Duration
}
```

#### Collect the Resource Status

The kubefed daemons have to periodically collect the status of the federated resources in each
cluster.

The resources can be filtered by the label `kubefed.io/managed=true` to exclude them from the rest of Kubernetes resources.
In addition to that, the kubefed daemon periodically checks the type of the federated
resources and create an `Informer` per `FederatedTypeConfig`.
The list of `FederatedTypeConfig` available resources is defined in the control plane cluster.
There is no need to create an `Informer` per kind of resource type in the Kubernetes cluster.
The `Informer` populates the cache with the status of the federated resources.

In order to know the `TargetType` of the federated resources, the control-plane exposes an endpoint to know which type of resources are federated:

* `/federatedtypeconfigs`: This endpoint returns the list of the `FederatedTypeConfig` resources at any time.


#### Desired State Reconciliation

Every daemon is responsible of keeping on sync which federated resources need to be created/updated/deleted
in its managed cluster.

To do so, the kubefed talks to the control-plane to be informed of which resources
need to be federated in the cluster.

Likewise this reconciliation loop is in charge of reverting any local changes done in a cluster to
a federated resource.
This ensures that the desired state defined in the control-plane cluster is enforced
in the target clusters.

In the `kubefed control-plane` section, we present the alternative pull-based reconciliation models
and how the daemon and control-plane work together to enforce the desired state at any time.

### Kubefed control-plane

The control-plane watches the kubefed clusters consuming the exposed endpoints to be aware
of issues in relation to the clusters and the federated resources.

A critical operation is to constantly watch the status of the federated resources.

<img src="./images/kubefedv2Example.jpg">

When requesting the status of a federated resource, the result needs to report the status
of the propagation and its own current state.

Another crucial operation is the reconciliation of the desired state, the control-plane
works as a centralized system where the customer defines what and where a federated resource
is created.
Keeping the desired state synced with the state of the clusters would define the success of this new architecture.

A common operation that would be consumed by the daemons consists on an endpoint to list the available federated types:

* `/federatedtypeconfigs`: This endpoint returns the list of the `FederatedTypeConfig` resources.
To avoid creating `Informers` in all the kubefed clusters against all the resources, the ideal approach
is to be aware of the types of resources federated at any time.

Next we present four alternatives or models to keep the desired state synced
across all the cluters.

#### Option 1 - Desired State Reconciliation - Subscriber/Publisher

This approach follows a subscriber/publisher model where the desired state
is published to the different kubefed clusters in the shape of events (distributed messaging).
These events represent the create/update/delete actions triggered against the federated
resources in the control-plane cluster.
In other words, any action triggered in a federated resource action forwards the respective event
to the kubefed member clusters defined in the `placement` property of the resource.

A creation of a federated resource publishes a `CreateEvent` similar to the `request.Reconcile` objects
in the controller-runtime (including the object itself).
This `CreateEvent` contains the object and would be published to the queue of the respective kubefed clusters.
There is a queue per kubefed cluster where to publish the events.
Once a message is published into the queue, the kubefed daemon receives it and
process the event which creates a new resource in the cluster.
Then the daemon is responsible of applying this new event to ensure the desired state.

If we compare this mechanism with how a controller works where an internal queue is created,
it is similar but in this case the controller does not ensures the desired state in the remote
clusters.
It forwards the events to the target kubefed clusters for them to reach the defined state.
If the event failed or could not be forwarded, then the event is re-queued.
This model presents a level triggered reconciliation which differs from Kubernetes
native edge triggered reconciliation.
To avoid all the out-of-sync problems of level triggered reconciliation, a full periodic
reconciliation should be required to pull the desired state from the control-plane cluster
and enforce that desired state.

With this model the reconciliation of the desired state is a job of the daemons.
This new mechanism decentralized the reconciliation of the desired state and reliefs
the control plane of all this workload.

Likewise the kubefed daemons are the responsible to listen on the channels to compute
coming events, and perform a full periodic reconciliation.

Note that the mechanism of creation of a publisher and subscribe to a specific queue takes place
when a kubefed cluster joins the federation.

##### Analysis

Obviously this new approach requires of a third party software (e.g. nats-server, nsq).
Likewise daemons need to perform a full periodic reconciliation and request to the
control plane the desired state. So the daemons need to be able to talk to the control plane
to do this full reconciliation.

In this approach the networking and computation load is minimal.
The data exchanged is equal to the amount of events triggered against the federated resources.

This pub/sub eliminates the need of the daemon to constantly ask for the desired state,
the control-plane publishes the events, and the daemon applies it.

The security of this communication would rely on the features of the used third party.

#### Option 2 - Desired State Reconciliation - Watch Federated types

Every kubefed daemon has to create a remote informer to watch any changes in the federated
types which all belong to the same group `types.kubefed.io`.
This remote informer is created against the control-plane cluster, so a kubeconfig
and the required permissions should be granted to all the kubefed clusters.
A controller uses that `Informer` and trigger the respective operations to reach the
desired state of the kubefed cluster.

The main challenge of this approach is filtering what federated resources a kubefed
daemon should be able to watch.
In other words, a watch should be only aware of changes in the resources assigned to this
cluster.
This could be done adding labels to the federated resources in addition to RBAC
settings in the control plane cluster to only allow `get` to the resources that
are to be federated out.

##### Analysis

This approach requires of a constant bi-directional net-link between control-plane and kubefed
to be able to reach the desired state.

The main challenge is an ideal filtering of the resources that each cluster should view.


#### Option 3 - Desired State Reconciliation - Crossed endpoints

In this approach, the kubefed daemon constantly fetches the list of federated resources
that represents the desired state.

To do so, the control-plane should exposes a specific endpoint per cluster:

* `/federatedresources/{cluster}`: this endpoint returns the list of all the federated
resources allocated to that cluster.

Obviously a cache should be used to reduce the response times and certain reconcile
loops should watch on changes in the federated types to keep these lists up-to-date
for each cluster.

Once the daemon queries that endpoint with the desired state, it compares
it with the current (local) state, and applies the desired changes such as resource deletion, updates
or creations.

As a possible modification to this alternative, a more efficient protocol can be defined
to reduce the amount of data exchange:

* The `control-plane` exposes two endpoints: one to define a `versionID` per cluster to report changes in the
state.
A `versionID` defines a current state. For instance, if new federated resources are created
or updated, the `versionID` should change.

* The `daemon` periodically asks if there is anything new for him to the control-plane.
The daemon has to store a `versionID` that represents the current (local) state for comparison with
the desired state defined for the cluster in the control-plane.
If the `versionID` differs then the daemon requests the desired state from the control-plane
`/federatedresources/{cluster}`.
Finally, the kubefed daemon needs to reconcile the desired state with the local state.

##### Analysis

This model is a bit more expensive in terms of computation and networking data exchanged.
The daemon has to collect all the federated resources and compute the list
to determine if a resource changed or not.

The computation of ensuring the desired state in comparison with the expected list of
resources brings more workload than necessary with other approaches to the daemons.

In addition, this solution requires to maintain all these API endpoints which is
tedious and hard to maintain.

#### Option 4 -- Centralized Resource Management -- GitOps

In order to avoid the `kubefed daemon --> control-plane` communication, I thought
about using a centralized component where the federated resources could be created
to be consumed by all the kubefed clusters.

This centralized component represents a hybrid approach of using the federation api to define configuration, and then generating a canonical form into a git repository for gitOps.
This alternative solution might represent a good solution for some use cases.

##### Analysis

As an administrator, the admins prefer to have the RBAC rules in the cluster,
and not outsourced to a git repository with rules about who gets to push what where.

With this alternative, the communication flow goes against the gitOps repository or
centralized datastore where the desired state is stored.

Analogously to the `Option 1`, this alternative adds a new third party in the game
which might increase the complexity of the whole architecture.

In terms of security, the system should rely on what each cluster is allowed to view,
likewise it is hard to understand how `overwrites` would work in a per cluster level.

### Controller Resource Propagation Status - creation/update/creation

The creation of a federated resource triggers an asynchronous process that updates
the state of the operation once applied to all the Kubefed clusters.

These actions trigger operations in the control-plane and kubefed clusters.

The following list represents the different steps when creating a resource:

1. Create federated resource.
2. Update status of the federated resource to reflect the initiation of the propagation (e.g. `Propagating`).
3. A propagation reconcile loop is triggered to update the the status of this resource until the completion of this operation.
4. As part of the reconciliation loop, the system checks which kubefed clusters
are specified as part of the `placement` for this resource. Next, the system verifies
if the federated resource exists, if it does the new status is changed to `Propagated`, and the
reconciliation loop ends successfully. Otherwise the reconcile request is re-queue
until the resource is created in the target cluster or an error is reported (`CreationTimedOut` or so).

A similar process applies to deletion and update operations on the federated resources.
However, in a deletion of a federated resource, this operation is considered completed when the resource
does not exist on any of the allocated clusters.

### Controller Resource Status

The control-plane periodically queries the kubefed cluster daemons to be aware of the
status of the federated resources on each cluster.
Note that this status differs from the propagation status and is oriented to report
issues on the federated resources.

To reduce the load of this periodic reconcile loop that updates the status of the federated
resources, the control-plane requests from the daemons the `notready` resources.

### Other Main Reconciliation Loops

For the `ServiceDNSRecord` controller, Kubefed `v2` should omit this functionality and rely on
other solutions such as service meshes.
The purpose of Kubefed should be to ultimately federate resources across clusters.
Another option is to expose an additional endpoint in the daemons to be aware of the
`serviceDNS`, therefore the control-plane does not need to create remote `Informer` against
all the kubefed clusters.

The future of `ServiceDNSRecord` controller is something to decide with the community.
It goes one step beyond by dealing with networking considerations when talking to apps deployed across clusters.

On the other hand, the controllers in charge of the scheduler preferences remains as part of the control-plane.
The enforcement of these preferences rely on the control-plane and daemons, so nothing would change there.

The rest of kubefed cluster related operations would be managed by controllers in the control-plane.

### Kubefed Security Concerns

With this new approach, the kubefed daemon only exposes read permissions to ensure
the reconciliation of the current status of the federated resources.
This differs from the current architecture where the control-plane has write access
to the kubefed clusters.

A difference is the `control-plane` and `kubefed daemons` which need to communicate between themselves.
We need to define which kind of the communication is necessary based on the chosen option among
the defined alternatives.
However, if a bi-directional communication is required, I believe a good approach would be to follow the [SPIFEE Trust domain federation](https://docs.google.com/document/d/1OC9nI2W04oghhbEDJpKdIUIw-G23YzWeHZxwGLIkB8k/edit) approach.
To federate identity and trust, you must exchange the trust bundles between the `control-plane`
and `kubefed daemon` of each cluster.


### User Stories

#### Story 1

Users want to avoid giving cluster admin access to something like federation,
some admins have expressed distaste at that requirement.
If the management cluster is compromised that would give access to the kubefed clusters.

With this new proposal, the management cluster only has read access to the kubefed clusters.

#### Story 2

Kubefed uses push reconciliation, all the workload relies on the control plane running
on the management cluster. Due to the existing design decisions, kubefed represents
a bottleneck when having to manage thousands of resources across many clusters.

With this new proposal, Kubefed would use a pull-based reconciliation mechanism
and most of the workload is shared between the kubefed agents and the control-plane.

#### Story 3

Customers might have an infrastructure setup where the kubefed clusters are behind
some NAT gateways, and consequently the control-plane cannot reach the kubefed clusters.

With this new approach, the kubefed daemon can register with the control plane and sets up
a bi-directional tunnel.
