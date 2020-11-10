/*
Copyright 2020 The Kubernetes Authors.

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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v4/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v4/merge"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

func TestWipeManagedFields(t *testing.T) {
	type testCase struct {
		name                                string
		liveManagedFields, newManagedFields fieldpath.ManagedFields
		manager                             string
		// newObject is not used in the actual test but helps to visualize what's going on (for now)
		liveObject, newObject, preparedObject typed.YAMLObject
		expectedManagedFields                 fieldpath.ManagedFields
	}

	tests := map[string]testCase{
		"does not own new fields removed by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a"), _P("b")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1}`,
			newObject:      `{"a": 1, "b": 2}`,
			preparedObject: `{"a": 1}`,
			expectedManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
		},
		"does not own modified fields removed by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1}`,
			newObject:      `{"a": 2}`,
			preparedObject: `{"a": 1}`,
			expectedManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(), "v1", false),
			},
		},
		"does not wipe added fields not removed by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a"), _P("b")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1}`,
			newObject:      `{"a": 1, "b": 2}`,
			preparedObject: `{"a": 1, "b": 2}`,
			expectedManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a"), _P("b")), "v1", false),
			},
		},
		"does not wipe modified fields not removed by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1}`,
			newObject:      `{"a": 2}`,
			preparedObject: `{"a": 2}`,
			expectedManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
		},
		"does remove removed fields not reset by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a"), _P("b")), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1, "b": 2}`,
			newObject:      `{"a": 1}`,
			preparedObject: `{"a": 1}`,
			expectedManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
		},
		"does not remove removed fields reset by prepare": {
			liveManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a"), _P("b")), "v1", false),
			},
			newManagedFields: fieldpath.ManagedFields{
				"test": fieldpath.NewVersionedSet(_NS(_P("a")), "v1", false),
			},
			manager:        "test",
			liveObject:     `{"a": 1, "b": 2}`,
			newObject:      `{"a": 1}`,
			preparedObject: `{"a": 1, "b": 2}`,
			expectedManagedFields: fieldpath.ManagedFields{
				// TODO: should `b` still be owned despite the applier not intending to own it?
				"test": fieldpath.NewVersionedSet(_NS(_P("a") /* _P("b") */), "v1", false),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			parser := DeducedParser

			parsedLiveObject, err := parser.Type("").FromYAML(FixTabsOrDie(test.liveObject))
			if err != nil {
				t.Fatalf("parsing liveObject: %v", err)
			}
			parsedPreparedObject, err := parser.Type("").FromYAML(FixTabsOrDie(test.preparedObject))
			if err != nil {
				t.Fatalf("parsing preparedObject: %v", err)
			}

			newManagedFields, err := merge.WipeManagedFields(
				test.liveManagedFields,
				test.newManagedFields,
				test.manager,
				parsedLiveObject,
				parsedPreparedObject,
			)
			if err != nil {
				t.Fatalf("wiping managedFields: %v", err)
			}

			if !test.expectedManagedFields.Equals(newManagedFields) {
				t.Fatalf("expected managedFields after wiping to be:\n%s\ngot:\n%s", test.expectedManagedFields, test.newManagedFields)
			}
		})
	}
}
