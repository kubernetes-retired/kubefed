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

package federate

import (
	"context"
	"io"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/enable"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

var (
	federate_long = `
		Federate creates a federated resource from a kubernetes resource.
		The target resource must exist in the cluster hosting the federation
		control plane. The control plane must have a FederatedTypeConfig
		for the type of the kubernetes resource. The new federated resource
		will be created with the same name and namespace (if namespaced) as
		the kubernetes resource.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the --host-cluster-context
		flag otherwise.`

	federate_example = `
		# Federate resource named "my-dep" in namespace "my-ns" of type identified by FederatedTypeConfig "deployment.apps"
		kubefed2 federate deployment.apps my-dep -n "my-ns" --host-cluster-context=cluster1`
	// TODO(irfanurrehman): implement —contents flag applicable to namespaces
)

type federateResource struct {
	options.SubcommandOptions
	typeName          string
	resourceName      string
	resourceNamespace string
}

func (j *federateResource) Bind(flags *pflag.FlagSet) {
	flags.StringVarP(&j.resourceNamespace, "namespace", "n", "default", "The namespace of the resource to federate.")
}

// Complete ensures that options are valid.
func (j *federateResource) Complete(args []string) error {
	if len(args) == 0 {
		return errors.New("FEDERATED-TYPE-NAME is required")
	}
	j.typeName = args[0]

	if len(args) == 1 {
		return errors.New("RESOURCE-NAME is required")
	}
	j.resourceName = args[1]

	if j.typeName == ctlutil.NamespaceName {
		glog.Infof("Resource to federate is a namespace. Given namespace will itself be the container for the federated namespace")
		j.resourceNamespace = ""
	}

	return nil
}

// NewCmdFederateResource defines the `federate` command that federates a
// Kubernetes resource of the given kubernetes type.
func NewCmdFederateResource(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &federateResource{}

	cmd := &cobra.Command{
		Use:     "federate FEDERATED-TYPE-NAME RESOURCE-NAME",
		Short:   "Federate creates a federated resource from a kubernetes resource",
		Long:    federate_long,
		Example: federate_example,
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

// Run is the implementation of the `federate resource` command.
func (j *federateResource) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get host cluster config")
	}

	qualifiedTypeName := ctlutil.QualifiedName{
		Namespace: j.FederationNamespace,
		Name:      j.typeName,
	}

	qualifiedResourceName := ctlutil.QualifiedName{
		Namespace: j.resourceNamespace,
		Name:      j.resourceName,
	}

	_, err = FederateResource(hostConfig, qualifiedTypeName, qualifiedResourceName, j.DryRun)
	return err
}

func FederateResource(hostConfig *rest.Config, qualifiedTypeName, qualifiedResourceName ctlutil.QualifiedName, dryrun bool) (*unstructured.Unstructured, error) {
	typeConfig, err := lookupTypeDetails(hostConfig, qualifiedTypeName)
	if err != nil {
		return nil, err
	}

	templateResource, err := getTargetResource(hostConfig, typeConfig, qualifiedResourceName)
	if err != nil {
		return nil, err
	}

	return createFedResource(hostConfig, typeConfig, templateResource, dryrun)
}

func lookupTypeDetails(config *rest.Config, qualifiedTypeName ctlutil.QualifiedName) (*fedv1a1.FederatedTypeConfig, error) {
	client, err := genericclient.New(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get federation client")
	}

	typeConfig := &fedv1a1.FederatedTypeConfig{}
	err = client.Get(context.TODO(), typeConfig, qualifiedTypeName.Namespace, qualifiedTypeName.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving FederatedTypeConfig %q", qualifiedTypeName)
	}

	_, err = enable.LookupAPIResource(config, typeConfig.Name, typeConfig.APIVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving API resource for FederatedTypeConfig %q", qualifiedTypeName)
	}

	glog.Infof("FederatedTypeConfig: %q found", qualifiedTypeName)
	return typeConfig, nil
}

func getTargetResource(hostConfig *rest.Config, typeConfig *fedv1a1.FederatedTypeConfig, qualifiedName ctlutil.QualifiedName) (*unstructured.Unstructured, error) {
	targetAPIResource := typeConfig.GetTarget()
	targetClient, err := ctlutil.NewResourceClient(hostConfig, &targetAPIResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating client for %s", targetAPIResource.Kind)
	}

	kind := targetAPIResource.Kind
	resource, err := targetClient.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving target %s %q", kind, qualifiedName)
	}

	glog.Infof("Target %s %q found", kind, qualifiedName)
	return resource, nil
}

func createFedResource(hostConfig *rest.Config, typeConfig *fedv1a1.FederatedTypeConfig, template *unstructured.Unstructured, dryrun bool) (*unstructured.Unstructured, error) {
	fedAPIResource := typeConfig.GetFederatedType()
	fedClient, err := ctlutil.NewResourceClient(hostConfig, &fedAPIResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating client for %s", fedAPIResource.Kind)
	}

	targetKind := typeConfig.GetTarget().Kind
	if targetKind == ctlutil.ServiceAccountKind {
		unstructured.RemoveNestedField(template.Object, ctlutil.SecretsField)
	}

	qualifiedName := ctlutil.NewQualifiedName(template)
	resourceNamespace := ""
	if typeConfig.GetTarget().Kind == ctlutil.NamespaceKind {
		resourceNamespace = qualifiedName.Name
	} else {
		resourceNamespace = qualifiedName.Namespace
	}
	fedResource := &unstructured.Unstructured{}
	SetBasicMetaFields(fedResource, fedAPIResource, qualifiedName.Name, resourceNamespace, "")
	RemoveUnwantedFields(template)

	fedKind := fedAPIResource.Kind
	qualifiedFedName := ctlutil.NewQualifiedName(fedResource)
	err = unstructured.SetNestedField(fedResource.Object, template.Object, ctlutil.SpecField, ctlutil.TemplateField)
	if err != nil {
		return nil, errors.Wrapf(err, "Error setting template into %s %q ", fedKind, qualifiedFedName)
	}

	err = unstructured.SetNestedStringMap(fedResource.Object, map[string]string{}, ctlutil.SpecField, ctlutil.PlacementField, ctlutil.ClusterSelectorField, ctlutil.MatchLabelsField)
	if err != nil {
		return nil, errors.Wrapf(err, "Error setting placement into %s %q", fedKind, qualifiedFedName)
	}

	var createdResource *unstructured.Unstructured = nil
	if !dryrun {
		createdResource, err = fedClient.Resources(resourceNamespace).Create(fedResource, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "Error creating %s %q", fedKind, qualifiedFedName)
		}
	}

	glog.Infof("Successfully created a %s from %s %q", fedKind, typeConfig.GetTarget().Kind, qualifiedName)
	return createdResource, nil
}
