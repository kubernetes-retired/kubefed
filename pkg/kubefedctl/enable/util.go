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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
)

func DecodeYAMLFromFile(filename string, obj interface{}) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return DecodeYAML(f, obj)
}

func DecodeYAML(r io.Reader, obj interface{}) error {
	decoder := yaml.NewYAMLToJSONDecoder(r)
	return decoder.Decode(obj)
}

func CrdForAPIResource(apiResource metav1.APIResource, validation *apiextv1b1.CustomResourceValidation, shortNames []string) *apiextv1b1.CustomResourceDefinition {
	scope := apiextv1b1.ClusterScoped
	if apiResource.Namespaced {
		scope = apiextv1b1.NamespaceScoped
	}
	return &apiextv1b1.CustomResourceDefinition{
		// Explicitly including TypeMeta will ensure it will be
		// serialized properly to yaml.
		TypeMeta: metav1.TypeMeta{
			Kind:       "CustomResourceDefinition",
			APIVersion: "apiextensions.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: typeconfig.GroupQualifiedName(apiResource),
		},
		Spec: apiextv1b1.CustomResourceDefinitionSpec{
			Group:   apiResource.Group,
			Version: apiResource.Version,
			Scope:   scope,
			Names: apiextv1b1.CustomResourceDefinitionNames{
				Plural:     apiResource.Name,
				Kind:       apiResource.Kind,
				ShortNames: shortNames,
			},
			Validation: validation,
		},
	}
}

func LookupAPIResource(config *rest.Config, key, targetVersion string) (*metav1.APIResource, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating discovery client")
	}

	resourceLists, err := client.ServerPreferredResources()
	if err != nil {
		return nil, errors.Wrap(err, "Error listing api resources")
	}

	// TODO(marun) Consider using a caching scheme ala kubectl
	lowerKey := strings.ToLower(key)
	var targetResource *metav1.APIResource
	var matchedResources []string
	var matchResource = func(resource metav1.APIResource, gv schema.GroupVersion) {
		if targetResource == nil {
			targetResource = resource.DeepCopy()
			targetResource.Group = gv.Group
			targetResource.Version = gv.Version
		}

		matchedResources = append(matchedResources, groupQualifiedName(resource.Name, gv.Group))
	}

	for _, resourceList := range resourceLists {
		// The list holds the GroupVersion for its list of APIResources
		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Error parsing GroupVersion")
		}
		if len(targetVersion) > 0 && gv.Version != targetVersion {
			continue
		}
		for _, resource := range resourceList.APIResources {
			if lowerKey == resource.Name ||
				lowerKey == resource.SingularName ||
				lowerKey == strings.ToLower(resource.Kind) ||
				lowerKey == fmt.Sprintf("%s.%s", resource.Name, gv.Group) {

				matchResource(resource, gv)
				continue
			}
			for _, shortName := range resource.ShortNames {
				if lowerKey == strings.ToLower(shortName) {
					matchResource(resource, gv)
					break
				}
			}
		}

	}
	if len(matchedResources) > 1 {
		return nil, errors.Errorf("Multiple resources are matched by %q: %s. A group-qualified plural name must be provided.", key, strings.Join(matchedResources, ", "))
	}

	if targetResource != nil {
		return targetResource, nil
	}

	return nil, errors.Errorf("Unable to find api resource named %q.", key)
}

func resourceKey(apiResource metav1.APIResource) string {
	var group string
	if len(apiResource.Group) == 0 {
		group = "core"
	} else {
		group = apiResource.Group
	}
	var version string
	if len(apiResource.Version) == 0 {
		version = "v1"
	} else {
		version = apiResource.Version
	}
	return fmt.Sprintf("%s.%s/%s", apiResource.Name, group, version)
}

func groupQualifiedName(name, group string) string {
	apiResource := metav1.APIResource{
		Name:  name,
		Group: group,
	}

	return typeconfig.GroupQualifiedName(apiResource)
}
