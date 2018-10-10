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

package framework_test

import (
	"errors"
	"testing"

	"sigs.k8s.io/structured-merge-diff/tests/framework"
)

type mockImplementation struct {
	applyFunc  func(live, config framework.YAMLObject, workflow string, force bool) (framework.YAMLObject, error)
	updateFunc func(live, config framework.YAMLObject, workflow string) (framework.YAMLObject, error)

	applyFuncCallCount  int
	updateFuncCallCount int
}

func (i *mockImplementation) Apply(live, config framework.YAMLObject, workflow string, force bool) (framework.YAMLObject, error) {
	i.applyFuncCallCount++
	if i.applyFunc != nil {
		return i.applyFunc(live, config, workflow, force)
	}
	return "", nil
}

func (i *mockImplementation) Update(live, config framework.YAMLObject, workflow string) (framework.YAMLObject, error) {
	i.updateFuncCallCount++
	if i.updateFunc != nil {
		return i.updateFunc(live, config, workflow)
	}
	return "", nil
}

func TestState(t *testing.T) {
	t.Run("does call Implementation.Apply on State.Apply", func(t *testing.T) {
		impl := &mockImplementation{}
		state := &framework.State{Implementation: impl}
		state.Apply("", "", false)

		if impl.applyFuncCallCount != 1 {
			t.Errorf("State.Apply should call Implementation.Apply exactly once, got: %v", impl.applyFuncCallCount)
		}
	})

	t.Run("does call Implementation.update on State.Update", func(t *testing.T) {
		impl := &mockImplementation{}
		state := &framework.State{Implementation: impl}
		state.Update("", "")

		if impl.updateFuncCallCount != 1 {
			t.Errorf("State.Update should call Implementation.update exactly once, got: %v", impl.updateFuncCallCount)
		}
	})

	t.Run("does not overwrite live on apply error", func(t *testing.T) {
		impl := &mockImplementation{}
		impl.applyFunc = func(live, config framework.YAMLObject, workflow string, force bool) (framework.YAMLObject, error) {
			return "", errors.New("")
		}

		state := &framework.State{Implementation: impl}
		state.Live = framework.YAMLObject("test")

		state.Apply("", "", false)
		if state.Live != "test" {
			t.Errorf("State.Apply should not overwrite live on error, got: %v", state.Live)
		}
	})

	t.Run("does not overwrite live on update error", func(t *testing.T) {
		impl := &mockImplementation{}
		impl.updateFunc = func(live, config framework.YAMLObject, workflow string) (framework.YAMLObject, error) {
			return "", errors.New("")
		}

		state := &framework.State{Implementation: impl}
		state.Live = framework.YAMLObject("test")

		state.Update("", "")
		if state.Live != "test" {
			t.Errorf("State.Update should not overwrite live on error, got: %v", state.Live)
		}
	})
}
