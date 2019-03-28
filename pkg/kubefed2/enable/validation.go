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

package enable

import (
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

func federatedTypeValidationSchema(templateSchema map[string]v1beta1.JSONSchemaProps) *v1beta1.CustomResourceValidation {
	schema := ValidationSchema(v1beta1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]v1beta1.JSONSchemaProps{
			"placement": {
				Type: "object",
				Properties: map[string]v1beta1.JSONSchemaProps{
					// clusterName allows a scheduling mechanism to explicitly
					// indicate placement. If clusterName is provided,
					// labelSelector will be ignored.
					"clusterNames": {
						Type: "array",
						Items: &v1beta1.JSONSchemaPropsOrArray{
							Schema: &v1beta1.JSONSchemaProps{
								Type: "string",
							},
						},
					},
					"clusterSelector": {
						Type: "object",
						Properties: map[string]v1beta1.JSONSchemaProps{
							"matchExpressions": {
								Type: "array",
								Items: &v1beta1.JSONSchemaPropsOrArray{
									Schema: &v1beta1.JSONSchemaProps{
										Type: "object",
										Properties: map[string]v1beta1.JSONSchemaProps{
											"key": {
												Type: "string",
											},
											"operator": {
												Type: "string",
											},
											"values": {
												Type: "array",
												Items: &v1beta1.JSONSchemaPropsOrArray{
													Schema: &v1beta1.JSONSchemaProps{
														Type: "string",
													},
												},
											},
										},
										Required: []string{
											"key",
											"operator",
										},
									},
								},
							},
							"matchLabels": {
								Type: "object",
								AdditionalProperties: &v1beta1.JSONSchemaPropsOrBool{
									Schema: &v1beta1.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
			"overrides": {
				Type: "array",
				Items: &v1beta1.JSONSchemaPropsOrArray{
					Schema: &v1beta1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]v1beta1.JSONSchemaProps{
							"clusterName": {
								Type: "string",
							},
							"clusterOverrides": {
								Type: "array",
								Items: &v1beta1.JSONSchemaPropsOrArray{
									Schema: &v1beta1.JSONSchemaProps{
										Type: "object",
										Properties: map[string]v1beta1.JSONSchemaProps{
											"path": {
												Type: "string",
											},
											"value": {
												// Supporting the override of an arbitrary field
												// precludes up-front validation.  Errors in
												// the definition of override values will need to
												// be caught during propagation.
												AnyOf: []v1beta1.JSONSchemaProps{
													{
														Type: "string",
													},
													{
														Type: "integer",
													},
													{
														Type: "boolean",
													},
													{
														Type: "object",
													},
													{
														Type: "array",
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if templateSchema != nil {
		specProperties := schema.OpenAPIV3Schema.Properties["spec"].Properties
		specProperties["template"] = v1beta1.JSONSchemaProps{
			Type:       "object",
			Properties: templateSchema,
		}
		// Add retainReplicas field to types that exposes a replicas
		// field that could be targeted by HPA.
		if templateSpec, ok := templateSchema["spec"]; ok {
			if replicasField, ok := templateSpec.Properties["replicas"]; ok {
				if replicasField.Type == "integer" && replicasField.Format == "int32" {
					specProperties[util.RetainReplicasField] = v1beta1.JSONSchemaProps{
						Type: "boolean",
					}
				}
			}
		}

	}
	return schema
}

func ValidationSchema(specProps v1beta1.JSONSchemaProps) *v1beta1.CustomResourceValidation {
	return &v1beta1.CustomResourceValidation{
		OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
			Properties: map[string]v1beta1.JSONSchemaProps{
				"apiVersion": {
					Type: "string",
				},
				"kind": {
					Type: "string",
				},
				// TODO(marun) Add a comprehensive schema for metadata
				"metadata": {
					Type: "object",
				},
				"spec": specProps,
			},
		},
	}
}
