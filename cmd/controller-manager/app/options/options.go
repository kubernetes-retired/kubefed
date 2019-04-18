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
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/spf13/pflag"
)

// Options contains everything necessary to create and run controller-manager.
type Options struct {
	Config                   *util.ControllerConfig
	FeatureGates             map[string]bool
	LimitedScope             bool
	LeaderElection           *util.LeaderElectionConfiguration
	ClusterHealthCheckConfig util.ClusterHealthCheckConfig
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Config.FederationNamespace, "federation-namespace", util.DefaultFederationSystemNamespace, "The namespace the federation control plane is deployed in.")
	o.setDefaults()
}

func (o *Options) setDefaults() {
	o.FeatureGates = make(map[string]bool)
	o.LimitedScope = false

	o.Config.ClusterNamespace = util.MulticlusterPublicNamespace
	o.Config.ClusterAvailableDelay = util.DefaultClusterAvailableDelay
	o.Config.ClusterUnavailableDelay = util.DefaultClusterUnavailableDelay

	o.LeaderElection.LeaseDuration = util.DefaultLeaderElectionLeaseDuration
	o.LeaderElection.RenewDeadline = util.DefaultLeaderElectionRenewDeadline
	o.LeaderElection.RetryPeriod = util.DefaultLeaderElectionRetryPeriod
	o.LeaderElection.ResourceLock = "configmaps"

	o.ClusterHealthCheckConfig.PeriodSeconds = util.DefaultClusterHealthCheckPeriod
	o.ClusterHealthCheckConfig.FailureThreshold = util.DefaultClusterHealthCheckFailureThreshold
	o.ClusterHealthCheckConfig.SuccessThreshold = util.DefaultClusterHealthCheckSuccessThreshold
	o.ClusterHealthCheckConfig.TimeoutSeconds = util.DefaultClusterHealthCheckTimeout
}

func NewOptions() *Options {
	return &Options{
		Config:         new(util.ControllerConfig),
		FeatureGates:   make(map[string]bool),
		LeaderElection: new(util.LeaderElectionConfiguration),
	}
}
