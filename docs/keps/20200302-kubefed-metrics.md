---
kep-number: 0
short-desc: Kubefed Custom Metrics
title: Kubefed Custom Metrics
authors:
  - "@hectorj2f"
reviewers:
  - "@jimmidyson"
  - "@pmorie"
  - "@xunpan"
approvers:
  - "@jimmidyson"
  - "@pmorie"
  - "@xunpan"
editor: TBD
creation-date: 2020-03-02
last-updated: 2020-03-02
status: provisional
---

# Kubefed Custom Metrics

## Table of Contents

* [Kubefed Custom Metrics](#kubefed-custom-metrics)
  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non\-Goals](#non-goals)
  * [Proposals](#proposals)
    * [Metrics](#metrics)
    * [Risks and Mitigations](#risks-and-mitigations)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Drawbacks](#drawbacks)
  * [Infrastructure Needed](#infrastructure-needed)

## Summary

This document describes the different metrics and valuable data that could be exposed
and consumed from Kubefed to create dashboards and better understand this engine.

## Motivation

We aim to define a generic strategy on how to identify, consume and expose
custom Kubefed metrics.


### Goals

* Identify which metrics should be exposed from Kubefed if possible.
* Define a set of Kubefed metrics that could be consumed by Prometheus tools.
* Specify the type of each metric (e.g histogram, gauge, counter, summary).
* Use these metrics to create Grafana dashboards.

### Non-Goals

* Technical details about the Grafana Dashbards.

## Proposals

Kubefed already exposes a small set of metrics. These are some of the default metrics provided by
the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/metrics), in particular, Kubefed only exposes the client-only metrics. The rest of metrics are not available because Kubefed was not implemented
using the `controller-runtime` utils.

The metrics
are exposed by a `/metrics` route on a [Prometheus friendly format](https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md).
A service monitor should be created to instruct Prometheus tools to scrape
the metrics from the Kubefed `metrics` service endpoint.

However the client-only metrics are not enough, and Kubefed custom metrics have to
be identified and exposed to better understand this engine and scalability challenges.


### Metrics

In the following we share a table with the relevant metrics:

Kubefed clusters states reflect the status of the cluster and is periodically checked.


The following metric aims to register the total number of Kubefed clusters on `ready`, `notready` and `offline` state:

* `kubefedcluster_total`: a gauge metric that holds the number Kubefed clusters in any of the three possible states.
   To identify the type of state, we add a label `state` to this metric with the value of the state.

In addition to these metrics, we should also store the time this whole operation takes:

* `cluster_health_status_duration_seconds`: this `histogram` metric holds the duration in seconds of the action that checks
the health status of a Kubefed cluster.

Kubefed needs to connect to the remote clusters to validate/create/delete all the federated resources
in the target clusters. When having many clusters, the time invested on connecting
to remote clusters might be relevant:

* `cluster_client_connection_duration_seconds`: this `histogram` metric holds the duration in seconds of the creation
of a Kubernetes client to a remote cluster. This operation normally implies to connect to
the remote server to get certain metadata.

Kubefed federates resources on target clusters, and one of its controllers triggers
a periodic reconciliation of all target federated resources.

* `reconcile_federated_resources_duration_seconds`: this `histogram` metric holds the duration in seconds of the action that
reconcile federated resources in the target clusters.

Another operation that is relevant to record is the creation/update/deletion of
the propagated resources. This action is handled by the called dispatchers in Kubefed.

For this metric, we could choose a single metric that will include additional labels
to distinguish the different operations:

* `dispatch_operation_duration_seconds`: this `histogram` metric holds the duration in seconds of the creation/update/deletion
of the different propagated resources. The label `action` will hold the `create`, `update` and `delete` operations.

Regarding cluster join/unjoin operations, these metrics are also convenient to register:

* `joined_cluster_total`: a gauge metric that holds the number joined clusters.

* `join_cluster_duration_seconds`: this `histogram` metric holds the duration in seconds of the join cluster action.

* `unjoin_cluster_duration_seconds`: this `histogram` metric holds the duration in seconds of the unjoin cluster action.

To keep track of the rest of controllers and its reconciliation time, we will use a generic metric:

* `controller_runtime_reconcile_duration_seconds`: is a `histogram` which keeps track of the duration
of reconciliations for other Kubefed controllers. A label `controller` will allow to distinguish
the different controllers.

In addition to these metrics, we could add counters to register common error types.
This approach would make easy to track their rate on a dashboard.


#### Alternatives

### Implementation Details/Notes/Constraints

All the identified metrics in this document might be added to Kubefed in an incremental manner.

### Risks and Mitigations

## Graduation Criteria

## Implementation History

## Drawbacks

## Infrastructure Needed
