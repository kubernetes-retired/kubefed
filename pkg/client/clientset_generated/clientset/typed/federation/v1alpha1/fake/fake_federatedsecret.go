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

// FakeFederatedSecrets implements FederatedSecretInterface
type FakeFederatedSecrets struct {
	Fake *FakeFederationV1alpha1
	ns   string
}

var federatedsecretsResource = schema.GroupVersionResource{Group: "federation.k8s.io", Version: "v1alpha1", Resource: "federatedsecrets"}

var federatedsecretsKind = schema.GroupVersionKind{Group: "federation.k8s.io", Version: "v1alpha1", Kind: "FederatedSecret"}

// Get takes name of the federatedSecret, and returns the corresponding federatedSecret object, and an error if there is any.
func (c *FakeFederatedSecrets) Get(name string, options v1.GetOptions) (result *v1alpha1.FederatedSecret, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(federatedsecretsResource, c.ns, name), &v1alpha1.FederatedSecret{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecret), err
}

// List takes label and field selectors, and returns the list of FederatedSecrets that match those selectors.
func (c *FakeFederatedSecrets) List(opts v1.ListOptions) (result *v1alpha1.FederatedSecretList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(federatedsecretsResource, federatedsecretsKind, c.ns, opts), &v1alpha1.FederatedSecretList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.FederatedSecretList{}
	for _, item := range obj.(*v1alpha1.FederatedSecretList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested federatedSecrets.
func (c *FakeFederatedSecrets) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(federatedsecretsResource, c.ns, opts))

}

// Create takes the representation of a federatedSecret and creates it.  Returns the server's representation of the federatedSecret, and an error, if there is any.
func (c *FakeFederatedSecrets) Create(federatedSecret *v1alpha1.FederatedSecret) (result *v1alpha1.FederatedSecret, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(federatedsecretsResource, c.ns, federatedSecret), &v1alpha1.FederatedSecret{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecret), err
}

// Update takes the representation of a federatedSecret and updates it. Returns the server's representation of the federatedSecret, and an error, if there is any.
func (c *FakeFederatedSecrets) Update(federatedSecret *v1alpha1.FederatedSecret) (result *v1alpha1.FederatedSecret, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(federatedsecretsResource, c.ns, federatedSecret), &v1alpha1.FederatedSecret{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecret), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeFederatedSecrets) UpdateStatus(federatedSecret *v1alpha1.FederatedSecret) (*v1alpha1.FederatedSecret, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(federatedsecretsResource, "status", c.ns, federatedSecret), &v1alpha1.FederatedSecret{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecret), err
}

// Delete takes name of the federatedSecret and deletes it. Returns an error if one occurs.
func (c *FakeFederatedSecrets) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(federatedsecretsResource, c.ns, name), &v1alpha1.FederatedSecret{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeFederatedSecrets) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(federatedsecretsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.FederatedSecretList{})
	return err
}

// Patch applies the patch and returns the patched federatedSecret.
func (c *FakeFederatedSecrets) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.FederatedSecret, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(federatedsecretsResource, c.ns, name, data, subresources...), &v1alpha1.FederatedSecret{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.FederatedSecret), err
}
