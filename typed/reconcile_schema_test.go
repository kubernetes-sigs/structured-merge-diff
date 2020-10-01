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

package typed_test

import (
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

type reconcileTestCase struct {
	name         string
	rootTypeName string
	oldSchema    typed.YAMLObject
	newSchema    typed.YAMLObject
	liveObject   typed.YAMLObject
	oldFields    *fieldpath.Set
	fixedFields  *fieldpath.Set
}

func granularSchema(version string) typed.YAMLObject {
	return typed.YAMLObject(fmt.Sprintf(`types:
- name: %s
  map:
    fields:
      - name: struct
        type:
          namedType: struct
      - name: list
        type:
          namedType: list
      - name: objectList
        type:
          namedType: objectList
      - name: stringMap
        type:
          namedType: stringMap
      - name: unchanged
        type:
          namedType: unchanged
- name: struct
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
    - name: string
      type:
        scalar: string
- name: list
  list:
    elementType:
      scalar: string
    elementRelationship: associative
- name: objectList
  list:
    elementType:
      namedType: listItem
    elementRelationship: associative
    keys:
      - keyA
      - keyB
- name: listItem
  map:
    fields:
    - name: keyA
      type:
        scalar: string
    - name: keyB
      type:
        scalar: string
    - name: value
      type:
        scalar: string
- name: stringMap
  map:
    elementType:
      scalar: string
- name: unchanged
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
`, version))
}

func atomicSchema(version string) typed.YAMLObject {
	return typed.YAMLObject(fmt.Sprintf(`types:
- name: %s
  map:
    fields:
      - name: struct
        type:
          namedType: struct
      - name: list
        type:
          namedType: list
      - name: objectList
        type:
          namedType: objectList
      - name: stringMap
        type:
          namedType: stringMap
      - name: unchanged
        type:
          namedType: unchanged
- name: struct
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
    - name: string
      type:
        scalar: string
    elementRelationship: atomic
- name: list
  list:
    elementType:
      scalar: string
    elementRelationship: atomic
- name: objectList
  list:
    elementType:
      namedType: listItem
    elementRelationship: atomic
- name: listItem
  map:
    fields:
    - name: keyA
      type:
        scalar: string
    - name: keyB
      type:
        scalar: string
    - name: value
      type:
        scalar: string
- name: stringMap
  map:
    elementType:
      scalar: string
    elementRelationship: atomic
- name: unchanged
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
`, version))
}

const basicLiveObject = typed.YAMLObject(`
struct:
  numeric: 1
  string: "two"
list:
  - one
  - two
objectList:
  - keyA: a1
    keyB: b1
    value: v1
  - keyA: a2
    keyB: b2
    value: v2
stringMap:
  key1: value1
unchanged:
  numeric: 10
`)

var reconcileCases = []reconcileTestCase{{
	name:         "granular-to-atomic",
	rootTypeName: "v1",
	oldSchema:    granularSchema("v1"),
	newSchema:    atomicSchema("v1"),
	liveObject:   basicLiveObject,
	oldFields: _NS(
		_P("struct", "numeric"),
		_P("list", _V("one")),
		_P("stringMap", "key1"),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "value"),
		_P("unchanged", "numeric"),
	),
	fixedFields: _NS(
		_P("struct"),
		_P("list"),
		_P("objectList"),
		_P("stringMap"),
		_P("unchanged", "numeric"),
	),
}, {
	name:         "atomic-to-granular",
	rootTypeName: "v1",
	oldSchema:    atomicSchema("v1"),
	newSchema:    granularSchema("v1"),
	liveObject:   basicLiveObject,
	oldFields: _NS(
		_P("struct"),
		_P("list"),
		_P("objectList"),
		_P("stringMap"),
		_P("unchanged", "numeric"),
	),
	fixedFields: _NS(
		_P("struct"),
		_P("struct", "numeric"),
		_P("struct", "string"),
		_P("list"),
		_P("list", _V("one")),
		_P("list", _V("two")),
		_P("objectList"),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1")),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "value"),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "keyA"),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "keyB"),
		_P("objectList", _KBF("keyA", "a2", "keyB", "b2")),
		_P("objectList", _KBF("keyA", "a2", "keyB", "b2"), "value"),
		_P("objectList", _KBF("keyA", "a2", "keyB", "b2"), "keyA"),
		_P("objectList", _KBF("keyA", "a2", "keyB", "b2"), "keyB"),
		_P("stringMap"),
		_P("stringMap", "key1"),
		_P("unchanged", "numeric"),
	),
}, {
	name:         "no-change-granular",
	rootTypeName: "v1",
	oldSchema:    granularSchema("v1"),
	newSchema:    granularSchema("v1"),
	liveObject:   basicLiveObject,
	oldFields: _NS(
		_P("struct", "numeric"),
		_P("list", _V("one")),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "value"),
		_P("stringMap", "key1"),
		_P("unchanged", "numeric"),
	),
	fixedFields: nil, // indicates no change
}, {
	name:         "no-change-atomic",
	rootTypeName: "v1",
	oldSchema:    atomicSchema("v1"),
	newSchema:    atomicSchema("v1"),
	liveObject:   basicLiveObject,
	oldFields: _NS(
		_P("struct"),
		_P("list"),
		_P("objectList"),
		_P("unchanged", "numeric"),
	),
	fixedFields: nil, // indicates no change
}, {
	name:         "no-change-empty-granular",
	rootTypeName: "v1",
	oldSchema:    granularSchema("v1"),
	newSchema:    granularSchema("v1"),
	liveObject: typed.YAMLObject(`
struct: {}
list: []
objectList:
  - keyA: a1
    keyB: b1
stringMap: {}
unchanged: {}
`),
	oldFields: _NS(
		_P("struct"),
		_P("list"),
		_P("objectList"),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1")),
		_P("objectList", _KBF("keyA", "a1", "keyB", "b1"), "value"),
		_P("unchanged"),
	),
	fixedFields: nil, // indicates no change
}}

func TestReconcileFieldSetWithSchema(t *testing.T) {
	for _, tt := range reconcileCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.testReconcileCase(t)
		})
	}
}

func (tt reconcileTestCase) testReconcileCase(t *testing.T) {
	parser, err := typed.NewParser(tt.newSchema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}
	pt := parser.Type(tt.rootTypeName)
	liveObject, err := pt.FromYAML(tt.liveObject)
	if err != nil {
		t.Fatalf("failed to parse/validate yaml: %v\n%v", err, tt.liveObject)
	}

	fixed, err := typed.ReconcileFieldSetWithSchema(tt.oldFields, liveObject)
	if err != nil {
		t.Fatalf("fixup errors: %v", err)
	}
	if tt.fixedFields == nil {
		if fixed != nil {
			t.Fatalf("expected fieldset to be null but got\n:%s", fixed.String())
		}
		return
	}

	if fixed == nil {
		t.Fatalf("expected fieldset to be\n:%s\n:but got null", tt.fixedFields.String())
	}

	if !fixed.Equals(tt.fixedFields) {
		t.Errorf("expected fieldset:\n%s\n:but got\n:%s", tt.fixedFields.String(), fixed.String())
	}
}
