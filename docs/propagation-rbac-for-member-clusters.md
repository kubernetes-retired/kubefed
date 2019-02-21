This guide shows a way to make a user propagate federated resources to a set of member clusters in kubernetes federation.

`FederatedNamespace` controls which cluster a federated namespace can be propogated to. It also determines which clusters a federated resource of the namespace can be propagated to. `RBAC` can be created to make a user only operate resources in the specified namespace. Here is an example to leverage both `FederatedNamespace` and `RBAC` to limit user propagation rights to a set of member clusters.

This example shows that `user1` only progagates federated resources to `cluster1` and `user2` can propagate federated resources to both `cluster2` and `cluster3`.

# Prerequisite
Federation control plane should be deployed with 3 member clusters: `cluster1`, `cluster2` and `cluster3`. Refer to [User Guide](https://github.com/kubernetes-sigs/federation-v2/blob/master/docs/userguide.md) for details in federation deployment.

# Creating FederatedNamespace
Administrator creates `FederatedNamespace` resources and specifies proper placement for different federated namespaces.
`namespace1` is propagated to member `cluster1`. `namespace2` is propagated to member `cluster2` and `cluster3`.
```bash
$ cat <<EOF | kubectl create -f -
---
apiVersion: v1                                  
kind: Namespace
metadata:
  name: namespace1
---
apiVersion: types.federation.k8s.io/v1alpha1                
kind: FederatedNamespace
metadata:
  name: namespace1
  namespace: namespace1
spec:
  placement:
    clusterNames:
    - cluster1
---
apiVersion: v1                                  
kind: Namespace
metadata:
  name: namespace2
---
apiVersion: types.federation.k8s.io/v1alpha1                
kind: FederatedNamespace
metadata:
  name: namespace2
  namespace: namespace2
spec:
  placement:
    clusterNames:
    - cluster2
    - cluster3
EOF
```

# Configuring RBAC
Administrator configures RBAC to limit a user to access the specified namespace. `user1` can operate federated resources in `namespace1` and `user2` can operate federated resources in `namespace2`.
```bash
$ cat <<EOF | kubectl create -f -
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: user1
  namespace: namespace1
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: namespaced-access
  namespace: namespace1
rules:
- apiGroups: ["types.federation.k8s.io"]
  resources: ["*"]
  verbs: ["*"]
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: user1
  namespace: namespace1
subjects:
- kind: ServiceAccount
  name: user1
  namespace: namespace1
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: namespaced-access
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: user2
  namespace: namespace2
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: namespaced-access
  namespace: namespace2
rules:
- apiGroups: ["types.federation.k8s.io"]
  resources: ["*"]
  verbs: ["*"]
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: user2
  namespace: namespace2
subjects:
- kind: ServiceAccount
  name: user2
  namespace: namespace2
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: namespaced-access
EOF
```
In this case, a user can only access the specified namespace. So, he can propagate his resources to the related member clusters with the namespace isolation.

# Deploying Federated Resource
End user can use the secret information to access the namespace. Create `Config` for `user2` as an example.
```bash
$ cat <<EOF > user2-config.yaml
apiVersion: v1
clusters:
- cluster:
    certificate-authority: /usr/local/work/minikube/.minikube/ca.crt
    server: https://192.168.99.100:8443
  name: cluster1
contexts:
- context:
    cluster: cluster1
    namespace: namespace2
    user: user2
  name: user2
current-context: user2
kind: Config
preferences: {}
users:
EOF
$ kubectl get sa -n namespace2 user2 -o "jsonpath={.secrets[0].name}";echo
user2-token-vh2sb
$ kubectl config set-credentials user2 --token `kubectl get secret  -n namespace2 user2-token-vh2sb -o "jsonpath={.data.token}" | base64 -d` --kubeconfig user2-config.yaml
```
`user2` creates `FederatedDeployment` in the host cluster to propagate resource to member clusters. With help of federation controller, federated resources can only be propagated to specified member clusters for the user.
```bash
$ cat <<EOF | kubectl --kubeconfig user2-config.yaml create -f -
---
apiVersion: types.federation.k8s.io/v1alpha1
kind: FederatedDeployment
metadata:
  name: test-deployment
  namespace: namespace2
spec:
  template:
    metadata:
      labels:
        app: nginx
    spec:
      replicas: 3
      selector:
        matchLabels:
          app: nginx
      template:
        metadata:
          labels:
            app: nginx
        spec:
          containers:
          - image: nginx
            name: nginx

  placement:
    clusterNames:
    - cluster1
    - cluster2
    - cluster3
EOF
$ kubectl get deployment -n namespace2 --context cluster1
No resources found.
$ kubectl get deployment -n namespace2 --context cluster2
NAME              DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
test-deployment   3         3         3            3           5m
$ kubectl get deployment -n namespace2 --context cluster3
NAME              DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
test-deployment   3         3         3            3           5m

$ kubectl get federateddeployment -n namespace2 --kubeconfig user2-config.yaml
NAME              AGE
test-deployment   1h
$ kubectl get federateddeployment -n namespace1 --kubeconfig user2-config.yaml
No resources found.
Error from server (Forbidden): federateddeployments.types.federation.k8s.io is forbidden: User "system:serviceaccount:namespace2:user2" cannot list federateddeployments.types.federation.k8s.io in the namespace "namespace1"
```
`Deployment` cannot be propagated to `cluster1` even it is specified in placement of `FederatedDeployment`.
