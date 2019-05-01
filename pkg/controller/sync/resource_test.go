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

package sync

import (
	"strings"
	"testing"

	kfenable "github.com/kubernetes-sigs/federation-v2/pkg/kubefedctl/enable"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGetTemplateHash(t *testing.T) {
	template := &unstructured.Unstructured{}
	yaml := `
kind: foo
spec:
  template:
    spec:
      foo:
`
	err := kfenable.DecodeYAML(strings.NewReader(yaml), template)
	if err != nil {
		t.Fatalf("An unexpected error occurred: %v", err)
	}
	hash, err := GetTemplateHash(template.Object)
	if err != nil {
		t.Fatalf("An unexpected error occurred: %v", err)
	}
	expectedHash := "a5b8d4352d5aed51c51b93900258ccf3"
	if hash != expectedHash {
		t.Fatalf("Expected %s, got %s", expectedHash, hash)
	}
}
