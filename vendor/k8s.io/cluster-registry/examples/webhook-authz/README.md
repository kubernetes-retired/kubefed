# Example Webhook authorizer

This directory contains the scaffolding necessary to use a webhook authorizer
with the standalone cluster registry. It is not meant to be used with the
aggregated cluster registry.

NOTE: this example is not meant for production use! Do not deploy this into a
production environment.

## Usage

1.  `cd` into this directory.

2.  Deploy a standalone cluster registry. You can use
    [crinit](https://github.com/kubernetes/cluster-registry/blob/master/docs/userguide.md#standalone).

3.  Create a self-signed SSL cert for the cluster registry webhook authorizer
    (or skip this step and use an existing SSL cert).

```sh
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -nodes
```

4.  Create a Secret that contains the files needed for the cluster registry
    webhook authorizer (i.e., the `rules.hcl` file, the configuration file that
    configures the cluster registry API server to use the webhook authorizer,
    and the SSL key and certificate). If you deployed your cluster registry into
    a namespace other than `clusterregistry`, change the namespace below
    accordingly.

```sh
kubectl create secret generic bitesize -n clusterregistry \
  --from-file=./rules.hcl --from-file=./webhook-config.yaml \
  --from-file=./key.pem --from-file=./cert.pem
```

5.  Create the webhook authorizer. Note that you may need to change the
    namespace if you did not deploy the cluster registry into the
    `clusterregistry` namespace.

```sh
kubectl apply -f ./authz.yaml -n clusterregistry
```

6.  Update the cluster registry deployment as follows.

```yaml
...
      - command:
        - ./clusterregistry
        ... existing flags ...
        - --authorization-always-allow=false
        - --authorization-webhook=true
        - --authorization-webhook-config-file=/etc/bitesize/webhook-config.yaml
        - --authorization-webhook-cache-authorized-ttl=5m
        - --authorization-webhook-cache-unauthorized-ttl=30s
        volumeMounts:
        ... existing volumeMounts ...
        - name: bitesize
          mountPath: /etc/bitesize
...
      volumes:
      ... existing volumes ...
      - name: bitesize
        secret:
          secretName: bitesize
...
```

> NOTE: You may run into an issue recreating the Pod because the persistent disk
> is already being used. Refer [here](/docs/userguide.md#multi-attach-error) for
> more info.

If everything is running properly, and you have admin credentials for this
cluster in your kubeconfig file, you should be able to access clusters as the
admin user:

```sh
$ kubectl get clusters --context <your_cluster_registry_context> --user <your_admin_user>
No resources found
```

and list them as `testuser`:

```sh
$ kubectl get clusters --context <your_cluster_registry_context> --as testuser
No resources found
```

but not access them as someone else:

```sh
$ kubectl get clusters --context <your_cluster_registry_context> --as bob
Error from server (Forbidden): clusters.clusterregistry.k8s.io is forbidden: User "bob" cannot list clusters.clusterregistry.k8s.io at the cluster scope: Not allowed
```

or create/update them as `testuser`:

```sh
$ kubectl apply -f ../samplecontainer/cluster.yaml --context <your_cluster_registry_context> --as testuser
Error from server (Forbidden): error when creating "../samplecontainer/cluster.yaml": clusters.clusterregistry.k8s.io is forbidden: User "testuser" cannot create clusters.clusterregistry.k8s.io at the cluster scope: Not allowed
```
