
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


package replicaplacementpreferences

import (
	"log"

	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"

	"github.com/marun/federation-v2/pkg/apis/federatedscheduling/v1alpha1"
	"github.com/marun/federation-v2/pkg/controller/sharedinformers"
	listers "github.com/marun/federation-v2/pkg/client/listers_generated/federatedscheduling/v1alpha1"
)

// +controller:group=federatedscheduling,version=v1alpha1,kind=ReplicaPlacementPreferences,resource=replicaplacementpreferences
type ReplicaPlacementPreferencesControllerImpl struct {
	builders.DefaultControllerFns

	// lister indexes properties about ReplicaPlacementPreferences
	lister listers.ReplicaPlacementPreferencesLister
}

// Init initializes the controller and is called by the generated code
// Register watches for additional resource types here.
func (c *ReplicaPlacementPreferencesControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
	// Use the lister for indexing replicaplacementpreferences labels
	c.lister = arguments.GetSharedInformers().Factory.Federatedscheduling().V1alpha1().ReplicaPlacementPreferences().Lister()
}

// Reconcile handles enqueued messages
func (c *ReplicaPlacementPreferencesControllerImpl) Reconcile(u *v1alpha1.ReplicaPlacementPreferences) error {
	// Implement controller logic here
	log.Printf("Running reconcile ReplicaPlacementPreferences for %s\n", u.Name)
	return nil
}

func (c *ReplicaPlacementPreferencesControllerImpl) Get(namespace, name string) (*v1alpha1.ReplicaPlacementPreferences, error) {
	return c.lister.ReplicaPlacementPreferences(namespace).Get(name)
}
