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
	v1alpha1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	scheme "github.com/marun/fnord/pkg/client/clientset_generated/clientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederationPlacementsGetter has a method to return a FederationPlacementInterface.
// A group's client should implement this interface.
type FederationPlacementsGetter interface {
	FederationPlacements(namespace string) FederationPlacementInterface
}

// FederationPlacementInterface has methods to work with FederationPlacement resources.
type FederationPlacementInterface interface {
	Create(*v1alpha1.FederationPlacement) (*v1alpha1.FederationPlacement, error)
	Update(*v1alpha1.FederationPlacement) (*v1alpha1.FederationPlacement, error)
	UpdateStatus(*v1alpha1.FederationPlacement) (*v1alpha1.FederationPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederationPlacement, error)
	List(opts v1.ListOptions) (*v1alpha1.FederationPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederationPlacement, err error)
	FederationPlacementExpansion
}

// federationPlacements implements FederationPlacementInterface
type federationPlacements struct {
	client rest.Interface
	ns     string
}

// newFederationPlacements returns a FederationPlacements
func newFederationPlacements(c *FederationV1alpha1Client, namespace string) *federationPlacements {
	return &federationPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federationPlacement, and returns the corresponding federationPlacement object, and an error if there is any.
func (c *federationPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederationPlacement, err error) {
	result = &v1alpha1.FederationPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederationPlacements that match those selectors.
func (c *federationPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederationPlacementList, err error) {
	result = &v1alpha1.FederationPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federationPlacements.
func (c *federationPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federationPlacement and creates it.  Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *federationPlacements) Create(federationPlacement *v1alpha1.FederationPlacement) (result *v1alpha1.FederationPlacement, err error) {
	result = &v1alpha1.FederationPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federationplacements").
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federationPlacement and updates it. Returns the server's representation of the federationPlacement, and an error, if there is any.
func (c *federationPlacements) Update(federationPlacement *v1alpha1.FederationPlacement) (result *v1alpha1.FederationPlacement, err error) {
	result = &v1alpha1.FederationPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(federationPlacement.Name).
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federationPlacements) UpdateStatus(federationPlacement *v1alpha1.FederationPlacement) (result *v1alpha1.FederationPlacement, err error) {
	result = &v1alpha1.FederationPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(federationPlacement.Name).
		SubResource("status").
		Body(federationPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federationPlacement and deletes it. Returns an error if one occurs.
func (c *federationPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federationplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federationPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federationplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federationPlacement.
func (c *federationPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederationPlacement, err error) {
	result = &v1alpha1.FederationPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federationplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
