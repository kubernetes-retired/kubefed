# User Guide

## What is the cluster registry?

The cluster registry is a Kubernetes-style API and API server that provides an
endpoint for interacting with a list of clusters and associated metadata. If it
helps, you can think of the cluster registry as a hosted kubeconfig file.
However, since it's a Kubernetes-style API, the cluster registry allows custom
annotations and filtering across labels, can be used with `kubectl` and
Kubernetes-style generated client libraries, and supports having controllers
watch for updates.

## Deploying a cluster registry

The `crinit` tool allows you to deploy a cluster registry into an existing
Kubernetes cluster. Run the following commands to download the latest nightly
build of the tool:

> Currently the tool is only released for Linux 64-bit. You'll need to build it
> yourself if you want to use it on a different platform. See
> [the development docs](development.md#Building-crinit).

```sh
PACKAGE=client
LATEST=$(curl https://storage.googleapis.com/crreleases/nightly/latest)
curl -O http://storage.googleapis.com/crreleases/nightly/$LATEST/clusterregistry-$PACKAGE.tar.gz
tar xzf clusterregistry-$PACKAGE.tar.gz
```

### Standalone

You can deploy the cluster registry as a standalone API server like so:

```sh
./crinit standalone init <cluster_registry_instance_name> --host-cluster-context=<your_cluster_context>
```

where `cluster_registry_instance_name` is the name you want to give this cluster
registry instance and `your_cluster_context` is a context entry in your
kubeconfig file referencing a cluster into which the cluster registry will be
deployed. The `cluster_registry_instance_name` will be used to name resources
that will be created in your cluster for the cluster registry, such as a Service
and a Deployment, and should be named appropriately. When the command completes,
it will create an entry in your kubeconfig file for the cluster registry API
server named with the provided `cluster_registry_instance_name`.
You can interact with it using kubectl:

```sh
$ kubectl get clusters --context <cluster_registry_instance_name>
No resources found
$
```

### Aggregated API server

Before deploying the cluster registry as an aggregated API server, take a look at
https://kubernetes.io/docs/tasks/access-kubernetes-api/configure-aggregation-layer/#enable-apiserver-flags
which talks about what `kube-apiserver` flags need to be enabled in order to
enable the aggregation layer.

Once the `kube-apiserver` aggregation layer is enabled, you can deploy the
cluster registry as an aggregated API server like so:

```sh
./crinit aggregated init <cluster_registry_context> --host-cluster-context=<your_cluster_context>
```

where `your_cluster_context` is a context entry in your kubeconfig file
referencing a cluster into which the cluster registry will be deployed and to
which the cluster registry API server will be added as an aggregated API server.
When the command completes, you will have a cluster registry running as an
aggregated API server with the cluster API server identified by
`your_cluster_context`. You can interact with it using kubectl:

```sh
$ kubectl get clusters --context <your_cluster_context>
No resources found
$
```

### Try it out!

In these examples, `context` is either the `cluster_registry_instance_name` if
you did a standalone deployment, or `your_cluster_context` if you did an
aggregated deployment.

Try creating a cluster:

```sh
kubectl apply -f - --context <context> <<EOF
kind: Cluster
apiVersion: clusterregistry.k8s.io/v1alpha1
metadata:
  name: test-cluster
spec:
  kubernetesApiEndpoints:
    serverEndpoints:
      - clientCidr: "0.0.0.0/0"
        serverAddress: "100.0.0.0"
EOF
```

And then reading it back:

```sh
kubectl get clusters --context <context>
```

### Advanced deployments

`crinit` takes care of a lot of the plumbing work to get a cluster registry
running, but the cluster registry can be run anywhere you like, as long as you
configure it correctly.

#### API server
Since the cluster registry is an API server, all of the guidance around
deploying and managing aggregated API servers applies.
[The Kubernetes docs](https://kubernetes.io/docs/concepts/api-extension/apiserver-aggregation/)
have some pointers about aggregation. You can also look at the
[API Server Concepts docs](https://github.com/kubernetes-incubator/apiserver-builder/tree/master/docs/concepts)
for some more technical guidance about aggregating. If you wish to run the
cluster registry as a standalone server, you will need to manage the
certificates and client authentication/authorization yourself. You can use any
authentication mode supported by the `clusterregistry` command, but you will
probably need to use an
[authorizing webhook](https://kubernetes.io/docs/admin/authorization/webhook/),
since the cluster registry does not provide RBAC support inherently.

#### etcd

`crinit` by default deploys `etcd` in a container in the same pod as the
`clusterregistry` container. This makes it easy to get started, but is not
necessarily a good strategy for deploying a cluster registry for production use.
You may want to look into the
[etcd operator](https://github.com/coreos/etcd-operator) for production
deployments.

## Interacting with the cluster registry

### kubectl

The cluster registry is a Kubernetes-style API server, and you can interact with
it using standard `kubectl` commands. It provides one API type, `clusters`,
which you can create, get, list and delete like any other Kubernetes object. See
[Try it out!](#try-it-out) above for some sample commands.

### Generated Go client

There is a generated Go client library for the cluster registry in
[/pkg/client](/pkg/client). You can vendor in the cluster registry repository
and use the client library directly from your Go code.

### OpenAPI spec

There is an OpenAPI spec file provided [here](/api/swagger.json). You can use
it to generate client libraries in a language of your choice.

## Troubleshooting

### Multi-Attach error

If you are updating the cluster registry deployment by hand, you may run into
errors like the following:

```
Multi-Attach error for volume "vol" Volume is already exclusively attached to one node and can't be attached to another
```

This is because the new `clusterregistry` `Pod` will not be able to claim the
persistent volume if it is started on a different node than the existing `Pod`.
A simple fix is to change the `deployement.spec.strategy.type` to `Recreate`.
There are more details in https://github.com/kubernetes/kubernetes/issues/48968.
