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
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/structured-merge-diff/v6/value"
)

func TestPathElementRoundTrip(t *testing.T) {
	type testCase struct {
		input       string
		pathElement PathElement
		output      string // if unset, input is expected as output
	}

	tests := []testCase{
		{input: `i:0`, pathElement: IndexElement(0)},
		{input: `i:1234`, pathElement: IndexElement(1234)},
		{input: `f:`, pathElement: FieldNameElement("")},
		{input: `f:spec`, pathElement: FieldNameElement("spec")},
		{input: `f:more-complicated-string`, pathElement: FieldNameElement("more-complicated-string")},
		{input: `f: string-with-spaces   `, pathElement: FieldNameElement(" string-with-spaces   ")},
		{input: `f:abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`, pathElement: FieldNameElement("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")},
		{input: `k:{"name":"my-container"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("my-container")})},
		{input: `k:{"name":"   name with spaces   "}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("   name with spaces   ")})},
		{input: `k:{"port":"8080","protocol":"TCP"}`, pathElement: KeyElement(value.Field{Name: "port", Value: value.NewValueInterface("8080")}, value.Field{Name: "protocol", Value: value.NewValueInterface("TCP")})},
		{input: `k:{"optionalField":null}`, pathElement: KeyElement(value.Field{Name: "optionalField", Value: value.NewValueInterface(nil)})},
		{input: `k:{"jsonField":{"A":1,"B":null,"C":"D","E":{"F":"G"}}}`, pathElement: KeyElement(value.Field{Name: "jsonField", Value: value.NewValueInterface(map[string]interface{}{"A": float64(1), "B": nil, "C": "D", "E": map[string]interface{}{"F": "G"}})})},
		{input: `k:{"listField":["1","2","3"]}`, pathElement: KeyElement(value.Field{Name: "listField", Value: value.NewValueInterface([]interface{}{"1", "2", "3"})})},
		{input: `v:null`, pathElement: ValueElement(value.NewValueInterface(nil))},
		{input: `v:"some-string"`, pathElement: ValueElement(value.NewValueInterface("some-string"))},
		{input: `v:1234`, pathElement: ValueElement(value.NewValueInterface(float64(1234)))},
		{input: `v:{"g":"0","f":"0","e":"0","d":"0","c":"0","b":"0","a":"0"}`, pathElement: ValueElement(value.NewValueInterface(map[string]interface{}{"a": "0", "b": "0", "c": "0", "d": "0", "e": "0", "f": "0", "g": "0"})), output: `v:{"a":"0","b":"0","c":"0","d":"0","e":"0","f":"0","g":"0"}`},
		{input: `v:{"some":"json"}`, pathElement: ValueElement(value.NewValueInterface(map[string]interface{}{"some": "json"}))},
		{input: `v:{"some":" some  with spaces  "}`, pathElement: ValueElement(value.NewValueInterface(map[string]interface{}{"some": " some  with spaces  "}))},
		{input: `k:{"name":"app-🚀"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app-🚀")})},
		{input: `k:{"name":"app-💻"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app-💻")})},
		{input: `k:{"name":"app with-unicøde"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("app with-unicøde")})},
		{input: `k:{"name":"你好世界"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("你好世界")})},
		{input: `k:{"name":"Привет, мир"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("Привет, мир")})},
		{input: `k:{"name":"नमस्ते दुनिया"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("नमस्ते दुनिया")})},
		{input: `k:{"name":"👋"}`, pathElement: KeyElement(value.Field{Name: "name", Value: value.NewValueInterface("👋")})},
		{input: `f:spec-🚀`, pathElement: FieldNameElement("spec-🚀")},
		{input: `f:spec-\n`, pathElement: FieldNameElement("spec-\\n")},         // no interpretation of escapes when decoding
		{input: `f:spec-\u0041`, pathElement: FieldNameElement("spec-\\u0041")}, // no interpretation of escapes when decoding
		{input: `k:{"duplicate":"1","duplicate":"2"}`, pathElement: KeyElement(value.Field{Name: "duplicate", Value: value.NewValueInterface("1")}, value.Field{Name: "duplicate", Value: value.NewValueInterface("2")})},
		{input: `v:{"duplicate":"1","duplicate":"2"}`, pathElement: ValueElement(value.NewValueInterface(map[string]interface{}{"duplicate": "2"})), output: `v:{"duplicate":"2"}`},
		{input: `k:{"duplicate":{"a":1,"a":2},"duplicate":{"b":1,"b":2}}`, pathElement: KeyElement(value.Field{Name: "duplicate", Value: value.NewValueInterface(map[string]any{"a": float64(2)})}, value.Field{Name: "duplicate", Value: value.NewValueInterface(map[string]any{"b": float64(2)})}), output: `k:{"duplicate":{"a":2},"duplicate":{"b":2}}`},
		{input: `v:{"duplicate":{"a":1,"a":2},"duplicate":{"b":1,"b":2}}`, pathElement: ValueElement(value.NewValueInterface(map[string]any{"duplicate": map[string]any{"b": float64(2)}})), output: `v:{"duplicate":{"b":2}}`},
		{input: `k:null`, pathElement: KeyElement([]value.Field{}...), output: `k:{}`},
		{input: `k:{}`, pathElement: KeyElement([]value.Field{}...)},
		{input: `k:{"key":{}}`, pathElement: KeyElement(value.Field{Name: "key", Value: value.NewValueInterface(map[string]any{})})},
		{input: `f:"`, pathElement: FieldNameElement(`"`)},
		{input: `f:\`, pathElement: FieldNameElement(`\`)},
		{input: `f:\\`, pathElement: FieldNameElement(`\\`)},
		{input: `v:"\""`, pathElement: ValueElement(value.NewValueInterface(`"`))},
		{input: `v:"\\"`, pathElement: ValueElement(value.NewValueInterface(`\`))},
		{input: `k:{"\"":{}}`, pathElement: KeyElement(value.Field{Name: `"`, Value: value.NewValueInterface(map[string]any{})})},
		{input: `k:{"\\":{}}`, pathElement: KeyElement(value.Field{Name: `\`, Value: value.NewValueInterface(map[string]any{})})},
		{input: `k:{"\\\\":{}}`, pathElement: KeyElement(value.Field{Name: `\\`, Value: value.NewValueInterface(map[string]any{})})},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			pe, err := DeserializePathElement(test.input)
			if err != nil {
				t.Fatalf("Failed to create path element: %v", err)
			}
			if !reflect.DeepEqual(pe, test.pathElement) {
				t.Fatalf("Expected round-trip:\ninput: %#v\noutput: %#v, diff: %s", test.pathElement, pe, cmp.Diff(test.pathElement, pe))
			}
			output, err := SerializePathElement(pe)
			if err != nil {
				t.Fatalf("Failed to create string from path element (%#v): %v", pe, err)
			}
			expectedOutput := test.input
			if len(test.output) > 0 {
				expectedOutput = test.output
			}
			if expectedOutput != output {
				t.Fatalf("Expected round-trip:\ninput: %v\noutput: %v", expectedOutput, output)
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

func TestInvalidUTF8(t *testing.T) {
	nonUTF8Input := "\xff\xfe"
	sanitizedOutput := "\ufffd\ufffd"

	// roundTripCases exercise behavior reading invalid utf8 characters into a field set from json, then marshaling it back
	roundTripCases := []struct {
		inputPathElement     string
		err                  bool
		marshaledPathElement string
	}{
		{
			inputPathElement:     `"f:` + nonUTF8Input + `"`,
			marshaledPathElement: `"f:` + sanitizedOutput + `"`,
		},
		{
			inputPathElement:     `"v:\"` + nonUTF8Input + `\""`,
			marshaledPathElement: `"v:\"` + sanitizedOutput + `\""`,
		},
		{
			inputPathElement:     `"k:{\"1` + nonUTF8Input + `\":{}}"`,
			marshaledPathElement: `"k:{\"1` + sanitizedOutput + `\":{}}"`,
		},
		{
			inputPathElement:     `"k:{\"2\":\"` + nonUTF8Input + `\"}"`,
			marshaledPathElement: `"k:{\"2\":\"` + sanitizedOutput + `\"}"`,
		},
		{
			inputPathElement:     `"k:{\"3\":{\"` + nonUTF8Input + `\":{}}}"`,
			marshaledPathElement: `"k:{\"3\":{\"` + sanitizedOutput + `\":{}}}"`,
		},
		{
			inputPathElement:     `"k:{\"4\":{\"key\":\"` + nonUTF8Input + `\"}}"`,
			marshaledPathElement: `"k:{\"4\":{\"key\":\"` + sanitizedOutput + `\"}}"`,
		},
	}

	for _, tc := range roundTripCases {
		t.Run("roundtrip/"+tc.inputPathElement, func(t *testing.T) {
			input := `{` + tc.inputPathElement + `:{}}`
			s := NewSet()
			err := s.FromJSON(strings.NewReader(input))
			if err != nil {
				t.Log("input:", input)
				t.Fatal(err)
			}
			output, err := s.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			expectedOutput := `{` + tc.marshaledPathElement + `:{}}`
			if string(output) != expectedOutput {
				t.Fatalf("didn't round-trip\nexpected: %v\ngot:      %v", expectedOutput, string(output))
			}
		})
	}

	// fromMemoryCases exercise constructing a field set in memory with invalid utf8 characters, then marshaling to json
	fromMemoryCases := []struct {
		pe                   PathElement
		marshaledPathElement string
	}{
		{
			pe:                   FieldNameElement(nonUTF8Input),
			marshaledPathElement: `"f:` + sanitizedOutput + `"`,
		},
		{
			pe:                   ValueElement(value.NewValueInterface(nonUTF8Input)),
			marshaledPathElement: `"v:\"` + sanitizedOutput + `\""`,
		},
		{
			pe:                   KeyElement(value.Field{Name: "1" + nonUTF8Input, Value: value.NewValueInterface(map[string]any{})}),
			marshaledPathElement: `"k:{\"1` + sanitizedOutput + `\":{}}"`,
		},
		{
			pe:                   KeyElement(value.Field{Name: "2", Value: value.NewValueInterface(nonUTF8Input)}),
			marshaledPathElement: `"k:{\"2\":\"` + sanitizedOutput + `\"}"`,
		},
		{
			pe:                   KeyElement(value.Field{Name: "3", Value: value.NewValueInterface(map[string]any{nonUTF8Input: ""})}),
			marshaledPathElement: `"k:{\"3\":{\"` + sanitizedOutput + `\":\"\"}}"`,
		},
	}

	for _, tc := range fromMemoryCases {
		t.Run("memory/"+tc.pe.String(), func(t *testing.T) {
			s := NewSet(Path{tc.pe})
			output, err := s.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			expectedOutput := `{` + tc.marshaledPathElement + `:{}}`
			if string(output) != expectedOutput {
				t.Log("want:", expectedOutput)
				t.Log("got: ", string(output))
				t.Logf("want (bytes): %#v", expectedOutput)
				t.Logf("got (bytes):  %#v", string(output))
				t.Fatalf("unexpected marshal value\nexpected: %v\ngot:      %v", expectedOutput, string(output))
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	htmlChars := "<>&"

	// roundTripCases exercise behavior reading html characters in various places in a field path then marshaling it back
	roundTripCases := []struct {
		inputPathElement     string
		err                  bool
		marshaledPathElement string
	}{
		{
			inputPathElement:     `"f:` + htmlChars + `"`,
			marshaledPathElement: `"f:` + htmlChars + `"`, // no escaping
		},
		{
			inputPathElement:     `"v:\"` + htmlChars + `\""`,
			marshaledPathElement: `"v:\"` + htmlChars + `\""`,
		},
		{
			inputPathElement:     `"k:{\"1` + htmlChars + `\":{}}"`,
			marshaledPathElement: `"k:{\"1` + htmlChars + `\":{}}"`, // no escaping
		},
		{
			inputPathElement:     `"k:{\"2\":\"` + htmlChars + `\"}"`,
			marshaledPathElement: `"k:{\"2\":\"` + htmlChars + `\"}"`,
		},
		{
			inputPathElement:     `"k:{\"3\":{\"` + htmlChars + `\":{}}}"`,
			marshaledPathElement: `"k:{\"3\":{\"` + htmlChars + `\":{}}}"`,
		},
		{
			inputPathElement:     `"k:{\"4\":{\"key\":\"` + htmlChars + `\"}}"`,
			marshaledPathElement: `"k:{\"4\":{\"key\":\"` + htmlChars + `\"}}"`,
		},
	}

	for _, tc := range roundTripCases {
		t.Run("roundtrip/"+tc.inputPathElement, func(t *testing.T) {
			input := `{` + tc.inputPathElement + `:{}}`
			s := NewSet()
			err := s.FromJSON(strings.NewReader(input))
			if err != nil {
				t.Log("input:", input)
				t.Fatal(err)
			}
			output, err := s.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			expectedOutput := `{` + tc.marshaledPathElement + `:{}}`
			if string(output) != expectedOutput {
				t.Fatalf("didn't round-trip\nexpected: %v\ngot:      %v", expectedOutput, string(output))
			}
		})
	}

	// fromMemoryCases exercise constructing a field set in memory with html special characters in various places, then marshaling to json
	fromMemoryCases := []struct {
		pe                   PathElement
		marshaledPathElement string
	}{
		{
			pe:                   FieldNameElement(htmlChars),
			marshaledPathElement: `"f:` + htmlChars + `"`, // no escaping
		},
		{
			pe:                   ValueElement(value.NewValueInterface(htmlChars)),
			marshaledPathElement: `"v:\"` + htmlChars + `\""`,
		},
		{
			pe:                   KeyElement(value.Field{Name: "1" + htmlChars, Value: value.NewValueInterface(map[string]any{})}),
			marshaledPathElement: `"k:{\"1` + htmlChars + `\":{}}"`, // no escaping
		},
		{
			pe:                   KeyElement(value.Field{Name: "2", Value: value.NewValueInterface(htmlChars)}),
			marshaledPathElement: `"k:{\"2\":\"` + htmlChars + `\"}"`,
		},
		{
			pe:                   KeyElement(value.Field{Name: "3", Value: value.NewValueInterface(map[string]any{htmlChars: ""})}),
			marshaledPathElement: `"k:{\"3\":{\"` + htmlChars + `\":\"\"}}"`,
		},
	}

	for _, tc := range fromMemoryCases {
		t.Run("memory/"+tc.pe.String(), func(t *testing.T) {
			s := NewSet(Path{tc.pe})
			output, err := s.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			expectedOutput := `{` + tc.marshaledPathElement + `:{}}`
			if string(output) != expectedOutput {
				t.Log("want:", expectedOutput)
				t.Log("got: ", string(output))
				t.Logf("want (bytes): %#v", expectedOutput)
				t.Logf("got (bytes):  %#v", string(output))
				t.Fatalf("unexpected marshal value\nexpected: %v\ngot:      %v", expectedOutput, string(output))
			}
		})
	}
}

func TestUnsupportedFloats(t *testing.T) {
	testcases := []struct {
		name   string
		path   PathElement
		output string
	}{
		{
			name: "NaN",
			path: ValueElement(value.NewValueInterface(math.NaN())),
		},
		{
			name: "Infinity",
			path: ValueElement(value.NewValueInterface(math.Inf(1))),
		},
		{
			name: "-Infinity",
			path: ValueElement(value.NewValueInterface(math.Inf(-1))),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := SerializePathElement(tc.path)
			if err == nil {
				t.Fatal("expected error, got none")
			}
		})
	}

}

func TestDeserializePathElementError(t *testing.T) {
	tests := []string{
		``,
		`no-colon`,
		`i:index is not a number`,
		`i:1.23`,
		`i:`,
		`v:`,
		`k:`,
		`v:invalid json`,
		`v:"\"`,
		`v:"""`,
		`v:`,
		`v:NaN`,
		`v:Infinity`,
		`v:-Infinity`,
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
		`k:true`,
		`k:1`,
		`k:""`,
		`k:[]`,
		`k:{`,
		`k:{"`,
		`k:{"key`,
		`k:{"key"`,
		`k:{"key",`,
		`k:{"key":`,
		`k:{"key":{`,
		`k:{"key":{}`,
		`k:{"\":{}}`,
		`k:{""":{}}`,
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

		// A null key element is equivalent to an empty one.
		{`k:{}`, KeyElement([]value.Field{}...)},
		{`k:null`, KeyElement([]value.Field{}...)},
		{`k: null `, KeyElement([]value.Field{}...)},
	}

	for _, test := range tests {
		t.Run(test.stringValue, func(t *testing.T) {
			pe, err := DeserializePathElement(test.stringValue)
			if err != nil {
				t.Fatalf("Failed to create path element: %v", err)
			}
			if !reflect.DeepEqual(test.pathElement, pe) {
				t.Fatalf("Expected:\n%#v\ngot:\n%#v\ndiff:\n%s", test.pathElement, pe, cmp.Diff(test.pathElement, pe))
			}
		})
	}
}
