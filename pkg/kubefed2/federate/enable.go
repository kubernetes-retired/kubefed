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
	"errors"
	"fmt"
	"io"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextv1b1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	apicommon "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/common"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	ctlutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefed2/util"
)

const (
	defaultComparisonField  = apicommon.ResourceVersionField
	defaultPrimitiveGroup   = "primitives.federation.k8s.io"
	defaultPrimitiveVersion = "v1alpha1"
)

var (
	enable_long = `
		Enables a Kubernetes API type (including a CRD) to be propagated
		to members of a federation.  Federation primitives will be
		generated as CRDs and a FederatedTypeConfig will be created to
		configure a sync controller.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the
		--host-cluster-context flag otherwise.`

	enable_example = `
		# Enable federation of Services with service type overrideable
		kubefed2 federate enable Service --override-paths=spec.type --host-cluster-context=cluster1`
)

type enableType struct {
	options.SubcommandOptions
	enableTypeOptions
}

type enableTypeOptions struct {
	targetName         string
	targetVersion      string
	rawComparisonField string
	primitiveVersion   string
	primitiveGroup     string
	output             string
	outputYAML         bool
	filename           string
	federateDirective  *FederateDirective
}

// Bind adds the join specific arguments to the flagset passed in as an
// argument.
func (o *enableTypeOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.targetVersion, "version", "", "Optional, the API version of the target type.")
	flags.StringVar(&o.rawComparisonField, "comparison-field", string(defaultComparisonField),
		fmt.Sprintf("The field in the target type to compare for equality. Valid values are %q (default) and %q.",
			apicommon.ResourceVersionField, apicommon.GenerationField,
		),
	)
	flags.StringVar(&o.primitiveGroup, "primitive-group", defaultPrimitiveGroup, "The name of the API group to use for generated federation primitives.")
	flags.StringVar(&o.primitiveVersion, "primitive-version", defaultPrimitiveVersion, "The API version to use for generated federation primitives.")
	flags.StringVarP(&o.output, "output", "o", "", "If provided, the resources that will be created in the API will be output to stdout in the provided format.  Valid values are ['yaml'].")
	flags.StringVarP(&o.filename, "filename", "f", "", "If provided, the command will be configured from the provided yaml file.  Only --output wll be accepted from the command line")
}

// NewCmdFederateEnable defines the `federate enable` command that
// enables federation of a Kubernetes API type.
func NewCmdFederateEnable(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &enableType{}

	cmd := &cobra.Command{
		Use:     "enable NAME",
		Short:   "Enables propagation of a Kubernetes API type",
		Long:    enable_long,
		Example: enable_example,
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
func (j *enableType) Complete(args []string) error {
	j.federateDirective = NewFederateDirective()
	fd := j.federateDirective

	if j.output == "yaml" {
		j.outputYAML = true
	} else if len(j.output) > 0 {
		return fmt.Errorf("Invalid value for --output: %s", j.output)
	}

	if len(j.filename) > 0 {
		err := DecodeYAMLFromFile(j.filename, fd)
		if err != nil {
			return fmt.Errorf("Failed to load yaml from file %q: %v", j.filename, err)
		}
		return nil
	}

	if len(args) == 0 {
		return errors.New("NAME is required")
	}
	fd.Name = args[0]

	if j.rawComparisonField == string(apicommon.ResourceVersionField) ||
		j.rawComparisonField == string(apicommon.GenerationField) {

		fd.Spec.ComparisonField = apicommon.VersionComparisonField(j.rawComparisonField)
	} else {
		return fmt.Errorf("comparison field must be %q or %q",
			apicommon.ResourceVersionField, apicommon.GenerationField,
		)
	}
	if len(j.targetVersion) > 0 {
		fd.Spec.TargetVersion = j.targetVersion
	}
	if len(j.primitiveGroup) > 0 {
		fd.Spec.PrimitiveGroup = j.primitiveGroup
	}
	if len(j.primitiveVersion) > 0 {
		fd.Spec.PrimitiveVersion = j.primitiveVersion
	}

	return nil
}

// Run is the implementation of the `federate enable` command.
func (j *enableType) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return fmt.Errorf("Failed to get host cluster config: %v", err)
	}

	resources, err := GetResources(hostConfig, j.federateDirective)
	if err != nil {
		return err
	}

	if j.outputYAML {
		concreteTypeConfig := resources.TypeConfig.(*fedv1a1.FederatedTypeConfig)
		objects := []pkgruntime.Object{concreteTypeConfig}
		for _, crd := range resources.CRDs {
			objects = append(objects, crd)
		}
		err := writeObjectsToYAML(objects, cmdOut)
		if err != nil {
			return fmt.Errorf("Failed to write objects to YAML: %v", err)
		}
	}

	if j.DryRun {
		// Avoid mutating the API
		return nil
	}

	err = CreateResources(cmdOut, hostConfig, resources, j.FederationNamespace)
	if err != nil {
		return err
	}

	return nil
}

type typeResources struct {
	TypeConfig typeconfig.Interface
	CRDs       []*apiextv1b1.CustomResourceDefinition
}

func GetResources(config *rest.Config, federateDirective *FederateDirective) (*typeResources, error) {
	apiResource, err := LookupAPIResource(config, federateDirective.Name, federateDirective.Spec.TargetVersion)
	if err != nil {
		return nil, err
	}
	glog.V(2).Infof("Found resource %q", resourceKey(*apiResource))

	typeConfig := typeConfigForTarget(*apiResource, federateDirective)

	accessor, err := newSchemaAccessor(config, *apiResource)
	if err != nil {
		return nil, fmt.Errorf("Error initializing validation schema accessor: %v", err)
	}
	crds, err := primitiveCRDs(typeConfig, accessor)
	if err != nil {
		return nil, err
	}

	return &typeResources{
		TypeConfig: typeConfig,
		CRDs:       crds,
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

	crdClient, err := apiextv1b1client.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Failed to create crd clientset: %v", err)
	}
	for _, crd := range resources.CRDs {
		_, err := crdClient.CustomResourceDefinitions().Create(crd)
		if err != nil {
			return fmt.Errorf("Error creating CRD %q: %v", crd.Name, err)
		}
		write(fmt.Sprintf("customresourcedefinition.apiextensions.k8s.io/%s created\n", crd.Name))
	}

	fedClient, err := util.FedClientset(config)
	if err != nil {
		return fmt.Errorf("Failed to get federation clientset: %v", err)
	}
	concreteTypeConfig := resources.TypeConfig.(*fedv1a1.FederatedTypeConfig)
	_, err = fedClient.CoreV1alpha1().FederatedTypeConfigs(namespace).Create(concreteTypeConfig)
	if err != nil {
		return fmt.Errorf("Error creating FederatedTypeConfig %q: %v", concreteTypeConfig.Name, err)
	}
	write(fmt.Sprintf("federatedtypeconfig.core.federation.k8s.io/%s created in namespace %s\n", concreteTypeConfig.Name, namespace))

	return nil
}

func typeConfigForTarget(apiResource metav1.APIResource, federateDirective *FederateDirective) typeconfig.Interface {
	spec := federateDirective.Spec
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
			ComparisonField:    spec.ComparisonField,
			PropagationEnabled: true,
			Placement: fedv1a1.APIResource{
				Kind: fmt.Sprintf("Federated%sPlacement", kind),
			},
			Override: &fedv1a1.APIResource{
				Kind: fmt.Sprintf("Federated%sOverride", kind),
			},
		},
	}
	if typeConfig.Name == ctlutil.NamespaceName {
		typeConfig.Spec.Template = typeConfig.Spec.Target
		typeConfig.Spec.Placement.Group = spec.PrimitiveGroup
		typeConfig.Spec.Placement.Version = spec.PrimitiveVersion
		typeConfig.Spec.Override.Group = spec.PrimitiveGroup
		typeConfig.Spec.Override.Version = spec.PrimitiveVersion
	} else {
		typeConfig.Spec.Template = fedv1a1.APIResource{
			Group:      spec.PrimitiveGroup,
			Version:    spec.PrimitiveVersion,
			Kind:       fmt.Sprintf("Federated%s", kind),
			PluralName: fmt.Sprintf("federated%s", pluralName),
		}
	}
	// Set defaults that would normally be set by the api
	fedv1a1.SetFederatedTypeConfigDefaults(typeConfig)
	return typeConfig
}

func primitiveCRDs(typeConfig typeconfig.Interface, accessor schemaAccessor) ([]*apiextv1b1.CustomResourceDefinition, error) {
	crds := []*apiextv1b1.CustomResourceDefinition{}

	// Namespaces do not require a template
	if typeConfig.GetTarget().Kind != ctlutil.NamespaceKind {
		templateSchema, err := templateValidationSchema(accessor)
		if err != nil {
			return nil, err
		}
		crds = append(crds, CrdForAPIResource(typeConfig.GetTemplate(), templateSchema))
	}

	placementSchema := placementValidationSchema()
	crds = append(crds, CrdForAPIResource(typeConfig.GetPlacement(), placementSchema))

	overrideSchema := overrideValidationSchema()
	crds = append(crds, CrdForAPIResource(*typeConfig.GetOverride(), overrideSchema))

	return crds, nil
}

func writeObjectsToYAML(objects []pkgruntime.Object, w io.Writer) error {
	for _, obj := range objects {
		w.Write([]byte("---\n"))
		err := writeObjectToYAML(obj, w)
		if err != nil {
			return fmt.Errorf("Error encoding resource to yaml: %v ", err)
		}
	}
	return nil
}

func writeObjectToYAML(obj pkgruntime.Object, w io.Writer) error {
	json, err := jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(obj)
	if err != nil {
		return err
	}

	// Convert to unstructured to filter status out of the output.  If
	// status is included in the yaml, attempting to create it in a
	// kube API will cause an error.
	unstructuredObj := &unstructured.Unstructured{}
	unstructured.UnstructuredJSONScheme.Decode(json, nil, unstructuredObj)
	delete(unstructuredObj.Object, "status")
	// Also remove unnecessary field
	metadataMap := unstructuredObj.Object["metadata"].(map[string]interface{})
	delete(metadataMap, "creationTimestamp")

	updatedJSON, err := unstructuredObj.MarshalJSON()
	if err != nil {
		return err
	}

	data, err := yaml.JSONToYAML(updatedJSON)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
