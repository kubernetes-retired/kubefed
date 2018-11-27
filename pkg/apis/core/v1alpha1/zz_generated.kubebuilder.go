/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SchemeGroupVersion is group version used to register these objects
var SchemeGroupVersion = schema.GroupVersion{Group: "core.federation.k8s.io", Version: "v1alpha1"}

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ClusterPropagatedVersion{},
		&ClusterPropagatedVersionList{},
		&FederatedCluster{},
		&FederatedClusterList{},
		&FederatedServiceStatus{},
		&FederatedServiceStatusList{},
		&FederatedTypeConfig{},
		&FederatedTypeConfigList{},
		&PropagatedVersion{},
		&PropagatedVersionList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterPropagatedVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPropagatedVersion `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedCluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedServiceStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedServiceStatus `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedTypeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedTypeConfig `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PropagatedVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PropagatedVersion `json:"items"`
}

// CRD Generation
func getFloat(f float64) *float64 {
	return &f
}

func getInt(i int64) *int64 {
	return &i
}

var (
	// Define CRDs for resources
	ClusterPropagatedVersionCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "clusterpropagatedversions.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "ClusterPropagatedVersion",
				Plural: "clusterpropagatedversions",
			},
			Scope: "Cluster",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Properties: map[string]v1beta1.JSONSchemaProps{
						"apiVersion": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"kind": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": v1beta1.JSONSchemaProps{
							Type: "object",
						},
						"spec": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
						"status": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"clusterVersions": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"version": v1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
										},
									},
								},
								"overridesVersion": v1beta1.JSONSchemaProps{
									Type: "string",
								},
								"templateVersion": v1beta1.JSONSchemaProps{
									Type: "string",
								},
							},
						},
					},
				},
			},
			Subresources: &v1beta1.CustomResourceSubresources{
				Status: &v1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
	// Define CRDs for resources
	FederatedClusterCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedclusters.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedCluster",
				Plural: "federatedclusters",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Properties: map[string]v1beta1.JSONSchemaProps{
						"apiVersion": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"kind": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": v1beta1.JSONSchemaProps{
							Type: "object",
						},
						"spec": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"clusterRef": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
								"secretRef": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"conditions": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"lastProbeTime": v1beta1.JSONSchemaProps{
													Type:   "string",
													Format: "date-time",
												},
												"lastTransitionTime": v1beta1.JSONSchemaProps{
													Type:   "string",
													Format: "date-time",
												},
												"message": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"reason": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"status": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"type": v1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
											Required: []string{
												"type",
												"status",
											}},
									},
								},
								"region": v1beta1.JSONSchemaProps{
									Type: "string",
								},
								"zone": v1beta1.JSONSchemaProps{
									Type: "string",
								},
							},
						},
					},
				},
			},
			Subresources: &v1beta1.CustomResourceSubresources{
				Status: &v1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
	// Define CRDs for resources
	FederatedServiceStatusCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedservicestatuses.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedServiceStatus",
				Plural: "federatedservicestatuses",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Properties: map[string]v1beta1.JSONSchemaProps{
						"apiVersion": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"clusterStatus": v1beta1.JSONSchemaProps{
							Type: "array",
							Items: &v1beta1.JSONSchemaPropsOrArray{
								Schema: &v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"clusterName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"status": v1beta1.JSONSchemaProps{
											Type:       "object",
											Properties: map[string]v1beta1.JSONSchemaProps{},
										},
									},
								},
							},
						},
						"kind": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": v1beta1.JSONSchemaProps{
							Type: "object",
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedTypeConfigCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedtypeconfigs.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedTypeConfig",
				Plural: "federatedtypeconfigs",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Properties: map[string]v1beta1.JSONSchemaProps{
						"apiVersion": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"kind": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": v1beta1.JSONSchemaProps{
							Type: "object",
						},
						"spec": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"comparisonField": v1beta1.JSONSchemaProps{
									Type: "string",
								},
								"enableStatus": v1beta1.JSONSchemaProps{
									Type: "boolean",
								},
								"namespaced": v1beta1.JSONSchemaProps{
									Type: "boolean",
								},
								"override": v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"group": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"kind": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"pluralName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"version": v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
									Required: []string{
										"kind",
									}},
								"placement": v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"group": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"kind": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"pluralName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"version": v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
									Required: []string{
										"kind",
									}},
								"propagationEnabled": v1beta1.JSONSchemaProps{
									Type: "boolean",
								},
								"status": v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"group": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"kind": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"pluralName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"version": v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
									Required: []string{
										"kind",
									}},
								"target": v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"group": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"kind": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"pluralName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"version": v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
									Required: []string{
										"kind",
									}},
								"template": v1beta1.JSONSchemaProps{
									Type: "object",
									Properties: map[string]v1beta1.JSONSchemaProps{
										"group": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"kind": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"pluralName": v1beta1.JSONSchemaProps{
											Type: "string",
										},
										"version": v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
									Required: []string{
										"kind",
									}},
							},
							Required: []string{
								"target",
								"namespaced",
								"comparisonField",
								"propagationEnabled",
								"template",
								"placement",
							}},
						"status": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"observedGeneration": v1beta1.JSONSchemaProps{
									Type:   "integer",
									Format: "int64",
								},
								"propagationController": v1beta1.JSONSchemaProps{
									Type: "string",
								},
								"statusController": v1beta1.JSONSchemaProps{
									Type: "string",
								},
							},
						},
					},
				},
			},
			Subresources: &v1beta1.CustomResourceSubresources{
				Status: &v1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
	// Define CRDs for resources
	PropagatedVersionCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "propagatedversions.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "PropagatedVersion",
				Plural: "propagatedversions",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Properties: map[string]v1beta1.JSONSchemaProps{
						"apiVersion": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"kind": v1beta1.JSONSchemaProps{
							Type: "string",
						},
						"metadata": v1beta1.JSONSchemaProps{
							Type: "object",
						},
						"spec": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
						"status": v1beta1.JSONSchemaProps{
							Type: "object",
							Properties: map[string]v1beta1.JSONSchemaProps{
								"clusterVersions": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"version": v1beta1.JSONSchemaProps{
													Type: "string",
												},
											},
										},
									},
								},
								"overridesVersion": v1beta1.JSONSchemaProps{
									Type: "string",
								},
								"templateVersion": v1beta1.JSONSchemaProps{
									Type: "string",
								},
							},
						},
					},
				},
			},
			Subresources: &v1beta1.CustomResourceSubresources{
				Status: &v1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
)
