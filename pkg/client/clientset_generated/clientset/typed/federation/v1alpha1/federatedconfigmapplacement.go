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

// FederatedConfigMapPlacementsGetter has a method to return a FederatedConfigMapPlacementInterface.
// A group's client should implement this interface.
type FederatedConfigMapPlacementsGetter interface {
	FederatedConfigMapPlacements(namespace string) FederatedConfigMapPlacementInterface
}

// FederatedConfigMapPlacementInterface has methods to work with FederatedConfigMapPlacement resources.
type FederatedConfigMapPlacementInterface interface {
	Create(*v1alpha1.FederatedConfigMapPlacement) (*v1alpha1.FederatedConfigMapPlacement, error)
	Update(*v1alpha1.FederatedConfigMapPlacement) (*v1alpha1.FederatedConfigMapPlacement, error)
	UpdateStatus(*v1alpha1.FederatedConfigMapPlacement) (*v1alpha1.FederatedConfigMapPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedConfigMapPlacement, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedConfigMapPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedConfigMapPlacement, err error)
	FederatedConfigMapPlacementExpansion
}

// federatedConfigMapPlacements implements FederatedConfigMapPlacementInterface
type federatedConfigMapPlacements struct {
	client rest.Interface
	ns     string
}

// newFederatedConfigMapPlacements returns a FederatedConfigMapPlacements
func newFederatedConfigMapPlacements(c *FederationV1alpha1Client, namespace string) *federatedConfigMapPlacements {
	return &federatedConfigMapPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedConfigMapPlacement, and returns the corresponding federatedConfigMapPlacement object, and an error if there is any.
func (c *federatedConfigMapPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	result = &v1alpha1.FederatedConfigMapPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedConfigMapPlacements that match those selectors.
func (c *federatedConfigMapPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedConfigMapPlacementList, err error) {
	result = &v1alpha1.FederatedConfigMapPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedConfigMapPlacements.
func (c *federatedConfigMapPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedConfigMapPlacement and creates it.  Returns the server's representation of the federatedConfigMapPlacement, and an error, if there is any.
func (c *federatedConfigMapPlacements) Create(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	result = &v1alpha1.FederatedConfigMapPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		Body(federatedConfigMapPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedConfigMapPlacement and updates it. Returns the server's representation of the federatedConfigMapPlacement, and an error, if there is any.
func (c *federatedConfigMapPlacements) Update(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	result = &v1alpha1.FederatedConfigMapPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		Name(federatedConfigMapPlacement.Name).
		Body(federatedConfigMapPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedConfigMapPlacements) UpdateStatus(federatedConfigMapPlacement *v1alpha1.FederatedConfigMapPlacement) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	result = &v1alpha1.FederatedConfigMapPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		Name(federatedConfigMapPlacement.Name).
		SubResource("status").
		Body(federatedConfigMapPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedConfigMapPlacement and deletes it. Returns an error if one occurs.
func (c *federatedConfigMapPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedConfigMapPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedConfigMapPlacement.
func (c *federatedConfigMapPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedConfigMapPlacement, err error) {
	result = &v1alpha1.FederatedConfigMapPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedconfigmapplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
