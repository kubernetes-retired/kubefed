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

package taint

import (
	"fmt"
	"strings"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
)

// parseTaints takes a spec which is an array and creates slices for new taints to be added, taints to be deleted.
func parseTaints(spec []string) ([]corev1.Taint, []corev1.Taint, error) {
	var taints, taintsToRemove []corev1.Taint
	uniqueTaints := map[corev1.TaintEffect]sets.String{}

	for _, taintSpec := range spec {
		if strings.Index(taintSpec, "=") != -1 && strings.Index(taintSpec, ":") != -1 {
			newTaint, err := parseTaint(taintSpec)
			if err != nil {
				return nil, nil, err
			}
			// validate if taint is unique by <key, effect>
			if len(uniqueTaints[newTaint.Effect]) > 0 && uniqueTaints[newTaint.Effect].Has(newTaint.Key) {
				return nil, nil, fmt.Errorf("duplicated taints with the same key and effect: %v", newTaint)
			}
			// add taint to existingTaints for uniqueness check
			if len(uniqueTaints[newTaint.Effect]) == 0 {
				uniqueTaints[newTaint.Effect] = sets.String{}
			}
			uniqueTaints[newTaint.Effect].Insert(newTaint.Key)

			taints = append(taints, newTaint)
		} else if strings.HasSuffix(taintSpec, "-") {
			taintKey := taintSpec[:len(taintSpec)-1]
			var effect corev1.TaintEffect
			if strings.Index(taintKey, ":") != -1 {
				parts := strings.Split(taintKey, ":")
				taintKey = parts[0]
				effect = corev1.TaintEffect(parts[1])
			}

			// If effect is specified, need to validate it.
			if len(effect) > 0 {
				err := validateTaintEffect(effect)
				if err != nil {
					return nil, nil, err
				}
			}
			taintsToRemove = append(taintsToRemove, corev1.Taint{Key: taintKey, Effect: effect})
		} else {
			return nil, nil, fmt.Errorf("unknown taint spec: %v", taintSpec)
		}
	}
	return taints, taintsToRemove, nil
}

// parseTaint parses a taint from a string. Taint must be of the format '<key>=<value>:<effect>'.
func parseTaint(st string) (corev1.Taint, error) {
	var taint corev1.Taint
	parts := strings.Split(st, "=")
	if len(parts) != 2 || len(parts[1]) == 0 || len(validation.IsQualifiedName(parts[0])) > 0 {
		return taint, fmt.Errorf("invalid taint spec: %v", st)
	}

	parts2 := strings.Split(parts[1], ":")

	errs := validation.IsValidLabelValue(parts2[0])
	if len(parts2) != 2 || len(errs) != 0 {
		return taint, fmt.Errorf("invalid taint spec: %v, %s", st, strings.Join(errs, "; "))
	}

	effect := corev1.TaintEffect(parts2[1])
	if err := validateTaintEffect(effect); err != nil {
		return taint, err
	}

	taint.Key = parts[0]
	taint.Value = parts2[0]
	taint.Effect = effect

	return taint, nil
}

func validateTaintEffect(effect corev1.TaintEffect) error {
	if effect != corev1.TaintEffectNoSchedule && effect != corev1.TaintEffectPreferNoSchedule && effect != corev1.TaintEffectNoExecute {
		return fmt.Errorf("invalid taint effect: %v, unsupported taint effect", effect)
	}

	return nil
}

// Return an updated set of taints by adding/updating/removing from the pre-existing taints on the cluster.
func applyTaints(currentTaints, addTaints, removeTaints []corev1.Taint) ([]corev1.Taint, error) {
	glog.V(4).Infof("Current taints: %v", currentTaints)
	glog.V(4).Infof("Add taints: %v", addTaints)
	glog.V(4).Infof("Remove taints: %v", removeTaints)

	var numAdded, numUpdated, numRemoved int

	newTaintsLoop:
	for _, addTaint := range addTaints {
		for i, taint := range currentTaints {
			if addTaint.Key == taint.Key && addTaint.Effect == taint.Effect {
				currentTaints[i].Value = addTaint.Value
				numUpdated++
				continue newTaintsLoop
			}
		}
		currentTaints = append(currentTaints, addTaint)
		numAdded++
	}

	newTaints := []corev1.Taint{}

	removeTaintsLoop:
	for _, taint := range currentTaints {
		for _, removeTaint := range removeTaints {
			if len(removeTaint.Effect) == 0 {
				if taint.Key == removeTaint.Key {
					numRemoved++
					continue removeTaintsLoop
				}
			} else {
				if taint.Key == removeTaint.Key && taint.Effect == removeTaint.Effect {
					numRemoved++
					continue removeTaintsLoop
				}
			}
		}
		newTaints = append(newTaints, taint)
	}

	glog.V(4).Infof("Updated taints: %v", newTaints)
	glog.Infof("Added %d taints, updated %d, removed %d", numAdded, numUpdated, numRemoved)
	return newTaints, nil

}

// CheckIfTaintsAlreadyExists checks if the cluster already has taints that we want to add and returns a string with taint keys.
func checkIfTaintsAlreadyExist(currentTaints, newTaints []corev1.Taint) string {
	alreadyExist := []string{}
	for _, newTaint := range newTaints {
		for _, taint := range currentTaints {
			if newTaint.Key == taint.Key && newTaint.Effect == taint.Effect {
				alreadyExist = append(alreadyExist, newTaint.Key)
			}
		}
	}
	return strings.Join(alreadyExist, ",")
}