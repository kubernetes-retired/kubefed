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

// FakeFederatedSecretOverrides implements FederatedSecretOverrideInterface
type FakeFederatedSecretOverrides struct {
	Fake *FakeFederation
	ns   string
}

var federatedsecretoverridesResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedsecretoverrides"}

var federatedsecretoverridesKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedSecretOverride"}

// Get takes name of the federatedSecretOverride, and returns the corresponding federatedSecretOverride object, and an error if there is any.
func (c *FakeFederatedSecretOverrides) Get(name string, options v1.GetOptions) (result *federation.FederatedSecretOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedsecretoverridesResource, c.ns, name), &federation.FederatedSecretOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedSecretOverride), err
}

// List takes label and field selectors, and returns the list of FederatedSecretOverrides that match those selectors.
func (c *FakeFederatedSecretOverrides) List(opts v1.ListOptions) (result *federation.FederatedSecretOverrideList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedsecretoverridesResource, federatedsecretoverridesKind, c.ns, opts), &federation.FederatedSecretOverrideList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedSecretOverrideList{}
	for _, item := range obj.(*federation.FederatedSecretOverrideList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedSecretOverrides.
func (c *FakeFederatedSecretOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedsecretoverridesResource, c.ns, opts))

}

// Create takes the representation of a federatedSecretOverride and creates it.  Returns the server's representation of the federatedSecretOverride, and an error, if there is any.
func (c *FakeFederatedSecretOverrides) Create(federatedSecretOverride *federation.FederatedSecretOverride) (result *federation.FederatedSecretOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedsecretoverridesResource, c.ns, federatedSecretOverride), &federation.FederatedSecretOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedSecretOverride), err
}

// Update takes the representation of a federatedSecretOverride and updates it. Returns the server's representation of the federatedSecretOverride, and an error, if there is any.
func (c *FakeFederatedSecretOverrides) Update(federatedSecretOverride *federation.FederatedSecretOverride) (result *federation.FederatedSecretOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedsecretoverridesResource, c.ns, federatedSecretOverride), &federation.FederatedSecretOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedSecretOverride), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedSecretOverrides) UpdateStatus(federatedSecretOverride *federation.FederatedSecretOverride) (*federation.FederatedSecretOverride, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedsecretoverridesResource, "status", c.ns, federatedSecretOverride), &federation.FederatedSecretOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedSecretOverride), err
}

// Delete takes name of the federatedSecretOverride and deletes it. Returns an error if one occurs.
func (c *FakeFederatedSecretOverrides) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedsecretoverridesResource, c.ns, name), &federation.FederatedSecretOverride{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedSecretOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedsecretoverridesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedSecretOverrideList{})
	return err
}

// Patch applies the patch and returns the patched federatedSecretOverride.
func (c *FakeFederatedSecretOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedSecretOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedsecretoverridesResource, c.ns, name, data, subresources...), &federation.FederatedSecretOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedSecretOverride), err
}
