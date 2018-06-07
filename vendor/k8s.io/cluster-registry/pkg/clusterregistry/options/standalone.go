/*
Copyright 2017 The Kubernetes Authors.

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

package options

import (
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
)

// StandaloneServerRunOptions contains runtime options for the cluster registry
// deployed as a standalone Kubernetes API server.
type StandaloneServerRunOptions struct {
	*serverRunOptions
	authentication *StandaloneAuthenticationOptions
	authorization  *StandaloneAuthorizationOptions
}

// NewStandaloneServerRunOptions creates a new StandaloneServerRunOptions
// object with default values.
func NewStandaloneServerRunOptions() *StandaloneServerRunOptions {
	return &StandaloneServerRunOptions{
		serverRunOptions: NewServerRunOptions(),
		authentication:   NewStandaloneAuthenticationOptions().WithAll(),
		authorization:    NewStandaloneAuthorizationOptions(),
	}
}

// AddFlags adds flags to the FlagSet for specific StandaloneServerRunOptions
// fields before calling the embedded AddFlags method to add the rest of
// the common options.
func (s *StandaloneServerRunOptions) AddFlags(fs *pflag.FlagSet) {
	s.authentication.AddFlags(fs)
	s.authorization.AddFlags(fs)
	s.serverRunOptions.AddFlags(fs)
}

// GenericServerRunOptions gets the embedded genericServerRunOptions field.
func (s *StandaloneServerRunOptions) GenericServerRunOptions() *options.ServerRunOptions {
	return s.genericServerRunOptions
}

// Etcd gets the embedded etcd field.
func (s *StandaloneServerRunOptions) Etcd() *options.EtcdOptions {
	return s.etcd
}

// SecureServing gets the embedded secureServing field.
func (s *StandaloneServerRunOptions) SecureServing() *options.SecureServingOptions {
	return s.secureServing
}

// Audit gets the embedded audit field.
func (s *StandaloneServerRunOptions) Audit() *options.AuditOptions {
	return s.audit
}

// Features gets the embedded features field.
func (s *StandaloneServerRunOptions) Features() *options.FeatureOptions {
	return s.features
}

// Validate checks any specific StandaloneServerRunOptions before calling the
// embedded Validate method for the common options.
func (s *StandaloneServerRunOptions) Validate() []error {
	return s.serverRunOptions.Validate()
}

// ApplyAuthentication applies the delegated authentication to the config.
func (s *StandaloneServerRunOptions) ApplyAuthentication(c *server.Config) error {
	return s.authentication.ApplyTo(c)
}

// ApplyAuthorization applies the delegated authorization to the config.
func (s *StandaloneServerRunOptions) ApplyAuthorization(c *server.Config) error {
	return s.authorization.ApplyTo(c)
}
