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
		{`k:{"duplicateKey":"value1","duplicateKey":"value2"}`, KeyElement(value.Field{Name: "duplicateKey", Value: value.NewValueInterface("value1")}, value.Field{Name: "duplicateKey", Value: value.NewValueInterface("value2")})},
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
