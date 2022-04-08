/*
Copyright 2019 The Kubernetes Authors.

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

package dispatch

import (
	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

// RetainClusterFields updates the desired object with values retained
// from the cluster object.
func RetainClusterFields(targetKind string, desiredObj, clusterObj, fedObj *unstructured.Unstructured) error {
	// Pass the same ResourceVersion as in the cluster object for update operation, otherwise operation will fail.
	desiredObj.SetResourceVersion(clusterObj.GetResourceVersion())

	// Retain finalizers and annotations since they will typically be set by
	// controllers in a member cluster.  It is still possible to set the fields
	// via overrides.
	desiredObj.SetFinalizers(clusterObj.GetFinalizers())
	desiredObj.SetAnnotations(clusterObj.GetAnnotations())

	if targetKind == util.ServiceKind {
		return retainServiceFields(desiredObj, clusterObj)
	}
	if targetKind == util.ServiceAccountKind {
		return retainServiceAccountFields(desiredObj, clusterObj)
	}
	return retainReplicas(desiredObj, clusterObj, fedObj)
}

func retainServiceFields(desiredObj, clusterObj *unstructured.Unstructured) error {
	// healthCheckNodePort is allocated by APIServer and unchangeable, so it should be retained while updating
	healthCheckNodePort, ok, err := unstructured.NestedInt64(clusterObj.Object, util.SpecField, util.HealthCheckNodePortField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving healthCheckNodePort from service")
	}
	if ok && healthCheckNodePort > 0 {
		if err = unstructured.SetNestedField(desiredObj.Object, healthCheckNodePort, util.SpecField, util.HealthCheckNodePortField); err != nil {
			return errors.Wrap(err, "Error setting healthCheckNodePort for service")
		}
	}

	// ClusterIP and NodePort are allocated to Service by cluster, so retain the same if any while updating

	// Retain clusterip and clusterips
	clusterIP, ok, err := unstructured.NestedString(clusterObj.Object, util.SpecField, util.ClusterIPField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving clusterIP from cluster service")
	}
	// !ok could indicate that a cluster ip was not assigned
	if ok && clusterIP != "" {
		err := unstructured.SetNestedField(desiredObj.Object, clusterIP, util.SpecField, util.ClusterIPField)
		if err != nil {
			return errors.Wrap(err, "Error setting clusterIP for service")
		}
	}
	clusterIPs, ok, err := unstructured.NestedStringSlice(clusterObj.Object, util.SpecField, util.ClusterIPsField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving clusterIPs from cluster service")
	}
	// !ok could indicate that cluster ips was not assigned
	if ok && len(clusterIPs) > 0 {
		err := unstructured.SetNestedStringSlice(desiredObj.Object, clusterIPs, util.SpecField, util.ClusterIPsField)
		if err != nil {
			return errors.Wrap(err, "Error setting clusterIPs for service")
		}
	}

	// Retain nodeports
	clusterPorts, ok, err := unstructured.NestedSlice(clusterObj.Object, util.SpecField, util.PortsField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving ports from cluster service")
	}
	if !ok {
		return nil
	}
	var desiredPorts []interface{}
	desiredPorts, ok, err = unstructured.NestedSlice(desiredObj.Object, util.SpecField, util.PortsField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving ports from service")
	}
	if !ok {
		desiredPorts = []interface{}{}
	}
	for desiredIndex := range desiredPorts {
		for clusterIndex := range clusterPorts {
			fPort := desiredPorts[desiredIndex].(map[string]interface{})
			cPort := clusterPorts[clusterIndex].(map[string]interface{})
			if !(fPort["name"] == cPort["name"] && fPort["protocol"] == cPort["protocol"] && fPort["port"] == cPort["port"]) {
				continue
			}
			nodePort, ok := cPort["nodePort"]
			if ok {
				fPort["nodePort"] = nodePort
			}
		}
	}
	err = unstructured.SetNestedSlice(desiredObj.Object, desiredPorts, util.SpecField, util.PortsField)
	if err != nil {
		return errors.Wrap(err, "Error setting ports for service")
	}

	return nil
}

// retainServiceAccountFields retains the 'secrets' field of a service account
// if the desired representation does not include a value for the field.  This
// ensures that the sync controller doesn't continually clear a generated
// secret from a service account, prompting continual regeneration by the
// service account controller in the member cluster.
//
// TODO(marun) Clearing a manually-set secrets field will require resetting
// placement.  Is there a better way to do this?
func retainServiceAccountFields(desiredObj, clusterObj *unstructured.Unstructured) error {
	// Check whether the secrets field is populated in the desired object.
	desiredSecrets, ok, err := unstructured.NestedSlice(desiredObj.Object, util.SecretsField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving secrets from desired service account")
	}
	if ok && len(desiredSecrets) > 0 {
		// Field is populated, so an update to the target resource does not
		// risk triggering a race with the service account controller.
		return nil
	}

	// Retrieve the secrets from the cluster object and retain them.
	secrets, ok, err := unstructured.NestedSlice(clusterObj.Object, util.SecretsField)
	if err != nil {
		return errors.Wrap(err, "Error retrieving secrets from service account")
	}
	if ok && len(secrets) > 0 {
		err := unstructured.SetNestedField(desiredObj.Object, secrets, util.SecretsField)
		if err != nil {
			return errors.Wrap(err, "Error setting secrets for service account")
		}
	}
	return nil
}

func retainReplicas(desiredObj, clusterObj, fedObj *unstructured.Unstructured) error {
	// Retain the replicas field if the federated object has been
	// configured to do so.  If the replicas field is intended to be
	// set by the in-cluster HPA controller, not retaining it will
	// thrash the scheduler.
	retainReplicas, ok, err := unstructured.NestedBool(fedObj.Object, util.SpecField, util.RetainReplicasField)
	if err != nil {
		return err
	}
	if ok && retainReplicas {
		replicas, ok, err := unstructured.NestedInt64(clusterObj.Object, util.SpecField, util.ReplicasField)
		if err != nil {
			return err
		}
		if ok {
			err := unstructured.SetNestedField(desiredObj.Object, replicas, util.SpecField, util.ReplicasField)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
