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
	"testing"

	"sigs.k8s.io/structured-merge-diff/v6/value"
)

func TestFromJSONTokenTypeErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"numberAsObjectKey", `{1:"v"}`},
		{"trueAsObjectKey", `{true:"v"}`},
		{"nullAsObjectKey", `{null:"v"}`},
		{"arrayAsObjectKey", `{[]:"v"}`},
		{"objectAsObjectKey", `{{}:"v"}`},

		{"trailingCommaInObject", `{"a":1,}`},
		{"trailingCommaInArray", `[1,2,]`},
		{"leadingCommaInObject", `{,"a":1}`},
		{"missingCommaBetweenEntries", `{"a":1 "b":2}`},
		{"missingValue", `{"a":}`},
		{"missingColon", `{"a" 1}`},
		{"unclosedObjectAfterOpen", `{`},
		{"unclosedObjectAfterKey", `{"a"`},
		{"unclosedObjectAfterColon", `{"a":`},
		{"unclosedObjectAfterValue", `{"a":1`},
		{"unclosedArray", `[1`},
		{"unclosedString", `"abc`},

		{"singleQuotedKey", `{'a':1}`},
		{"unquotedKey", `{a:1}`},
		{"lineComment", "{// c\n\"a\":1}"},
		{"blockComment", `{/*c*/"a":1}`},

		{"hexEscape", `"\x41"`},
		{"invalidEscape", `"\q"`},

		{"infinity", `Infinity`},
		{"nan", `NaN`},
		{"leadingPlus", `+1`},
		{"trailingDot", `1.`},
		{"bareDecimal", `.5`},
		{"emptyExponent", `1e`},

		{"doubleCommaInObject", `{"a":1,,"b":2}`},
		{"objectClosedWithBracket", `{"a":1]`},
		{"arrayClosedWithBrace", `[1,2}`},
		{"nestedObjectClosedWithBracket", `{"a":{]}`},

		{"bareCloseBrace", `}`},
		{"bareCloseBracket", `]`},
		{"bareComma", `,`},
		{"bareColon", `:`},

		// TODO: Trailing data should not be allowed
		//{"trailingClosed", `{"f:a":{}}}`},
		//{"trailingObject", `{"f:a":{}}{}`},
		//{"trailingValue", `{"f:a":{}}"string"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if v, err := value.FromJSON([]byte(tc.input)); err == nil {
				t.Fatalf("expected error for %q, parsed: %#v", tc.input, v)
			}
		})
	}
}
