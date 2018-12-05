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

package federate

import (
	"fmt"
	"io"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

var (
	disable_long = `
		Disables propagation of a Kubernetes API type.  This command
		can also optionally delete the API resources added by the enable
		command.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the
		--host-cluster-context flag otherwise.`

	disable_example = `
		# Disable propagation of the Service type
		kubefed2 federate disable Service

		# Disable propagation of the Service type and delete API resources
		kubefed2 federate disable Service --delete-from-api`
)

type disableType struct {
	options.SubcommandOptions
	disableTypeOptions
}

type disableTypeOptions struct {
	targetName string
	delete     bool
}

// Bind adds the join specific arguments to the flagset passed in as an
// argument.
func (o *disableTypeOptions) Bind(flags *pflag.FlagSet) {
	flags.BoolVar(&o.delete, "delete-from-api", false, "Whether to remove the API resources added by 'enable'.")
}

// NewCmdFederateDisable defines the `federate disable` command that
// disables federation of a Kubernetes API type.
func NewCmdFederateDisable(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &disableType{}

	cmd := &cobra.Command{
		Use:     "disable NAME",
		Short:   "Disables propagation of a Kubernetes API type",
		Long:    disable_long,
		Example: disable_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *disableType) Complete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("NAME is required")
	}
	j.targetName = args[0]

	return nil
}

// Run is the implementation of the `federate disable` command.
func (j *disableType) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to get host cluster config: %v", err)
	}

	typeConfigName := ctlutil.QualifiedName{
		Namespace: j.FederationNamespace,
		Name:      j.targetName,
	}

	err = DisableFederation(cmdOut, hostConfig, typeConfigName, j.delete, j.DryRun)
	if err != nil {
		return err
	}

	return nil
}

func DisableFederation(cmdOut io.Writer, config *rest.Config, typeConfigName ctlutil.QualifiedName, delete, dryRun bool) error {
	fedClient, err := util.FedClientset(config)
	if err != nil {
		return fmt.Errorf("Failed to get federation clientset: %v", err)
	}
	typeConfig, err := fedClient.CoreV1alpha1().FederatedTypeConfigs(typeConfigName.Namespace).Get(typeConfigName.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Error retrieving FederatedTypeConfig %q: %v", typeConfigName, err)
	}

	if dryRun {
		return nil
	}

	write := func(data string) {
		if cmdOut != nil {
			cmdOut.Write([]byte(data))
		}
	}

	if typeConfig.Spec.PropagationEnabled {
		typeConfig.Spec.PropagationEnabled = false
		_, err = fedClient.CoreV1alpha1().FederatedTypeConfigs(typeConfig.Namespace).Update(typeConfig)
		if err != nil {
			return fmt.Errorf("Error disabling propagation for FederatedTypeConfig %q: %v", typeConfigName, err)
		}
		write(fmt.Sprintf("Disabled propagation for FederatedTypeConfig %q\n", typeConfigName))
	} else {
		write(fmt.Sprintf("Propagation already disabled for FederatedTypeConfig %q\n", typeConfigName))
	}
	if !delete {
		return nil
	}

	// TODO(marun) consider waiting for the sync controller to be stopped before attempting deletion
	deletePrimitives(config, typeConfig, write)
	err = fedClient.CoreV1alpha1().FederatedTypeConfigs(typeConfigName.Namespace).Delete(typeConfigName.Name, nil)
	if err != nil {
		return fmt.Errorf("Error deleting FederatedTypeConfig %q: %v", typeConfigName, err)
	}
	write(fmt.Sprintf("federatedtypeconfig %q deleted\n", typeConfigName))

	return nil
}

func deletePrimitives(config *rest.Config, typeConfig typeconfig.Interface, write func(string)) error {
	client, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Error creating crd client: %v", err)
	}

	failedDeletion := []string{}
	crdNames := primitiveCRDNames(typeConfig)
	for _, crdName := range crdNames {
		err := client.CustomResourceDefinitions().Delete(crdName, nil)
		if err != nil && !errors.IsNotFound(err) {
			glog.Errorf("Failed to delete crd %q: %v", crdName, err)
			failedDeletion = append(failedDeletion, crdName)
			continue
		}
		write(fmt.Sprintf("customresourcedefinition %q deleted\n", crdName))
	}
	if len(failedDeletion) > 0 {
		return fmt.Errorf("The following crds were not deleted successfully (see error log for details): %v", failedDeletion)
	}

	return nil
}

func primitiveCRDNames(typeConfig typeconfig.Interface) []string {
	names := []string{
		typeconfig.GroupQualifiedName(typeConfig.GetTemplate()),
		typeconfig.GroupQualifiedName(typeConfig.GetPlacement()),
	}
	overrideAPIResource := typeConfig.GetOverride()
	if overrideAPIResource != nil {
		names = append(names, typeconfig.GroupQualifiedName(*overrideAPIResource))
	}
	return names
}
