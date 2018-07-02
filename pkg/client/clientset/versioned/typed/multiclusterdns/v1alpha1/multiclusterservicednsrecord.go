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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	scheme "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MultiClusterServiceDNSRecordsGetter has a method to return a MultiClusterServiceDNSRecordInterface.
// A group's client should implement this interface.
type MultiClusterServiceDNSRecordsGetter interface {
	MultiClusterServiceDNSRecords(namespace string) MultiClusterServiceDNSRecordInterface
}

// MultiClusterServiceDNSRecordInterface has methods to work with MultiClusterServiceDNSRecord resources.
type MultiClusterServiceDNSRecordInterface interface {
	Create(*v1alpha1.MultiClusterServiceDNSRecord) (*v1alpha1.MultiClusterServiceDNSRecord, error)
	Update(*v1alpha1.MultiClusterServiceDNSRecord) (*v1alpha1.MultiClusterServiceDNSRecord, error)
	UpdateStatus(*v1alpha1.MultiClusterServiceDNSRecord) (*v1alpha1.MultiClusterServiceDNSRecord, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MultiClusterServiceDNSRecord, error)
	List(opts v1.ListOptions) (*v1alpha1.MultiClusterServiceDNSRecordList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MultiClusterServiceDNSRecord, err error)
	MultiClusterServiceDNSRecordExpansion
}

// multiClusterServiceDNSRecords implements MultiClusterServiceDNSRecordInterface
type multiClusterServiceDNSRecords struct {
	client rest.Interface
	ns     string
}

// newMultiClusterServiceDNSRecords returns a MultiClusterServiceDNSRecords
func newMultiClusterServiceDNSRecords(c *MulticlusterdnsV1alpha1Client, namespace string) *multiClusterServiceDNSRecords {
	return &multiClusterServiceDNSRecords{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the multiClusterServiceDNSRecord, and returns the corresponding multiClusterServiceDNSRecord object, and an error if there is any.
func (c *multiClusterServiceDNSRecords) Get(name string, options v1.GetOptions) (result *v1alpha1.MultiClusterServiceDNSRecord, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecord{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MultiClusterServiceDNSRecords that match those selectors.
func (c *multiClusterServiceDNSRecords) List(opts v1.ListOptions) (result *v1alpha1.MultiClusterServiceDNSRecordList, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecordList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested multiClusterServiceDNSRecords.
func (c *multiClusterServiceDNSRecords) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a multiClusterServiceDNSRecord and creates it.  Returns the server's representation of the multiClusterServiceDNSRecord, and an error, if there is any.
func (c *multiClusterServiceDNSRecords) Create(multiClusterServiceDNSRecord *v1alpha1.MultiClusterServiceDNSRecord) (result *v1alpha1.MultiClusterServiceDNSRecord, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecord{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		Body(multiClusterServiceDNSRecord).
		Do().
		Into(result)
	return
}

// Update takes the representation of a multiClusterServiceDNSRecord and updates it. Returns the server's representation of the multiClusterServiceDNSRecord, and an error, if there is any.
func (c *multiClusterServiceDNSRecords) Update(multiClusterServiceDNSRecord *v1alpha1.MultiClusterServiceDNSRecord) (result *v1alpha1.MultiClusterServiceDNSRecord, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecord{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		Name(multiClusterServiceDNSRecord.Name).
		Body(multiClusterServiceDNSRecord).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *multiClusterServiceDNSRecords) UpdateStatus(multiClusterServiceDNSRecord *v1alpha1.MultiClusterServiceDNSRecord) (result *v1alpha1.MultiClusterServiceDNSRecord, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecord{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		Name(multiClusterServiceDNSRecord.Name).
		SubResource("status").
		Body(multiClusterServiceDNSRecord).
		Do().
		Into(result)
	return
}

// Delete takes name of the multiClusterServiceDNSRecord and deletes it. Returns an error if one occurs.
func (c *multiClusterServiceDNSRecords) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *multiClusterServiceDNSRecords) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched multiClusterServiceDNSRecord.
func (c *multiClusterServiceDNSRecords) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MultiClusterServiceDNSRecord, err error) {
	result = &v1alpha1.MultiClusterServiceDNSRecord{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("multiclusterservicednsrecords").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
