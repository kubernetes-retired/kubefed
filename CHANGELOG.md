# Unreleased

# v0.8.1
- [#1435](https://github.com/kubernetes-sigs/kubefed/pull/1435)
  fix: Support Kubernetes pre-release versions in kubefed chart
- [#1432](https://github.com/kubernetes-sigs/kubefed/pull/1432)
  build(deps): Upgrade and align k8s.io dependency versions
- [#1424](https://github.com/kubernetes-sigs/kubefed/pull/1424)
  fix(postinstall-job): Allow postinstall job to choose a docker repo/image
- [#1428](https://github.com/kubernetes-sigs/kubefed/pull/1428)
  fix: retry on recoverable propagation failure
- [#1416](https://github.com/kubernetes-sigs/kubefed/pull/1416)
  feat: Make restclient config configurable
# v0.8.0
- [#1332](https://github.com/kubernetes-sigs/kubefed/pull/1332)
  fix: Error on existing secrets
- [#1379](https://github.com/kubernetes-sigs/kubefed/pull/1379)
  feat: make objects in workqueue "comparable" to avoid multiple reconciliations on one object at same time
- [#1388](https://github.com/kubernetes-sigs/kubefed/pull/1388)
  fix: added blank import to ensure correct workqueue metric registration
- [#1389](https://github.com/kubernetes-sigs/kubefed/pull/1389)
  fix: use merge-patch on finalizer operations to resolve racing conflicts
- [#1393](https://github.com/kubernetes-sigs/kubefed/pull/1393)
  feat: add intersection behavior for RSP
- [#1399](https://github.com/kubernetes-sigs/kubefed/pull/1399)
  docs: correct command line example for test
- [#1400](https://github.com/kubernetes-sigs/kubefed/pull/1400)
  feat: make concurrency of the worker configurable
- [#1409](https://github.com/kubernetes-sigs/kubefed/pull/1409)
  feat: Use patch to replace update in generic client
- [#1410](https://github.com/kubernetes-sigs/kubefed/pull/1410)
  fix: fixed false api resource error log in kubefedctl
- [#1415](https://github.com/kubernetes-sigs/kubefed/pull/1415)
  feat: Update controller-runtime to v0.8.3
- [#1420](https://github.com/kubernetes-sigs/kubefed/pull/1420)
  build: Switch to Github Actions from Travis
- [#1421](https://github.com/kubernetes-sigs/kubefed/pull/1421)
  docs: update README.md to reflect current projectstate
- [#1422](https://github.com/kubernetes-sigs/kubefed/pull/1422)
  build: Remove need for TTY when running in Github Actions
- [#1425](https://github.com/kubernetes-sigs/kubefed/pull/1425)
  feat: Upgrade to controller-runtime 0.9.0
- [#1426](https://github.com/kubernetes-sigs/kubefed/pull/1426)
  chore: Upgrade golangci-lint and helm
- [#1427](https://github.com/kubernetes-sigs/kubefed/pull/1427)
  chore: Kind and Kubernetes upgrade

# v0.7.0
- [#1380](https://github.com/kubernetes-sigs/kubefed/pull/1380) 
  fix: infinite reconciliation loop.
- [#1385](https://github.com/kubernetes-sigs/kubefed/pull/1385) 
  fix: replica rebalance reconciliation.
- [#1377](https://github.com/kubernetes-sigs/kubefed/pull/1377) 
  feat: add a proxy url field to kubefed clusters.
- [#1382](https://github.com/kubernetes-sigs/kubefed/pull/1382) 
  fix: register workqueue metrics in controller-runtime firstly.
- [#1371](https://github.com/kubernetes-sigs/kubefed/pull/1371) 
  chore: enable workqueue metrics for all controllers.
- [#1360](https://github.com/kubernetes-sigs/kubefed/pull/1360) 
  chore: remove FederatedIngress feature.
- [#1367](https://github.com/kubernetes-sigs/kubefed/pull/1367) 
  fix: webhook command.
- [#1361](https://github.com/kubernetes-sigs/kubefed/pull/1361) 
  feature: kubefedcluster use cadata in kubeconfig file.
- [#1355](https://github.com/kubernetes-sigs/kubefed/pull/1355) 
  feature: support DeleteOptions when deleting resources in member clusters.
- [#1357](https://github.com/kubernetes-sigs/kubefed/pull/1357) 
  chore: Upgrade dependencies.
- [#1351](https://github.com/kubernetes-sigs/kubefed/pull/1351) 
  chore: remove feature CrossClusterServiceDiscovery.

# v0.6.1
- [#1346](https://github.com/kubernetes-sigs/kubefed/pull/1346)
  fix: upgrade path broken from older versions than v0.6.0.
- [#1347](https://github.com/kubernetes-sigs/kubefed/pull/1347)
  chore: retain healthCheckNodePort for service when updating.
- [#1334](https://github.com/kubernetes-sigs/kubefed/pull/1334)
  chore: exec enable cmd ignore some apiservices errors.
# v0.6.0
- [#1328](https://github.com/kubernetes-sigs/kubefed/pull/1328)
  docs: optimize chart readme. 
- [#1292](https://github.com/kubernetes-sigs/kubefed/pull/1292)
  feat: collect remote resource status when enabled. 
- [#1325](https://github.com/kubernetes-sigs/kubefed/pull/1325)
  chore: add helm parameter imagePullSecrets. 
- [#1324](https://github.com/kubernetes-sigs/kubefed/pull/1324)
  chore: improve some of the deployment and build scripts. 
- [#1323](https://github.com/kubernetes-sigs/kubefed/pull/1323)
  make create-clusters.sh work based on kind document. 
- [#1297](https://github.com/kubernetes-sigs/kubefed/pull/1297)
  feat: Transition from apiextensions.k8s.io/v1beta1 to apiextensions.k8s.io/v1.
# v0.5.1
- [#1318](https://github.com/kubernetes-sigs/kubefed/pull/1318)
  chore: make certain cert-manager properties configurable.
- [#1315](https://github.com/kubernetes-sigs/kubefed/pull/1315)
  fix: klog verbosity detection.
# v0.5.0
- [#1310](https://github.com/kubernetes-sigs/kubefed/pull/1310)
  chore: Add labels to service.
- [#1308](https://github.com/kubernetes-sigs/kubefed/pull/1308) 
  chore: upgrade tools.
- [#1306](https://github.com/kubernetes-sigs/kubefed/pull/1306) 
  chore: Remove need for insecure registry.  
- [#1305](https://github.com/kubernetes-sigs/kubefed/pull/1305) 
  chore: disabled FederatedIngress by default.
- [#1302](https://github.com/kubernetes-sigs/kubefed/pull/1302)
  chore: set the tolerations and nodeSelector  
- [#1301](https://github.com/kubernetes-sigs/kubefed/pull/1301) 
  chore: upgrade kubernetes dependencies.
- [#1300](https://github.com/kubernetes-sigs/kubefed/pull/1300)
  doc: fixing some typos in user guide.
- [#1298](https://github.com/kubernetes-sigs/kubefed/pull/1298)
  chore: disable CrossClusterDiscovery feature by default
- [#1294](https://github.com/kubernetes-sigs/kubefed/pull/1294)
  fix: prefer core resources when enabling a type.
# v0.4.1
- [#1289](https://github.com/kubernetes-sigs/kubefed/pull/1289) 
  chore: use cert-mananager.io/v1 group/version
- [#1286](https://github.com/kubernetes-sigs/kubefed/pull/1286) 
  docs: clean up redundant text 
- [#1282](https://github.com/kubernetes-sigs/kubefed/pull/1282) 
  chore: webhook functions now return AdmissionResponse
- [#1280]((https://github.com/kubernetes-sigs/kubefed/pull/1280)
  docs: Add quickstart guide
- [#1279]((https://github.com/kubernetes-sigs/kubefed/pull/1279)
  Add a shortname to replicaschedulingpreference
- [#1274]((https://github.com/kubernetes-sigs/kubefed/pull/1274)
  Update linters and gofmt scripts 
# v0.4.0
- [#1260](https://github.com/kubernetes-sigs/kubefed/pull/1260)
  Helm 3 chart migration.
- [#1269](https://github.com/kubernetes-sigs/kubefed/pull/1270)
  Go 1.14.7 and dependency updates.
- [#1263](https://github.com/kubernetes-sigs/kubefed/pull/1263)
  Migrate to controller-runtime webhook server from unmaintained /openshift generic-admission-server.
- [#1261](https://github.com/kubernetes-sigs/kubefed/pull/1261)
  Split helm chart values for webhook and controller-manager deployments.
# v0.3.1
-  [#1251](https://github.com/kubernetes-sigs/kubefed/pull/1251)
   Update dependency to kubernetes 1.18.6.
-  [#1248](https://github.com/kubernetes-sigs/kubefed/pull/1248)
   Fix service status not working issue.
-  [#1245](https://github.com/kubernetes-sigs/kubefed/pull/1245)
   Set default log level for kubefed controller manager.
-  [#1244](https://github.com/kubernetes-sigs/kubefed/pull/1244)
   Kubernetes 1.18.5 dependency 1.18.5 and other upgrades.
-  [#1239](https://github.com/kubernetes-sigs/kubefed/pull/1239)
   Correct docs for a Cluster Selector case.
-  [#1236](https://github.com/kubernetes-sigs/kubefed/pull/1236)
   Increase resource limits for kubefed controller.
-  [#1232](https://github.com/kubernetes-sigs/kubefed/pull/1232)
   Docs: adding documentation to add an array element using overrides.
-  [#1231](https://github.com/kubernetes-sigs/kubefed/pull/1231)
   Cleanup legacy context after delete cluster.
-  [#1229](https://github.com/kubernetes-sigs/kubefed/pull/1229)
   Correct steps in ingress and service with coredns document.
-  [#1228](https://github.com/kubernetes-sigs/kubefed/pull/1228)
   Upgrage ingress to 0.32.
-  [#1221](https://github.com/kubernetes-sigs/kubefed/pull/1221)
   Chore: make cluster creation work with kind v0.7.0.
# v0.3.0
-  [#1218](https://github.com/kubernetes-sigs/kubefed/pull/1218)
   chore: Cleanup travis config about dep
-  [#1216](https://github.com/kubernetes-sigs/kubefed/pull/1216)
   fix: We shouldn't expect invidual error msg printed from std packages
-  [#1212](https://github.com/kubernetes-sigs/kubefed/pull/1212)
   test: add test to validate the controller actions to keep the cluster data 
-  [#1209](https://github.com/kubernetes-sigs/kubefed/pull/1209)
   typo: changing 'federeate' for 'federate
-  [#1207](https://github.com/kubernetes-sigs/kubefed/pull/1207)
   fix: enable command for crd resources
-  [#1200](https://github.com/kubernetes-sigs/kubefed/pull/1200)
   Bump Helm version to 2.16.3
-  [#1196](https://github.com/kubernetes-sigs/kubefed/pull/1196)
   feat: add custom kubefed metrics
-  [#1181](https://github.com/kubernetes-sigs/kubefed/issues/1181)
   fix: namespaced condition for api resource  
# v0.2.0-alpha.1
-  [#1129](https://github.com/kubernetes-sigs/kubefed/pull/1129)
   An empty `spec.placement.clusters` field will now always result in
   no clusters being selected. Previously an empty `clusters` field
   could result in `spec.placement.clusterSelector` being used.
-  [#1107](https://github.com/kubernetes-sigs/kubefed/pull/1107)
  `status.observedGeneration` is now recorded by the sync controller
   for federated resources to provide an indication of whether the
   current state of a resource has been processed.
-  [#1121](https://github.com/kubernetes-sigs/kubefed/pull/1121) Update
   `kubefedctl federate` shorthand option for `--enable-type` to `-t` instead
   of `-e` to avoid confusing error message when only one dash is accidentally
   used e.g. `-enable-type`, resulting in a valid parsing of flags but
   erroneous use of the option.
-  [#1184](https://github.com/kubernetes-sigs/kubefed/pull/1184) Update dependencies to controller-runtime v0.5.0 and Kubernetes 1.17.3 
-  [#1195](https://github.com/kubernetes-sigs/kubefed/pull/1195) Use unstructured runtime.Object for more detailed reflector logging
-  [#1193](https://github.com/kubernetes-sigs/kubefed/pull/1193) Serve controller default metrics
-  [#1192](https://github.com/kubernetes-sigs/kubefed/pull/1192) Add pprof profiling support

# v0.1.0-rc6
-  [#1099](https://github.com/kubernetes-sigs/kubefed/pull/1099)
   Updates to propagation status are now only made in response to
   propagation to member clusters or errors in propagation. Previously
   propagation status was updated every time a federated resource was
   reconciled which could result in unnecessary resource consumption.
-  [#1098](https://github.com/kubernetes-sigs/kubefed/pull/1098)
   Propagated version is now only updated when changed.
-  [#1087](https://github.com/kubernetes-sigs/kubefed/issues/1087)
   The ReplicaScheduling controller now correctly updates existing
   overrides of `/spec/replicas`. Previously the controller was able
   to create and remove overrides for the `replicas` field but would
   fail to update them.
-  [#1076](https://github.com/kubernetes-sigs/kubefed/pull/1076)
   All `kubefedctl` commands now default `--host-cluster-context` to the
   current context in log messages.
-  [#1086](https://github.com/kubernetes-sigs/kubefed/pull/1086)
   `kubefedctl federate` now removes all metadata fields except labels
   from the template of federated resources created from a
   non-federated resource. Previously `metadata.annotations` and
   `metadata.finalizers` were not removed which could result in
   propagation errors.
-  [#1079](https://github.com/kubernetes-sigs/kubefed/issues/1079) The
   spec field is now required in generated federated types. For types
   generated previously, a check has been added so that a missing spec
   field does not cause a nil pointer exception.

# v0.1.0-rc5
-  [#1058](https://github.com/kubernetes-sigs/kubefed/issues/1058)
   KubeFedConfig spec.scope is now immutable.
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
