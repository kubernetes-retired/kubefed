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

// FederatedDeploymentsGetter has a method to return a FederatedDeploymentInterface.
// A group's client should implement this interface.
type FederatedDeploymentsGetter interface {
	FederatedDeployments(namespace string) FederatedDeploymentInterface
}

// FederatedDeploymentInterface has methods to work with FederatedDeployment resources.
type FederatedDeploymentInterface interface {
	Create(*v1alpha1.FederatedDeployment) (*v1alpha1.FederatedDeployment, error)
	Update(*v1alpha1.FederatedDeployment) (*v1alpha1.FederatedDeployment, error)
	UpdateStatus(*v1alpha1.FederatedDeployment) (*v1alpha1.FederatedDeployment, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.FederatedDeployment, error)
	List(opts v1.ListOptions) (*v1alpha1.FederatedDeploymentList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeployment, err error)
	FederatedDeploymentExpansion
}

// federatedDeployments implements FederatedDeploymentInterface
type federatedDeployments struct {
	client rest.Interface
	ns     string
}

// newFederatedDeployments returns a FederatedDeployments
func newFederatedDeployments(c *FederationV1alpha1Client, namespace string) *federatedDeployments {
	return &federatedDeployments{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedDeployment, and returns the corresponding federatedDeployment object, and an error if there is any.
func (c *federatedDeployments) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedDeployment, err error) {
	result = &v1alpha1.FederatedDeployment{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeployments").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedDeployments that match those selectors.
func (c *federatedDeployments) List(opts v1.ListOptions) (result *v1alpha1.FederatedDeploymentList, err error) {
	result = &v1alpha1.FederatedDeploymentList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federateddeployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedDeployments.
func (c *federatedDeployments) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federateddeployments").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedDeployment and creates it.  Returns the server's representation of the federatedDeployment, and an error, if there is any.
func (c *federatedDeployments) Create(federatedDeployment *v1alpha1.FederatedDeployment) (result *v1alpha1.FederatedDeployment, err error) {
	result = &v1alpha1.FederatedDeployment{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federateddeployments").
		Body(federatedDeployment).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedDeployment and updates it. Returns the server's representation of the federatedDeployment, and an error, if there is any.
func (c *federatedDeployments) Update(federatedDeployment *v1alpha1.FederatedDeployment) (result *v1alpha1.FederatedDeployment, err error) {
	result = &v1alpha1.FederatedDeployment{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeployments").
		Name(federatedDeployment.Name).
		Body(federatedDeployment).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedDeployments) UpdateStatus(federatedDeployment *v1alpha1.FederatedDeployment) (result *v1alpha1.FederatedDeployment, err error) {
	result = &v1alpha1.FederatedDeployment{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federateddeployments").
		Name(federatedDeployment.Name).
		SubResource("status").
		Body(federatedDeployment).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedDeployment and deletes it. Returns an error if one occurs.
func (c *federatedDeployments) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeployments").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedDeployments) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federateddeployments").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedDeployment.
func (c *federatedDeployments) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeployment, err error) {
	result = &v1alpha1.FederatedDeployment{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federateddeployments").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
