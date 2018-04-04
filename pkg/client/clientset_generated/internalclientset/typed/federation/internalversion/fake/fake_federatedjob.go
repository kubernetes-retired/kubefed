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
package fake

import (
	federation "github.com/marun/federation-v2/pkg/apis/federation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedJobs implements FederatedJobInterface
type FakeFederatedJobs struct {
	Fake *FakeFederation
	ns   string
}

var federatedjobsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedjobs"}

var federatedjobsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedJob"}

// Get takes name of the federatedJob, and returns the corresponding federatedJob object, and an error if there is any.
func (c *FakeFederatedJobs) Get(name string, options v1.GetOptions) (result *federation.FederatedJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedjobsResource, c.ns, name), &federation.FederatedJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJob), err
}

// List takes label and field selectors, and returns the list of FederatedJobs that match those selectors.
func (c *FakeFederatedJobs) List(opts v1.ListOptions) (result *federation.FederatedJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedjobsResource, federatedjobsKind, c.ns, opts), &federation.FederatedJobList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedJobList{}
	for _, item := range obj.(*federation.FederatedJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedJobs.
func (c *FakeFederatedJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedjobsResource, c.ns, opts))

}

// Create takes the representation of a federatedJob and creates it.  Returns the server's representation of the federatedJob, and an error, if there is any.
func (c *FakeFederatedJobs) Create(federatedJob *federation.FederatedJob) (result *federation.FederatedJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedjobsResource, c.ns, federatedJob), &federation.FederatedJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJob), err
}

// Update takes the representation of a federatedJob and updates it. Returns the server's representation of the federatedJob, and an error, if there is any.
func (c *FakeFederatedJobs) Update(federatedJob *federation.FederatedJob) (result *federation.FederatedJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedjobsResource, c.ns, federatedJob), &federation.FederatedJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJob), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedJobs) UpdateStatus(federatedJob *federation.FederatedJob) (*federation.FederatedJob, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedjobsResource, "status", c.ns, federatedJob), &federation.FederatedJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJob), err
}

// Delete takes name of the federatedJob and deletes it. Returns an error if one occurs.
func (c *FakeFederatedJobs) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedjobsResource, c.ns, name), &federation.FederatedJob{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedjobsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedJobList{})
	return err
}

// Patch applies the patch and returns the patched federatedJob.
func (c *FakeFederatedJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedjobsResource, c.ns, name, data, subresources...), &federation.FederatedJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJob), err
}
