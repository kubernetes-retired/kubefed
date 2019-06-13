/*
Copyright 2019 The Kubernetes Authors.

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

package webhook

import (
	"os"

	"github.com/openshift/generic-admission-server/pkg/apiserver"
	"github.com/openshift/generic-admission-server/pkg/cmd/server"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubefed/pkg/controller/webhook/federatedtypeconfig"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedcluster"
	"sigs.k8s.io/kubefed/pkg/controller/webhook/kubefedconfig"
)

func NewWebhookCommand(stopChan <-chan struct{}) *cobra.Command {
	admissionHooks := []apiserver.AdmissionHook{
		&federatedtypeconfig.FederatedTypeConfigAdmissionHook{},
		&kubefedcluster.KubeFedClusterAdmissionHook{},
		&kubefedconfig.KubeFedConfigAdmissionHook{},
	}

	cmd := server.NewCommandStartAdmissionServer(os.Stdout, os.Stderr, stopChan, admissionHooks...)
	cmd.Use = "webhook"
	cmd.Short = "Start a kubefed webhook server"
	cmd.Long = "Start a kubefed webhook server"

	return cmd
}
