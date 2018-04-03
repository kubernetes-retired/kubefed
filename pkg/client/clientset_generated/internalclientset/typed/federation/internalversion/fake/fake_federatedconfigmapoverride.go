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

// FakeFederatedConfigMapOverrides implements FederatedConfigMapOverrideInterface
type FakeFederatedConfigMapOverrides struct {
	Fake *FakeFederation
	ns   string
}

var federatedconfigmapoverridesResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "", Resource: "federatedconfigmapoverrides"}

var federatedconfigmapoverridesKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "", Kind: "FederatedConfigMapOverride"}

// Get takes name of the federatedConfigMapOverride, and returns the corresponding federatedConfigMapOverride object, and an error if there is any.
func (c *FakeFederatedConfigMapOverrides) Get(name string, options v1.GetOptions) (result *federation.FederatedConfigMapOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedconfigmapoverridesResource, c.ns, name), &federation.FederatedConfigMapOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedConfigMapOverride), err
}

// List takes label and field selectors, and returns the list of FederatedConfigMapOverrides that match those selectors.
func (c *FakeFederatedConfigMapOverrides) List(opts v1.ListOptions) (result *federation.FederatedConfigMapOverrideList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedconfigmapoverridesResource, federatedconfigmapoverridesKind, c.ns, opts), &federation.FederatedConfigMapOverrideList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &federation.FederatedConfigMapOverrideList{}
	for _, item := range obj.(*federation.FederatedConfigMapOverrideList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedConfigMapOverrides.
func (c *FakeFederatedConfigMapOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedconfigmapoverridesResource, c.ns, opts))

}

// Create takes the representation of a federatedConfigMapOverride and creates it.  Returns the server's representation of the federatedConfigMapOverride, and an error, if there is any.
func (c *FakeFederatedConfigMapOverrides) Create(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (result *federation.FederatedConfigMapOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedconfigmapoverridesResource, c.ns, federatedConfigMapOverride), &federation.FederatedConfigMapOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedConfigMapOverride), err
}

// Update takes the representation of a federatedConfigMapOverride and updates it. Returns the server's representation of the federatedConfigMapOverride, and an error, if there is any.
func (c *FakeFederatedConfigMapOverrides) Update(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (result *federation.FederatedConfigMapOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedconfigmapoverridesResource, c.ns, federatedConfigMapOverride), &federation.FederatedConfigMapOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedConfigMapOverride), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedConfigMapOverrides) UpdateStatus(federatedConfigMapOverride *federation.FederatedConfigMapOverride) (*federation.FederatedConfigMapOverride, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedconfigmapoverridesResource, "status", c.ns, federatedConfigMapOverride), &federation.FederatedConfigMapOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedConfigMapOverride), err
}

// Delete takes name of the federatedConfigMapOverride and deletes it. Returns an error if one occurs.
func (c *FakeFederatedConfigMapOverrides) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedconfigmapoverridesResource, c.ns, name), &federation.FederatedConfigMapOverride{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedConfigMapOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedconfigmapoverridesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &federation.FederatedConfigMapOverrideList{})
	return err
}

// Patch applies the patch and returns the patched federatedConfigMapOverride.
func (c *FakeFederatedConfigMapOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *federation.FederatedConfigMapOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedconfigmapoverridesResource, c.ns, name, data, subresources...), &federation.FederatedConfigMapOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*federation.FederatedConfigMapOverride), err
}
