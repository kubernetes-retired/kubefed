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

package kubefed2

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/enable"
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
		# Disable propagation of the kubernetes API type 'Deployment', named
		in FederatedTypeConfig as 'deployments.apps'
		kubefed2 disable deployments.apps

		# Disable propagation of the kubernetes API type 'Deployment', named
		in FederatedTypeConfig as 'deployments.apps', and delete corresponding
		Federated API resources
		kubefed2 disable deployments.apps --delete-from-api`
)

type disableType struct {
	options.GlobalSubcommandOptions
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

// NewCmdTypeDisable defines the `disable` command that
// disables federation of a Kubernetes API type.
func NewCmdTypeDisable(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &disableType{}

	cmd := &cobra.Command{
		Use:     "disable NAME",
		Short:   "Disables propagation of a Kubernetes API type",
		Long:    disable_long,
		Example: disable_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				glog.Fatalf("Error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("Error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.GlobalSubcommandBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *disableType) Complete(args []string) error {
	if len(args) == 0 {
		return errors.New("NAME is required")
	}
	j.targetName = args[0]

	return nil
}

// Run is the implementation of the `disable` command.
func (j *disableType) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get host cluster config")
	}

	// If . is specified, the target name is assumed as a group qualified name.
	// In such case, ignore the lookup to make sure deletion of a federatedtypeconfig
	// for which the corresponding target has been removed.
	name := j.targetName
	if !strings.Contains(j.targetName, ".") {
		apiResource, err := enable.LookupAPIResource(hostConfig, j.targetName, "")
		if err != nil {
			return err
		}
		name = typeconfig.GroupQualifiedName(*apiResource)
	}

	typeConfigName := ctlutil.QualifiedName{
		Namespace: j.FederationNamespace,
		Name:      name,
	}

	return DisableFederation(cmdOut, hostConfig, typeConfigName, j.delete, j.DryRun)
}

func DisableFederation(cmdOut io.Writer, config *rest.Config, typeConfigName ctlutil.QualifiedName, delete, dryRun bool) error {
	client, err := genericclient.New(config)
	if err != nil {
		return errors.Wrap(err, "Failed to get federation clientset")
	}
	typeConfig := &fedv1a1.FederatedTypeConfig{}
	err = client.Get(context.TODO(), typeConfig, typeConfigName.Namespace, typeConfigName.Name)
	if err != nil {
		return errors.Wrapf(err, "Error retrieving FederatedTypeConfig %q", typeConfigName)
	}

	if dryRun {
		return nil
	}

	write := func(data string) {
		if cmdOut == nil {
			return
		}

		if _, err := cmdOut.Write([]byte(data)); err != nil {
			glog.Fatalf("Unexpected err: %v\n", err)
		}
	}

	if typeConfig.Spec.PropagationEnabled {
		typeConfig.Spec.PropagationEnabled = false
		err = client.Update(context.TODO(), typeConfig)
		if err != nil {
			return errors.Wrapf(err, "Error disabling propagation for FederatedTypeConfig %q", typeConfigName)
		}
		write(fmt.Sprintf("Disabled propagation for FederatedTypeConfig %q\n", typeConfigName))
	} else {
		write(fmt.Sprintf("Propagation already disabled for FederatedTypeConfig %q\n", typeConfigName))
	}
	if !delete {
		return nil
	}

	// TODO(marun) consider waiting for the sync controller to be stopped before attempting deletion
	err = deleteFederatedType(config, typeConfig, write)
	if err != nil {
		return err
	}

	err = client.Delete(context.TODO(), typeConfig, typeConfig.Namespace, typeConfig.Name)
	if err != nil {
		return errors.Wrapf(err, "Error deleting FederatedTypeConfig %q", typeConfigName)
	}
	write(fmt.Sprintf("federatedtypeconfig %q deleted\n", typeConfigName))

	return nil
}

func deleteFederatedType(config *rest.Config, typeConfig typeconfig.Interface, write func(string)) error {
	client, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Error creating crd client")
	}

	crdName := typeconfig.GroupQualifiedName(typeConfig.GetFederatedType())
	err = client.CustomResourceDefinitions().Delete(crdName, nil)
	if err != nil {
		return errors.Wrapf(err, "Error deleting crd %q", crdName)
	}
	write(fmt.Sprintf("customresourcedefinition %q deleted\n", crdName))
	return nil
}
