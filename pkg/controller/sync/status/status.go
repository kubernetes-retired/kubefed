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
	"reflect"
	"time"

	"github.com/pkg/errors"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

type PropagationStatus string

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
	ManagedLabelFalse      PropagationStatus = "ManagedLabelFalse"

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
	Name         string            `json:"name"`
	Status       PropagationStatus `json:"status,omitempty"`
	RemoteStatus interface{}       `json:"remoteStatus,omitempty"`
}

type GenericCondition struct {
	// Type of cluster condition
	Type ConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status apiv1.ConditionStatus `json:"status"`
	// Last time reconciliation resulted in an error or the last time a
	// change was propagated to member clusters.
	// +optional
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason AggregateReason `json:"reason,omitempty"`
}

type GenericFederatedStatus struct {
	ObservedGeneration int64                  `json:"observedGeneration,omitempty"`
	Conditions         []*GenericCondition    `json:"conditions,omitempty"`
	Clusters           []GenericClusterStatus `json:"clusters,omitempty"`
}

type GenericFederatedResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status *GenericFederatedStatus `json:"status,omitempty"`
}

type PropagationStatusMap map[string]PropagationStatus

type CollectedPropagationStatus struct {
	StatusMap        PropagationStatusMap
	ResourcesUpdated bool
}

type CollectedResourceStatus struct {
	StatusMap        map[string]interface{}
	ResourcesUpdated bool
}

// SetFederatedStatus sets the conditions and clusters fields of the
// federated resource's object map. Returns a boolean indication of
// whether status should be written to the API.
func SetFederatedStatus(fedObject *unstructured.Unstructured, reason AggregateReason, collectedStatus CollectedPropagationStatus, collectedResourceStatus CollectedResourceStatus, resourceStatusCollection bool) (bool, error) {
	resource := &GenericFederatedResource{}
	err := util.UnstructuredToInterface(fedObject, resource)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to unmarshall to generic resource")
	}
	if resource.Status == nil {
		resource.Status = &GenericFederatedStatus{}
	}

	changed := resource.Status.update(fedObject.GetGeneration(), reason, collectedStatus, collectedResourceStatus, resourceStatusCollection)
	if !changed {
		return false, nil
	}

	resourceJSON, err := json.Marshal(resource)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to marshall generic status to json")
	}
	resourceObj := &unstructured.Unstructured{}
	err = resourceObj.UnmarshalJSON(resourceJSON)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to marshall generic resource json to unstructured")
	}

	klog.V(4).Infof("Setting the status of federated object '%v' and resource object '%v'", fedObject.GetName(), resourceObj.GetName())
	fedObject.Object[util.StatusField] = resourceObj.Object[util.StatusField]

	return true, nil
}

// update ensures that the status reflects the given generation, reason
// and collected status. Returns a boolean indication of whether the
// status has been changed.
func (s *GenericFederatedStatus) update(generation int64, reason AggregateReason,
	collectedStatus CollectedPropagationStatus, collectedResourceStatus CollectedResourceStatus, resourceStatusCollection bool) bool {
	generationUpdated := s.ObservedGeneration != generation
	if generationUpdated {
		s.ObservedGeneration = generation
	}

	// Identify whether one or more clusters could not be reconciled
	// successfully.
	if reason == AggregateSuccess {
		for cluster, value := range collectedStatus.StatusMap {
			rawStatus := collectedResourceStatus.StatusMap[cluster]
			if value != ClusterPropagationOK || (resourceStatusCollection && rawStatus == nil) {
				klog.V(4).Infof("Check the cluster '%v' with resource status '%v' and propStatus '%v' whose resource status collection is: '%v'", cluster, rawStatus, value, resourceStatusCollection)
				reason = CheckClusters
				break
			}
		}
	}

	clustersChanged := s.setClusters(collectedStatus.StatusMap, collectedResourceStatus.StatusMap, resourceStatusCollection)

	// Indicate that changes were propagated if either status.clusters
	// was changed or if existing resources were updated (which could
	// occur even if status.clusters was unchanged).
	// TODO (hectorj2f): re-consider this new condition or add a new one for the resource status update or not.
	changesPropagated := clustersChanged || len(collectedStatus.StatusMap) > 0 && len(collectedResourceStatus.StatusMap) > 0 && collectedStatus.ResourcesUpdated

	propStatusUpdated := s.setPropagationCondition(reason, changesPropagated)

	statusUpdated := generationUpdated || propStatusUpdated

	klog.V(4).Infof("Value of flags: propStatusUpdated: '%v'; statusUpdated '%v'; changesPropagated '%v'", propStatusUpdated, statusUpdated, changesPropagated)
	return statusUpdated
}

// setClusters sets the status.clusters slice from propagation and resource status
// maps. Returns a boolean indication of whether the status.clusters was
// modified.
func (s *GenericFederatedStatus) setClusters(statusMap PropagationStatusMap, resourceStatusMap map[string]interface{}, resourceStatusCollection bool) bool {
	if !s.clustersDiffer(statusMap, resourceStatusMap, resourceStatusCollection) {
		return false
	}
	s.Clusters = []GenericClusterStatus{}
	for clusterName, status := range statusMap {
		rawResourceStatus := resourceStatusMap[clusterName]
		s.Clusters = append(s.Clusters, GenericClusterStatus{
			Name:         clusterName,
			Status:       status,
			RemoteStatus: rawResourceStatus,
		})
	}
	return true
}

// clustersDiffer checks whether `status.clusters` differs from the
// given status map.
func (s *GenericFederatedStatus) clustersDiffer(statusMap PropagationStatusMap, resourceStatusMap map[string]interface{}, resourceStatusCollection bool) bool {
	if len(s.Clusters) != len(statusMap) || resourceStatusCollection && (len(s.Clusters) != len(resourceStatusMap)) {
		klog.V(4).Infof("Clusters differs from the size: clusters = %v, statusMap = %v, resourceStatusMap = %v", s.Clusters, statusMap, resourceStatusMap )
		return true
	}
	for _, status := range s.Clusters {
		if statusMap[status.Name] != status.Status {
			return true
		}
		if !reflect.DeepEqual(resourceStatusMap[status.Name], status.RemoteStatus) {
			klog.V(4).Infof("Clusters resource status differ: %v VS %v", resourceStatusMap[status.Name], status.RemoteStatus)
			return true
		}
	}
	return false
}

// setPropagationCondition ensures that the Propagation condition is
// updated to reflect the given reason.  The type of the condition is
// derived from the reason (empty -> True, not empty -> False).
func (s *GenericFederatedStatus) setPropagationCondition(reason AggregateReason, changesPropagated bool) bool {
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

	now := time.Now().UTC().Format(time.RFC3339)

	transition := newCondition || !(propCondition.Status == newStatus && propCondition.Reason == reason)
	if transition {
		propCondition.LastTransitionTime = now
		propCondition.Status = newStatus
		propCondition.Reason = reason
	}

	updateRequired := changesPropagated || transition
	if updateRequired {
		propCondition.LastUpdateTime = now
	}

	return updateRequired
}
