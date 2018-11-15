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
	InstallCRDs          bool
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.InstallCRDs, "install-crds", true, "install the CRDs used by the controller as part of startup")
	fs.Var(flagutil.NewMapStringBool(&o.FeatureGates), "feature-gates", "A set of key=value pairs that describe feature gates for alpha/experimental features. "+
		"Options are:\n"+strings.Join(utilfeature.DefaultFeatureGate.KnownFeatures(), "\n"))

	fs.StringVar(&o.Config.FederationNamespace, "federation-namespace", util.DefaultFederationSystemNamespace, "The namespace the federation control plane is deployed in.")
	fs.StringVar(&o.Config.ClusterNamespace, "registry-namespace", util.MulticlusterPublicNamespace, "The cluster registry namespace.")
	fs.DurationVar(&o.Config.ClusterAvailableDelay, "cluster-available-delay", util.DefaultClusterAvailableDelay, "Time to wait before reconciling on a healthy cluster.")
	fs.DurationVar(&o.Config.ClusterUnavailableDelay, "cluster-unavailable-delay", util.DefaultClusterUnavailableDelay, "Time to wait before giving up on an unhealthy cluster.")

	fs.BoolVar(&o.LimitedScope, "limited-scope", false, "Whether the federation namespace will be the only target for federation.")
	fs.DurationVar(&o.ClusterMonitorPeriod, "cluster-monitor-period", time.Second*40, "How often to monitor the cluster health")
}

func NewOptions() *Options {
	return &Options{
		Config:       new(util.ControllerConfig),
		FeatureGates: make(map[string]bool),
	}
}
