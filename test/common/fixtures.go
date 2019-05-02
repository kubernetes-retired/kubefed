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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	kfenable "github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/enable"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	path := fixturePath()
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrapf(err, "Error reading fixture from path %q", path)
	}

	fixtures := make(map[string]*unstructured.Unstructured)
	suffix := ".yaml"
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), suffix) {
			continue
		}

		typeConfigName := strings.TrimSuffix(file.Name(), suffix)

		filename := filepath.Join(path, file.Name())
		fixture := &unstructured.Unstructured{}
		err := kfenable.DecodeYAMLFromFile(filename, fixture)
		if err != nil {
			return nil, errors.Wrapf(err, "Error reading fixture for %q", typeConfigName)
		}
		fixtures[typeConfigName] = fixture
	}

	return fixtures, nil
}
