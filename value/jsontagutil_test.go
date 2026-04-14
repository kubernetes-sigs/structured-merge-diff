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

package value

import (
	"reflect"
	"testing"
)

type innerType struct {
	X string `json:"x"`
}

type testEmbeddedEmptyTag struct {
	innerType `json:""`
}

type testEmbeddedInlineTag struct {
	innerType `json:",inline"`
}

type testNonEmbeddedInlineTag struct {
	Field string `json:",inline"`
}

func TestLookupJsonTagsInline(t *testing.T) {
	tests := []struct {
		name           string
		structType     reflect.Type
		fieldIndex     int
		expectedInline bool
	}{
		{
			name:           "embedded with empty json tag",
			structType:     reflect.TypeOf(testEmbeddedEmptyTag{}),
			fieldIndex:     0,
			expectedInline: true,
		},
		{
			name:           "embedded with ,inline tag",
			structType:     reflect.TypeOf(testEmbeddedInlineTag{}),
			fieldIndex:     0,
			expectedInline: true,
		},
		{
			name:           "non-embedded with ,inline tag",
			structType:     reflect.TypeOf(testNonEmbeddedInlineTag{}),
			fieldIndex:     0,
			expectedInline: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.structType.Field(tc.fieldIndex)
			_, _, inline, _, _ := lookupJsonTags(f)
			if inline != tc.expectedInline {
				t.Errorf("lookupJsonTags(%v) inline = %v, want %v", f.Name, inline, tc.expectedInline)
			}
		})
	}
}
