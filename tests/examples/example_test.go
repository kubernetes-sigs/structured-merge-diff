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

	"sigs.k8s.io/structured-merge-diff/tests/framework"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// TestExample shows how to use the test framework
func TestExample(t *testing.T) {
	state := &framework.State{Implementation: &mockImplementation{}}

	err := state.Apply(`
list:
- a
- b
- c
`, "default", false)
	err = framework.CheckExpectedConflicts(err, nil)
	if err != nil {
		t.Errorf("encountered unexpected conflicts: %v", err)
	}

	err = state.Apply(`
list:
- a
- b
- c
- d
	`, "default", false)

	err = framework.CheckExpectedConflicts(err, framework.Conflicts{
		framework.Conflict{Field: "someConflict"},
	})
	if err != nil {
		t.Errorf("encountered unexpected conflicts: %v", err)
	}
}

type mockImplementation struct{}

func (i *mockImplementation) Apply(live, config typed.YAMLObject, workflow string, force bool) (typed.YAMLObject, error) {
	return "", nil
}

func (i *mockImplementation) Update(live, config typed.YAMLObject, workflow string) (typed.YAMLObject, error) {
	return "", nil
}
