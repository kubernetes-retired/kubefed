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

package federatedingress

import (
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	"k8s.io/client-go/tools/record"

	corev1alpha1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	corev1alpha1client "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned/typed/core/v1alpha1"
	corev1alpha1informer "github.com/kubernetes-sigs/federation-v2/pkg/client/informers/externalversions/core/v1alpha1"
	corev1alpha1lister "github.com/kubernetes-sigs/federation-v2/pkg/client/listers/core/v1alpha1"

	"github.com/kubernetes-sigs/federation-v2/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for FederatedIngress resources goes here.

func (bc *FederatedIngressController) Reconcile(k types.ReconcileKey) error {
	// INSERT YOUR CODE HERE
	log.Printf("Implement the Reconcile function on federatedingress.FederatedIngressController to reconcile %s\n", k.Name)
	return nil
}

// +kubebuilder:controller:group=core,version=v1alpha1,kind=FederatedIngress,resource=federatedingresses
type FederatedIngressController struct {
	// INSERT ADDITIONAL FIELDS HERE
	federatedingressLister corev1alpha1lister.FederatedIngressLister
	federatedingressclient corev1alpha1client.CoreV1alpha1Interface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	federatedingressrecorder record.EventRecorder
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &FederatedIngressController{
		federatedingressLister: arguments.ControllerManager.GetInformerProvider(&corev1alpha1.FederatedIngress{}).(corev1alpha1informer.FederatedIngressInformer).Lister(),

		federatedingressclient:   arguments.Clientset.CoreV1alpha1(),
		federatedingressrecorder: arguments.CreateRecorder("FederatedIngressController"),
	}

	// Create a new controller that will call FederatedIngressController.Reconcile on changes to FederatedIngresss
	gc := &controller.GenericController{
		Name:             "FederatedIngressController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&corev1alpha1.FederatedIngress{}); err != nil {
		return gc, err
	}

	// IMPORTANT:
	// To watch additional resource types - such as those created by your controller - add gc.Watch* function calls here
	// Watch function calls will transform each object event into a FederatedIngress Key to be reconciled by the controller.
	//
	// **********
	// For any new Watched types, you MUST add the appropriate // +kubebuilder:informer and // +kubebuilder:rbac
	// annotations to the FederatedIngressController and run "kubebuilder generate.
	// This will generate the code to start the informers and create the RBAC rules needed for running in a cluster.
	// See:
	// https://godoc.org/github.com/kubernetes-sigs/kubebuilder/pkg/gen/controller#example-package
	// **********

	return gc, nil
}
