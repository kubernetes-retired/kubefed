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

package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	clientset "k8s.io/cluster-registry/pkg/client/clientset/versioned"
	informers "k8s.io/cluster-registry/pkg/client/informers/externalversions"
)

var (
	masterURL  string
	kubeconfig string
	slackURL   string
)

// setUpSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
func setUpSignalHandler() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

func main() {
	flag.Parse()

	stopCh := setUpSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	clusterClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building cluster clientset: %s", err.Error())
	}

	clusterInformerFactory := informers.NewSharedInformerFactory(clusterClient, time.Second*30)

	controller := NewSlackController(kubeClient, clusterClient, clusterInformerFactory, slackURL)

	go clusterInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value provided in the default context in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&slackURL, "slack-url", "", "The URL of a Slack Incoming Webhook to which messages will be posted. Must be non-empty, or this controller will be ineffectual. See https://api.slack.com/incoming-webhooks.")
}
