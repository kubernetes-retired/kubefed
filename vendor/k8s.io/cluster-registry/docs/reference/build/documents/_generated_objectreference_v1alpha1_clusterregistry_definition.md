## ObjectReference v1alpha1

Group        | Version     | Kind
------------ | ---------- | -----------
`clusterregistry` | `v1alpha1` | `ObjectReference`



ObjectReference contains enough information to let you inspect or modify the referred object.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#authinfo-v1alpha1">AuthInfo v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`kind`<br /> *string*    | Kind contains the kind of the referent, e.g., Secret or ConfigMap More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
`name`<br /> *string*    | Name contains the name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
`namespace`<br /> *string*    | Namespace contains the namespace of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/

