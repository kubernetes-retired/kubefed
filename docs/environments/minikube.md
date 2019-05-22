<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Minikube](#minikube)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Minikube

[Minikube](https://kubernetes.io/docs/getting-started-guides/minikube/)
provides one of the quickest way to set-up clusters for use with KubeFed.

**NOTE:** You will need to use a minikube version that supports
deploying a kubernetes cluster >= 1.13. [Recently
released](https://github.com/kubernetes/minikube/releases/latest)
versions of minikube (>= `0.32.0`) will satisfy this requirement.

Once you have minikube installed run:

```bash
minikube start -p cluster1 --kubernetes-version v1.13.4
minikube start -p cluster2 --kubernetes-version v1.13.4
```

Even though the `minikube` cluster has been started, you'll want to verify all
your `minikube` components are up and ready by examining the state of the
kubernetes components in the clusters via:

```bash
kubectl get all --all-namespaces
```

After all pods reach a Running status, you can return to the [User Guide](../userguide.md) to deploy the cluster
registry and KubeFed control plane.
