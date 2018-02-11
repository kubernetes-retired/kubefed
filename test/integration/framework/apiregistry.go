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
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/pborman/uuid"

	"github.com/kubernetes-sig-testing/frameworks/integration"
	"k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

// ApiRegistryFixture manages a api registry apiserver
type ApiRegistryFixture struct {
	EtcdUrl             string
	Host                string
	SecureConfigFixture *SecureConfigFixture
	ApiRegistry         *integration.APIServer
}

func SetUpApiRegistryFixture(t *testing.T) *ApiRegistryFixture {
	f := &ApiRegistryFixture{}
	f.setUp(t)
	return f
}

func (f *ApiRegistryFixture) setUp(t *testing.T) {
	defer TearDownOnPanic(t, f)

	f.EtcdUrl = SetUpEtcd(t)
	f.SecureConfigFixture = SetUpSecureConfigFixture(t)

	// TODO(marun) ensure resiliency in the face of another process
	// taking the port

	port, err := FindFreeLocalPort()
	if err != nil {
		t.Fatal(err)
	}

	bindAddress := "127.0.0.1"
	f.Host = fmt.Sprintf("https://%s:%d", bindAddress, port)
	url, err := url.Parse(f.Host)
	if err != nil {
		t.Fatalf("Error parsing url: %v", err)
	}

	args := []string{
		"standalone",
		"--etcd-servers", f.EtcdUrl,
		"--client-ca-file", f.SecureConfigFixture.CACertFile,
		"--cert-dir", f.SecureConfigFixture.CertDir,
		"--bind-address", bindAddress,
		"--secure-port", strconv.Itoa(port),
		"--etcd-prefix", uuid.New(),
	}

	apiServer := &integration.APIServer{
		Name: "clusterregistry",
		URL:  url,
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}
	err = apiServer.Start()
	if err != nil {
		t.Fatalf("Error starting api registry apiserver: %v", err)
	}
	f.ApiRegistry = apiServer
}

func (f *ApiRegistryFixture) TearDown(t *testing.T) {
	if f.ApiRegistry != nil {
		f.ApiRegistry.Stop()
		f.ApiRegistry = nil
	}
	if f.SecureConfigFixture != nil {
		f.SecureConfigFixture.TearDown(t)
		f.SecureConfigFixture = nil
	}
	if len(f.EtcdUrl) > 0 {
		TearDownEtcd(t)
		f.EtcdUrl = ""
	}
}

func (f *ApiRegistryFixture) NewClient(t *testing.T, userAgent string) clientset.Interface {
	config := f.SecureConfigFixture.NewClientConfig(t, f.Host, userAgent)
	return clientset.NewForConfigOrDie(config)
}
