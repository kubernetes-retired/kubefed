## ClusterCondition v1alpha1

Group        | Version     | Kind
------------ | ---------- | -----------
`clusterregistry` | `v1alpha1` | `ClusterCondition`



ClusterCondition contains condition information for a cluster.

<aside class="notice">
Appears In:

<ul> 
<li><a href="#clusterstatus-v1alpha1">ClusterStatus v1alpha1</a></li>
</ul></aside>

Field        | Description
------------ | -----------
`lastHeartbeatTime`<br /> *[Time](#time-v1)*    | LastHeartbeatTime is the last time this condition was updated.
`lastTransitionTime`<br /> *[Time](#time-v1)*    | LastTransitionTime is the last time the condition changed from one status to another.
`message`<br /> *string*    | Message is a human-readable message indicating details about the last status change.
`reason`<br /> *string*    | Reason is a (brief) reason for the condition&#39;s last status change.
`status`<br /> *string*    | Status is the status of the condition. One of True, False, Unknown.
`type`<br /> *string*    | Type is the type of the cluster condition.

