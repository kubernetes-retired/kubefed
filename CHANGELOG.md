# Unreleased
-  [#722](https://github.com/kubernetes-sigs/federation-v2/issues/722) -
   Removal of the FederatedTypeConfig for namespaces now disables all
   namespaced sync controllers. Additionally, the namespace FederatedTypeConfig
   must always exist prior to starting any namespaced sync controller.
 - [#612](https://github.com/kubernetes-sigs/federation-v2/pull/612) -
   Label managed resources in member clusters and only watch resources
   so labeled to minimize the memory usage of the federated control
   plane.
 - [#721](https://github.com/kubernetes-sigs/federation-v2/issues/721) -
   kubefed2 disable now deletes the FederatedTypeConfig rather than set
   propagationEnabled, waits for the sync controller to shut down, and
   optionally removes the federated type CRD.
 - [#825](https://github.com/kubernetes-sigs/federation-v2/pull/825) -
   kubefed2 tool is renamed to kubefedctl.

# v0.0.9
-  [#776](https://github.com/kubernetes-sigs/federation-v2/pull/776) -
   Switch to use `scope` instead of `limitedScope` to specify if it is
   `Namespaced` or `Cluster` scoped federation deployment.
-  [#797](https://github.com/kubernetes-sigs/federation-v2/pull/797) -
   Cross-cluster service discovery now works for multi-zone clusters.
   There is an update to FederatedClusters and ServiceDNSRecord API
   types wherein the zone field is changed to zones.
-  [#720](https://github.com/kubernetes-sigs/federation-v2/issues/720) -
   `kubefed2 enable` now succeeds if federation of the type is already
   enabled.
 - [#738](https://github.com/kubernetes-sigs/federation-v2/issues/738) -
   Cleanup `kubefed2 enable` required arguments and remove unnecessary
   `--registry-namespace` option from `kubefed2 <enable|disable>`.
 - [#737](https://github.com/kubernetes-sigs/federation-v2/pull/737) -
   Switch to use FederationConfig resource rather than command line
   options for federation controller configuration management
 - [#549](https://github.com/kubernetes-sigs/federation-v2/pull/549) -
   As a result of watching only labled resources, unlabled resources
   in unselected clusters will no longer be deleted.

# v0.0.8
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
