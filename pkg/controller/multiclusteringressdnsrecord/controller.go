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

package multiclusteringressdnsrecord

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/client-go/tools/record"

	multiclusterdnsv1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/multiclusterdns/v1alpha1"
	multiclusterdnsv1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/multiclusterdns/v1alpha1"
	multiclusterdnsv1alpha1informer "github.com/kubernetes-sigs/federation-v2/pkg/client/informers/externalversions/multiclusterdns/v1alpha1"
	multiclusterdnsv1alpha1lister "github.com/kubernetes-sigs/federation-v2/pkg/client/listers/multiclusterdns/v1alpha1"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for MultiClusterIngressDNSRecord resources goes here.

func (bc *MultiClusterIngressDNSRecordController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on multiclusteringressdnsrecord.MultiClusterIngressDNSRecordController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:controller:group=multiclusterdns,version=v1alpha1,kind=MultiClusterIngressDNSRecord,resource=multiclusteringressdnsrecords
type MultiClusterIngressDNSRecordController struct {
	// INSERT ADDITIONAL FIELDS HERE
	multiclusteringressdnsrecordLister multiclusterdnsv1alpha1lister.MultiClusterIngressDNSRecordLister
	multiclusteringressdnsrecordclient multiclusterdnsv1alpha1client.MulticlusterdnsV1alpha1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	multiclusteringressdnsrecordrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &MultiClusterIngressDNSRecordController{
		multiclusteringressdnsrecordLister: arguments.ControllerManager.GetInformerProvider(&multiclusterdnsv1alpha1.MultiClusterIngressDNSRecord{}).(multiclusterdnsv1alpha1informer.MultiClusterIngressDNSRecordInformer).Lister(),

		multiclusteringressdnsrecordclient:   arguments.Clientset.MulticlusterdnsV1alpha1(),
		multiclusteringressdnsrecordrecorder: arguments.CreateRecorder("MultiClusterIngressDNSRecordController"),
	}

	// Create a new controller that will call MultiClusterIngressDNSRecordController.Reconcile on changes to MultiClusterIngressDNSRecords
	gc := &controller.GenericController{
		Name:             "MultiClusterIngressDNSRecordController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&multiclusterdnsv1alpha1.MultiClusterIngressDNSRecord{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a MultiClusterIngressDNSRecord Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the MultiClusterIngressDNSRecordController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	return gc, nil
}
