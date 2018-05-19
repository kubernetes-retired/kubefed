
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


package federatednamespaceplacement

import (
	"log"

	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federation/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sharedinformers"
	listers "github.com/kubernetes-sigs/federation-v2/pkg/client/listers_generated/federation/v1alpha1"
)

// +controller:group=federation,version=v1alpha1,kind=FederatedNamespacePlacement,resource=federatednamespaceplacements
type FederatedNamespacePlacementControllerImpl struct {
	builders.DefaultControllerFns

	// lister indexes properties about FederatedNamespacePlacement
	lister listers.FederatedNamespacePlacementLister
}

// Init initializes the controller and is called by the generated code
// Register watches for additional resource types here.
func (c *FederatedNamespacePlacementControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
	// Use the lister for indexing federatednamespaceplacements labels
	c.lister = arguments.GetSharedInformers().Factory.Federation().V1alpha1().FederatedNamespacePlacements().Lister()
}

// Reconcile handles enqueued messages
func (c *FederatedNamespacePlacementControllerImpl) Reconcile(u *v1alpha1.FederatedNamespacePlacement) error {
	// Implement controller logic here
	log.Printf("Running reconcile FederatedNamespacePlacement for %s\n", u.Name)
	return nil
}

func (c *FederatedNamespacePlacementControllerImpl) Get(namespace, name string) (*v1alpha1.FederatedNamespacePlacement, error) {
	return c.lister.Get(name)
}
