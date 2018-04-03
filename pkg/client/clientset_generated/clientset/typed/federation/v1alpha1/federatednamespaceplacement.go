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
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	scheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedNamespacePlacementsGetter has a method to return a FederatedNamespacePlacementInterface.
// A group's client should implement this interface.
type FederatedNamespacePlacementsGetter interface {
	FederatedNamespacePlacements() FederatedNamespacePlacementInterface
}

// FederatedNamespacePlacementInterface has methods to work with FederatedNamespacePlacement resources.
type FederatedNamespacePlacementInterface interface {
	Create(*v1alpha1.FederatedNamespacePlacement) (*v1alpha1.FederatedNamespacePlacement, error)
	Update(*v1alpha1.FederatedNamespacePlacement) (*v1alpha1.FederatedNamespacePlacement, error)
	UpdateStatus(*v1alpha1.FederatedNamespacePlacement) (*v1alpha1.FederatedNamespacePlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedNamespacePlacement, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedNamespacePlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedNamespacePlacement, err error)
	FederatedNamespacePlacementExpansion
}

// federatedNamespacePlacements implements FederatedNamespacePlacementInterface
type federatedNamespacePlacements struct {
	client rest.Interface
}

// newFederatedNamespacePlacements returns a FederatedNamespacePlacements
func newFederatedNamespacePlacements(c *FederationV1alpha1Client) *federatedNamespacePlacements {
	return &federatedNamespacePlacements{
		client: c.RESTClient(),
	}
}

// Get takes name of the federatedNamespacePlacement, and returns the corresponding federatedNamespacePlacement object, and an error if there is any.
func (c *federatedNamespacePlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	result = &v1alpha1.FederatedNamespacePlacement{}
	err = c.client.Get().
		Resource("federatednamespaceplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedNamespacePlacements that match those selectors.
func (c *federatedNamespacePlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedNamespacePlacementList, err error) {
	result = &v1alpha1.FederatedNamespacePlacementList{}
	err = c.client.Get().
		Resource("federatednamespaceplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedNamespacePlacements.
func (c *federatedNamespacePlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("federatednamespaceplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedNamespacePlacement and creates it.  Returns the server's representation of the federatedNamespacePlacement, and an error, if there is any.
func (c *federatedNamespacePlacements) Create(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	result = &v1alpha1.FederatedNamespacePlacement{}
	err = c.client.Post().
		Resource("federatednamespaceplacements").
		Body(federatedNamespacePlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedNamespacePlacement and updates it. Returns the server's representation of the federatedNamespacePlacement, and an error, if there is any.
func (c *federatedNamespacePlacements) Update(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	result = &v1alpha1.FederatedNamespacePlacement{}
	err = c.client.Put().
		Resource("federatednamespaceplacements").
		Name(federatedNamespacePlacement.Name).
		Body(federatedNamespacePlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedNamespacePlacements) UpdateStatus(federatedNamespacePlacement *v1alpha1.FederatedNamespacePlacement) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	result = &v1alpha1.FederatedNamespacePlacement{}
	err = c.client.Put().
		Resource("federatednamespaceplacements").
		Name(federatedNamespacePlacement.Name).
		SubResource("status").
		Body(federatedNamespacePlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedNamespacePlacement and deletes it. Returns an error if one occurs.
func (c *federatedNamespacePlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("federatednamespaceplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedNamespacePlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("federatednamespaceplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedNamespacePlacement.
func (c *federatedNamespacePlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedNamespacePlacement, err error) {
	result = &v1alpha1.FederatedNamespacePlacement{}
	err = c.client.Patch(pt).
		Resource("federatednamespaceplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
