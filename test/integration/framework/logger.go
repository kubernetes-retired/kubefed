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
	"testing"

	"github.com/kubernetes-sigs/federation-v2/test/common"
)

type integrationLogger struct {
	t *testing.T
}

func NewIntegrationLogger(t *testing.T) common.TestLogger {
	return &integrationLogger{
		t: t,
	}
}

func (l *integrationLogger) Errorf(format string, args ...interface{}) {
	l.t.Errorf(format, args...)
}

func (l *integrationLogger) Fatal(args ...interface{}) {
	l.t.Fatal(args...)
}

func (l *integrationLogger) Fatalf(format string, args ...interface{}) {
	l.t.Fatalf(format, args...)
}

func (l *integrationLogger) Log(args ...interface{}) {
	l.t.Log(args...)
}

func (l *integrationLogger) Logf(format string, args ...interface{}) {
	l.t.Logf(format, args...)
}
