[![Build Status](https://travis-ci.org/kubernetes-sigs/federation-v2.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/federation-v2 "Travis")
[![Image Repository on Quay](https://quay.io/repository/kubernetes-multicluster/federation-v2/status "Image Repository on Quay")](https://quay.io/repository/kubernetes-multicluster/federation-v2)

# Kubernetes Federation v2

This repo contains an in-progress prototype of some of the
foundational aspects of V2 of Kubernetes Federation.  The prototype
builds on the sync controller (a.k.a. push reconciler) from
[Federation v1](https://github.com/kubernetes/federation/) to iterate
on the API concepts laid down in the [brainstorming
doc](https://docs.google.com/document/d/159cQGlfgXo6O4WxXyWzjZiPoIuiHVl933B43xhmqPEE/edit#)
and further refined in the [architecture
doc](https://docs.google.com/document/d/1ihWETo-zE8U_QNuzw5ECxOWX0Df_2BVfO3lC4OesKRQ/edit#).
Access to both documents is available to members of the
[kubernetes-sig-multicluster google
group](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster).

## Concepts

<p align="center"><img src="docs/images/concepts.png" width="711"></p>

Federation is configured with two types of information:

- **Type configuration** declares which API types federation should handle
- **Cluster configuration** declares which clusters federation should target

**Propagation** refers to the mechanism that distributes resources to federated
clusters.

Type configuration has three fundamental concepts:

- **Templates** define the representation of a resource common across clusters
- **Placement** defines which clusters the resource is intended to appear in
- **Overrides** define per-cluster field-level variation to apply to the template

These three abstractions provide a concise representation of a resource intended
to appear in multiple clusters. They encode the minimum information required for
**propagation** and are well-suited to serve as the glue between any given
propagation mechanism and higher-order behaviors like policy-based placement and
dynamic scheduling.

These fundamental concepts provide building blocks that can be used by
higher-level APIs:

- **Status** collects the status of resources distributed by federation across all federated clusters
- **Policy** determines which subset of clusters a resource is allowed to be distributed to
- **Scheduling** refers to a decision-making capability that can decide how 
  workloads should be spread across different clusters similar to how a human
  operator would

## Features

| Feature | Maturity |
|---------|----------|
| [Push propagation of arbitrary types to remote clusters](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/userguide.md#example) | Alpha |
| [CLI utility (`kubefedctl`)](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/userguide.md#operations) | Alpha |
| [Generate Federation APIs without writing code](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/userguide.md#enabling-federation-of-an-api-type) | Alpha |
| [Multicluster Service DNS via `external-dns`](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/servicedns-with-externaldns.md) | Alpha |
| [Multicluster Ingress DNS via `external-dns`](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/ingressdns-with-externaldns.md) | Alpha |
| [Replica Scheduling Preferences](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/userguide.md#replicaschedulingpreference) | Alpha |

## Guides

### User Guide

Take a look at our [user guide](docs/userguide.md) if you are interested in
using Federation v2.

### Development Guide

Take a look at our [development guide](docs/development.md) if you are
interested in contributing.

## Code of Conduct

Participation in the Kubernetes community is governed by the
[Kubernetes Code of Conduct](./code-of-conduct.md).
