/*
Copyright 2019 The Kubernetes Authors.

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
	"strings"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"

	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/enable"
)

var systemMetadataFields = []string{"selfLink", "uid", "resourceVersion", "generation", "creationTimestamp", "deletionTimestamp", "deletionGracePeriodSeconds"}

func RemoveUnwantedFields(resource *unstructured.Unstructured) {
	for _, field := range systemMetadataFields {
		unstructured.RemoveNestedField(resource.Object, "metadata", field)
		// For resources with pod template subresource (jobs, deployments, replicasets)
		unstructured.RemoveNestedField(resource.Object, "spec", "template", "metadata", field)
	}
	unstructured.RemoveNestedField(resource.Object, "metadata", "name")
	unstructured.RemoveNestedField(resource.Object, "metadata", "namespace")
	unstructured.RemoveNestedField(resource.Object, "apiVersion")
	unstructured.RemoveNestedField(resource.Object, "kind")
	unstructured.RemoveNestedField(resource.Object, "status")
}

func SetBasicMetaFields(resource *unstructured.Unstructured, apiResource metav1.APIResource, name, namespace, generateName string) {
	resource.SetKind(apiResource.Kind)
	gv := schema.GroupVersion{Group: apiResource.Group, Version: apiResource.Version}
	resource.SetAPIVersion(gv.String())
	resource.SetName(name)
	if generateName != "" {
		resource.SetGenerateName(generateName)
	}
	if apiResource.Namespaced {
		resource.SetNamespace(namespace)
	}
}

func namespacedAPIResourceMap(config *rest.Config) (map[string]metav1.APIResource, error) {
	apiResourceLists, err := enable.GetServerPreferredResources(config)
	if err != nil {
		return nil, err
	}

	apiResources := make(map[string]metav1.APIResource)
	for _, apiResourceList := range apiResourceLists {
		if len(apiResourceList.APIResources) == 0 {
			continue
		}

		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing GroupVersion")
		}

		group := gv.Group
		for _, apiResource := range apiResourceList.APIResources {
			if !apiResource.Namespaced || isFederatedAPIResource(apiResource.Kind, group) ||
				apiResourceMatchesSkipNames(apiResource, group) {
				continue
			}

			// TODO(irfanurrehman): Define a strategy to federate the latest available version
			// Include a kind only once (if found under multiple group/versions).
			if _, ok := apiResources[apiResource.Kind]; ok {
				continue
			}

			// The individual apiResources do not have the group and version set
			apiResource.Group = group
			apiResource.Version = gv.Version

			apiResources[apiResource.Kind] = apiResource
		}
	}

	return apiResources, nil
}

func apiResourceMatchesSkipNames(apiResource metav1.APIResource, group string) bool {
	for _, name := range controllerCreatedAPIResourceNames {
		if name == "" {
			continue
		}
		if enable.NameMatchesResource(name, apiResource, group) {
			return true
		}
	}
	return false
}

func isFederatedAPIResource(kind, group string) bool {
	return strings.HasPrefix(kind, federationKindPrefix) && group == enable.DefaultFederationGroup
}

// resources stores a list of resources for an api type
type resources struct {
	// resource type information
	apiResource metav1.APIResource
	// resource list
	resources []*unstructured.Unstructured
}

func getResourcesInNamespace(config *rest.Config, namespace string) ([]resources, error) {
	apiResources, err := namespacedAPIResourceMap(config)
	if err != nil {
		return nil, err
	}

	resourcesInNamespace := []resources{}
	for _, apiResource := range apiResources {
		client, err := ctlutil.NewResourceClient(config, &apiResource)
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating client for %s", apiResource.Kind)
		}

		resourceList, err := client.Resources(namespace).List(metav1.ListOptions{})
		if apierrors.IsNotFound(err) || resourceList == nil {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "Error listing resources for %s", apiResource.Kind)
		}

		// It would be a waste of cycles to iterate through empty slices while federating resource
		if len(resourceList.Items) == 0 {
			continue
		}

		targetResources := resources{apiResource: apiResource}
		for _, resource := range resourceList.Items {
			targetResources.resources = append(targetResources.resources, &resource)
		}
		resourcesInNamespace = append(resourcesInNamespace, targetResources)
	}

	return resourcesInNamespace, nil
}
