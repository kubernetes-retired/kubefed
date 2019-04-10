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

package enable

import (
	"context"
	"fmt"
	"io"

	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "github.com/kubernetes-sigs/federation-v2/pkg/client/generic"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

const (
	DefaultFederationGroup   = "types.federation.k8s.io"
	DefaultFederationVersion = "v1alpha1"
)

var (
	enable_long = `
		Enables a Kubernetes API type (including a CRD) to be propagated
		to members of a federation.  A CRD for the federated type will be
		generated and a FederatedTypeConfig will be created to configure
		a sync controller.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the
		--host-cluster-context flag otherwise.`

	enable_example = `
		# Enable federation of Deployments
		kubefed2 enable deployments.apps --host-cluster-context=cluster1`
)

type enableType struct {
	options.SubcommandOptions
	enableTypeOptions
}

type enableTypeOptions struct {
	targetName          string
	targetVersion       string
	federationVersion   string
	federationGroup     string
	output              string
	outputYAML          bool
	filename            string
	enableTypeDirective *EnableTypeDirective
}

// Bind adds the join specific arguments to the flagset passed in as an
// argument.
func (o *enableTypeOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.targetVersion, "version", "", "Optional, the API version of the target type.")
	flags.StringVar(&o.federationGroup, "federation-group", DefaultFederationGroup, "The name of the API group to use for the generated federation type.")
	flags.StringVar(&o.federationVersion, "federation-version", DefaultFederationVersion, "The API version to use for the generated federation type.")
	flags.StringVarP(&o.output, "output", "o", "", "If provided, the resources that would be created in the API by the command are instead output to stdout in the provided format.  Valid values are ['yaml'].")
	flags.StringVarP(&o.filename, "filename", "f", "", "If provided, the command will be configured from the provided yaml file.  Only --output will be accepted from the command line")
}

// NewCmdTypeEnable defines the `enable` command that
// enables federation of a Kubernetes API type.
func NewCmdTypeEnable(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &enableType{}

	cmd := &cobra.Command{
		Use:     "enable NAME",
		Short:   "Enables propagation of a Kubernetes API type",
		Long:    enable_long,
		Example: enable_example,
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
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

// Complete ensures that options are valid and marshals them if necessary.
func (j *enableType) Complete(args []string) error {
	j.enableTypeDirective = NewEnableTypeDirective()
	fd := j.enableTypeDirective

	if j.output == "yaml" {
		j.outputYAML = true
	} else if len(j.output) > 0 {
		return errors.Errorf("Invalid value for --output: %s", j.output)
	}

	if len(j.filename) > 0 {
		err := DecodeYAMLFromFile(j.filename, fd)
		if err != nil {
			return errors.Wrapf(err, "Failed to load yaml from file %q", j.filename)
		}
		return nil
	}

	if len(args) == 0 {
		return errors.New("NAME is required")
	}
	fd.Name = args[0]

	if len(j.targetVersion) > 0 {
		fd.Spec.TargetVersion = j.targetVersion
	}
	if len(j.federationGroup) > 0 {
		fd.Spec.FederationGroup = j.federationGroup
	}
	if len(j.federationVersion) > 0 {
		fd.Spec.FederationVersion = j.federationVersion
	}

	return nil
}

// Run is the implementation of the `enable` command.
func (j *enableType) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get host cluster config")
	}

	resources, err := GetResources(hostConfig, j.enableTypeDirective)
	if err != nil {
		return err
	}

	if j.outputYAML {
		concreteTypeConfig := resources.TypeConfig.(*fedv1a1.FederatedTypeConfig)
		objects := []pkgruntime.Object{concreteTypeConfig, resources.CRD}
		err := writeObjectsToYAML(objects, cmdOut)
		if err != nil {
			return errors.Wrap(err, "Failed to write objects to YAML")
		}
		// -o yaml implies dry run
		return nil
	}

	if j.DryRun {
		// Avoid mutating the API
		return nil
	}

	return CreateResources(cmdOut, hostConfig, resources, j.FederationNamespace)
}

type typeResources struct {
	TypeConfig typeconfig.Interface
	CRD        *apiextv1b1.CustomResourceDefinition
}

func GetResources(config *rest.Config, enableTypeDirective *EnableTypeDirective) (*typeResources, error) {
	apiResource, err := LookupAPIResource(config, enableTypeDirective.Name, enableTypeDirective.Spec.TargetVersion)
	if err != nil {
		return nil, err
	}
	glog.V(2).Infof("Found type %q", resourceKey(*apiResource))

	typeConfig := GenerateTypeConfigForTarget(*apiResource, enableTypeDirective)

	accessor, err := newSchemaAccessor(config, *apiResource)
	if err != nil {
		return nil, errors.Wrap(err, "Error initializing validation schema accessor")
	}

	shortNames := []string{}
	for _, shortName := range apiResource.ShortNames {
		shortNames = append(shortNames, fmt.Sprintf("f%s", shortName))
	}

	crd := federatedTypeCRD(typeConfig, accessor, shortNames)

	return &typeResources{
		TypeConfig: typeConfig,
		CRD:        crd,
	}, nil
}

// TODO(marun) Allow updates to the configuration for a type that has
// already been enabled for federation.  This would likely involve
// updating the version of the target type and the validation of the schema.
func CreateResources(cmdOut io.Writer, config *rest.Config, resources *typeResources, namespace string) error {
	write := func(data string) {
		if cmdOut != nil {
			cmdOut.Write([]byte(data))
		}
	}

	hostClientset, err := util.HostClientset(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create host clientset")
	}
	_, err = hostClientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "Federation system namespace %q does not exist", namespace)
	} else if err != nil {
		return errors.Wrapf(err, "Error attempting to determine whether federation system namespace %q exists", namespace)
	}

	crdClient, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "Failed to create crd clientset")
	}

	existingCRD, err := crdClient.CustomResourceDefinitions().Get(resources.CRD.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = crdClient.CustomResourceDefinitions().Create(resources.CRD)
		if err != nil {
			return errors.Wrapf(err, "Error creating CRD %q", resources.CRD.Name)
		}
		write(fmt.Sprintf("customresourcedefinition.apiextensions.k8s.io/%s created\n", resources.CRD.Name))
	} else if err != nil {
		return errors.Wrapf(err, "Error getting CRD %q", resources.CRD.Name)
	} else {
		existingCRD.Spec = resources.CRD.Spec
		_, err = crdClient.CustomResourceDefinitions().Update(existingCRD)
		if err != nil {
			return errors.Wrapf(err, "Error updating CRD %q", resources.CRD.Name)
		}
		write(fmt.Sprintf("customresourcedefinition.apiextensions.k8s.io/%s updated\n", resources.CRD.Name))
	}

	client, err := genericclient.New(config)
	if err != nil {
		return errors.Wrap(err, "Failed to get federation clientset")
	}
	concreteTypeConfig := resources.TypeConfig.(*fedv1a1.FederatedTypeConfig)
	concreteTypeConfig.Namespace = namespace
	err = client.Create(context.TODO(), concreteTypeConfig)
	if err != nil {
		return errors.Wrapf(err, "Error creating FederatedTypeConfig %q", concreteTypeConfig.Name)
	}
	write(fmt.Sprintf("federatedtypeconfig.core.federation.k8s.io/%s created in namespace %s\n", concreteTypeConfig.Name, namespace))

	return nil
}

func GenerateTypeConfigForTarget(apiResource metav1.APIResource, enableTypeDirective *EnableTypeDirective) typeconfig.Interface {
	spec := enableTypeDirective.Spec
	kind := apiResource.Kind
	pluralName := apiResource.Name
	typeConfig := &fedv1a1.FederatedTypeConfig{
		// Explicitly including TypeMeta will ensure it will be
		// serialized properly to yaml.
		TypeMeta: metav1.TypeMeta{
			Kind:       "FederatedTypeConfig",
			APIVersion: "core.federation.k8s.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: typeconfig.GroupQualifiedName(apiResource),
		},
		Spec: fedv1a1.FederatedTypeConfigSpec{
			Target: fedv1a1.APIResource{
				Version: apiResource.Version,
				Kind:    kind,
			},
			Namespaced:         apiResource.Namespaced,
			PropagationEnabled: true,
			FederatedType: fedv1a1.APIResource{
				Group:      spec.FederationGroup,
				Version:    spec.FederationVersion,
				Kind:       fmt.Sprintf("Federated%s", kind),
				PluralName: fmt.Sprintf("federated%s", pluralName),
			},
		},
	}

	// Set defaults that would normally be set by the api
	fedv1a1.SetFederatedTypeConfigDefaults(typeConfig)
	return typeConfig
}

func federatedTypeCRD(typeConfig typeconfig.Interface, accessor schemaAccessor, shortNames []string) *apiextv1b1.CustomResourceDefinition {
	var templateSchema map[string]apiextv1b1.JSONSchemaProps
	templateSchema = accessor.templateSchema()
	schema := federatedTypeValidationSchema(templateSchema)
	return CrdForAPIResource(typeConfig.GetFederatedType(), schema, shortNames)
}

func writeObjectsToYAML(objects []pkgruntime.Object, w io.Writer) error {
	for _, obj := range objects {
		w.Write([]byte("---\n"))
		err := writeObjectToYAML(obj, w)
		if err != nil {
			return errors.Wrap(err, "Error encoding object to yaml")
		}
	}
	return nil
}

func writeObjectToYAML(obj pkgruntime.Object, w io.Writer) error {
	json, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(obj)
	if err != nil {
		return err
	}

	unstructuredObj := &unstructured.Unstructured{}
	unstructured.UnstructuredJSONScheme.Decode(json, nil, unstructuredObj)
	return util.WriteUnstructuredToYaml(unstructuredObj, w)
}
