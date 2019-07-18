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

// Package options contains flags and options for initializing controller-manager
package options

import (
	"github.com/spf13/pflag"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"sigs.k8s.io/kubefed/pkg/controller/util"
)

// Options contains everything necessary to create and run controller-manager.
type Options struct {
	Config                   *util.ControllerConfig
	FeatureGates             map[string]bool
	Scope                    apiextv1b1.ResourceScope
	LeaderElection           *util.LeaderElectionConfiguration
	ClusterHealthCheckConfig *util.ClusterHealthCheckConfig
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Config.KubeFedNamespace, "kubefed-namespace", "", "The namespace the KubeFed control plane is deployed in.")
}

func NewOptions() *Options {
	return &Options{
		Config:                   new(util.ControllerConfig),
		FeatureGates:             make(map[string]bool),
		LeaderElection:           new(util.LeaderElectionConfiguration),
		ClusterHealthCheckConfig: new(util.ClusterHealthCheckConfig),
	}
}
