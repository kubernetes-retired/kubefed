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
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/pborman/uuid"

	"github.com/kubernetes-incubator/apiserver-builder/pkg/cmd/server"
	"github.com/marun/fnord/pkg/apis"
	"github.com/marun/fnord/pkg/client/clientset_generated/clientset"
	"github.com/marun/fnord/pkg/openapi"
)

// FederationApiFixture manages a federation api server
type FederationApiFixture struct {
	stopCh              chan struct{}
	EtcdUrl             string
	Host                string
	SecureConfigFixture *SecureConfigFixture
}

func SetUpFederationApiFixture(t *testing.T) *FederationApiFixture {
	f := &FederationApiFixture{}
	f.setUp(t)
	return f
}

func (f *FederationApiFixture) setUp(t *testing.T) {
	defer TearDownOnPanic(t, f)

	f.EtcdUrl = SetUpEtcd(t)
	f.SecureConfigFixture = SetUpSecureConfigFixture(t)

	// TODO(marun) ensure resiliency in the face of another process
	// taking the port.

	port, err := FindFreeLocalPort()
	if err != nil {
		t.Fatal(err)
	}

	f.stopCh = make(chan struct{})

	server.GetOpenApiDefinition = openapi.GetOpenAPIDefinitions
	cmd, _ := server.NewCommandStartServer(uuid.New(), os.Stdout, os.Stderr, apis.GetAllApiBuilders(), f.stopCh, "Api", "v0")
	bindAddress := "127.0.0.1"
	cmd.SetArgs([]string{
		"--etcd-servers", f.EtcdUrl,
		"--client-ca-file", f.SecureConfigFixture.CACertFile,
		"--cert-dir", f.SecureConfigFixture.CertDir,
		"--bind-address", bindAddress,
		"--secure-port", strconv.Itoa(port),
		"--delegated-auth=false",
	})

	f.Host = fmt.Sprintf("https://%s:%d", bindAddress, port)

	go func() {
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}()
}

func (f *FederationApiFixture) TearDown(t *testing.T) {
	if f.stopCh != nil {
		close(f.stopCh)
		f.stopCh = nil
	}
	if len(f.EtcdUrl) > 0 {
		TearDownEtcd(t)
		f.EtcdUrl = ""
	}
	if f.SecureConfigFixture != nil {
		f.SecureConfigFixture.TearDown(t)
		f.SecureConfigFixture = nil
	}
}

func (f *FederationApiFixture) NewClient(t *testing.T, userAgent string) clientset.Interface {
	config := f.SecureConfigFixture.NewClientConfig(t, f.Host, userAgent)
	return clientset.NewForConfigOrDie(config)
}
