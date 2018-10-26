# Minikube

[Minikube](https://kubernetes.io/docs/getting-started-guides/minikube/) provides the quickest way to set-up clusters for
use with the Federation v2 control plane.

**NOTE:** You will need to use a minikube version that supports deploying a
kubernetes cluster >= 1.11. Currently there is no released version of minikube
that supports kube v1.11 with profiles so you'll need to either:

- Build minikube from master by following these
   [instructions](https://github.com/kubernetes/minikube/blob/master/docs/contributors/build_guide.md).
- Or use a recent CI build such as [this one from PR
   2943](http://storage.googleapis.com/minikube-builds/2943/minikube-linux-amd64).

Once you have minikube installed run:

```bash
minikube start -p cluster1 --kubernetes-version v1.11.0
minikube start -p cluster2 --kubernetes-version v1.11.0
```

Even though the `minikube` cluster has been started, you'll want to verify all
your `minikube` components are up and ready by examining the state of the
kubernetes components in the clusters via:

```bash
kubectl get all --all-namespaces
```

After all pods reach a Running status, you can return to the [User Guide](../userguide.md) to deploy the cluster
registry and Federation v2 control plane.
