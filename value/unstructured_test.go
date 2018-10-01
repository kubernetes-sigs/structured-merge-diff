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

package value

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnstructured(t *testing.T) {
	objects := []string{
		`{}`,
		// Valid yaml that isn't parsed right due to our use of MapSlice:
		// `[{}]`,
		// These two are also valid, and they do parse, but I'm not sure
		// they construct the right object:
		// `[]`,
		// `["a",{},"b",null]`,
		`foo: bar`,
		`foo:
  - bar
  - baz
qux: [1, 2]`,
		`1.5`,
		`true`,
		`"foo"`,
		`false`,
		`a:
  a: null
  b: null
  c: null
  d: null
z:
  d: null
  c: null
  b: null
  a: null
`,
		`foo:
  baz:
    bar:
      qux: [true, false, 1, "1"]
`,
		// TODO: I'd like to test random objects.
	}

	for i := range objects {
		b := []byte(objects[i])
		t.Run(fmt.Sprintf("unstructured-ordered-%v", i), func(t *testing.T) {
			t.Parallel()
			runUnstructuredTestOrdered(t, b)
		})
		t.Run(fmt.Sprintf("unstructured-unordered-%v", i), func(t *testing.T) {
			t.Parallel()
			runUnstructuredTestUnordered(t, b)
		})
	}
}

func runUnstructuredTestOrdered(t *testing.T, input []byte) {
	var decoded interface{}
	// this enables order sensitivity; note the yaml package is broken
	// for e.g. documents that have root-level arrays.
	var ms yaml.MapSlice
	if err := yaml.Unmarshal(input, &ms); err == nil {
		decoded = ms
	} else if err := yaml.Unmarshal(input, &decoded); err != nil {
		t.Fatalf("failed to decode (%v):\n%s", err, input)
	}

	v, err := FromUnstructured(decoded)
	if err != nil {
		t.Fatalf("failed to interpret (%v):\n%s", err, input)
	}

	dcheck, _ := yaml.Marshal(decoded)

	encoded := v.ToUnstructured(true)
	echeck, err := yaml.Marshal(encoded)
	if err != nil {
		t.Fatalf("unstructured rendered an unencodable output: %v", err)
	}

	if string(dcheck) != string(echeck) {
		t.Fatalf("From/To were not inverse.\n\ndecoded: %#v\n\nencoded: %#v\n\ndecoded:\n%s\n\nencoded:\n%s", decoded, encoded, dcheck, echeck)
	}
}

func runUnstructuredTestUnordered(t *testing.T, input []byte) {
	var decoded interface{}
	err := yaml.Unmarshal(input, &decoded)
	if err != nil {
		t.Fatalf("failed to decode (%v):\n%s", err, input)
	}

	v, err := FromUnstructured(decoded)
	if err != nil {
		t.Fatalf("failed to interpret (%v):\n%s", err, input)
	}

	dcheck, _ := yaml.Marshal(decoded)

	encoded := v.ToUnstructured(false)
	echeck, err := yaml.Marshal(encoded)
	if err != nil {
		t.Fatalf("unstructured rendered an unencodable output: %v", err)
	}

	if string(dcheck) != string(echeck) {
		t.Fatalf("From/To were not inverse.\n\ndecoded: %#v\n\nencoded: %#v\n\ndecoded:\n%s\n\nencoded:\n%s", decoded, encoded, dcheck, echeck)
	}
}
