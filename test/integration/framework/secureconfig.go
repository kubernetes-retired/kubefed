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
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/kubernetes-sigs/federation-v2/test/common"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/cert"
)

type SecureConfigFixture struct {
	CertDir      string
	ServerCAFile string
	Key          *rsa.PrivateKey
	CACert       *x509.Certificate
	CACertFile   string
}

func SetUpSecureConfigFixture(tl common.TestLogger) *SecureConfigFixture {
	f := &SecureConfigFixture{}
	f.setUp(tl)
	return f
}

func (f *SecureConfigFixture) setUp(tl common.TestLogger) {
	defer TearDownOnPanic(tl, f)

	var err error

	f.CertDir, err = ioutil.TempDir("", "fed-test-certs")
	if err != nil {
		tl.Fatal(err)
	}
	f.ServerCAFile = path.Join(f.CertDir, "apiserver.crt")

	f.Key, err = cert.NewPrivateKey()
	if err != nil {
		tl.Fatal(err)
	}

	f.CACert, err = cert.NewSelfSignedCACert(cert.Config{CommonName: "client-ca"}, f.Key)
	if err != nil {
		tl.Fatal(err)
	}

	caCertFile, err := ioutil.TempFile(f.CertDir, "client-ca.crt")
	if err != nil {
		tl.Fatal(err)
	}
	f.CACertFile = caCertFile.Name()

	if err := ioutil.WriteFile(f.CACertFile, cert.EncodeCertPEM(f.CACert), 0644); err != nil {
		tl.Fatal(err)
	}
}

func (f *SecureConfigFixture) TearDown(tl common.TestLogger) {
	if len(f.CertDir) >= 0 {
		os.RemoveAll(f.CertDir)
	}
}

func (f *SecureConfigFixture) NewClientConfig(tl common.TestLogger, host string) *rest.Config {
	// The server ca file is written on startup, and may not be immediately available
	f.waitForServerCAFile(tl)

	config := &rest.Config{
		Host:            host,
		TLSClientConfig: f.newTLSClientConfig(tl),
	}
	return config
}

func (f *SecureConfigFixture) waitForServerCAFile(tl common.TestLogger) {
	err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (done bool, err error) {
		if _, err := os.Stat(f.ServerCAFile); err == nil {
			return true, nil
		}
		return false, err
	})
	if err != nil {
		tl.Fatalf("Error reading CA file: %s: %v\nHas the server been started?", f.ServerCAFile, err)
	}
}

func (f *SecureConfigFixture) newTLSClientConfig(tl common.TestLogger) rest.TLSClientConfig {
	key, err := cert.NewPrivateKey()
	if err != nil {
		tl.Fatal(err)
	}

	clientCert, err := cert.NewSignedCert(
		cert.Config{
			CommonName: "federation-test",
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
		key, f.CACert, f.Key,
	)
	if err != nil {
		tl.Fatal(err)
	}

	return rest.TLSClientConfig{
		CertData: cert.EncodeCertPEM(clientCert),
		KeyData:  cert.EncodePrivateKeyPEM(key),
		CAFile:   f.ServerCAFile,
	}
}
