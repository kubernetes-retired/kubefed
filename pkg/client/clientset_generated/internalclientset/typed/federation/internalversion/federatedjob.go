/*
Copyright 2018 The Federation v2 Authors.

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
package internalversion

import (
	federation "github.com/marun/federation-v2/pkg/apis/federation"
	scheme "github.com/marun/federation-v2/pkg/client/clientset_generated/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// FederatedJobsGetter has a method to return a FederatedJobInterface.
// A group's client should implement this interface.
type FederatedJobsGetter interface {
	FederatedJobs(namespace string) FederatedJobInterface
}

// FederatedJobInterface has methods to work with FederatedJob resources.
type FederatedJobInterface interface {
	Create(*federation.FederatedJob) (*federation.FederatedJob, error)
	Update(*federation.FederatedJob) (*federation.FederatedJob, error)
	UpdateStatus(*federation.FederatedJob) (*federation.FederatedJob, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*federation.FederatedJob, error)
	List(opts v1.ListOptions) (*federation.FederatedJobList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedJob, err error)
	FederatedJobExpansion
}

// federatedJobs implements FederatedJobInterface
type federatedJobs struct {
	client rest.Interface
	ns     string
}

// newFederatedJobs returns a FederatedJobs
func newFederatedJobs(c *FederationClient, namespace string) *federatedJobs {
	return &federatedJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the federatedJob, and returns the corresponding federatedJob object, and an error if there is any.
func (c *federatedJobs) Get(name string, options v1.GetOptions) (result *federation.FederatedJob, err error) {
	result = &federation.FederatedJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedjobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of FederatedJobs that match those selectors.
func (c *federatedJobs) List(opts v1.ListOptions) (result *federation.FederatedJobList, err error) {
	result = &federation.FederatedJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("federatedjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested federatedJobs.
func (c *federatedJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("federatedjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a federatedJob and creates it.  Returns the server's representation of the federatedJob, and an error, if there is any.
func (c *federatedJobs) Create(federatedJob *federation.FederatedJob) (result *federation.FederatedJob, err error) {
	result = &federation.FederatedJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("federatedjobs").
		Body(federatedJob).
		Do().
		Into(result)
	return
}

// Update takes the representation of a federatedJob and updates it. Returns the server's representation of the federatedJob, and an error, if there is any.
func (c *federatedJobs) Update(federatedJob *federation.FederatedJob) (result *federation.FederatedJob, err error) {
	result = &federation.FederatedJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedjobs").
		Name(federatedJob.Name).
		Body(federatedJob).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *federatedJobs) UpdateStatus(federatedJob *federation.FederatedJob) (result *federation.FederatedJob, err error) {
	result = &federation.FederatedJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("federatedjobs").
		Name(federatedJob.Name).
		SubResource("status").
		Body(federatedJob).
		Do().
		Into(result)
	return
}

// Delete takes name of the federatedJob and deletes it. Returns an error if one occurs.
func (c *federatedJobs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedjobs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *federatedJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("federatedjobs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched federatedJob.
func (c *federatedJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedJob, err error) {
	result = &federation.FederatedJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("federatedjobs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
