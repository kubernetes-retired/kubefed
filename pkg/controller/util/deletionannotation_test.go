/*
Copyright 2021 The Kubernetes Authors.

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

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteOptions(t *testing.T) {
	fedObj := &unstructured.Unstructured{}

	fedObjOrphan := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					DeleteOptionAnnotation: "{\"propagationPolicy\":\"Orphan\"}",
				},
			},
		},
	}

	fedObjGrace := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					DeleteOptionAnnotation: "{\"gracePeriodSeconds\":5}",
				},
			},
		},
	}

	fedObjGraceAndOrphan := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"annotations": map[string]interface{}{
					DeleteOptionAnnotation: "{\"propagationPolicy\":\"Orphan\",\"gracePeriodSeconds\":5}",
				},
			},
		},
	}

	actOpt0 := client.DeleteOptions{}
	opt0, _ := GetDeleteOptions(fedObj)
	actOpt0.ApplyOptions(opt0)
	expOpt0 := client.DeleteOptions{}
	assert.Equal(t, expOpt0.AsDeleteOptions(), actOpt0.AsDeleteOptions())

	actOpt1 := client.DeleteOptions{}
	opt1, _ := GetDeleteOptions(fedObjOrphan)
	actOpt1.ApplyOptions(opt1)
	prop := metav1.DeletePropagationOrphan
	expOpt1 := client.DeleteOptions{PropagationPolicy: &prop}
	assert.Equal(t, expOpt1.AsDeleteOptions(), actOpt1.AsDeleteOptions())

	actOpt2 := client.DeleteOptions{}
	opt2, _ := GetDeleteOptions(fedObjGrace)
	actOpt2.ApplyOptions(opt2)
	seconds := int64(5)
	expOpt2 := client.DeleteOptions{GracePeriodSeconds: &seconds}
	assert.Equal(t, expOpt2.AsDeleteOptions(), actOpt2.AsDeleteOptions())

	actOpt3 := client.DeleteOptions{}
	opt3, _ := GetDeleteOptions(fedObjGraceAndOrphan)
	actOpt3.ApplyOptions(opt3)
	expOpt3 := client.DeleteOptions{GracePeriodSeconds: &seconds, PropagationPolicy: &prop}
	assert.Equal(t, expOpt3.AsDeleteOptions(), actOpt3.AsDeleteOptions())
}
