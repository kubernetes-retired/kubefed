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

package leaderelection

import (
	"context"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"sigs.k8s.io/kubefed/cmd/controller-manager/app/options"
)

func NewKubeFedLeaderElector(opts *options.Options, fnStartControllers func(*options.Options, <-chan struct{})) (*leaderelection.LeaderElector, error) {
	const component = "kubefed-controller-manager"
	kubeConfig := restclient.CopyConfig(opts.Config.KubeConfig)
	restclient.AddUserAgent(kubeConfig, "kubefed-leader-election")
	leaderElectionClient := kubeclient.NewForConfigOrDie(kubeConfig)

	hostname, err := os.Hostname()
	if err != nil {
		klog.Infof("unable to get hostname: %v", err)
		return nil, err
	}

	// Prepare event clients.
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&corev1.EventSinkImpl{Interface: leaderElectionClient.CoreV1().Events(opts.Config.KubeFedNamespace)})
	eventRecorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: component})

	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())
	rl, err := resourcelock.New(string(opts.LeaderElection.ResourceLock),
		opts.Config.KubeFedNamespace,
		component,
		leaderElectionClient.CoreV1(),
		leaderElectionClient.CoordinationV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: eventRecorder,
		})
	if err != nil {
		klog.Infof("couldn't create resource lock: %v", err)
		return nil, err
	}

	return leaderelection.NewLeaderElector(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: opts.LeaderElection.LeaseDuration,
		RenewDeadline: opts.LeaderElection.RenewDeadline,
		RetryPeriod:   opts.LeaderElection.RetryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Info("promoted as leader")
				stopChan := ctx.Done()
				fnStartControllers(opts, stopChan)
				<-stopChan
			},
			OnStoppedLeading: func() {
				klog.Info("leader election lost")
			},
		},
	})
}
