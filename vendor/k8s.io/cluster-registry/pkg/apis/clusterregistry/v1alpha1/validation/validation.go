/*
Copyright 2017 The Kubernetes Authors.

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

// Package validation defines validation routines for the clusterregistry API.
package validation

import (
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterregistry "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
)

// ValidateCluster ensures that a newly-created Cluster is valid.
func ValidateCluster(cluster *clusterregistry.Cluster) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&cluster.ObjectMeta, false, validation.ValidateClusterName, field.NewPath("metadata"))
	allErrs = append(allErrs, validateClusterName(cluster)...)
	return allErrs
}

// ValidateClusterUpdate ensures that an update to a Cluster is valid.
func ValidateClusterUpdate(cluster, oldCluster *clusterregistry.Cluster) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&cluster.ObjectMeta, &oldCluster.ObjectMeta, field.NewPath("metadata"))
	if cluster.Name != oldCluster.Name {
		allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "name"),
			cluster.Name+" != "+oldCluster.Name, "cannot change cluster name"))
	}
	allErrs = append(allErrs, validateClusterName(cluster)...)
	return allErrs
}

func validateClusterName(cluster *clusterregistry.Cluster) field.ErrorList {
	if len(cluster.ClusterName) > 0 {
		return field.ErrorList{field.Invalid(field.NewPath("metadata", "clusterName"), "len(ClusterName) > 0", "clusterName is not used and must not be set.")}
	}
	return field.ErrorList{}
}
