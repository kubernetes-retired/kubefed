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
	"time"

	"k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"

	"github.com/spf13/pflag"
)

// Options interface contains required methods to be implemented by the cluster
// registry API server subcommand modes: standalone and
// aggregated.
type Options interface {
	GenericServerRunOptions() *genericoptions.ServerRunOptions
	Etcd() *genericoptions.EtcdOptions
	SecureServing() *genericoptions.SecureServingOptions
	Audit() *genericoptions.AuditOptions
	Features() *genericoptions.FeatureOptions
	Validate() []error
	ApplyAuthentication(*server.Config) error
	ApplyAuthorization(c *server.Config) error
}

// ServerRunOptions contains runtime options for the cluster registry.
type serverRunOptions struct {
	genericServerRunOptions *genericoptions.ServerRunOptions
	etcd                    *genericoptions.EtcdOptions
	secureServing           *genericoptions.SecureServingOptions
	audit                   *genericoptions.AuditOptions
	features                *genericoptions.FeatureOptions

	eventTTL time.Duration
}

// NewServerRunOptions creates a new ServerRunOptions object with default values.
func NewServerRunOptions() *serverRunOptions {
	o := &serverRunOptions{
		genericServerRunOptions: genericoptions.NewServerRunOptions(),
		etcd:          genericoptions.NewEtcdOptions(storagebackend.NewDefaultConfig("/registry/clusterregistry.kubernetes.io", nil)),
		secureServing: genericoptions.NewSecureServingOptions(),
		audit:         genericoptions.NewAuditOptions(),
		features:      genericoptions.NewFeatureOptions(),

		eventTTL: 1 * time.Hour,
	}
	o.etcd.DefaultStorageMediaType = "application/vnd.kubernetes.protobuf"
	return o
}

// AddFlags adds flags for serverRunOptions fields to be specified via FlagSet.
func (s *serverRunOptions) AddFlags(fs *pflag.FlagSet) {
	s.genericServerRunOptions.AddUniversalFlags(fs)
	s.etcd.AddFlags(fs)
	s.secureServing.AddFlags(fs)
	s.audit.AddFlags(fs)
	s.features.AddFlags(fs)

	fs.DurationVar(&s.eventTTL, "event-ttl", s.eventTTL, "Amount of time to retain events.")
}
