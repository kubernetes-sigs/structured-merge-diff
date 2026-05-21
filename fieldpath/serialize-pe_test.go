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

package fieldpath

import (
	"reflect"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v6/value"
)

func TestPathElementRoundTrip(t *testing.T) {
	type testCase struct {
		stringValue string
		pathElement PathElement
	}

	tests := []testCase{
		{`i:0`, IndexElement(0)},
		{`i:1234`, IndexElement(1234)},
		{`f:`, FieldNameElement("")},
		{`f:spec`, FieldNameElement("spec")},
		{`f:more-complicated-string`, FieldNameElement("more-complicated-string")},
		{`f: string-with-spaces   `, FieldNameElement(" string-with-spaces   ")},
		{`f:abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`, FieldNameElement("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")},
		{`k:{"name":"my-container"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("my-container")})},
		{`k:{"name":"   name with spaces   "}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("   name with spaces   ")})},
		{`k:{"port":"8080","protocol":"TCP"}`, KeyElement(value.Field{Name: "port", Value: value.NewValueInterface("8080")}, value.Field{Name: "protocol", Value: value.NewValueInterface("TCP")})},
		{`k:{"optionalField":null}`, KeyElement(value.Field{Name: "optionalField", Value: value.NewValueInterface(nil)})},
		{`k:{"jsonField":{"A":1,"B":null,"C":"D","E":{"F":"G"}}}`, KeyElement(value.Field{Name: "jsonField", Value: value.NewValueInterface(map[string]interface{}{"A": float64(1), "B": nil, "C": "D", "E": map[string]interface{}{"F": "G"}})})},
		{`k:{"listField":["1","2","3"]}`, KeyElement(value.Field{Name: "listField", Value: value.NewValueInterface([]interface{}{"1", "2", "3"})})},
		{`v:null`, ValueElement(value.NewValueInterface(nil))},
		{`v:"some-string"`, ValueElement(value.NewValueInterface("some-string"))},
		{`v:1234`, ValueElement(value.NewValueInterface(float64(1234)))},
		{`v:{"some":"json"}`, ValueElement(value.NewValueInterface(map[string]interface{}{"some": "json"}))},
		{`v:{"some":" some  with spaces  "}`, ValueElement(value.NewValueInterface(map[string]interface{}{"some": " some  with spaces  "}))},
		{`k:{"name":"app-🚀"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app-🚀")})},
		{`k:{"name":"app-💻"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app-💻")})},
		{`k:{"name":"app with-unicøde"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app with-unicøde")})},
		{`k:{"name":"你好世界"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("你好世界")})},
		{`k:{"name":"Привет, мир"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("Привет, мир")})},
		{`k:{"name":"नमस्ते दुनिया"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("नमस्ते दुनिया")})},
		{`k:{"name":"👋"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("👋")})},
		{`f:spec-🚀`, FieldNameElement("spec-🚀")},
		{`f:spec-\n`, FieldNameElement("spec-\\n")},
	}

	for _, test := range tests {
		t.Run(test.stringValue, func(t *testing.T) {
			pe, err := DeserializePathElement(test.stringValue)
			if err != nil {
				t.Fatalf("Failed to create path element: %v", err)
			}
			if !reflect.DeepEqual(pe, test.pathElement) {
				t.Fatalf("Expected round-trip:\ninput: %#v\noutput: %#v", test.pathElement, pe)
			}
			output, err := SerializePathElement(pe)
			if err != nil {
				t.Fatalf("Failed to create string from path element (%#v): %v", pe, err)
			}
			if test.stringValue != output {
				t.Fatalf("Expected round-trip:\ninput: %v\noutput: %v", test.stringValue, output)
			}
		})
	}
}

func TestPathElementIgnoreUnknown(t *testing.T) {
	_, err := DeserializePathElement("r:Hello")
	if err != ErrUnknownPathElementType {
		t.Fatalf("Unknown qualifiers must not return an invalid path element")
	}
}

func TestDeserializePathElementError(t *testing.T) {
	tests := []string{
		``,
		`no-colon`,
		`i:index is not a number`,
		`i:1.23`,
		`i:`,
		`v:invalid json`,
		`v:`,
		`k:invalid json`,
		`k:{"name":invalid}`,
		`v:{"some":" \x41"}`, // This is an invalid JSON string because \x41 is not a valid escape sequence.
		`v`,
		`k`,
		`f`,
		`i`,
		`v:{"a":"b"`,
		`k:{"a":"b"`,
		`i: 0`,
		`i:0 `,
		`v:{"some":"json"} {"other":"json"}`, // multiple values
		`k:{"name":"my-container"} {"other":"my-container"}`, // multiple keys
		`v:{"some":"json"} {"other":"json"`,                  // multiple values with malformed trailing data
		`k:{"name":"my-container"} {"other":"my-container"`,  // multiple keys with malformed trailing data
		`v:{"some":"json"} garbage`,
		`k:{"name":"my-container"} garbage`,
	}
	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			pe, err := DeserializePathElement(test)
			if err == nil {
				t.Fatalf("Expected error, no error found. got: %#v, %s", pe, pe)
			}
		})
	}
}

func TestDeserializePathElementSuccess(t *testing.T) {
	type testCase struct {
		stringValue string
		pathElement PathElement
	}

	tests := []testCase{
		// Leading whitespace
		{`v: {"some":"json"}`, ValueElement(value.NewValueInterface(map[string]interface{}{"some": "json"}))},
		{`k: {"name":"my-container"}`, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("my-container")})},

		// Trailing whitespace
		{`v:{"some":"json"} `, ValueElement(value.NewValueInterface(map[string]interface{}{"some": "json"}))},
		{`k:{"name":"my-container"} `, KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("my-container")})},

		// Single-byte escapes in map key of key element (`k`)
		{`k:{"name\u002dcontainer":"my-container"}`, KeyElement(value.Field{Name: "name-container", Value: value.NewValueInterface("my-container")})},
		{`k:{"name\nwith\nnewlines":"my-container"}`, KeyElement(value.Field{Name: "name\nwith\nnewlines", Value: value.NewValueInterface("my-container")})},
		{`k:{"name\"quoted\"":"my-container"}`, KeyElement(value.Field{Name: `name"quoted"`, Value: value.NewValueInterface("my-container")})},

		// Multi-byte escapes in map key of key element (`k`)
		{`k:{"name-\ud83d\ude80":"my-container"}`, KeyElement(value.Field{Name: "name-🚀", Value: value.NewValueInterface("my-container")})},
		{`k:{"\u4f60\u597d":"\u4e16\u754c"}`, KeyElement(value.Field{Name: "你好", Value: value.NewValueInterface("世界")})},

		// Single-byte escapes in value element (`v`)
		{`v:"value\u002dcontainer"`, ValueElement(value.NewValueInterface("value-container"))},
		{`v:"value\nwith\nnewlines"`, ValueElement(value.NewValueInterface("value\nwith\nnewlines"))},
		{`v:"value\"quoted\""`, ValueElement(value.NewValueInterface(`value"quoted"`))},

		// Multi-byte escapes in value element (`v`)
		{`v:"value-\ud83d\ude80"`, ValueElement(value.NewValueInterface("value-🚀"))},
		{`v:"\u4f60\u597d"`, ValueElement(value.NewValueInterface("你好"))},

		// Unescaped UTF-8 in key/value
		{`k:{"name-🚀":"my-container"}`, KeyElement(value.Field{Name: "name-🚀", Value: value.NewValueInterface("my-container")})},
		{`v:"value-🚀"`, ValueElement(value.NewValueInterface("value-🚀"))},
	}

	for _, test := range tests {
		t.Run(test.stringValue, func(t *testing.T) {
			pe, err := DeserializePathElement(test.stringValue)
			if err != nil {
				t.Fatalf("Failed to create path element: %v", err)
			}
			if !reflect.DeepEqual(pe, test.pathElement) {
				t.Fatalf("Expected:\n%#v\ngot:\n%#v", test.pathElement, pe)
			}
		})
	}
}
