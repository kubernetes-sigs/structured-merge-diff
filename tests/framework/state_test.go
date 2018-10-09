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
	"testing"

	"sigs.k8s.io/structured-merge-diff/tests/framework"
)

type MockImplementation struct {
	applyFunc    func(live, config framework.YAMLObject, workflow string, force bool) (framework.YAMLObject, error)
	nonApplyFunc func(live, config framework.YAMLObject, workflow string) (framework.YAMLObject, error)

	applyFuncCallCount    int
	nonApplyFuncCallCount int
}

func (i *MockImplementation) Apply(live, config framework.YAMLObject, workflow string, force bool) (framework.YAMLObject, error) {
	i.applyFuncCallCount++
	if i.applyFunc != nil {
		return i.applyFunc(live, config, workflow, force)
	}
	return "", nil
}

func (i *MockImplementation) NonApply(live, config framework.YAMLObject, workflow string) (framework.YAMLObject, error) {
	i.nonApplyFuncCallCount++
	if i.nonApplyFunc != nil {
		return i.nonApplyFunc(live, config, workflow)
	}
	return "", nil
}

func TestState(t *testing.T) {
	t.Run("does call Implementation.Apply on State.Apply", func(t *testing.T) {
		impl := &MockImplementation{}
		state := &framework.State{Implementation: impl}
		state.Apply("", "", false)

		if impl.applyFuncCallCount != 1 {
			t.Errorf("State.Apply should call Implementation.Apply exactly once, got: %v", impl.applyFuncCallCount)
		}
	})

	t.Run("does call Implementation.NonApply on State.Update", func(t *testing.T) {
		impl := &MockImplementation{}
		state := &framework.State{Implementation: impl}
		state.Update("", "")

		if impl.nonApplyFuncCallCount != 1 {
			t.Errorf("State.Update should call Implementation.NonApply exactly once, got: %v", impl.nonApplyFuncCallCount)
		}
	})

	t.Run("does call Implementation.NonApply on State.Patch", func(t *testing.T) {
		impl := &MockImplementation{}
		state := &framework.State{Implementation: impl}
		state.Patch("", "")

		if impl.nonApplyFuncCallCount != 1 {
			t.Errorf("State.Patch should call Implementation.NonApply exactly once, got: %v", impl.nonApplyFuncCallCount)
		}
	})
}
