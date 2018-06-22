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

/*
The apis package describes the comment directives that may be applied to apis / resources
*/
package apis

// Resource annotates a type as a resource
const Resource = "// +resource:path="

// Maximum annotates a go struct field for CRD validation
const Maximum = "// +kubebuilder:validation:Maximum="

// ExclusiveMaximum annotates a go struct field for CRD validation
const ExclusiveMaximum = "// +kubebuilder:validation:ExclusiveMaximum="

// Minimum annotates a go struct field for CRD validation
const Minimum = "// +kubebuilder:validation:Minimum="

// ExclusiveMinimum annotates a go struct field for CRD validation
const ExclusiveMinimum = "// +kubebuilder:validation:ExclusiveMinimum="
