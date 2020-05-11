---
kep-number: 0
short-desc: Cleanup federated resource on unjoin
title: Cleanup Federated Resource
authors:
  - "@jr0d"
reviewers:
  - "@jimmidyson"
  - "@hectorj2f"
approvers:
  - "@jimmidyson"
editor: TBD
creation-date: 2020-05-18
last-updated: 2020-05-04
status: provisional
---

# Cleanup federated resources

## Table of Contents

* [Cleanup Federated Resources](#cleanup-federated-resources)
  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
    * [User Story](#user-stroy)
    * [Goals](#goals)
  * [Proposals](#proposal)
    * [Implementation Details](#implementation-details)
      * [Best Effort Strategy](#best-effort-strategy)
      * [Required Strategy](#required-strategy)
    * [Addition ClusterConditionTypes (optional)](#additional-clusterconfigtypes-optional)
      * [Unjoining ClusterConfigurationType](#unjoining-clusterconfigurationtype)
      * [Failed ClusterConfigurationType](#failed-clusterconfigurationtype)
  * [Risks](#risks)
    * [Adding Cluster Conditions](#adding-cluster-conditions)

This document proposes a mechanism to specify that a federated resource should be removed from a managed cluster when leaving federation.

## Motivation

In practice, federation is often used to join clusters to some higher level centralized management plane, identity management for example. To achieve this, federation is used to propagate a configuration state to clusters which overrides an existing configuration state. When a cluster is unjoined, the federated configuration becomes invalid requiring the operator to manually remove the conflicting resources.

### User Story

As a cluster operator using kubefed, I would like to create resources which will automatically be removed from managed clusters during unjoin.

### Goals

* Define how users can indicate that a federated resource should be removed when a cluster leaves federation
* Define removal strategies

## Proposals

The APIResource struct should be updated to contain a `RemoveStrategy` member. `RemoveStrategy` will initially support two values, `BestEffort` and `Required`.

  * The `BestEffort` strategy will **not** prevent a cluster from unjoining when there are errors removing the resource.
  * The `Required` strategy will halt the unjoin process and set the KubefedCluster resource in a failed state when encountering errors. Unjoining is blocked until all resources with the `Required` `RemoveStrategy` have been removed successfully.

By default, `RemoveStrategy` will be nil on all federated types. A user must explicitly set a removal strategy on the resource they are creating. Types without a removal strategy will not be removed.


### Implementation details

* Update APIResource to contain a new member:
  ```
    // RemoveStrategy indicates what strategy should be employed when removing
    // federated resources during nnjoin
    RemoveStrategy *RemoveStrategy `json:"removeStrategy,omitempty"`
  ```

  ```
    type RemoveStrategy string

    const (
      RemoveStrategyBestEffort RemoveStrategy = "BestEffort"
      RemoveStrategyRequired   RemoveStrategy = "Required"
    )
  ```
* When labels are applied to federated resources (ApplyOverrides), add the following label if RemoveStrategy is !nil:
  * `kubefed.io/remove: best-effort || required`
* During unjoin, query for managed (`kubefed.io/managed: true`) resources with the `kubefed.io/remove` label: `[{ key: "kubefed.io/remove", operator: "Exists" }]`

#### Best Effort Strategy

Errors or timeouts that occur while removing resources with the best effort removal strategy should be logged but should not halt the unjoining process.

#### Required Strategy

Resources with the "required" removal strategy must be removed successfully on unjoin. After removing the resource, the client must verify that the resource is no longer present. If a resource cannot be removed:

* UnjoinCluster should return a formatted error containing a list of resources which could not be removed
* A ClusterCondition should be added to the KubefedCluster status with a message indicating which resources could not be removed
* (optional) Set the ConditionType to `Failed`


### Additional ClusterConditionTypes (optional)

Presently, there are only two ClusterConditionTypes, `Ready` and `Offline`. Given the `required` remove strategy, a `KubefedCluster` may end up in a state where the cluster is `Ready` but is in the process of "Unjoining". Adding additional conditions will make it easier to communicate federation life cycle status to the cluster operator.

#### Unjoining ClusterConditionType

In continuation of the existing nomenclature, the `Unjoining` condition should be set on a cluster when the operated initiates `UnjoinCluster`.

#### Failed ClusterConditionType

In the event of an error, such as failure to remove a resource with a `required` remove strategy, the cluster should enter a `Failed` condition.

## Risks

### Adding Cluster Conditions

Adding additional cluster states (conditions) may be outside the scope of this proposal. It is not necessary to achieve the goals of the proposal, but does aid in operator experience. At the same time, it also increases implementation complexity.
