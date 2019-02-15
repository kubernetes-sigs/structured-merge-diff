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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/merge"
)

func TestMultipleAppliersSet(t *testing.T) {
	tests := map[string]TestCase{
		"remove_one": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						- name: b
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: c
					`,
				},
				Apply{
					Manager:    "apply-one",
					APIVersion: "v3",
					Object: `
						list:
						- name: a
					`,
				},
			},
			Object: `
				list:
				- name: a
				- name: c
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("a")), "name"),
					),
					APIVersion: "v3",
				},
				"apply-two": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("c")), "name"),
					),
					APIVersion: "v2",
				},
			},
		},
		"same_value_no_conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						  value: 0
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: a
						  value: 0
					`,
				},
			},
			Object: `
				list:
				- name: a
				  value: 0
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("a")), "name"),
						_P("list", _KBF("name", _SV("a")), "value"),
					),
					APIVersion: "v1",
				},
				"apply-two": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("a")), "name"),
						_P("list", _KBF("name", _SV("a")), "value"),
					),
					APIVersion: "v2",
				},
			},
		},
		"change_value_yes_conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						  value: 0
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: a
						  value: 1
					`,
					Conflicts: merge.Conflicts{
						merge.Conflict{Manager: "apply-one", Path: _P("list", _KBF("name", _SV("a")), "value")},
					},
				},
			},
			Object: `
				list:
				- name: a
				  value: 0
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("a")), "name"),
						_P("list", _KBF("name", _SV("a")), "value"),
					),
					APIVersion: "v1",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(associativeListParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMultipleAppliersSetBroken(t *testing.T) {
	tests := map[string]TestCase{
		"remove_one_keep_one": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						- name: b
						- name: c
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: c
						- name: d
					`,
				},
				Apply{
					Manager:    "apply-one",
					APIVersion: "v3",
					Object: `
						list:
						- name: a
					`,
				},
			},
			Object: `
				list:
				- name: a
				- name: c
				- name: d
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("a")), "name"),
					),
					APIVersion: "v3",
				},
				"apply-two": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _KBF("name", _SV("c")), "name"),
						_P("list", _KBF("name", _SV("d")), "name"),
					),
					APIVersion: "v2",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.Test(associativeListParser) == nil {
				t.Fatal("Broken test passed")
			}
		})
	}
}
