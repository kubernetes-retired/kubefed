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

package federate

import (
	"fmt"
	"strings"

	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func templateValidationSchema(accessor schemaAccessor) (*v1beta1.CustomResourceValidation, error) {
	var properties map[string]v1beta1.JSONSchemaProps
	if accessor != nil {
		properties = accessor.templateSchema()
	}
	return ValidationSchema(v1beta1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]v1beta1.JSONSchemaProps{
			"template": {
				Type:       "object",
				Properties: properties,
			},
		},
	}), nil
}

func placementValidationSchema() *v1beta1.CustomResourceValidation {
	return ValidationSchema(v1beta1.JSONSchemaProps{
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
	})
}

func overrideValidationSchema(accessor schemaAccessor, overridePaths map[string][]string) (*v1beta1.CustomResourceValidation, error) {
	// No schema accessor means the validation schema cannot be determined
	if accessor == nil {
		return nil, nil
	}
	fieldsSchema := map[string]v1beta1.JSONSchemaProps{
		"clusterName": {
			Type: "string",
		},
	}
	for overrideName, pathEntries := range overridePaths {
		fieldSchema, err := accessor.schemaForField(pathEntries)
		errMsg := fmt.Sprintf("Unable to find validation schema for %q", strings.Join(pathEntries, "."))
		if err != nil {
			return nil, fmt.Errorf("%s: %v", errMsg, err)
		}
		if fieldSchema == nil {
			return nil, fmt.Errorf(errMsg)
		}
		fieldsSchema[overrideName] = *fieldSchema
	}

	return ValidationSchema(v1beta1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]v1beta1.JSONSchemaProps{
			"overrides": {
				Type: "array",
				Items: &v1beta1.JSONSchemaPropsOrArray{
					Schema: &v1beta1.JSONSchemaProps{
						Type:       "object",
						Properties: fieldsSchema,
					},
				},
			},
		},
	}), nil
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
				"metadata": {
					Type: "object",
				},
				"spec": specProps,
			},
		},
	}
}
