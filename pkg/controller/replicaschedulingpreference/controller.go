
/*
Copyright 2018 The Federation v2 Authors.

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


package replicaschedulingpreference

import (
	"log"

	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/federatedscheduling/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/sharedinformers"
	listers "github.com/kubernetes-sigs/federation-v2/pkg/client/listers_generated/federatedscheduling/v1alpha1"
)

// +controller:group=federatedscheduling,version=v1alpha1,kind=ReplicaSchedulingPreference,resource=replicaschedulingpreferences
type ReplicaSchedulingPreferenceControllerImpl struct {
	builders.DefaultControllerFns

	// lister indexes properties about ReplicaSchedulingPreference
	lister listers.ReplicaSchedulingPreferenceLister
}

// Init initializes the controller and is called by the generated code
// Register watches for additional resource types here.
func (c *ReplicaSchedulingPreferenceControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
	// Use the lister for indexing replicaschedulingpreferences labels
	c.lister = arguments.GetSharedInformers().Factory.Federatedscheduling().V1alpha1().ReplicaSchedulingPreferences().Lister()
}

// Reconcile handles enqueued messages
func (c *ReplicaSchedulingPreferenceControllerImpl) Reconcile(u *v1alpha1.ReplicaSchedulingPreference) error {
	// Implement controller logic here
	log.Printf("Running reconcile ReplicaSchedulingPreference for %s\n", u.Name)
	return nil
}

func (c *ReplicaSchedulingPreferenceControllerImpl) Get(namespace, name string) (*v1alpha1.ReplicaSchedulingPreference, error) {
	return c.lister.ReplicaSchedulingPreferences(namespace).Get(name)
}
