# Unreleased
 - [#622](https://github.com/kubernetes-sigs/federation-v2/pull/622) -
   Switched the sync controller to using a new finalizer
   (`federation.k8s.io/sync-controller` instead of
   `federation.kubernetes.io/delete-from-underlying-clusters`) and
   replaced the use of the kube `orphan` finalizer in favor of an
   annotation to avoid conflicting with the garbage collector.  This
   change in finalizer usage represents a breaking change since
   resources reconciled by previous versions of the sync controller
   will have the old finalizer.  The old finalizer would need to be
   manually removed from a resource for that resource to be garbage
   collected after deletion.
- [#698](https://github.com/kubernetes-sigs/federation-v2/pull/698) -
   Fix the generated CRD schema of scalable resources to define the
   `retainReplicas` of type `boolean` rather than the invalid type
   `bool`.
