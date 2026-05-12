/*
Copyright 2026 The Kubernetes Authors.

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

package value_test

import (
	"reflect"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v6/value"
)

func TestFromJSONTokenTypeErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		// Empty and whitespace
		{"empty", ``},
		{"singleSpace", ` `},
		{"multipleSpaces", `     `},
		{"tab", "\t"},
		{"newline", "\n"},
		{"carriageReturn", "\r"},
		{"mixedWhitespace", " \t\n\r "},

		// Wrong map key types
		{"numberAsObjectKey", `{1:"v"}`},
		{"trueAsObjectKey", `{true:"v"}`},
		{"nullAsObjectKey", `{null:"v"}`},
		{"arrayAsObjectKey", `{[]:"v"}`},
		{"objectAsObjectKey", `{{}:"v"}`},

		// Misplaced tokens
		{"trailingCommaInObject", `{"a":1,}`},
		{"trailingCommaInArray", `[1,2,]`},
		{"leadingCommaInObject", `{,"a":1}`},
		{"missingCommaBetweenEntries", `{"a":1 "b":2}`},
		{"missingValue", `{"a":}`},
		{"missingColon", `{"a" 1}`},
		{"doubleCommaInObject", `{"a":1,,"b":2}`},
		{"objectClosedWithBracket", `{"a":1]`},
		{"arrayClosedWithBrace", `[1,2}`},
		{"nestedObjectClosedWithBracket", `{"a":{]}`},

		// Unexpected start tokens
		{"bareCloseBrace", `}`},
		{"bareCloseBracket", `]`},
		{"bareComma", `,`},
		{"bareColon", `:`},

		// Invalid JSON
		{"singleQuotedKey", `{'a':1}`},
		{"unquotedKey", `{a:1}`},
		{"lineComment", "{// c\n\"a\":1}"},
		{"blockComment", `{/*c*/"a":1}`},
		{"hexEscape", `"\x41"`},
		{"invalidEscape", `"\q"`},
		{"infinity", `Infinity`},
		{"nan", `NaN`},
		{"leadingPlus", `+1`},

		// Invalid numbers
		{"trailingDot", `1.`},
		{"bareDecimal", `.5`},
		{"emptyExponent", `1e`},
		{"justMinusSign", `-`},
		{"justDecimalPoint", `.`},
		{"trailingExponentSign", `1e+`},
		{"minusThenDecimal", `-1.`},

		// Truncated and unclosed
		{"truncatedTrue", `tru`},
		{"truncatedFalse", `fals`},
		{"truncatedNull", `nul`},
		{"unclosedString", `"abc`},
		{"unclosedStringEndsInBackslash", `"abc\`},
		{"unclosedStringEndsInUnicodeEscape", `"\u00`},
		{"unclosedArrayEmpty", `[`},
		{"unclosedArrayAfterValue", `[1`},
		{"unclosedArrayAfterComma", `[1,`},
		{"unclosedArrayAfterCommaWhitespace", `[1, `},
		{"unclosedNestedArray", `[[`},
		{"unclosedNestedArrayAfterValue", `[[1`},
		{"unclosedObjectAfterOpen", `{`},
		{"unclosedObjectAfterKey", `{"a"`},
		{"unclosedObjectAfterColon", `{"a":`},
		{"unclosedObjectAfterValue", `{"a":1`},
		{"unclosedObjectAfterComma", `{"a":1,`},
		{"unclosedNestedObject", `{"a":{`},
		{"unclosedNestedObjectAfterKey", `{"a":{"b"`},

		// Unexpected trailing data
		{"trailingClosed", `{"f:a":{}}}`},
		{"trailingObject", `{"f:a":{}}{}`},
		{"trailingValue", `{"f:a":{}}"string"`},
		{"twoNumbers", `1 2`},
		{"twoNumbersTabSeparated", "1\t2"},
		{"twoNumbersNewlineSeparated", "1\n2"},
		{"numberThenObject", `1 {}`},
		{"objectThenNumber", `{} 1`},
		{"objectThenObject", `{}{}`},
		{"objectThenObjectWithWhitespace", `{} {}`},
		{"arrayThenArray", `[][]`},
		{"nullThenNull", `null null`},
		{"trueThenFalse", `true false`},
		{"stringThenString", `"a" "b"`},
		{"valueThenTrailingComma", `1,`},
		{"valueThenTrailingColon", `1:`},
		{"valueThenTrailingBrace", `1}`},
		{"valueThenTrailingBracket", `1]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if v, err := value.FromJSON([]byte(tc.input)); err == nil {
				t.Fatalf("expected error for %q, parsed: %#v", tc.input, v)
			}
		})
	}
}

func TestFromJSONValid(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		canonical string
	}{
		// Valid values
		{"bareNull", `null`, `null`},
		{"bareTrue", `true`, `true`},
		{"bareFalse", `false`, `false`},
		{"bareNumber", `42`, `42`},
		{"bareString", `"x"`, `"x"`},
		{"bareEmptyString", `""`, `""`},
		{"bareEmptyObject", `{}`, `{}`},
		{"bareEmptyArray", `[]`, `[]`},

		// Leading whitespace
		{"leadingSpaceBeforeNumber", `  42`, `42`},
		{"leadingTabBeforeBool", "\ttrue", `true`},
		{"leadingNewlineBeforeString", "\n\"x\"", `"x"`},
		{"leadingMixedBeforeObject", " \t\n{\"a\":1}", `{"a":1}`},
		{"leadingMixedBeforeArray", " \t\n[1,2]", `[1,2]`},
		{"leadingMixedBeforeNull", " \t\nnull", `null`},

		// Trailing whitespace
		{"trailingSpaceAfterNumber", `42  `, `42`},
		{"trailingTabAfterBool", "false\t", `false`},
		{"trailingNewlineAfterString", "\"x\"\n", `"x"`},
		{"trailingMixedAfterObject", "{\"a\":1} \t\n", `{"a":1}`},
		{"trailingMixedAfterArray", "[1,2] \t\n\r", `[1,2]`},

		// Surrounding whitespace
		{"surroundingWhitespace", " \t\n {\"a\":1} \t\n", `{"a":1}`},
		{"surroundingWhitespaceAroundNull", " null ", `null`},
		{"surroundingWhitespaceAroundBool", " true ", `true`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := value.FromJSON([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.input, err)
			}
			want, err := value.FromJSON([]byte(tc.canonical))
			if err != nil {
				t.Fatalf("unexpected error parsing canonical %q: %v", tc.canonical, err)
			}
			if !reflect.DeepEqual(got.Unstructured(), want.Unstructured()) {
				t.Fatalf("for %q: got %#v, want %#v (from %q)",
					tc.input, got.Unstructured(), want.Unstructured(), tc.canonical)
			}
		})
	}
}
