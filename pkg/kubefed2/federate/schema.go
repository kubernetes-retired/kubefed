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

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/util/proto"
	"k8s.io/kubectl/pkg/framework/openapi"
)

type schemaAccessor interface {
	templateSchema() map[string]apiextv1b1.JSONSchemaProps
	schemaForField(pathEntries []string) (*apiextv1b1.JSONSchemaProps, error)
}

func newSchemaAccessor(config *rest.Config, apiResource metav1.APIResource) (schemaAccessor, error) {
	// Assume the resource may be a CRD, and fall back to OpenAPI if that is not the case.
	crdAccessor, err := newCRDSchemaAccessor(config, apiResource)
	if err != nil {
		return nil, err
	}
	if crdAccessor != nil {
		return crdAccessor, nil
	}
	return newOpenAPISchemaAccessor(config, apiResource)
}

type crdSchemaAccessor struct {
	validation *apiextv1b1.CustomResourceValidation
}

func newCRDSchemaAccessor(config *rest.Config, apiResource metav1.APIResource) (schemaAccessor, error) {
	// CRDs must have a group
	if len(apiResource.Group) == 0 {
		return nil, nil
	}
	// Check whether the target resource is a crd
	crdClient, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create crd clientset: %v", err)
	}
	crdName := fmt.Sprintf("%s.%s", apiResource.Name, apiResource.Group)
	crd, err := crdClient.CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("Error attempting retrieval of crd %q: %v", crdName, err)
	}
	return &crdSchemaAccessor{validation: crd.Spec.Validation}, nil
}

func (a *crdSchemaAccessor) templateSchema() map[string]apiextv1b1.JSONSchemaProps {
	if a.validation.OpenAPIV3Schema != nil {
		return a.validation.OpenAPIV3Schema.Properties
	}
	return nil
}

func (a *crdSchemaAccessor) schemaForField(pathEntries []string) (*apiextv1b1.JSONSchemaProps, error) {
	if a.validation == nil || a.validation.OpenAPIV3Schema == nil ||
		a.validation.OpenAPIV3Schema.Properties == nil {

		return nil, fmt.Errorf("Validation schema not available for target CRD")
	}
	schemaMap := a.validation.OpenAPIV3Schema.Properties
	path := strings.Join(pathEntries, ".")
	var schema *apiextv1b1.JSONSchemaProps
	for _, pathEntry := range pathEntries {
		foundSchema, ok := schemaMap[pathEntry]
		if !ok {
			return nil, fmt.Errorf("Error finding schema for %q: %q missing from local map", path, pathEntry)
		}
		schema = &foundSchema
		if schema.Properties != nil {
			schemaMap = schema.Properties
		}
	}
	return schema, nil
}

type openAPISchemaAccessor struct {
	targetResource proto.Schema
}

func newOpenAPISchemaAccessor(config *rest.Config, apiResource metav1.APIResource) (schemaAccessor, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("Error creating discovery client: %v", err)
	}
	resources, err := openapi.NewOpenAPIGetter(client).Get()
	if err != nil {
		return nil, fmt.Errorf("Error loading openapi schema: %v", err)
	}
	gvk := schema.GroupVersionKind{
		Group:   apiResource.Group,
		Version: apiResource.Version,
		Kind:    apiResource.Kind,
	}
	targetResource := resources.LookupResource(gvk)
	if targetResource == nil {
		return nil, fmt.Errorf("Unable to find openapi schema for %q", gvk)
	}
	return &openAPISchemaAccessor{
		targetResource: targetResource,
	}, nil
}

func (a *openAPISchemaAccessor) templateSchema() map[string]apiextv1b1.JSONSchemaProps {
	var templateSchema *apiextv1b1.JSONSchemaProps
	visitor := &jsonSchemaVistor{
		collect: func(schema apiextv1b1.JSONSchemaProps) {
			templateSchema = &schema
		},
	}
	a.targetResource.Accept(visitor)

	return templateSchema.Properties
}

func (a *openAPISchemaAccessor) schemaForField(pathEntries []string) (*apiextv1b1.JSONSchemaProps, error) {
	var fieldSchema *apiextv1b1.JSONSchemaProps
	visitor := &fieldSchemaVistor{
		collect: func(schema apiextv1b1.JSONSchemaProps) {
			fieldSchema = &schema
		},
		pathEntries: pathEntries,
	}
	a.targetResource.Accept(visitor)
	return fieldSchema, nil
}

// jsonSchemaVistor converts proto.Schema resources into json schema.
// A local visitor (and associated callback) is intended to be created
// whenever a function needs to recurse.
//
// TODO(marun) Generate more extensive schema if/when openapi schema
// provides more detail as per https://github.com/ant31/crd-validation
type jsonSchemaVistor struct {
	collect func(schema apiextv1b1.JSONSchemaProps)
}

func (v *jsonSchemaVistor) VisitArray(a *proto.Array) {
	arraySchema := apiextv1b1.JSONSchemaProps{
		Type:  "array",
		Items: &apiextv1b1.JSONSchemaPropsOrArray{},
	}
	localVisitor := &jsonSchemaVistor{
		collect: func(schema apiextv1b1.JSONSchemaProps) {
			arraySchema.Items.Schema = &schema
		},
	}
	a.SubType.Accept(localVisitor)
	v.collect(arraySchema)
}

func (v *jsonSchemaVistor) VisitMap(m *proto.Map) {
	mapSchema := apiextv1b1.JSONSchemaProps{
		Type: "object",
		AdditionalProperties: &apiextv1b1.JSONSchemaPropsOrBool{
			Allows: true,
		},
	}
	localVisitor := &jsonSchemaVistor{
		collect: func(schema apiextv1b1.JSONSchemaProps) {
			mapSchema.AdditionalProperties.Schema = &schema
		},
	}
	m.SubType.Accept(localVisitor)
	v.collect(mapSchema)
}

func (v *jsonSchemaVistor) VisitPrimitive(p *proto.Primitive) {
	schema := schemaForPrimitive(p)
	v.collect(schema)
}

func (v *jsonSchemaVistor) VisitKind(k *proto.Kind) {
	kindSchema := apiextv1b1.JSONSchemaProps{
		Type:       "object",
		Properties: make(map[string]apiextv1b1.JSONSchemaProps),
		Required:   k.RequiredFields,
	}
	for key, fieldSchema := range k.Fields {
		// Status cannot be defined for a template
		if key == "status" {
			continue
		}
		localVisitor := &jsonSchemaVistor{
			collect: func(schema apiextv1b1.JSONSchemaProps) {
				kindSchema.Properties[key] = schema
			},
		}
		fieldSchema.Accept(localVisitor)
	}
	v.collect(kindSchema)
}

func (v *jsonSchemaVistor) VisitReference(r proto.Reference) {
	r.SubSchema().Accept(v)
}

// fieldSchemaVistor determines the type and format of the given field
// path.  Only primitive fields are supported, and this schema is
// likely to be deprecated in favor of generic overrides in the near
// future.
type fieldSchemaVistor struct {
	collect     func(schema apiextv1b1.JSONSchemaProps)
	pathEntries []string
}

func (v *fieldSchemaVistor) VisitArray(a *proto.Array) {
	// Arrays are not supported as override targets
}

func (v *fieldSchemaVistor) VisitMap(m *proto.Map) {
	// Maps are only supported as direct override targets (
	// e.g. secret 'data' field).  The simple path-based override
	// mechanism doesn't have a way of expressing a path that includes
	// an entry in a map (e.g. spec.mydata["foo"]).
	mapSchema := apiextv1b1.JSONSchemaProps{
		Type: "object",
		AdditionalProperties: &apiextv1b1.JSONSchemaPropsOrBool{
			Allows: true,
		},
	}
	localVisitor := &jsonSchemaVistor{
		collect: func(schema apiextv1b1.JSONSchemaProps) {
			mapSchema.AdditionalProperties.Schema = &schema
		},
	}
	m.SubType.Accept(localVisitor)
	v.collect(mapSchema)
}

func (v *fieldSchemaVistor) VisitPrimitive(p *proto.Primitive) {
	schema := schemaForPrimitive(p)
	v.collect(schema)
}

func (v *fieldSchemaVistor) VisitKind(k *proto.Kind) {
	for key, fieldSchema := range k.Fields {
		if key == v.pathEntries[0] {
			localVisitor := &fieldSchemaVistor{
				collect:     v.collect,
				pathEntries: v.pathEntries[1:],
			}
			fieldSchema.Accept(localVisitor)
			break
		}
	}
}

func (v *fieldSchemaVistor) VisitReference(r proto.Reference) {
	r.SubSchema().Accept(v)
}

func schemaForPrimitive(p *proto.Primitive) apiextv1b1.JSONSchemaProps {
	schema := apiextv1b1.JSONSchemaProps{}

	if p.Format == "int-or-string" {
		schema.AnyOf = []apiextv1b1.JSONSchemaProps{
			{
				Type:   "integer",
				Format: "int32",
			},
			{
				Type: "string",
			},
		}
		return schema
	}

	if len(p.Format) > 0 {
		schema.Format = p.Format
	}
	schema.Type = p.Type
	return schema
}
