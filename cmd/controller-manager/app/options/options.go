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
	"strings"
	"time"

	"github.com/spf13/pflag"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	flagutil "k8s.io/apiserver/pkg/util/flag"

	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
)

// Options contains everything necessary to create and run controller-manager.
type Options struct {
	Config               *util.ControllerConfig
	FeatureGates         map[string]bool
	ClusterMonitorPeriod time.Duration
	LimitedScope         bool
	LeaderElection       *util.LeaderElectionConfiguration
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.Var(flagutil.NewMapStringBool(&o.FeatureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	fs.StringVar(&o.Config.FederationNamespace, "federation-namespace", util.DefaultFederationSystemNamespace, "The namespace the federation control plane is deployed in.")
	fs.StringVar(&o.Config.ClusterNamespace, "registry-namespace", util.MulticlusterPublicNamespace, "The cluster registry namespace.")
	fs.DurationVar(&o.Config.ClusterAvailableDelay, "cluster-available-delay", util.DefaultClusterAvailableDelay, "Time to wait before reconciling on a healthy cluster.")
	fs.DurationVar(&o.Config.ClusterUnavailableDelay, "cluster-unavailable-delay", util.DefaultClusterUnavailableDelay, "Time to wait before giving up on an unhealthy cluster.")

	fs.BoolVar(&o.LimitedScope, "limited-scope", false, "Whether the federation namespace will be the only target for federation.")
	fs.DurationVar(&o.ClusterMonitorPeriod, "cluster-monitor-period", time.Second*40, "How often to monitor the cluster health")

	fs.DurationVar(&o.LeaderElection.LeaseDuration, "leader-elect-lease-duration", util.DefaultLeaderElectionLeaseDuration, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&o.LeaderElection.RenewDeadline, "leader-elect-renew-deadline", util.DefaultLeaderElectionRenewDeadline, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&o.LeaderElection.RetryPeriod, "leader-elect-retry-period", util.DefaultLeaderElectionRetryPeriod, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
	fs.StringVar(&o.LeaderElection.ResourceLock, "leader-elect-resource-lock", "configmaps", ""+
		"The type of resource object that is used for locking during "+
		"leader election. Supported options are `configmaps` (default) and `endpoints`.")
}

func NewOptions() *Options {
	return &Options{
		Config:         new(util.ControllerConfig),
		FeatureGates:   make(map[string]bool),
		LeaderElection: new(util.LeaderElectionConfiguration),
	}
}
