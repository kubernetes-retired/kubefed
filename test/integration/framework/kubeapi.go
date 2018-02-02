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
	clientset "k8s.io/client-go/kubernetes"
)

// KubernetesApiFixture manages a kubernetes api server
type KubernetesApiFixture struct {
	EtcdUrl             string
	Host                string
	SecureConfigFixture *SecureConfigFixture
	ApiServer           *integration.APIServer
}

func SetUpKubernetesApiFixture(t *testing.T) *KubernetesApiFixture {
	f := &KubernetesApiFixture{}
	f.setUp(t)
	return f
}

func (f *KubernetesApiFixture) setUp(t *testing.T) {
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
		"--etcd-servers", f.EtcdUrl,
		"--client-ca-file", f.SecureConfigFixture.CACertFile,
		"--cert-dir", f.SecureConfigFixture.CertDir,
		"--bind-address", bindAddress,
		"--secure-port", strconv.Itoa(port),
		"--insecure-port", "0",
		"--etcd-prefix", uuid.New(),
	}

	apiServer := &integration.APIServer{
		URL:  url,
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}
	err = apiServer.Start()
	if err != nil {
		t.Fatalf("Error starting kubernetes apiserver: %v", err)
	}
	f.ApiServer = apiServer
}

func (f *KubernetesApiFixture) TearDown(t *testing.T) {
	if f.ApiServer != nil {
		f.ApiServer.Stop()
		f.ApiServer = nil
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

func (f *KubernetesApiFixture) NewClient(t *testing.T, userAgent string) clientset.Interface {
	config := f.SecureConfigFixture.NewClientConfig(t, f.Host, userAgent)
	return clientset.NewForConfigOrDie(config)
}
