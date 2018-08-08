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
		&FederatedCluster{},
		&FederatedClusterList{},
		&FederatedConfigMap{},
		&FederatedConfigMapList{},
		&FederatedConfigMapOverride{},
		&FederatedConfigMapOverrideList{},
		&FederatedConfigMapPlacement{},
		&FederatedConfigMapPlacementList{},
		&FederatedDeployment{},
		&FederatedDeploymentList{},
		&FederatedDeploymentOverride{},
		&FederatedDeploymentOverrideList{},
		&FederatedDeploymentPlacement{},
		&FederatedDeploymentPlacementList{},
		&FederatedIngress{},
		&FederatedIngressList{},
		&FederatedIngressPlacement{},
		&FederatedIngressPlacementList{},
		&FederatedJob{},
		&FederatedJobList{},
		&FederatedJobOverride{},
		&FederatedJobOverrideList{},
		&FederatedJobPlacement{},
		&FederatedJobPlacementList{},
		&FederatedNamespacePlacement{},
		&FederatedNamespacePlacementList{},
		&FederatedReplicaSet{},
		&FederatedReplicaSetList{},
		&FederatedReplicaSetOverride{},
		&FederatedReplicaSetOverrideList{},
		&FederatedReplicaSetPlacement{},
		&FederatedReplicaSetPlacementList{},
		&FederatedSecret{},
		&FederatedSecretList{},
		&FederatedSecretOverride{},
		&FederatedSecretOverrideList{},
		&FederatedSecretPlacement{},
		&FederatedSecretPlacementList{},
		&FederatedService{},
		&FederatedServiceList{},
		&FederatedServicePlacement{},
		&FederatedServicePlacementList{},
		&FederatedTypeConfig{},
		&FederatedTypeConfigList{},
		&PropagatedVersion{},
		&PropagatedVersionList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedCluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedConfigMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedConfigMap `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedConfigMapOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedConfigMapOverride `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedConfigMapPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedConfigMapPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedDeployment `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedDeploymentOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedDeploymentOverride `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedDeploymentPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedDeploymentPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedIngressList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedIngress `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedIngressPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedIngressPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedJob `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedJobOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedJobOverride `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedJobPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedJobPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedNamespacePlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedNamespacePlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedReplicaSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedReplicaSet `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedReplicaSetOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedReplicaSetOverride `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedReplicaSetPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedReplicaSetPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedSecret `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedSecretOverrideList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedSecretOverride `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedSecretPlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedSecretPlacement `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedService `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FederatedServicePlacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FederatedServicePlacement `json:"items"`
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
					Type: "object",
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
	FederatedConfigMapCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedconfigmaps.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedConfigMap",
				Plural: "federatedconfigmaps",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedConfigMapOverrideCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedconfigmapoverrides.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedConfigMapOverride",
				Plural: "federatedconfigmapoverrides",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"overrides": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"data": v1beta1.JSONSchemaProps{
													Type: "object",
												},
											},
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedConfigMapPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedconfigmapplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedConfigMapPlacement",
				Plural: "federatedconfigmapplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedDeploymentCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federateddeployments.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedDeployment",
				Plural: "federateddeployments",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedDeploymentOverrideCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federateddeploymentoverrides.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedDeploymentOverride",
				Plural: "federateddeploymentoverrides",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"overrides": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"replicas": v1beta1.JSONSchemaProps{
													Type:   "integer",
													Format: "int32",
												},
											},
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedDeploymentPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federateddeploymentplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedDeploymentPlacement",
				Plural: "federateddeploymentplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedIngressCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedingresses.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedIngress",
				Plural: "federatedingresses",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedIngressPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedingressplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedIngressPlacement",
				Plural: "federatedingressplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedJobCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedjobs.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedJob",
				Plural: "federatedjobs",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedJobOverrideCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedjoboverrides.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedJobOverride",
				Plural: "federatedjoboverrides",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"overrides": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"parallelism": v1beta1.JSONSchemaProps{
													Type:   "integer",
													Format: "int32",
												},
											},
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedJobPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedjobplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedJobPlacement",
				Plural: "federatedjobplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedNamespacePlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatednamespaceplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedNamespacePlacement",
				Plural: "federatednamespaceplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedReplicaSetCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedreplicasets.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedReplicaSet",
				Plural: "federatedreplicasets",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedReplicaSetOverrideCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedreplicasetoverrides.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedReplicaSetOverride",
				Plural: "federatedreplicasetoverrides",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"overrides": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"replicas": v1beta1.JSONSchemaProps{
													Type:   "integer",
													Format: "int32",
												},
											},
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedReplicaSetPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedreplicasetplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedReplicaSetPlacement",
				Plural: "federatedreplicasetplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedSecretCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedsecrets.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedSecret",
				Plural: "federatedsecrets",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedSecretOverrideCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedsecretoverrides.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedSecretOverride",
				Plural: "federatedsecretoverrides",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"overrides": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "object",
											Properties: map[string]v1beta1.JSONSchemaProps{
												"clusterName": v1beta1.JSONSchemaProps{
													Type: "string",
												},
												"data": v1beta1.JSONSchemaProps{
													Type: "object",
												},
											},
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedSecretPlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedsecretplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedSecretPlacement",
				Plural: "federatedsecretplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedServiceCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedservices.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedService",
				Plural: "federatedservices",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"template": v1beta1.JSONSchemaProps{
									Type:       "object",
									Properties: map[string]v1beta1.JSONSchemaProps{},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
			},
		},
	}
	// Define CRDs for resources
	FederatedServicePlacementCRD = v1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "federatedserviceplacements.core.federation.k8s.io",
		},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   "core.federation.k8s.io",
			Version: "v1alpha1",
			Names: v1beta1.CustomResourceDefinitionNames{
				Kind:   "FederatedServicePlacement",
				Plural: "federatedserviceplacements",
			},
			Scope: "Namespaced",
			Validation: &v1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
					Type: "object",
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
								"clusterNames": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
							},
						},
						"status": v1beta1.JSONSchemaProps{
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
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
					Type: "object",
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
								"overridePath": v1beta1.JSONSchemaProps{
									Type: "array",
									Items: &v1beta1.JSONSchemaPropsOrArray{
										Schema: &v1beta1.JSONSchemaProps{
											Type: "string",
										},
									},
								},
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
							Type:       "object",
							Properties: map[string]v1beta1.JSONSchemaProps{},
						},
					},
				},
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
					Type: "object",
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
