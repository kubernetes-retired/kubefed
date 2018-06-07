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
	multiclusterdns "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeMultiClusterServiceDNSRecords implements MultiClusterServiceDNSRecordInterface
type FakeMultiClusterServiceDNSRecords struct {
	Fake *FakeMulticlusterdns
	ns   string
}

var multiclusterservicednsrecordsResource = schema.GroupVersionResource{Group: "multiclusterdns.k8s.io", Version: "", Resource: "multiclusterservicednsrecords"}

var multiclusterservicednsrecordsKind = schema.GroupVersionKind{Group: "multiclusterdns.k8s.io", Version: "", Kind: "MultiClusterServiceDNSRecord"}

// Get takes name of the multiClusterServiceDNSRecord, and returns the corresponding multiClusterServiceDNSRecord object, and an error if there is any.
func (c *FakeMultiClusterServiceDNSRecords) Get(name string, options v1.GetOptions) (result *multiclusterdns.MultiClusterServiceDNSRecord, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(multiclusterservicednsrecordsResource, c.ns, name), &multiclusterdns.MultiClusterServiceDNSRecord{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterdns.MultiClusterServiceDNSRecord), err
}

// List takes label and field selectors, and returns the list of MultiClusterServiceDNSRecords that match those selectors.
func (c *FakeMultiClusterServiceDNSRecords) List(opts v1.ListOptions) (result *multiclusterdns.MultiClusterServiceDNSRecordList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(multiclusterservicednsrecordsResource, multiclusterservicednsrecordsKind, c.ns, opts), &multiclusterdns.MultiClusterServiceDNSRecordList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &multiclusterdns.MultiClusterServiceDNSRecordList{}
	for _, item := range obj.(*multiclusterdns.MultiClusterServiceDNSRecordList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested multiClusterServiceDNSRecords.
func (c *FakeMultiClusterServiceDNSRecords) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(multiclusterservicednsrecordsResource, c.ns, opts))

}

// Create takes the representation of a multiClusterServiceDNSRecord and creates it.  Returns the server's representation of the multiClusterServiceDNSRecord, and an error, if there is any.
func (c *FakeMultiClusterServiceDNSRecords) Create(multiClusterServiceDNSRecord *multiclusterdns.MultiClusterServiceDNSRecord) (result *multiclusterdns.MultiClusterServiceDNSRecord, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(multiclusterservicednsrecordsResource, c.ns, multiClusterServiceDNSRecord), &multiclusterdns.MultiClusterServiceDNSRecord{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterdns.MultiClusterServiceDNSRecord), err
}

// Update takes the representation of a multiClusterServiceDNSRecord and updates it. Returns the server's representation of the multiClusterServiceDNSRecord, and an error, if there is any.
func (c *FakeMultiClusterServiceDNSRecords) Update(multiClusterServiceDNSRecord *multiclusterdns.MultiClusterServiceDNSRecord) (result *multiclusterdns.MultiClusterServiceDNSRecord, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(multiclusterservicednsrecordsResource, c.ns, multiClusterServiceDNSRecord), &multiclusterdns.MultiClusterServiceDNSRecord{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterdns.MultiClusterServiceDNSRecord), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeMultiClusterServiceDNSRecords) UpdateStatus(multiClusterServiceDNSRecord *multiclusterdns.MultiClusterServiceDNSRecord) (*multiclusterdns.MultiClusterServiceDNSRecord, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(multiclusterservicednsrecordsResource, "status", c.ns, multiClusterServiceDNSRecord), &multiclusterdns.MultiClusterServiceDNSRecord{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterdns.MultiClusterServiceDNSRecord), err
}

// Delete takes name of the multiClusterServiceDNSRecord and deletes it. Returns an error if one occurs.
func (c *FakeMultiClusterServiceDNSRecords) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(multiclusterservicednsrecordsResource, c.ns, name), &multiclusterdns.MultiClusterServiceDNSRecord{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeMultiClusterServiceDNSRecords) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(multiclusterservicednsrecordsResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &multiclusterdns.MultiClusterServiceDNSRecordList{})
	return err
}

// Patch applies the patch and returns the patched multiClusterServiceDNSRecord.
func (c *FakeMultiClusterServiceDNSRecords) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *multiclusterdns.MultiClusterServiceDNSRecord, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(multiclusterservicednsrecordsResource, c.ns, name, data, subresources...), &multiclusterdns.MultiClusterServiceDNSRecord{})

	if obj == nil {
		return nil, err
	}
	return obj.(*multiclusterdns.MultiClusterServiceDNSRecord), err
}
