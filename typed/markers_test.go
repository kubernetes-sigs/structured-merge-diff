/*
Copyright 2025 The Kubernetes Authors.

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
	"strings"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

func TestExtractItems(t *testing.T) {
	tests := []struct {
		name           string
		schema         typed.YAMLObject
		value          typed.YAMLObject
		expectedUnset  *fieldpath.Set
		expectedValue  typed.YAMLObject
		expectedErrors typed.ValidationErrors
	}{
		{
			name: "scalar on marker",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                    - name: field
                      type:
                        scalar: string`,
			value:         `{"field": {"k8s_io__value": "unset"}}`,
			expectedUnset: _NS(_P("field")),
			expectedValue: `{}`,
		},
		{
			name: "marker on atomic list",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicList
                    type:
                      list:
                        elementType:
                          scalar: string
                        elementRelationship: atomic`,
			value:         `{"atomicList": {"k8s_io__value": "unset"}}`,
			expectedUnset: _NS(_P("atomicList")),
			expectedValue: `{}`,
		},
		{
			name: "invalid marker in atomic list element",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicList
                    type:
                      list:
                        elementType:
                          scalar: string
                        elementRelationship: atomic`,
			value: `{"atomicList": ["foo", {"k8s_io__value": "unset"}]}`,
			expectedErrors: typed.ValidationErrors{{
				ErrorMessage: "failed to extract markers: .atomicList: markers are not allowed in the contents of atomics",
			}},
		},
		{
			name: "marker on atomic map",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicMap
                    type:
                      map:
                        elementType:
                          scalar: string
                        elementRelationship: atomic`,
			value:         `{"atomicMap": {"k8s_io__value": "unset"}}`,
			expectedUnset: _NS(_P("atomicMap")),
			expectedValue: `{}`,
		},
		{
			name: "invalid marker in atomic map value",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicMap
                    type:
                      map:
                        elementType:
                          scalar: string
                        elementRelationship: atomic`,
			value: `{"atomicMap": {"key": {"k8s_io__value": "unset"}}}`,
			expectedErrors: typed.ValidationErrors{{
				ErrorMessage: "failed to extract markers: .atomicMap: markers are not allowed in the contents of atomics",
			}},
		},
		{
			name: "marker in atomic struct",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicMap
                    type:
                      map:
                        fields:
                        - name: field1
                          type:
                            scalar: string
                        - name: field2
                          type:
                            scalar: string
                        elementRelationship: atomic`,
			value:         `{"atomicMap": {"k8s_io__value": "unset"}}`,
			expectedUnset: _NS(_P("atomicMap")),
			expectedValue: `{}`,
		},
		{
			name: "invalid marker in atomic struct field",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: atomicMap
                    type:
                      map:
                        fields:
                        - name: field1
                          type:
                            scalar: string
                        - name: field2
                          type:
                            scalar: string
                        elementRelationship: atomic`,
			value: `{"atomicMap": {"field1": {"k8s_io__value": "unset"}, "field2": "value"}}`,
			expectedErrors: typed.ValidationErrors{{
				ErrorMessage: "failed to extract markers: .atomicMap: markers are not allowed in the contents of atomics",
			}},
		},
		{
			name: "marker in separable map",
			schema: `
              types:
              - name: TestType
                map:
                  elementType:
                    scalar: string
                  elementRelationship: separable`,
			value:         `{"key": {"k8s_io__value": "unset"}}`,
			expectedUnset: _NS(_P("key")),
			expectedValue: `{}`,
		},
		{
			name: "marker in associative list",
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: items
                    type:
                      list:
                        elementType:
                          map:
                            fields:
                            - name: key
                              type:
                                scalar: string
                            - name: value
                              type:
                                scalar: string
                        elementRelationship: associative
                        keys: ["key"]`,
			value:         `{"items": [{"key": "k1", "value": {"k8s_io__value": "unset"}}]}`,
			expectedUnset: _NS(_P("items", _KBF("key", "k1"), "value")),
			expectedValue: `{"items": [{"key": "k1"}]}`,
		},
		{
			name: "map is orphaned after extracting markers", // The orphaned map should be removed
			schema: `
              types:
              - name: TestType
                map:
                  fields:
                  - name: nested
                    type:
                      map:
                        fields:
                        - name: field1
                          type:
                            scalar: string
                        - name: field2
                          type:
                            scalar: string`,
			value:         `{"nested": {"field1": {"k8s_io__value": "unset"}, "field2": {"k8s_io__value": "unset"}}}`,
			expectedUnset: _NS(_P("nested", "field1"), _P("nested", "field2")),
			expectedValue: `{}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser, err := typed.NewParser(tc.schema)
			if err != nil {
				t.Fatalf("failed to create schema: %v", err)
			}
			tv, err := parser.Type("TestType").FromYAML(tc.value, typed.AllowDuplicates, typed.AllowMarkers)
			if err != nil {
				t.Fatalf("Failed to parse value: %v", err)
			}

			objectNoMarkers, markers, err := tv.ExtractMarkers()

			if tc.expectedErrors != nil {
				for _, expectedErr := range tc.expectedErrors {
					if err == nil {
						t.Errorf("Expected errors '%v' but got none", tc.expectedErrors)
					} else if !strings.Contains(err.Error(), expectedErr.ErrorMessage) {
						t.Errorf("Expected errors %v but got %v", tc.expectedErrors, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no errors but got %v", err)
			}

			if tc.expectedUnset != nil {
				if !tc.expectedUnset.Equals(markers.Unset) {
					t.Errorf("Expected unset markers %v but got %v", tc.expectedUnset, markers.Unset)
				}
			}

			expectedValue, err := parser.Type("TestType").FromYAML(tc.expectedValue, typed.AllowDuplicates)
			if err != nil {
				t.Fatalf("Failed to parse value: %v", err)
			}

			compare, err := objectNoMarkers.Compare(expectedValue)
			if err != nil {
				t.Fatalf("Failed to compare values: %v", err)
			}
			if !compare.IsSame() {
				t.Errorf("Expected value %v but got %v", tc.expectedValue, objectNoMarkers.AsValue())
			}
		})
	}
}
