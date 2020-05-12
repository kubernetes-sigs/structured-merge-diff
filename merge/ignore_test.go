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

	"sigs.k8s.io/structured-merge-diff/v3/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v3/internal/fixture"
)

func TestIgnoredFields(t *testing.T) {
	tests := map[string]TestCase{
		"update_does_not_own_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("string"),
					),
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
					"v1",
					false,
				),
			},
		},
		"update_does_not_steal_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
				},
				Update{
					Manager:    "default2",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "no string"
					`,
					IgnoredFields: fieldpath.NewSet(fieldpath.MakePathOrDie("string")),
				},
			},
			Object: `
				numeric: 1
				string: "no string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
						fieldpath.MakePathOrDie("string"),
					),
					"v1",
					false,
				),
			},
		},
		"update_does_not_own_deep_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object:     `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("obj"),
					),
				},
			},
			Object: `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
					"v1",
					false,
				),
			},
		},
		"apply_does_not_own_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("string"),
					),
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
					"v1",
					true,
				),
			},
		},
		"apply_does_not_steal_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
				},
				Apply{
					Manager:    "default2",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "no string"
					`,
					IgnoredFields: fieldpath.NewSet(fieldpath.MakePathOrDie("string")),
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
						fieldpath.MakePathOrDie("string"),
					),
					"v1",
					true,
				),
				"default2": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
					"v1",
					true,
				),
			},
		},
		"apply_does_not_own_deep_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object:     `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("obj"),
					),
				},
			},
			Object: `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
					"v1",
					true,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(DeducedParser); err != nil {
				t.Fatal("Should fail:", err)
			}
		})
	}
}
