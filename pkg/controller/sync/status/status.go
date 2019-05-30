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

package status

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type PropagationStatus string

type AggregateStatus string

type AggregateReason string

type ConditionType string

const (
	ClusterPropagationOK PropagationStatus = ""
	WaitingForRemoval    PropagationStatus = "WaitingForRemoval"

	// Cluster-specific errors
	ClusterNotReady        PropagationStatus = "ClusterNotReady"
	CachedRetrievalFailed  PropagationStatus = "CachedRetrievalFailed"
	ComputeResourceFailed  PropagationStatus = "ComputeResourceFailed"
	ApplyOverridesFailed   PropagationStatus = "ApplyOverridesFailed"
	CreationFailed         PropagationStatus = "CreationFailed"
	UpdateFailed           PropagationStatus = "UpdateFailed"
	DeletionFailed         PropagationStatus = "DeletionFailed"
	LabelRemovalFailed     PropagationStatus = "LabelRemovalFailed"
	RetrievalFailed        PropagationStatus = "RetrievalFailed"
	AlreadyExists          PropagationStatus = "AlreadyExists"
	FieldRetentionFailed   PropagationStatus = "FieldRetentionFailed"
	VersionRetrievalFailed PropagationStatus = "VersionRetrievalFailed"
	ClientRetrievalFailed  PropagationStatus = "ClientRetrievalFailed"

	// Operation timeout errors
	CreationTimedOut     PropagationStatus = "CreationTimedOut"
	UpdateTimedOut       PropagationStatus = "UpdateTimedOut"
	DeletionTimedOut     PropagationStatus = "DeletionTimedOut"
	LabelRemovalTimedOut PropagationStatus = "LabelRemovalTimedOut"

	AggregateSuccess       AggregateReason = ""
	ClusterRetrievalFailed AggregateReason = "ClusterRetrievalFailed"
	ComputePlacementFailed AggregateReason = "ComputePlacementFailed"
	CheckClusters          AggregateReason = "CheckClusters"
	NamespaceNotFederated  AggregateReason = "NamespaceNotFederated"

	PropagationConditionType ConditionType = "Propagation"
)

type GenericClusterStatus struct {
	Name   string            `json:"name"`
	Status PropagationStatus `json:"status,omitempty"`
}

type GenericCondition struct {
	// Type of cluster condition
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status apiv1.ConditionStatus `json:"status"`
	// Last time the condition was checked.
	// +optional
	LastProbeTime string `json:"lastProbeTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason AggregateReason `json:"reason,omitempty"`
}

type GenericPropagationStatus struct {
	Conditions []*GenericCondition    `json:"conditions,omitempty"`
	Clusters   []GenericClusterStatus `json:"clusters,omitempty"`
}

type GenericFederatedStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status *GenericPropagationStatus `json:"status,omitempty"`
}

type PropagationStatusMap map[string]PropagationStatus

// SetPropagationStatus sets the conditions and clusters fields of the
// federated resource's object map from the provided reason and
// cluster status map.
func SetPropagationStatus(fedObject *unstructured.Unstructured, reason AggregateReason, statusMap PropagationStatusMap) error {
	status := &GenericFederatedStatus{}
	err := util.UnstructuredToInterface(fedObject, status)
	if err != nil {
		return errors.Wrapf(err, "Failed to unmarshall to generic status")
	}
	if status.Status == nil {
		status.Status = &GenericPropagationStatus{}
	}
	propStatus := status.Status

	// Identify whether one or more clusters could not be reconciled
	// successfully.
	if reason == AggregateSuccess && statusMap != nil {
		for _, value := range statusMap {
			if value != ClusterPropagationOK {
				reason = CheckClusters
				break
			}
		}
	}
	propStatus.setPropagationCondition(reason)
	propStatus.setClusterStatus(statusMap)

	statusJSON, err := json.Marshal(status)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshall generic status to json")
	}
	statusObj := &unstructured.Unstructured{}
	err = statusObj.UnmarshalJSON(statusJSON)
	if err != nil {
		return errors.Wrapf(err, "Failed to marshall generic status json to unstructured")
	}
	fedObject.Object[util.StatusField] = statusObj.Object[util.StatusField]

	return nil
}

// setPropagationCondition ensures that the Propagation condition is
// updated to reflect the given reason.  The type of the condition is
// derived from the reason (empty -> True, not empty -> False).
func (s *GenericPropagationStatus) setPropagationCondition(reason AggregateReason) {
	// Determine the appropriate status from the reason.
	var newStatus apiv1.ConditionStatus
	if reason == AggregateSuccess {
		newStatus = apiv1.ConditionTrue
	} else {
		newStatus = apiv1.ConditionFalse
	}

	if s.Conditions == nil {
		s.Conditions = []*GenericCondition{}
	}
	var propCondition *GenericCondition
	for _, condition := range s.Conditions {
		if condition.Type == PropagationConditionType {
			propCondition = condition
			break
		}
	}

	newCondition := propCondition == nil
	if newCondition {
		propCondition = &GenericCondition{
			Type: PropagationConditionType,
		}
		s.Conditions = append(s.Conditions, propCondition)
	}

	// Determine whether the latest status represents a change from
	// the old that requires updating the transition time.
	transition := newCondition || propCondition.Status != newStatus
	if transition {
		propCondition.Status = newStatus
		propCondition.LastTransitionTime = time.Now().UTC().Format(time.RFC3339)

	}

	propCondition.Reason = reason
	propCondition.LastProbeTime = time.Now().UTC().Format(time.RFC3339)

}

// setClusterStatus sets the cluster status slice from a propagation
// status map.
func (s *GenericPropagationStatus) setClusterStatus(statusMap PropagationStatusMap) {
	s.Clusters = []GenericClusterStatus{}
	for clusterName, status := range statusMap {
		s.Clusters = append(s.Clusters, GenericClusterStatus{
			Name:   clusterName,
			Status: status,
		})
	}
}
