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
	v1alpha1 "github.com/marun/fnord/pkg/apis/federation/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedSecretOverrideses implements FederatedSecretOverridesInterface
type FakeFederatedSecretOverrideses struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federatedsecretoverridesesResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatedsecretoverrideses"}

var federatedsecretoverridesesKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedSecretOverrides"}

// Get takes name of the federatedSecretOverrides, and returns the corresponding federatedSecretOverrides object, and an error if there is any.
func (c *FakeFederatedSecretOverrideses) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedSecretOverrides, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedsecretoverridesesResource, c.ns, name), &v1alpha1.FederatedSecretOverrides{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretOverrides), err
}

// List takes label and field selectors, and returns the list of FederatedSecretOverrideses that match those selectors.
func (c *FakeFederatedSecretOverrideses) List(opts v1.ListOptions) (result *v1alpha1.FederatedSecretOverridesList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedsecretoverridesesResource, federatedsecretoverridesesKind, c.ns, opts), &v1alpha1.FederatedSecretOverridesList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedSecretOverridesList{}
	for _, item := range obj.(*v1alpha1.FederatedSecretOverridesList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedSecretOverrideses.
func (c *FakeFederatedSecretOverrideses) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedsecretoverridesesResource, c.ns, opts))

}

// Create takes the representation of a federatedSecretOverrides and creates it.  Returns the server's representation of the federatedSecretOverrides, and an error, if there is any.
func (c *FakeFederatedSecretOverrideses) Create(federatedSecretOverrides *v1alpha1.FederatedSecretOverrides) (result *v1alpha1.FederatedSecretOverrides, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedsecretoverridesesResource, c.ns, federatedSecretOverrides), &v1alpha1.FederatedSecretOverrides{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretOverrides), err
}

// Update takes the representation of a federatedSecretOverrides and updates it. Returns the server's representation of the federatedSecretOverrides, and an error, if there is any.
func (c *FakeFederatedSecretOverrideses) Update(federatedSecretOverrides *v1alpha1.FederatedSecretOverrides) (result *v1alpha1.FederatedSecretOverrides, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedsecretoverridesesResource, c.ns, federatedSecretOverrides), &v1alpha1.FederatedSecretOverrides{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretOverrides), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedSecretOverrideses) UpdateStatus(federatedSecretOverrides *v1alpha1.FederatedSecretOverrides) (*v1alpha1.FederatedSecretOverrides, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedsecretoverridesesResource, "status", c.ns, federatedSecretOverrides), &v1alpha1.FederatedSecretOverrides{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretOverrides), err
}

// Delete takes name of the federatedSecretOverrides and deletes it. Returns an error if one occurs.
func (c *FakeFederatedSecretOverrideses) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedsecretoverridesesResource, c.ns, name), &v1alpha1.FederatedSecretOverrides{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedSecretOverrideses) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedsecretoverridesesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedSecretOverridesList{})
	return err
}

// Patch applies the patch and returns the patched federatedSecretOverrides.
func (c *FakeFederatedSecretOverrideses) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedSecretOverrides, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedsecretoverridesesResource, c.ns, name, data, subresources...), &v1alpha1.FederatedSecretOverrides{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecretOverrides), err
}
