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

// FederatedDeploymentPlacementsGetter has a method to return a FederatedDeploymentPlacementInterface.
// A group's client should implement this interface.
type FederatedDeploymentPlacementsGetter interface {
	FederatedDeploymentPlacements(namespace string) FederatedDeploymentPlacementInterface
}

// FederatedDeploymentPlacementInterface has methods to work with FederatedDeploymentPlacement resources.
type FederatedDeploymentPlacementInterface interface {
	Create(*v1alpha1.FederatedDeploymentPlacement) (*v1alpha1.FederatedDeploymentPlacement, error)
	Update(*v1alpha1.FederatedDeploymentPlacement) (*v1alpha1.FederatedDeploymentPlacement, error)
	UpdateStatus(*v1alpha1.FederatedDeploymentPlacement) (*v1alpha1.FederatedDeploymentPlacement, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedDeploymentPlacement, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedDeploymentPlacementList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentPlacement, err error)
	FederatedDeploymentPlacementExpansion
}

// federatedDeploymentPlacements implements FederatedDeploymentPlacementInterface
type federatedDeploymentPlacements struct {
	client rest.Interface
	ns     string
}

// newFederatedDeploymentPlacements returns a FederatedDeploymentPlacements
func newFederatedDeploymentPlacements(c *FederationV1alpha1Client, namespace string) *federatedDeploymentPlacements {
	return &federatedDeploymentPlacements{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedDeploymentPlacement, and returns the corresponding federatedDeploymentPlacement object, and an error if there is any.
func (c *federatedDeploymentPlacements) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	result = &v1alpha1.FederatedDeploymentPlacement{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedDeploymentPlacements that match those selectors.
func (c *federatedDeploymentPlacements) List(opts v1.ListOptions) (result *v1alpha1.FederatedDeploymentPlacementList, err error) {
	result = &v1alpha1.FederatedDeploymentPlacementList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedDeploymentPlacements.
func (c *federatedDeploymentPlacements) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedDeploymentPlacement and creates it.  Returns the server's representation of the federatedDeploymentPlacement, and an error, if there is any.
func (c *federatedDeploymentPlacements) Create(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	result = &v1alpha1.FederatedDeploymentPlacement{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		Body(federatedDeploymentPlacement).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedDeploymentPlacement and updates it. Returns the server's representation of the federatedDeploymentPlacement, and an error, if there is any.
func (c *federatedDeploymentPlacements) Update(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	result = &v1alpha1.FederatedDeploymentPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		Name(federatedDeploymentPlacement.Name).
		Body(federatedDeploymentPlacement).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedDeploymentPlacements) UpdateStatus(federatedDeploymentPlacement *v1alpha1.FederatedDeploymentPlacement) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	result = &v1alpha1.FederatedDeploymentPlacement{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		Name(federatedDeploymentPlacement.Name).
		SubResource("status").
		Body(federatedDeploymentPlacement).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedDeploymentPlacement and deletes it. Returns an error if one occurs.
func (c *federatedDeploymentPlacements) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedDeploymentPlacements) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedDeploymentPlacement.
func (c *federatedDeploymentPlacements) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentPlacement, err error) {
	result = &v1alpha1.FederatedDeploymentPlacement{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federateddeploymentplacements").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
