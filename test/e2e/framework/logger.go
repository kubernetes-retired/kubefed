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

package framework

import (
	"github.com/kubernetes-sigs/federation-v2/test/common"
)

type e2eLogger struct{}

func NewE2ELogger() common.TestLogger {
	return e2eLogger{}
}

func (e2eLogger) Errorf(format string, args ...interface{}) {
	Errorf(format, args...)
}

func (e2eLogger) Fatal(args ...interface{}) {
	// TODO(marun) Is there a nicer way to do this?
	Failf("%v", args)
}

func (e2eLogger) Fatalf(format string, args ...interface{}) {
	Failf(format, args...)
}

func (e2eLogger) Log(args ...interface{}) {
	// TODO(marun) Is there a nicer way to do this?
	Logf("%v", args)
}

func (e2eLogger) Logf(format string, args ...interface{}) {
	Logf(format, args...)
}
