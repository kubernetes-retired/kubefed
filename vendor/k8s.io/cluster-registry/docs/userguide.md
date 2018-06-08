# User Guide

## What is the cluster registry?

The cluster registry is a Kubernetes-style API that provides an endpoint for
interacting with a list of clusters and associated metadata. If it helps, you
can think of the cluster registry as a hosted kubeconfig file. However, since
it's a Kubernetes-style API, the cluster registry allows custom annotations and
filtering across labels, can be used with `kubectl` and Kubernetes-style
generated client libraries, and supports having controllers watch for updates.

## Setting up a cluster registry

The cluster registry API is defined as a [Kubernetes custom resource
definition](https://kubernetes.io/docs/concepts/api-extension/custom-resources/#customresourcedefinitions).
The [YAML for the CRD](/cluster-registry-crd.yaml) is stored in the cluster
registry repo. In order to set up the cluster registry, you must have an
existing Kubernetes API server running that supports the `apiextensions.k8s.io`
API group.

You can set up the cluster registry like so:

```sh
cd $CR_REPO_ROOT
kubectl apply -f cluster-registry-crd.yaml
```

This will register the cluster registry API with your currently-selected
context. To interact with it, use `kubectl`:

```sh
$ kubectl get clusters
No resources found
$
```

### Try it out!

Try creating a cluster:

```sh
kubectl apply -f - <<EOF
kind: Cluster
apiVersion: clusterregistry.k8s.io/v1alpha1
metadata:
  name: test-cluster
  namespace: default
spec:
  kubernetesApiEndpoints:
    serverEndpoints:
      - clientCIDR: "0.0.0.0/0"
        serverAddress: "100.0.0.0"
EOF
```

And then reading it back:

```sh
kubectl get clusters
```

## Interacting with the cluster registry

### kubectl

The cluster registry is a Kubernetes-style API, and you can interact with it
using standard `kubectl` commands. It provides one API type, `clusters`, which
you can create, get, list and delete like any other Kubernetes object. See [Try
it out!](#try-it-out) above for some sample commands.

### Generated Go client

There is a generated Go client library for the cluster registry in
[/pkg/client](/pkg/client). You can vendor in the cluster registry repository
and use the client library directly from your Go code.

### OpenAPI spec

There is an OpenAPI spec file provided
[here](/docs/reference/openapi-spec/swagger.json). You can use it to generate
client libraries in a language of your choice.
