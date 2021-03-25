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
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeduplicate(t *testing.T) {
	// single event for reconciliation
	singleEvent := QualifiedName{
		Namespace: "ns",
		Name:      "name",
	}

	// counters for reconciliation and enqueue
	var reconcileCount, enqueueCount int32 = 0, 0
	addReconcileCount := func() {
		atomic.AddInt32(&reconcileCount, 1)
	}
	addEnqueueCount := func() {
		atomic.AddInt32(&enqueueCount, 1)
	}

	var worker ReconcileWorker

	// an utility function to enqueue single event for specified times
	duplicateEnqueue := func(times int) {
		for i := 0; i < times; i++ {
			worker.Enqueue(singleEvent)
			addEnqueueCount()
		}
	}

	// new worker
	doOnce := sync.Once{}
	worker = NewReconcileWorker("test deduplicate",
		func(qualifiedName QualifiedName) ReconciliationStatus {
			addReconcileCount()
			// enqueue same events for 5 times during the reconciliation itself
			doOnce.Do(func() {
				duplicateEnqueue(5)
			})
			return StatusAllOK
		},
		WorkerTiming{},
	)

	// run worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker.Run(ctx.Done())

	// enqueue single event for 10 times
	duplicateEnqueue(10)

	// wait for all events enqueued and reconciled
	// FIXME: there is not a good way to detect whether all enqueued events have been reconciled, so we just wait for a while
	time.Sleep(1 * time.Second)

	// check counters
	if enqueueCount != 15 {
		t.Errorf("expected enqueue count 15 but got %d", enqueueCount)
	}
	if reconcileCount != 2 {
		t.Errorf("expected reconcile count 2 but got %d", reconcileCount)
	}

	t.Logf("the enqueued (before or during reconciliation) 15 same events have been squashed to 2")
}
