/*
Copyright 2025 The Kubernetes Authors.

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

	"sigs.k8s.io/structured-merge-diff/v4/merge"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v4/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

var unsetFieldsParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: unsetFields
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
    - name: string
      type:
        scalar: string
    - name: bool
      type:
        scalar: boolean
    - name: atomicList
      type:
        list:
          elementType:
            scalar: numeric
          elementRelationship: atomic
    - name: atomicStruct
      type:
       map:
         fields:
         - name: name
           type:
             scalar: string
         - name: value
           type:
             scalar: numeric
         elementRelationship: atomic
    - name: atomicMap
      type:
        map:
          elementType:
            scalar: numeric
          elementRelationship: atomic
    - name: associativeList
      type:
        list:
          elementType:
            namedType: nameValueType
          elementRelationship: associative
          keys:
          - name
    - name: multiKeyAssociativeList
      type:
        list:
          elementType:
            namedType: multiKeyType
          elementRelationship: associative
          keys:
          - stringKey
          - numericKey
    - name: granularMap
      type:
        map:
          elementType:
            scalar: numeric
          elementRelationship: granular
- name: nameValueType
  map:
    fields:
    - name: name
      type:
        scalar: string
    - name: value
      type:
        scalar: numeric
- name: multiKeyType
  map:
    fields:
    - name: stringKey
      type:
        scalar: string
    - name: numericKey
      type:
        scalar: numeric
      default: 1
    - name: value
      type:
        scalar: numeric`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("unsetFields")}
}()

func TestUnset(t *testing.T) {
	tests := map[string]TestCase{
		"unset_scalars": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						numeric: 1
						string: "string"
						bool: true
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						numeric: {k8s_io__value: unset}
						string: {k8s_io__value: unset}
						bool: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
						_P("string"),
						_P("bool"),
					),
					"v1",
					true,
				),
			},
		},
		"unset_atomic_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicList: [1, 2]
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						atomicList: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicList"),
					),
					"v1",
					true,
				),
			},
		},
		"unset_atomic_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicMap: {k1: 1, k2: 2}
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						atomicMap: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicMap"),
					),
					"v1",
					true,
				),
			},
		},
		"unset_one_entry_of_associative_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						associativeList: [{name: "a", value: 1}, {name: "b", value: 2}]
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						associativeList: [{name: "b", k8s_io__value: unset}]
					`,
					APIVersion: "v1",
				},
			},
			Object: `
				associativeList: [{name: "a", value: 1}]
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "a")),
						_P("associativeList", _KBF("name", "a"), "name"),
						_P("associativeList", _KBF("name", "a"), "value"),
					),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "b")), // Tracks ownership of the whole unset map entry
					),
					"v1",
					true,
				),
			},
		},
		"unset_all_entries_of_associative_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						associativeList: [
							{name: "a", value: 1},
							{name: "b", value: 2}
						]
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						associativeList: [
							{name: "a", k8s_io__value: unset},
							{name: "b", k8s_io__value: unset}
						]
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "a")),
						_P("associativeList", _KBF("name", "b")),
					),
					"v1",
					true,
				),
			},
		},
		"unset_granular_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						granularMap: {"a": 1, "b": 2}
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						granularMap: {"b": {k8s_io__value: unset}}
					`,
					APIVersion: "v1",
				},
			},
			Object: `
				granularMap: {"a": 1}
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("granularMap", "a"),
					),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("granularMap", "b"), // Tracks ownership of the whole unset map entry
					),
					"v1",
					true,
				),
			},
		},
		"unset_atomic_struct": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicStruct: {name: a, value: 1}
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						atomicStruct: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicStruct"),
					),
					"v1",
					true,
				),
			},
		},

		// conflict tests
		"conflict_unset_scalar": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						numeric: 1
						string: "string"
						bool: true
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						numeric: {k8s_io__value: unset}
						string: {k8s_io__value: unset}
						bool: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Conflicts: merge.Conflicts{
						{Manager: "m1", Path: _P("numeric")},
						{Manager: "m1", Path: _P("string")},
						{Manager: "m1", Path: _P("bool")},
					},
				},
			},
			Object: `
				numeric: 1
				string: "string"
				bool: true
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("numeric"),
						_P("string"),
						_P("bool"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict_unset_atomic_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicList: [1, 2]
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						atomicList: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Conflicts: merge.Conflicts{
						{Manager: "m1", Path: _P("atomicList")},
					},
				},
			},
			Object: `
				atomicList: [1, 2]
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicList"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict_unset_atomic_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicMap: {k1: 1, k2: 2}
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						atomicMap: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Conflicts: merge.Conflicts{
						{Manager: "m1", Path: _P("atomicMap")},
					},
				},
			},
			Object: `
				atomicMap: {k1: 1, k2: 2}
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicMap"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict_unset_associative_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						associativeList: [{name: "a", value: 1}, {name: "b", value: 2}]
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						associativeList: [{name: "b", k8s_io__value: unset}]
					`,
					APIVersion: "v1",
					Conflicts: merge.Conflicts{
						{Manager: "m1", Path: _P("associativeList", _KBF("name", "b"))},
					},
				},
			},
			Object: `
				associativeList: [{name: "a", value: 1}, {name: "b", value: 2}]
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "a")),
						_P("associativeList", _KBF("name", "a"), "name"),
						_P("associativeList", _KBF("name", "a"), "value"),
						_P("associativeList", _KBF("name", "b")),
						_P("associativeList", _KBF("name", "b"), "name"),
						_P("associativeList", _KBF("name", "b"), "value"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict_unset_granular_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						granularMap: {"a": 1, "b": 2}
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						granularMap: {"b": {k8s_io__value: unset}}
					`,
					APIVersion: "v1",
					Conflicts: []merge.Conflict{
						{Manager: "m1", Path: _P("granularMap", "b")},
					},
				},
			},
			Object: `
				granularMap: {"a": 1, "b": 2}
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("granularMap", "a"),
						_P("granularMap", "b"),
					),
					"v1",
					true,
				),
			},
		},
		"conflict_unset_atomic_struct": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicStruct: {name: a, value: 1}
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						atomicStruct: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Conflicts: []merge.Conflict{
						{Manager: "m1", Path: _P("atomicStruct")},
					},
				},
			},
			Object: `
				atomicStruct: {name: a, value: 1}
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("atomicStruct"),
					),
					"v1",
					true,
				),
			},
		},

		// Field management of owned unset fields
		"unset_associative_list_key_ownership_respected": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						associativeList:
						- name: item1
						  value: 1
						- name: item2
						  value: 2
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						associativeList:
						- name: item1
						  k8s_io__value: unset
					`,
					APIVersion: "v1",
				},
				// Subsequent operation from m1 should not be able to set the item
				Apply{
					Manager: "m1",
					Object: `
						associativeList:
						- name: item1
						  value: 5
						- name: item2
						  value: 2
					`,
					APIVersion: "v1",
					Conflicts: []merge.Conflict{
						{Manager: "m2", Path: _P("associativeList", _KBF("name", "item1"))},
					},
				},
			},
			Object: `
				associativeList:
				- name: item2
				  value: 2
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "item2")),
						_P("associativeList", _KBF("name", "item2"), "name"),
						_P("associativeList", _KBF("name", "item2"), "value"),
					),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("associativeList", _KBF("name", "item1")),
					),
					"v1",
					true,
				),
			},
		},
		"unset_scalar_ownership_respected": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						string: test
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						string: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
				// Subsequent operation from m1 should not be able to set the field
				Apply{
					Manager: "m1",
					Object: `
						string: newvalue
					`,
					APIVersion: "v1",
					Conflicts: []merge.Conflict{
						{Manager: "m2", Path: _P("string")},
					},
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("string"),
					),
					"v1",
					true,
				),
			},
		},

		// shared ownership of unset fields
		"unset_same_field_multiple_managers": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						string: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "m2",
					Object: `
						string: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
				},
			},
			Object:     ``,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("string"),
					),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("string"),
					),
					"v1",
					true,
				),
			},
		},

		// defaulteed map keys
		"unset_multi_key_with_default_associative_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						multiKeyAssociativeList:
						- stringKey: "a"
						  numericKey: 1
						  value: 1
						- stringKey: "b" 
						  numericKey: 2
						  value: 2
					`,
					APIVersion: "v1",
				},
				ForceApply{
					Manager: "m2",
					Object: `
						multiKeyAssociativeList:
						- stringKey: "a" # numericKey should default to 1.
						  k8s_io__value: unset
					`,
					APIVersion: "v1",
				},
			},
			Object: `
				multiKeyAssociativeList:
				- stringKey: "b"
				  numericKey: 2 
				  value: 2
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"m1": fieldpath.NewVersionedSet(
					_NS(
						_P("multiKeyAssociativeList", _KBF("stringKey", "b", "numericKey", 2)),
						_P("multiKeyAssociativeList", _KBF("stringKey", "b", "numericKey", 2), "stringKey"),
						_P("multiKeyAssociativeList", _KBF("stringKey", "b", "numericKey", 2), "numericKey"),
						_P("multiKeyAssociativeList", _KBF("stringKey", "b", "numericKey", 2), "value"),
					),
					"v1",
					true,
				),
				"m2": fieldpath.NewVersionedSet(
					_NS(
						_P("multiKeyAssociativeList", _KBF("stringKey", "a", "numericKey", 1)),
					),
					"v1",
					true,
				),
			},
		},

		// Validation tests
		"invalid_marker_on_granular_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						granularMap: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Error:      "failed to extract markers: .granularMap: markers are only allowed on atomic maps and associative list entries",
				},
			},
		},
		"invalid_marker_on_associative_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						associativeList: {k8s_io__value: unset}
					`,
					APIVersion: "v1",
					Error:      "failed to extract markers: .associativeList: markers are only allowed on atomic lists",
				},
			},
		},
		"invalid_marker_in_atomic_list": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicList: [
							{k8s_io__value: unset}
						]
					`,
					APIVersion: "v1",
					Error:      "failed to extract markers: .atomicList: markers are not allowed in the contents of atomics",
				},
			},
		},
		"invalid_marker_in_atomic_map": {
			Ops: []Operation{
				Apply{
					Manager: "m1",
					Object: `
						atomicMap: {k1: {k8s_io__value: unset}}
					`,
					APIVersion: "v1",
					Error:      "failed to extract markers: .atomicMap: markers are not allowed in the contents of atomics",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// We don't use TestOptionCombinations because we expect these tests to pass with EnableUnsetMarkers=true
			test.EnableUnsetMarkers = true
			if err := test.Test(unsetFieldsParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}
