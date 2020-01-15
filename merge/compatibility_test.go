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
	"sigs.k8s.io/structured-merge-diff/v3/typed"
)

var atomicListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: associative
  map:
    fields:
      - name: list
        type:
          namedType: associativeList
- name: atomic
  map:
    fields:
      - name: list
        type:
          namedType: atomicList
- name: associativeList
  list:
    elementType:
      namedType: myElement
    elementRelationship: associative
    keys:
    - name
- name: atomicList
  list:
    elementType:
      namedType: myElement
    elementRelationship: atomic
- name: myElement
  map:
    fields:
    - name: name
      type:
        scalar: string
    - name: value
      type:
        scalar: numeric
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestListsTopologyChange(t *testing.T) {
	tests := map[string]TestCase{
		"broken_atomic_doesnt_own_former_associative": {
			Ops: []Operation{
				Update{
					Manager: "one",
					Object: `
						list:
						- name: a
						  value: 1
					`,
					APIVersion: "associative",
				},
				Update{
					Manager: "other",
					Object: `
						list:
						- name: b
						  value: 2
					`,
					APIVersion: "atomic",
				},
			},
			Managed: fieldpath.ManagedFields{
				"one": fieldpath.NewVersionedSet(
					_NS(
						_P("list"),
					),
					"associative",
					false,
				),
				"other": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "b")),
						_P("list", _KBF("name", "b"), "name"),
						_P("list", _KBF("name", "b"), "value"),
					),
					"atomic",
					false,
				),
			},
		},
		"broken_associative_doesnt_own_former_atomic": {
			Ops: []Operation{
				Update{
					Manager: "one",
					Object: `
						list:
						- name: a
						  value: 1
					`,
					APIVersion: "atomic",
				},
				Update{
					Manager: "other",
					Object: `
						list:
						- name: b
						  value: 2
					`,
					APIVersion: "associative",
				},
			},

			Managed: fieldpath.ManagedFields{
				"one": fieldpath.NewVersionedSet(
					_NS(
						_P("list"),
					),
					"atomic",
					false,
				),
				"other": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "b")),
						_P("list", _KBF("name", "b"), "name"),
						_P("list", _KBF("name", "b"), "value"),
					),
					"associative",
					false,
				),
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
