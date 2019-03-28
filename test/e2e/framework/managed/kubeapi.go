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

package managed

import (
	"net/url"
	"os"

	"github.com/pborman/uuid"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/federation-v2/test/common"
	"sigs.k8s.io/testing_frameworks/integration"
)

// KubernetesApiFixture manages a kubernetes api server
type KubernetesApiFixture struct {
	EtcdUrl   *url.URL
	Host      string
	ApiServer *integration.APIServer
	IsPrimary bool
}

func SetUpKubernetesApiFixture(tl common.TestLogger) *KubernetesApiFixture {
	f := &KubernetesApiFixture{}
	f.setUp(tl)
	return f
}

func (f *KubernetesApiFixture) setUp(tl common.TestLogger) {
	defer TearDownOnPanic(tl, f)

	f.EtcdUrl = SetUpEtcd(tl)

	// TODO(marun) Enable https apiserver for integration.APIServer

	args := []string{
		"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
		"--cert-dir={{ .CertDir }}",
		"--insecure-port={{ if .URL }}{{ .URL.Port }}{{ end }}",
		"--insecure-bind-address={{ if .URL }}{{ .URL.Hostname }}{{ end }}",
		"--secure-port=0",
		"--etcd-prefix", uuid.New(),
	}

	apiServer := &integration.APIServer{
		EtcdURL: f.EtcdUrl,
		Args:    args,
		Out:     os.Stdout,
		Err:     os.Stderr,
	}
	err := apiServer.Start()
	if err != nil {
		tl.Fatalf("Error starting kubernetes apiserver: %v", err)
	}
	f.ApiServer = apiServer
	f.Host = f.ApiServer.URL.String()
}

func (f *KubernetesApiFixture) TearDown(tl common.TestLogger) {
	if f.ApiServer != nil {
		f.ApiServer.Stop()
		f.ApiServer = nil
	}
	if f.EtcdUrl != nil {
		TearDownEtcd(tl)
		f.EtcdUrl = nil
	}
}

func (f *KubernetesApiFixture) NewClient(tl common.TestLogger, userAgent string) clientset.Interface {
	config := f.NewConfig(tl)
	rest.AddUserAgent(config, userAgent)
	return clientset.NewForConfigOrDie(config)
}

func (f *KubernetesApiFixture) NewConfig(tl common.TestLogger) *rest.Config {
	return &rest.Config{Host: f.Host}
}
