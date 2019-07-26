# Unreleased
-  [#1052](https://github.com/kubernetes-sigs/kubefed/pull/1052)
   Support has been added for varying the `apiVersion` of target
   resources. This is intended to allow a federated type to manage
   more than one version of the target type across member clusters.
   `apiVersion` can be set either in the template of a federated
   resource or via override.
-  [#951](https://github.com/kubernetes-sigs/kubefed/issues/951)
   Propagation status for a namespaced federated resource whose
   containing namespace is not federated now indicates an unhealthy
   state.
-  [#1053](https://github.com/kubernetes-sigs/kubefed/pull/1053) API group
   changed from kubefed.k8s.io to kubefed.io.

# v0.1.0-rc4
-  [#908](https://github.com/kubernetes-sigs/kubefed/issues/908) Adds admission
   webhook validations for KubeFedCluster API.
-  [#982](https://github.com/kubernetes-sigs/kubefed/issues/982) To
   ensure compatibility with controllers in member clusters,
   metadata.finalizers and metadata.annotations can no longer be set
   from the template of a federated resource and values for these
   fields are always retained. The addition of jsonpatch overrides
   ensures that it is still possible to add or remove entries from
   these collections.
-  [#1013](https://github.com/kubernetes-sigs/kubefed/issues/1013) Add support
   for defaulting KubeFedConfigs using mutating admission webhook.
-  [#1038](https://github.com/kubernetes-sigs/kubefed/pull/1038) Removed template validation schema from Federated API's to facilitate upgrade scenarios.
-  [#690](https://github.com/kubernetes-sigs/kubefed/issues/690) Extends `kubefedctl`
   by adding the `orphaning-deletion` command, which allows to `enable` or `disable`
   leaving managed resources, when their relevant federated resource is deleted.
   In addition, the command allows to check current `status` of the orphaning deletion
   mode.
-  [#1044](https://github.com/kubernetes-sigs/kubefed/issues/1044) If a target namespace
   of `federate` and all `orphaning-deletion` commands is not specified, use the namespace from
   the client kubeconfig context.

# v0.1.0-rc3
-  [#520](https://github.com/kubernetes-sigs/kubefed/issues/520) Adds support
   for jsonpath overrides.
-  [#965](https://github.com/kubernetes-sigs/kubefed/issues/965) Adds admission
   webhook support for namespace-scoped deployments.
-  [#941](https://github.com/kubernetes-sigs/kubefed/issues/941) Adds
   admission webhook validations for KubeFedConfig API.
-  [#909](https://github.com/kubernetes-sigs/kubefed/issues/909) Adds
   admission webhook validations for FederatedTypeConfig API.

# v0.1.0-rc1
-  [#887](https://github.com/kubernetes-sigs/kubefed/pull/887) Updates
   KubefedConfig API based on Kubernetes API conventions.
-  [#885](https://github.com/kubernetes-sigs/kubefed/pull/885) Updates
   FederatedTypeConfig API based on Kubernetes API conventions.
-  [#886](https://sigs.k8s.io/kubefed/issues/886)
   The ca bundle for a member cluster is now stored as an optional
   field of KubefedCluster since a ca bundle may not be required for
   all clusters and is not sensitive information that requires storage
   in a secret.
-  [#865](https://sigs.k8s.io/kubefed/pull/865)
   github repo is renamed to kubefed. Following changes are done:
   - API group changed from federation.k8s.io to kubefed.k8s.io
   - FederatedCluster API is renamed to KubefedCluster
   - FederationConfig API is renamed to KubefedConfig
   - golang imports changed to use vanity url sigs.k8s.io/kubefed
   - docker image is renamed to quay.io/kubernetes-multicluster/kubefed
   - helm chart is renamed to kubefed
   - All role, rolebindings, service-account begining with federation- prefix renamed to kubefed-
-  [#875](https://sigs.k8s.io/kubefed/issues/875)
   Insecure member clusters are no longer supported due to
   KubefedCluster.SecretRef being made a required field.
-  [#869](https://sigs.k8s.io/kubefed/issues/869)
   The api endpoint of a member cluster is now stored in
   KubefedCluster instead of a cluster registry Cluster.
-  [#688](https://sigs.k8s.io/kubefed/issues/688)
   Cluster references in placement are now objects instead of strings
   to ensure extensibility.
-  [#832](https://sigs.k8s.io/kubefed/issues/832)
   `kubefedctl federate` can take input from a file via `--filename`
   option and stdin via `--filename -` option.
-  [#868](https://sigs.k8s.io/kubefed/issues/868)
   The default kubefed system namespace has been changed from
   `federation-system` to `kube-federation-system`.  The `kube-`
   prefix is reserved for system namespaces and including it avoids
   having the kubefed namespace conflict with a user namespace.
-  [#740](https://sigs.k8s.io/kubefed/issues/740)
   Propagation status is now recorded for all federated resources.
-  [#844](https://sigs.k8s.io/kubefed/pull/844)
   `kubefedctl federate` namespace with its content gets an option to
   skip API Resources via `--skip-api-resources`.

# v0.0.10
-  [#722](https://sigs.k8s.io/kubefed/issues/722) -
   Removal of the FederatedTypeConfig for namespaces now disables all
   namespaced sync controllers. Additionally, the namespace FederatedTypeConfig
   must always exist prior to starting any namespaced sync controller.
 - [#612](https://sigs.k8s.io/kubefed/pull/612) -
   Label managed resources in member clusters and only watch resources
   so labeled to minimize the memory usage of the federated control
   plane.
 - [#721](https://sigs.k8s.io/kubefed/issues/721) -
   kubefed2 disable now deletes the FederatedTypeConfig rather than set
   propagationEnabled, waits for the sync controller to shut down, and
   optionally removes the federated type CRD.
 - [#825](https://sigs.k8s.io/kubefed/pull/825) -
   kubefed2 tool is renamed to kubefedctl.
 - [#741](https://sigs.k8s.io/kubefed/pull/741) -
   Added conversion of a namespace and its contents to federated
   equivalents via `kubefedctl federate ns <namespace> --contents`.

# v0.0.9
-  [#776](https://sigs.k8s.io/kubefed/pull/776) -
   Switch to use `scope` instead of `limitedScope` to specify if it is
   `Namespaced` or `Cluster` scoped federation deployment.
-  [#797](https://sigs.k8s.io/kubefed/pull/797) -
   Cross-cluster service discovery now works for multi-zone clusters.
   There is an update to KubefedClusters and ServiceDNSRecord API
   types wherein the zone field is changed to zones.
-  [#720](https://sigs.k8s.io/kubefed/issues/720) -
   `kubefed2 enable` now succeeds if federation of the type is already
   enabled.
 - [#738](https://sigs.k8s.io/kubefed/issues/738) -
   Cleanup `kubefed2 enable` required arguments and remove unnecessary
   `--registry-namespace` option from `kubefed2 <enable|disable>`.
 - [#737](https://sigs.k8s.io/kubefed/pull/737) -
   Switch to use KubefedConfig resource rather than command line
   options for kubefed controller configuration management
 - [#549](https://sigs.k8s.io/kubefed/pull/549) -
   As a result of watching only labled resources, unlabled resources
   in unselected clusters will no longer be deleted.
 - [#663](https://sigs.k8s.io/kubefed/pull/663)
   `kubefed2 federate` now supports the `--enable-type` flag to optionally
   enable the given `type` for propagation.


# v0.0.8
 - [#652](https://sigs.k8s.io/kubefed/pull/652) -
   Switch to sourcing the template for a FederatedNamespace from a
   field rather than the containing namespace.  This ensures
   uniformity in template handling across all federated types.
 - [#716](https://sigs.k8s.io/kubefed/pull/716) -
   Upgrade kubebuilder version to v1.0.8
   - Removed generated typed clients for federation apis from tree.
     Use generic client to operate on federation apis as shown
     [here](https://github.com/kubernetes-sigs/controller-runtime/blob/master/pkg/client/example_test.go)
   - Helm based deployment method will be the only supported
     deployment method to install kubefed control plane.
 - [#622](https://sigs.k8s.io/kubefed/pull/622) -
   Switched the sync controller to using a new finalizer
   (`kubefed.k8s.io/sync-controller` instead of
   `federation.kubernetes.io/delete-from-underlying-clusters`) and
   replaced the use of the kube `orphan` finalizer in favor of an
   annotation to avoid conflicting with the garbage collector.  This
   change in finalizer usage represents a breaking change since
   resources reconciled by previous versions of the sync controller
   will have the old finalizer.  The old finalizer would need to be
   manually removed from a resource for that resource to be garbage
   collected after deletion.
- [#698](https://sigs.k8s.io/kubefed/pull/698) -
   Fix the generated CRD schema of scalable resources to define the
   `retainReplicas` of type `boolean` rather than the invalid type
   `bool`.
- [#662](https://sigs.k8s.io/kubefed/pull/662)
   Federating a resource could now function without a federation API.
- [#661](https://sigs.k8s.io/kubefed/pull/661)
   Accept group qualified type name for type in federate resource.
- [#660](https://sigs.k8s.io/kubefed/pull/660)
   `kubefed2 federate` has been updated to support output to yaml via
   `-o yaml`. YAML output would still require a Kubernetes API endpoint
    to function.
