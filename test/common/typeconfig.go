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

package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/typeconfig"
	"github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	"github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	typeConfigs         []typeconfig.Interface
	namespaceTypeConfig typeconfig.Interface
)

func TypeConfigsOrDie(tl TestLogger) []typeconfig.Interface {
	if typeConfigs == nil {
		var err error
		typeConfigs, err = FederatedTypeConfigs()
		if err != nil {
			tl.Fatalf("Error loading type configs: %v", err)
		}
	}
	return typeConfigs
}

func NamespaceTypeConfigOrDie(tl TestLogger) typeconfig.Interface {
	if namespaceTypeConfig == nil {
		for _, typeConfig := range TypeConfigsOrDie(tl) {
			if typeConfig.GetTarget().Kind == util.NamespaceKind {
				namespaceTypeConfig = typeConfig
				break
			}
		}
		if namespaceTypeConfig == nil {
			tl.Fatalf("Unable to find namespace type config")
		}
	}
	return namespaceTypeConfig
}

func FederatedTypeConfigs() ([]typeconfig.Interface, error) {
	path := typeConfigPath()
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	typeConfigs := []typeconfig.Interface{}
	for _, f := range files {
		filename := filepath.Join(path, f.Name())
		if !strings.HasSuffix(filename, ".yaml") {
			continue
		}
		obj, err := typeConfigFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("Error loading %s: %v", filename, err)
		}
		typeConfigs = append(typeConfigs, obj)
	}
	return typeConfigs, nil
}

func typeConfigPath() string {
	// Get the directory of this file
	_, filename, _, _ := runtime.Caller(0)
	commonPath := filepath.Dir(filename)
	testPath := filepath.Dir(commonPath)
	rootPath := filepath.Dir(testPath)
	return filepath.Join(rootPath, "config", "federatedtypes")
}

func typeConfigFromFile(filename string) (typeconfig.Interface, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewYAMLToJSONDecoder(f)
	obj := &v1alpha1.FederatedTypeConfig{}
	err = decoder.Decode(obj)
	if err != nil {
		return nil, err
	}
	v1alpha1.SetFederatedTypeConfigDefaults(obj)
	return obj, nil
}
