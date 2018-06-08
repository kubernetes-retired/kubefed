# Cluster Registry

[![GoReportCard Widget]][GoReportCard] [![GoDoc Widget]][GoDoc] [![Slack Widget]][Slack]

[GoDoc]: https://godoc.org/k8s.io/cluster-registry
[GoDoc Widget]: https://godoc.org/k8s.io/cluster-registry?status.svg
[Slack]: http://slack.kubernetes.io#sig-multicluster
[Slack Widget]: https://s3.eu-central-1.amazonaws.com/ngtuna/join-us-on-slack.png
[GoReportCard Widget]: https://goreportcard.com/badge/k8s.io/cluster-registry
[GoReportCard]: https://goreportcard.com/report/k8s.io/cluster-registry

A lightweight tool for maintaining a list of clusters and associated metadata.

# What is it?

The cluster registry helps you keep track of and perform operations on your
clusters. This repository contains an implementation of the Cluster Registry API
([code](https://github.com/kubernetes/cluster-registry/tree/master/pkg/apis/clusterregistry),
[design](docs/api_design.md)) as a Kubernetes Custom Resource Definition, which
is the canonical representation of the cluster registry.

# Documentation

Documentation is in the [`docs`](docs) directory.

Most directories containing Go code have package documentation, which you can
view on [Godoc](https://godoc.org/k8s.io/cluster-registry). Most directories
that have interesting content but do not have Go code have README.md files that
briefly describe their contents.

# Getting involved

The cluster registry is still a young project, but we welcome your
contributions, suggestions and input! Please reach out to the
[kubernetes-sig-multicluster](https://groups.google.com/forum/#!forum/kubernetes-sig-multicluster)
mailing list, or find us on
[Slack](https://github.com/kubernetes/community/blob/master/communication.md#social-media)
in [#sig-multicluster](https://kubernetes.slack.com/messages/sig-multicluster/).

## Maintainers

-   [@font](https://github.com/font)
-   [@madhusudancs](https://github.com/madhusudancs)
-   [@perotinus](https://github.com/perotinus)

# Development

Basic instructions for working in the cluster-registry repo are
[here](docs/development.md).
