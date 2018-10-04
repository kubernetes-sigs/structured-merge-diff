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

	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"

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
	name:         "struct grab bag",
	rootTypeName: "myStruct",
	schema: `types:
- name: myStruct
  struct:
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
    - name: setStr
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
    - name: setBool
      type:
        list:
          elementType:
            scalar: boolean
          elementRelationship: associative
    - name: setNumeric
      type:
        list:
          elementType:
            scalar: numeric
          elementRelationship: associative
`,
	validObjects: []string{
		`{"numeric":1}`,
		`{"numeric":3.14159}`,
		`{"string":"aoeu"}`,
		`{"bool":true}`,
		`{"bool":false}`,
		`{"setStr":["a","b","c"]}`,
		`{"setBool":[true,false]}`,
		`{"setNumeric":[1,2,3,3.14159]}`,
	},
	invalidObjects: []string{
		`{"numeric":null}`,
		`{"numeric":["foo"]}`,
		`{"numeric":{"a":1}}`,
		`{"numeric":"foo"}`,
		`{"numeric":true}`,
		`{"string":null}`,
		`{"string":1}`,
		`{"string":3.5}`,
		`{"string":true}`,
		`{"string":{"a":1}}`,
		`{"string":["foo"]}`,
		`{"bool":null}`,
		`{"bool":1}`,
		`{"bool":3.5}`,
		`{"bool":"aoeu"}`,
		`{"bool":{"a":1}}`,
		`{"bool":["foo"]}`,
		`{"setStr":["a","a"]}`,
		`{"setBool":[true,false,true]}`,
		`{"setNumeric":[1,2,3,3.14159,1]}`,
		`{"setStr":[1]}`,
		`{"setStr":[true]}`,
		`{"setStr":[1.5]}`,
		`{"setStr":[null]}`,
		`{"setStr":[{}]}`,
		`{"setStr":[[]]}`,
		`{"setBool":[true,false,true]}`,
		`{"setBool":[1]}`,
		`{"setBool":[1.5]}`,
		`{"setBool":[null]}`,
		`{"setBool":[{}]}`,
		`{"setBool":[[]]}`,
		`{"setBool":["a"]}`,
		`{"setNumeric":[1,2,3,3.14159,1]}`,
		`{"setNumeric":[null]}`,
		`{"setNumeric":[true]}`,
		`{"setNumeric":["a"]}`,
		`{"setNumeric":[[]]}`,
		`{"setNumeric":[{}]}`,
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
    - name: atomicList
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
		`{"atomicList":["a","a","a"]}`,
	},
	invalidObjects: []string{
		`{"key":true,"value":1}`,
		`{"list":{"key":true,"value":1}}`,
		`{"list":[{"key":true,"value":1}]}`,
		`{"list":[{"key":[],"value":1}]}`,
		`{"list":[{"key":{},"value":1}]}`,
		`{"list":[{"key":1.5,"value":1}]}`,
		`{"list":[{"key":1,"value":1}]}`,
		`{"list":[{"key":null,"value":1}]}`,
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
