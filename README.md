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

<p align="center"><img src="docs/images/propagation.png" width="711"></p>

# Concepts

The following abstractions support the propagation of a logical
federated type:

- Template: defines the representation of the resource common across clusters
- Placement: defines which clusters the resource is intended to appear in
- Override: optionally defines per-cluster field-level variation to apply to the template

These 3 abstractions provide a concise representation of a resource
intended to appear in multiple clusters.  Since the details encoded by
the abstractions are the minimum required for propagation, they are
well-suited to serve as the glue between any given propagation
mechanism and higher-order behaviors like policy-based placement and
dynamic scheduling.

# Guides

## Development Guide

Take a look at our [development guide](docs/development.md) if you are
interested in contributing.

## Code of Conduct

Participation in the Kubernetes community is governed by the
[Kubernetes Code of Conduct](./code-of-conduct.md).
