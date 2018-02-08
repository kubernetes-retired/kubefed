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
	"net"
	"strconv"
	"testing"
)

// FindFreeLocalPort returns the number of an available port number on
// the loopback interface.  Useful for determining the port to launch
// a server on.  Error handling required - there is a non-zero chance
// that the returned port number will be bound by another process
// after this function returns.
func FindFreeLocalPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	_, portStr, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, err
	}
	return port, nil

}

// SetUp is likely to be fixture-specific, but TearDown needs to be
// consistent to enable TearDownOnPanic.
type TestFixture interface {
	TearDown(t *testing.T)
}

// TearDownOnPanic can be used to ensure cleanup on setup failure.
func TearDownOnPanic(t *testing.T, f TestFixture) {
	if r := recover(); r != nil {
		f.TearDown(t)
		panic(r)
	}
}
