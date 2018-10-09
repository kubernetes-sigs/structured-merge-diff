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

func TestConflictsErrorString(t *testing.T) {
	err := framework.Conflicts{
		framework.Conflict{Field: "field1"},
		framework.Conflict{Field: "field2"},
	}

	if err.Error() != "field1, field2" {
		t.Error("invalid conflicts error string")
	}
}

func TestCheckExpectedConflicts(t *testing.T) {
	t.Run("do fail on non-conflict error", func(t *testing.T) {
		if err := framework.CheckExpectedConflicts(errors.New("foo"), nil); err == nil {
			t.Error("should fail on non-conflict error")
		}
	})

	t.Run("do not fail on nil error", func(t *testing.T) {
		if err := framework.CheckExpectedConflicts(nil, nil); err != nil {
			t.Error("should not fail on nil error")
		}
	})

	t.Run("do fail when expectations are nil", func(t *testing.T) {
		err := framework.Conflicts{
			framework.Conflict{Field: "field1"},
			framework.Conflict{Field: "field2"},
		}

		if err := framework.CheckExpectedConflicts(err, nil); err == nil {
			t.Error("should fail when expectations are nil")
		}
	})

	t.Run("do fail on unexpected conflicts", func(t *testing.T) {
		err := framework.Conflicts{
			framework.Conflict{Field: "field1"},
			framework.Conflict{Field: "field2"},
		}

		expect := framework.Conflicts{
			framework.Conflict{Field: "field1"},
			framework.Conflict{Field: "field3"},
		}

		if err := framework.CheckExpectedConflicts(err, expect); err == nil {
			t.Error("should fail on unexpected conflict")
		}
	})

	t.Run("do not fail on expected conflicts", func(t *testing.T) {
		err := framework.Conflicts{
			framework.Conflict{Field: "field1"},
			framework.Conflict{Field: "field2"},
		}

		expect := framework.Conflicts{
			framework.Conflict{Field: "field1"},
			framework.Conflict{Field: "field2"},
		}

		if err := framework.CheckExpectedConflicts(err, expect); err != nil {
			t.Errorf("should not fail on expected conflict: %v\ngot: %v", expect.Error(), err.Error())
		}
	})
}
