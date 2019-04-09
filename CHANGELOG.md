# Unreleased
 - [#652](https://github.com/kubernetes-sigs/federation-v2/pull/652) -
   Switch to sourcing the template for a FederatedNamespace from a
   field rather than the containing namespace.  This ensures
   uniformity in template handling across all federated types.
 - [#716](https://github.com/kubernetes-sigs/federation-v2/pull/716) -
   Upgrade kubebuilder version to v1.0.8
   - Removed generated typed clients for federation apis from tree.
     Use generic client to operate on federation apis as shown
     [here](https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/example_test.go)
   - Helm based deployment method will be the only supported
     deployment method to install federation control plane.
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
