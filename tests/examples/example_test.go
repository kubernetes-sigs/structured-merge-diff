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

package tests

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/tests/framework"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// TestExample shows how to use the test framework
func TestExample(t *testing.T) {
	state := &framework.State{Updater: &merge.Updater{}}
	parser, err := typed.NewParser(`types:
- name: lists
  struct:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}

	config, err := parser.FromYAML(`
list:
- a
- b
- c
`, "lists")
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config, err = parser.FromYAML(`
list:
- a
- b
- c
- d`, "lists")
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}
	err = state.Apply(config, "default", false)

	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// The following is wrong because the code doesn't work yet.
	_, err = state.Live.Compare(config)
	if err == nil {
		t.Fatalf("Succeeded to compare live with config")
	}
}
