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

package podanalyzer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	api_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctlutil "sigs.k8s.io/kubefed/pkg/controller/util"
)

func TestAnalyze(t *testing.T) {
	now := time.Now()
	podRunning := newPod("p1",
		api_v1.PodStatus{
			Phase: api_v1.PodRunning,
			Conditions: []api_v1.PodCondition{
				{
					Type:   api_v1.PodReady,
					Status: api_v1.ConditionTrue,
				},
			},
		})
	podUnschedulableTimeElapsed := newPod("pU",
		api_v1.PodStatus{
			Phase: api_v1.PodPending,
			Conditions: []api_v1.PodCondition{
				{
					Type:               api_v1.PodScheduled,
					Status:             api_v1.ConditionFalse,
					Reason:             api_v1.PodReasonUnschedulable,
					LastTransitionTime: metav1.Time{Time: now.Add(-10 * time.Minute)},
				},
			},
		})
	podUnschedulableRightNow := newPod("pU",
		api_v1.PodStatus{
			Phase: api_v1.PodPending,
			Conditions: []api_v1.PodCondition{
				{
					Type:               api_v1.PodScheduled,
					Status:             api_v1.ConditionFalse,
					Reason:             api_v1.PodReasonUnschedulable,
					LastTransitionTime: metav1.Time{Time: now},
				},
			},
		})
	podOther := newPod("pO",
		api_v1.PodStatus{
			Phase:      api_v1.PodPending,
			Conditions: []api_v1.PodCondition{},
		})

	result, status := AnalyzePods(&api_v1.PodList{Items: []api_v1.Pod{*podRunning, *podRunning,
		*podRunning, *podUnschedulableTimeElapsed, *podUnschedulableTimeElapsed, *podUnschedulableRightNow}}, now)
	assert.Equal(t, PodAnalysisResult{
		Total:           6,
		RunningAndReady: 3,
		Unschedulable:   2,
	}, result)
	assert.Equal(t, status, ctlutil.StatusNeedsRecheck)

	result, status = AnalyzePods(&api_v1.PodList{Items: []api_v1.Pod{*podOther}}, now)
	assert.Equal(t, PodAnalysisResult{
		Total:           1,
		RunningAndReady: 0,
		Unschedulable:   0,
	}, result)
	assert.Equal(t, status, ctlutil.StatusAllOK)
}

func newPod(name string, status api_v1.PodStatus) *api_v1.Pod {
	return &api_v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Status: status,
	}
}
