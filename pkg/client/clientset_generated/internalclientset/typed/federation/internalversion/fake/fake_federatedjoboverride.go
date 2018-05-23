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
	federation "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedJobOverrides implements FederatedJobOverrideInterface
type FakeFederatedJobOverrides struct {
	Fake *FakeFederation
	ns   string
}

var federatedjoboverridesResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedjoboverrides"}

var federatedjoboverridesKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedJobOverride"}

// Get takes name of the federatedJobOverride, and returns the corresponding federatedJobOverride object, and an error if there is any.
func (c *FakeFederatedJobOverrides) Get(name string, options v1.GetOptions) (result *federation.FederatedJobOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedjoboverridesResource, c.ns, name), &federation.FederatedJobOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJobOverride), err
}

// List takes label and field selectors, and returns the list of FederatedJobOverrides that match those selectors.
func (c *FakeFederatedJobOverrides) List(opts v1.ListOptions) (result *federation.FederatedJobOverrideList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedjoboverridesResource, federatedjoboverridesKind, c.ns, opts), &federation.FederatedJobOverrideList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedJobOverrideList{}
	for _, item := range obj.(*federation.FederatedJobOverrideList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedJobOverrides.
func (c *FakeFederatedJobOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedjoboverridesResource, c.ns, opts))

}

// Create takes the representation of a federatedJobOverride and creates it.  Returns the server's representation of the federatedJobOverride, and an error, if there is any.
func (c *FakeFederatedJobOverrides) Create(federatedJobOverride *federation.FederatedJobOverride) (result *federation.FederatedJobOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedjoboverridesResource, c.ns, federatedJobOverride), &federation.FederatedJobOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJobOverride), err
}

// Update takes the representation of a federatedJobOverride and updates it. Returns the server's representation of the federatedJobOverride, and an error, if there is any.
func (c *FakeFederatedJobOverrides) Update(federatedJobOverride *federation.FederatedJobOverride) (result *federation.FederatedJobOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedjoboverridesResource, c.ns, federatedJobOverride), &federation.FederatedJobOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJobOverride), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedJobOverrides) UpdateStatus(federatedJobOverride *federation.FederatedJobOverride) (*federation.FederatedJobOverride, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedjoboverridesResource, "status", c.ns, federatedJobOverride), &federation.FederatedJobOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJobOverride), err
}

// Delete takes name of the federatedJobOverride and deletes it. Returns an error if one occurs.
func (c *FakeFederatedJobOverrides) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedjoboverridesResource, c.ns, name), &federation.FederatedJobOverride{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedJobOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedjoboverridesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedJobOverrideList{})
	return err
}

// Patch applies the patch and returns the patched federatedJobOverride.
func (c *FakeFederatedJobOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedJobOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedjoboverridesResource, c.ns, name, data, subresources...), &federation.FederatedJobOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedJobOverride), err
}
