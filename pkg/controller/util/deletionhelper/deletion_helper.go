/*
Copyright 2016 The Kubernetes Authors.

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

// Package to help federation controllers to delete federated resources from
// underlying clusters when the resource is deleted from federation control
// plane.
package deletionhelper

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	finalizersutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util/finalizers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
)

const (
	// Add this finalizer to a federation resource if the resource should be
	// deleted from all underlying clusters before being deleted from
	// federation control plane.
	// This is ignored if FinalizerOrphan is also present on the resource.
	// In that case, both finalizers are removed from the resource and the
	// resource is deleted from federation control plane without affecting
	// the underlying clusters.
	FinalizerDeleteFromUnderlyingClusters string = "federation.kubernetes.io/delete-from-underlying-clusters"
)

type UpdateObjFunc func(runtime.Object) (runtime.Object, error)
type ObjNameFunc func(runtime.Object) string

type DeletionHelper struct {
	updateObjFunc UpdateObjFunc
	objNameFunc   ObjNameFunc
	informer      util.FederatedInformer
	updater       util.FederatedUpdater
}

func NewDeletionHelper(
	updateObjFunc UpdateObjFunc, objNameFunc ObjNameFunc,
	informer util.FederatedInformer, updater util.FederatedUpdater) *DeletionHelper {
	return &DeletionHelper{
		updateObjFunc: updateObjFunc,
		objNameFunc:   objNameFunc,
		informer:      informer,
		updater:       updater,
	}
}

// Ensures that the given object has both FinalizerDeleteFromUnderlyingClusters
// and FinalizerOrphan finalizers.
// We do this so that the controller is always notified when a federation resource is deleted.
// If user deletes the resource with nil DeleteOptions or
// DeletionOptions.OrphanDependents = true then the apiserver removes the orphan finalizer
// and deletion helper does a cascading deletion.
// Otherwise, deletion helper just removes the federation resource and orphans
// the corresponding resources in underlying clusters.
// This method should be called before creating objects in underlying clusters.
func (dh *DeletionHelper) EnsureFinalizers(obj runtime.Object) (
	runtime.Object, error) {
	finalizers := sets.String{}
	hasFinalizer, err := finalizersutil.HasFinalizer(obj, FinalizerDeleteFromUnderlyingClusters)
	if err != nil {
		return nil, err
	}
	if !hasFinalizer {
		finalizers.Insert(FinalizerDeleteFromUnderlyingClusters)
	}
	hasFinalizer, err = finalizersutil.HasFinalizer(obj, metav1.FinalizerOrphanDependents)
	if err != nil {
		return nil, err
	}
	if !hasFinalizer {
		finalizers.Insert(metav1.FinalizerOrphanDependents)
	}
	if finalizers.Len() != 0 {
		klog.V(2).Infof("Adding finalizers %v to %s", finalizers.List(), dh.objNameFunc(obj))
		return dh.addFinalizers(obj, finalizers)
	}
	return nil, nil
}

// Deletes the resources corresponding to the given federated resource from
// all underlying clusters, unless it has the FinalizerOrphan finalizer.
// Removes FinalizerOrphan and FinalizerDeleteFromUnderlyingClusters finalizers
// when done.
// Callers are expected to keep calling this (with appropriate backoff) until
// it succeeds.
func (dh *DeletionHelper) HandleObjectInUnderlyingClusters(obj runtime.Object, targetKey string, skipDelete func(runtime.Object) bool) (
	runtime.Object, error) {
	objName := dh.objNameFunc(obj)
	klog.V(2).Infof("Handling deletion of federated dependents for object: %s", objName)
	hasFinalizer, err := finalizersutil.HasFinalizer(obj, FinalizerDeleteFromUnderlyingClusters)
	if err != nil {
		return obj, err
	}
	if !hasFinalizer {
		klog.V(2).Infof("obj does not have %s finalizer. Nothing to do", FinalizerDeleteFromUnderlyingClusters)
		return obj, nil
	}
	hasOrphanFinalizer, err := finalizersutil.HasFinalizer(obj, metav1.FinalizerOrphanDependents)
	if err != nil {
		return obj, err
	}
	if hasOrphanFinalizer {
		klog.V(2).Infof("Found finalizer orphan. Nothing to do, just remove the finalizer")
		// If the obj has FinalizerOrphan finalizer, then we need to orphan the
		// corresponding objects in underlying clusters.
		// Just remove both the finalizers in that case.
		finalizers := sets.NewString(FinalizerDeleteFromUnderlyingClusters, metav1.FinalizerOrphanDependents)
		return dh.removeFinalizers(obj, finalizers)
	}

	klog.V(2).Infof("Deleting obj %s from underlying clusters", objName)
	// Else, we need to delete the obj from all underlying clusters.
	unreadyClusters, err := dh.informer.GetUnreadyClusters()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a list of unready clusters")
	}
	// TODO: Handle the case when cluster resource is watched after this is executed.
	// This can happen if a namespace is deleted before its creation had been
	// observed in all underlying clusters.
	clusterNsObjs, err := dh.informer.GetTargetStore().GetFromAllClusters(targetKey)
	klog.V(3).Infof("Found %d objects in underlying clusters", len(clusterNsObjs))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get object %s from underlying clusters", objName)
	}
	operations := make([]util.FederatedOperation, 0)
	for _, clusterNsObj := range clusterNsObjs {
		clusterObj := clusterNsObj.Object.(runtime.Object)
		if skipDelete(clusterObj) {
			continue
		}
		operations = append(operations, util.FederatedOperation{
			Type:        util.OperationTypeDelete,
			ClusterName: clusterNsObj.ClusterName,
			Obj:         clusterObj,
			Key:         targetKey,
		})
	}
	_, operationalErrors := dh.updater.Update(operations)
	if len(operationalErrors) > 0 {
		return nil, errors.Errorf("failed to execute deletions for obj %s: %v", objName, operationalErrors)
	}
	if len(operations) > 0 {
		// We have deleted a bunch of resources.
		// Wait for the store to observe all the deletions.
		var clusterNames []string
		for _, op := range operations {
			clusterNames = append(clusterNames, op.ClusterName)
		}
		return nil, errors.Errorf("waiting for object %s to be deleted from clusters: %s", objName, strings.Join(clusterNames, ", "))
	}

	// We have now deleted the object from all *ready* clusters.
	// But still need to wait for clusters that are not ready to ensure that
	// the object has been deleted from *all* clusters.
	if len(unreadyClusters) != 0 {
		var clusterNames []string
		for _, cluster := range unreadyClusters {
			clusterNames = append(clusterNames, cluster.Name)
		}
		return nil, errors.Errorf("waiting for clusters %s to become ready to verify that obj %s has been deleted", strings.Join(clusterNames, ", "), objName)
	}

	// All done. Just remove the finalizer.
	return dh.removeFinalizers(obj, sets.NewString(FinalizerDeleteFromUnderlyingClusters))
}

// Adds the given finalizers to the given objects ObjectMeta.
func (dh *DeletionHelper) addFinalizers(obj runtime.Object, finalizers sets.String) (runtime.Object, error) {
	isUpdated, err := finalizersutil.AddFinalizers(obj, finalizers)
	if err != nil || !isUpdated {
		return nil, err
	}
	// Send the update to apiserver.
	updatedObj, err := dh.updateObjFunc(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add finalizers %v to object %s", finalizers, dh.objNameFunc(obj))
	}
	return updatedObj, nil
}

// Removes the given finalizers from the given objects ObjectMeta.
func (dh *DeletionHelper) removeFinalizers(obj runtime.Object, finalizers sets.String) (runtime.Object, error) {
	isUpdated, err := finalizersutil.RemoveFinalizers(obj, finalizers)
	if err != nil || !isUpdated {
		return obj, err
	}
	// Send the update to apiserver.
	updatedObj, err := dh.updateObjFunc(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to remove finalizers %v from object %s", finalizers, dh.objNameFunc(obj))
	}
	return updatedObj, nil
}
