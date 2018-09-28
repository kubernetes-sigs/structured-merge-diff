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

package typed

import (
	"fmt"
	"testing"

	"github.com/kubernetes-sigs/structured-merge-diff/schema"
	"github.com/kubernetes-sigs/structured-merge-diff/value"

	"gopkg.in/yaml.v2"
)

type validationTestCase struct {
	name           string
	rootTypeName   string
	schema         string
	validObjects   []string
	invalidObjects []string
}

var validationCases = []validationTestCase{{
	name:         "simple pair",
	rootTypeName: "stringPair",
	schema: `types:
- name: stringPair
  struct:
    fields:
    - name: key
      type:
        scalar: string
    - name: value
      type:
        untyped: {}
`,
	validObjects: []string{
		`{"key":"foo","value":1}`,
		`{"key":"foo","value":{}}`,
		`{"key":"foo","value":null}`,
		`{"key":"foo"}`,
		`{"key":"foo","value":true}`,
		`{"key":"foo","value":true}`,
	},
	invalidObjects: []string{
		`{"key":true,"value":1}`,
		`{"key":1,"value":{}}`,
		`{"key":false,"value":null}`,
		`{"key":null}`,
		`{"key":[1, 2]}`,
		`{"key":{"foo":true}}`,
	},
}, {
	name:         "associative list",
	rootTypeName: "myRoot",
	schema: `types:
- name: myRoot
  struct:
    fields:
    - name: list
      type:
        namedType: myList
    - name: list2
      type:
        namedType: mySet
    - name: list3
      type:
        namedType: mySequence
- name: myList
  list:
    elementType:
      namedType: myElement
    elementRelationship: associative
    keys:
    - key
    - id
- name: mySet
  list:
    elementType:
      scalar: string
    elementRelationship: associative
- name: mySequence
  list:
    elementType:
      scalar: string
    elementRelationship: atomic
- name: myElement
  struct:
    fields:
    - name: key
      type:
        scalar: string
    - name: id
      type:
        scalar: numeric
    - name: value
      type:
        namedType: myValue
    - name: bv
      type:
        scalar: boolean
    - name: nv
      type:
        scalar: numeric
- name: myValue
  map:
    elementType:
      scalar: string
`,
	validObjects: []string{
		`{"list":[]}`,
		`{"list":[{"key":"a","id":1,"value":{"a":"a"}}]}`,
		`{"list":[{"key":"a","id":1},{"key":"a","id":2},{"key":"b","id":1}]}`,
		`{"list2":[]}`,
		`{"list2":["a"]}`,
		`{"list2":["a","b"]}`,
		`{"list2":["a","b","c"]}`,
		`{"list3":["a","a","a"]}`,
	},
	invalidObjects: []string{
		`{"key":true,"value":1}`,
		`{"list":{"key":true,"value":1}}`,
		`{"list":[{},{}]}`,
		`{"list":[{},null]}`,
		`{"list":[[]]}`,
		`{"list":[null]}`,
		`{"list":[{}]}`,
		`{"list":[{"value":{"a":"a"},"bv":true,"nv":3.14}]}`,
		`{"list":[{"key":"a","id":1,"value":{"a":1}}]}`,
		`{"list":[{"key":"a","id":1},{"key":"a","id":1}]}`,
		`{"list":[{"key":"a","id":1,"value":{"a":"a"},"bv":"true","nv":3.14}]}`,
		`{"list":[{"key":"a","id":1,"value":{"a":"a"},"bv":true,"nv":false}]}`,
		`{"list2":[null]}`,
		`{"list2":["a","a"]}`,
		`{"list2":[1]}`,
		`{"list2":[true]}`,
	},
}}

func (tt validationTestCase) test(t *testing.T) {
	var s schema.Schema
	err := yaml.Unmarshal([]byte(tt.schema), &s)
	if err != nil {
		t.Fatalf("unable to unmarshal schema")
	}

	for i, v := range tt.validObjects {
		v := v
		t.Run(fmt.Sprintf("%v-valid-%v", tt.name, i), func(t *testing.T) {
			t.Parallel()
			val, err := value.FromYAML([]byte(v))
			if err != nil {
				t.Fatalf("unable to interpret yaml: %v\n%v", err, v)
			}
			t.Logf("parsed object:\v%v", val.HumanReadable())
			_, err = AsTyped(val, &s, tt.rootTypeName)
			if err != nil {
				t.Errorf("got validation errors: %v", err)
			}
		})
	}

	for i, iv := range tt.invalidObjects {
		iv := iv
		t.Run(fmt.Sprintf("%v-invalid-%v", tt.name, i), func(t *testing.T) {
			t.Parallel()
			val, err := value.FromYAML([]byte(iv))
			if err != nil {
				t.Fatalf("unable to interpret yaml: %v\n%v", err, iv)
			}
			t.Logf("parsed object:\v%v", val.HumanReadable())
			_, err = AsTyped(val, &s, tt.rootTypeName)
			if err == nil {
				t.Errorf("didn't get validation errors!")
			}
		})
	}
}

func TestSchemaValidation(t *testing.T) {
	for _, tt := range validationCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}
