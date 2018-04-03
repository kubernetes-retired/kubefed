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
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeFederatedDeploymentOverrides implements FederatedDeploymentOverrideInterface
type FakeFederatedDeploymentOverrides struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federateddeploymentoverridesResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federateddeploymentoverrides"}

var federateddeploymentoverridesKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedDeploymentOverride"}

// Get takes name of the federatedDeploymentOverride, and returns the corresponding federatedDeploymentOverride object, and an error if there is any.
func (c *FakeFederatedDeploymentOverrides) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federateddeploymentoverridesResource, c.ns, name), &v1alpha1.FederatedDeploymentOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentOverride), err
}

// List takes label and field selectors, and returns the list of FederatedDeploymentOverrides that match those selectors.
func (c *FakeFederatedDeploymentOverrides) List(opts v1.ListOptions) (result *v1alpha1.FederatedDeploymentOverrideList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federateddeploymentoverridesResource, federateddeploymentoverridesKind, c.ns, opts), &v1alpha1.FederatedDeploymentOverrideList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedDeploymentOverrideList{}
	for _, item := range obj.(*v1alpha1.FederatedDeploymentOverrideList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedDeploymentOverrides.
func (c *FakeFederatedDeploymentOverrides) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federateddeploymentoverridesResource, c.ns, opts))

}

// Create takes the representation of a federatedDeploymentOverride and creates it.  Returns the server's representation of the federatedDeploymentOverride, and an error, if there is any.
func (c *FakeFederatedDeploymentOverrides) Create(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federateddeploymentoverridesResource, c.ns, federatedDeploymentOverride), &v1alpha1.FederatedDeploymentOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentOverride), err
}

// Update takes the representation of a federatedDeploymentOverride and updates it. Returns the server's representation of the federatedDeploymentOverride, and an error, if there is any.
func (c *FakeFederatedDeploymentOverrides) Update(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federateddeploymentoverridesResource, c.ns, federatedDeploymentOverride), &v1alpha1.FederatedDeploymentOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentOverride), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedDeploymentOverrides) UpdateStatus(federatedDeploymentOverride *v1alpha1.FederatedDeploymentOverride) (*v1alpha1.FederatedDeploymentOverride, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federateddeploymentoverridesResource, "status", c.ns, federatedDeploymentOverride), &v1alpha1.FederatedDeploymentOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentOverride), err
}

// Delete takes name of the federatedDeploymentOverride and deletes it. Returns an error if one occurs.
func (c *FakeFederatedDeploymentOverrides) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federateddeploymentoverridesResource, c.ns, name), &v1alpha1.FederatedDeploymentOverride{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedDeploymentOverrides) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federateddeploymentoverridesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedDeploymentOverrideList{})
	return err
}

// Patch applies the patch and returns the patched federatedDeploymentOverride.
func (c *FakeFederatedDeploymentOverrides) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedDeploymentOverride, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federateddeploymentoverridesResource, c.ns, name, data, subresources...), &v1alpha1.FederatedDeploymentOverride{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedDeploymentOverride), err
}
