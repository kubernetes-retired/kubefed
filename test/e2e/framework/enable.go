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

package framework

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"

	kfenable "github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/enable"
	"github.com/kubernetes-sigs/federation-v2/test/common"
)

func LoadEnableTypeDirectives(tl common.TestLogger) []*kfenable.EnableTypeDirective {
	path := enableTypeDirectivesPath(tl)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		tl.Fatalf("Error reading EnableTypeDirective resources from path %q: %v", path, err)
	}
	enableTypeDirectives := []*kfenable.EnableTypeDirective{}
	suffix := ".yaml"
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), suffix) {
			continue
		}
		filename := filepath.Join(path, file.Name())
		obj := kfenable.NewEnableTypeDirective()
		err := kfenable.DecodeYAMLFromFile(filename, obj)
		if err != nil {
			tl.Fatalf("Error loading EnableTypeDirective from file %q: %v", filename, err)
		}
		enableTypeDirectives = append(enableTypeDirectives, obj)
	}
	return enableTypeDirectives
}

func enableTypeDirectivesPath(tl common.TestLogger) string {
	// Get the directory of the current executable
	_, filename, _, _ := runtime.Caller(0)
	managedPath := filepath.Dir(filename)
	path, err := filepath.Abs(fmt.Sprintf("%s/../../../config/enabletypedirectives", managedPath))
	if err != nil {
		tl.Fatalf("Error discovering the path to FederatedType resources: %v", err)
	}
	return path
}
