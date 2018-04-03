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

	"github.com/pborman/uuid"

	"github.com/kubernetes-sig-testing/frameworks/integration"
	"github.com/kubernetes-sigs/federation-v2/test/common"
	"k8s.io/client-go/rest"
	"k8s.io/cluster-registry/pkg/client/clientset_generated/clientset"
)

// ClusterRegistryApiFixture manages a api registry apiserver
type ClusterRegistryApiFixture struct {
	EtcdUrl             string
	Host                string
	SecureConfigFixture *SecureConfigFixture
	ClusterRegistryApi  *integration.APIServer
}

func SetUpClusterRegistryApiFixture(tl common.TestLogger) *ClusterRegistryApiFixture {
	f := &ClusterRegistryApiFixture{}
	f.setUp(tl)
	return f
}

func (f *ClusterRegistryApiFixture) setUp(tl common.TestLogger) {
	defer TearDownOnPanic(tl, f)

	f.EtcdUrl = SetUpEtcd(tl)
	f.SecureConfigFixture = SetUpSecureConfigFixture(tl)

	// TODO(marun) ensure resiliency in the face of another process
	// taking the port

	port, err := FindFreeLocalPort()
	if err != nil {
		tl.Fatal(err)
	}

	bindAddress := "127.0.0.1"
	f.Host = fmt.Sprintf("https://%s:%d", bindAddress, port)
	url, err := url.Parse(f.Host)
	if err != nil {
		tl.Fatalf("Error parsing url: %v", err)
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
		tl.Fatalf("Error starting api registry apiserver: %v", err)
	}
	f.ClusterRegistryApi = apiServer
}

func (f *ClusterRegistryApiFixture) TearDown(tl common.TestLogger) {
	if f.ClusterRegistryApi != nil {
		f.ClusterRegistryApi.Stop()
		f.ClusterRegistryApi = nil
	}
	if f.SecureConfigFixture != nil {
		f.SecureConfigFixture.TearDown(tl)
		f.SecureConfigFixture = nil
	}
	if len(f.EtcdUrl) > 0 {
		TearDownEtcd(tl)
		f.EtcdUrl = ""
	}
}

func (f *ClusterRegistryApiFixture) NewClient(tl common.TestLogger, userAgent string) clientset.Interface {
	config := f.NewConfig(tl)
	rest.AddUserAgent(config, userAgent)
	return clientset.NewForConfigOrDie(config)
}

func (f *ClusterRegistryApiFixture) NewConfig(tl common.TestLogger) *rest.Config {
	return f.SecureConfigFixture.NewClientConfig(tl, f.Host)
}
