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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(
							_P("string"),
						),
					},
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(
							_P("string"),
						),
					},
				},
			},
			Object: `
				numeric: 1
				string: "no string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
						_P("string"),
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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(
							_P("obj"),
						),
					},
				},
			},
			Object: `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(
							_P("string"),
						),
					},
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(_P("string")),
					},
				},
			},
			Object: `
				numeric: 1
				string: "some string"
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
						_P("string"),
					),
					"v1",
					true,
				),
				"default2": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
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
					IgnoredFields: map[fieldpath.APIVersion]*fieldpath.Set{
						"v1": _NS(
							_P("obj"),
						),
					},
				},
			},
			Object: `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
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

func TestIgnoredFieldsUsesVersions(t *testing.T) {
	ignored := map[fieldpath.APIVersion]*fieldpath.Set{
		"v1": _NS(
			_P("mapOfMapsRecursive", "c"),
		),
		"v2": _NS(
			_P("mapOfMapsRecursive", "cc"),
		),
		"v3": _NS(
			_P("mapOfMapsRecursive", "ccc"),
		),
		"v4": _NS(
			_P("mapOfMapsRecursive", "cccc"),
		),
	}
	test := TestCase{
		Ops: []Operation{
			Apply{
				Manager: "apply-one",
				Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						  c:
						    d:
					`,
				APIVersion:    "v1",
				IgnoredFields: ignored,
			},
			Apply{
				Manager: "apply-two",
				Object: `
						mapOfMapsRecursive:
						  aa:
						  cc:
						    dd:
					`,
				APIVersion:    "v2",
				IgnoredFields: ignored,
			},
			Apply{
				Manager: "apply-one",
				Object: `
						mapOfMapsRecursive:
					`,
				APIVersion:    "v4",
				IgnoredFields: ignored,
			},
		},
		// note that this still contains cccc due to ignored fields not being removed from the update result
		Object: `
				mapOfMapsRecursive:
				  aaaa:
				  cccc:
				    dddd:
			`,
		APIVersion: "v4",
		Managed: fieldpath.ManagedFields{
			"apply-two": fieldpath.NewVersionedSet(
				_NS(
					_P("mapOfMapsRecursive", "aa"),
				),
				"v2",
				false,
			),
		},
	}

	if err := test.TestWithConverter(nestedTypeParser, repeatingConverter{nestedTypeParser}); err != nil {
		t.Fatal(err)
	}
}
