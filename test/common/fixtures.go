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
	"bytes"
	"strings"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	kfenable "sigs.k8s.io/kubefed/pkg/kubefedctl/enable"
)

var fixtures map[string]*unstructured.Unstructured

func TypeConfigFixturesOrDie(tl TestLogger) map[string]*unstructured.Unstructured {
	if fixtures == nil {
		var err error
		fixtures, err = typeConfigFixtures()
		if err != nil {
			tl.Fatalf("Error loading type config fixtures: %v", err)
		}
	}
	return fixtures
}

func typeConfigFixtures() (map[string]*unstructured.Unstructured, error) {
	fixtures := make(map[string]*unstructured.Unstructured)
	for _, file := range AssetNames() {
		fixture := &unstructured.Unstructured{}
		typeConfigName, err := DecodeYamlFromBindata(file, "test/common/fixtures/", fixture)
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading a fixture from %q", typeConfigName)
		}
		if len(typeConfigName) != 0 {
			fixtures[typeConfigName] = fixture
		}
	}

	return fixtures, nil
}

func DecodeYamlFromBindata(filename, prefix string, obj interface{}) (string, error) {
	if !strings.HasPrefix(filename, prefix) {
		return "", nil
	}
	yaml := MustAsset(filename)
	name := strings.TrimSuffix(strings.TrimPrefix(filename, prefix), ".yaml")
	err := kfenable.DecodeYAML(bytes.NewBuffer(yaml), obj)
	if err != nil {
		return "", err
	}
	return name, nil
}
