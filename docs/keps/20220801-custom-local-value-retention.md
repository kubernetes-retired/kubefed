# KubeFed - Custom Local Value Retention

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

In most cases, the KubeFed sync controller will overwrite any changes made to resources it manages in member clusters. But in some cases, some fields of the resource might be managed by other controllers which should be retained. Although KubeFed has some [built-in retention rules](https://github.com/kubernetes-sigs/kubefed/blob/master/docs/userguide.md#local-value-retention), it is impossible to cover every use case. A custom local value retention mechanism is needed to handle these various use cases.

## Motivation

The built-in retention rules cannot handle every use case of outside users.

### Goals

* Make local value retention customizable at resource type level.

### Non Goals

* Make local value retention customizable for every single resource object.
* Completely replace the built-in retention rules.

## Proposal

Extend current `FederatedTypeConfig`'s spec to include a new field `retention` as below:

```go
// FederatedTypeConfigSpec defines the desired state of FederatedTypeConfig.
type FederatedTypeConfigSpec struct {
	// The configuration of the target type. If not set, the pluralName and
	// groupName fields will be set from the metadata.name of this resource. The
	// kind field must be set.
	TargetType APIResource `json:"targetType"`
	// Whether or not propagation to member clusters should be enabled.
	Propagation PropagationMode `json:"propagation"`
	// Configuration for the federated type that defines (via
	// template, placement and overrides fields) how the target type
	// should appear in multiple cluster.
	FederatedType APIResource `json:"federatedType"`
	// Configuration for the status type that holds information about which type
	// holds the status of the federated resource. If not provided, the group
	// and version will default to those provided for the federated type api
	// resource.
	// +optional
	StatusType *APIResource `json:"statusType,omitempty"`
	// Whether or not Status object should be populated.
	// +optional
	StatusCollection *StatusCollectionMode `json:"statusCollection,omitempty"`
	// Custom retention fields of the target type. If provided, the value of fields
	// included will be retained on local resources if it is not set in the template
	// of the corresponding federated resource.
	// +optional
	Retention *Retention `json:"retention,omitempty"`
}

// Retention defines which fields should be retained on local resources.
type Retention struct {
	// Path of the fields need to be retained.
	Fields []string `json:"fields,omitempty"`
	// Key of the labels need to be retained.
	Labels []string `json:"labels,omitempty"`
}
```

Users can specify additional fields or labels to be retained on local resources by simply modifying their corresponding `FederatedTypeConfig`s. The modification will lead to a restart of the sync controller for that type to apply the custom retention rules on the fly.

For fields of custom retention, the format is somewhat like the key of field selector which would look like `spec.foo`. For fields of array object, `[]` should be appended after the corresponding array field like `spec.foo[].bar`. This is a specialized format that we need to doc.

The custom retention will happen in the same stage of the built-in retention, that is, before applying overrides on update. So it is still possible to set the fields via overrides which keeps the same behavior with built-in retention.

We might notice that there are two retention types of built-in retention: `Always` and `Conditional`. To make custom retention simple and flexible, its type will always be `Conditional` which means that the retention would only happen if users do not specify a value (including empty value) of the field in the template of the corresponding federated resource. This enables users to set values in the template of some federated resources to bypass the retention for them. In other words, users shouldn't provide a value of the field they want to be retained on local resources in the template nor overrides of the corresponding federated resource.

### User Stories

#### Story 1

Users deliver admission webhooks via KubeFed, in such case they apply a federated resource as below:

```yaml
apiVersion: types.kubefed.io/v1beta1
kind: FederatedValidatingWebhookConfiguration
metadata:
  name: foo
spec:
  template:
    spec:
      webhooks:
      - admissionReviewVersions:
        - v1
        clientConfig:
          service:
            name: foobar-controller
            namespace: kube-system
            path: /validate-foo
            port: 443
        name: validating-foo
        rules:
        - apiGroups:
          - bar.io
          apiVersions:
          - v1
          operations:
          - CREATE
          - UPDATE
          - DELETE
          resources:
          - foo
          scope: '*'
        sideEffects: None
  placement:
    clusters:
    - name: cluster1
    - name: cluster2
```

At the same time, they use [CA injector of cert-manager](https://cert-manager.io/docs/concepts/ca-injector/) to populate the `caBundle` field of `ValidatingWebhookConfiguration` in each member cluster. However, since that field is not set in the template of `FederatedValidatingWebhookConfiguration`, the value populated by cert-manager in member cluster will be erased by KubeFed which make the CA injector of cert-manager unusable. In such case, custom value retention is needed.

To use custom local value retention, users modify the `FederatedTypeConfig` of `ValidatingWebhookConfiguration` as below:

```yaml
apiVersion: core.kubefed.io/v1beta1
kind: FederatedTypeConfig
metadata:
  name: validatingwebhookconfigurations.admissionregistration.k8s.io
  namespace: kube-federation-system
spec:
  federatedType:
    group: types.kubefed.io
    kind: FederatedValidatingWebhookConfiguration
    pluralName: federatedvalidatingwebhookconfigurations
    scope: Cluster
    version: v1beta1
  propagation: Enabled
  targetType:
    group: admissionregistration.k8s.io
    kind: ValidatingWebhookConfiguration
    pluralName: validatingwebhookconfigurations
    scope: Cluster
    version: v1
  retention:
    fields:
    - spec.webhooks[].clientConfig.caBundle
```

This way, the `caBundle` field of `ValidatingWebhookConfiguration` in member clusters would be retained and both KubeFed and cert-manager are happy.

Furthermore, if users want to specify a global `caBundle` for another `FederatedValidatingWebhookConfiguration` resource, just specify that in template or overrides then it will be propagated to member clusters with no conflict with the custom retention setting.

## Alternatives

Another approach would be adding the custom local value retention settings to federated resource itself. This way we can do more granular settings but with more verbosity. This can also be considered as an override based on the settings in `FederatedTypeConfig` as this KEP proposed.
