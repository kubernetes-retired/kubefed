

-----------
# Cluster v1alpha1



Group        | Version     | Kind
------------ | ---------- | -----------
`clusterregistry` | `v1alpha1` | `Cluster`







Cluster contains information about a cluster in a cluster registry.



Field        | Description
------------ | -----------
`apiVersion`<br /> *string*    | APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources
`kind`<br /> *string*    | Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
`metadata`<br /> *[ObjectMeta](#objectmeta-v1)*    | Standard object&#39;s metadata. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
`spec`<br /> *[ClusterSpec](#clusterspec-v1alpha1)*    | Spec is the specification of the cluster. This may or may not be reconciled by an active controller.
`status`<br /> *[ClusterStatus](#clusterstatus-v1alpha1)*    | Status is the status of the cluster.


### ClusterSpec v1alpha1

<aside class="notice">
Appears In:

<ul>
<li><a href="#cluster-v1alpha1">Cluster v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`authInfo`<br /> *[AuthInfo](#authinfo-v1alpha1)*    | AuthInfo contains public information that can be used to authenticate to and authorize with this cluster. It is not meant to store private information (e.g., tokens or client certificates) and cluster registry implementations are not expected to provide hardened storage for secrets.
`kubernetesApiEndpoints`<br /> *[KubernetesAPIEndpoints](#kubernetesapiendpoints-v1alpha1)*    | KubernetesAPIEndpoints represents the endpoints of the API server for this cluster.

### ClusterStatus v1alpha1

<aside class="notice">
Appears In:

<ul>
<li><a href="#cluster-v1alpha1">Cluster v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`conditions`<br /> *[ClusterCondition](#clustercondition-v1alpha1) array*    | Conditions contains the different condition statuses for this cluster.





