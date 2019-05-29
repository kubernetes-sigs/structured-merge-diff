/*
Copyright 2019 The Kubernetes Authors.
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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// portListParser sets the default value of key "protocol" to "TCP"
var portListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: containerPorts
        type:
          list:
            elementType:
              map:
                fields:
                - name: port
                  type:
                    scalar: numeric
                - name: protocol
                  type:
                    scalar: string
                - name: name
                  type:
                    scalar: string
            elementRelationship: associative
            keys:
            - port
            - protocol
            defaultedKeys:
            - fieldName: protocol
              defaultValue: "\"TCP\""
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestDefaultKeysFlat(t *testing.T) {
	tests := map[string]TestCase{
		"apply_missing_defaulted_key_A": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
					`,
				},
			},
			Object: `
				containerPorts:
				- port: 80
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
					),
					"v1",
					false,
				),
			},
		},
		"apply_missing_defaulted_key_B": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
						- port: 80
						  protocol: UDP
					`,
				},
			},
			Object: `
				containerPorts:
				- port: 80
				- port: 80
				  protocol: UDP
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP")), "protocol"),
					),
					"v1",
					false,
				),
			},
		},
		"apply_missing_defaulted_key_with_conflict": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
						  protocol: TCP
						  name: foo
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
						  name: bar
					`,
					Conflicts: merge.Conflicts{
						merge.Conflict{Manager: "apply-one", Path: _P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "name")},
					},
				},
			},
			Object: `
				containerPorts:
				- port: 80
				  protocol: TCP
				  name: foo
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "protocol"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "name"),
					),
					"v1",
					false,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(portListParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDefaultKeysFlatErrors(t *testing.T) {
	tests := map[string]TestCase{
		"apply_missing_undefaulted_defaulted_key": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- protocol: TCP
					`,
				},
			},
		},
		"apply_missing_defaulted_key_ambiguous_A": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
						- port: 80
					`,
				},
			},
		},
		"apply_missing_defaulted_key_ambiguous_B": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						containerPorts:
						- port: 80
						- port: 80
						  protocol: TCP
					`,
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.Test(portListParser) == nil {
				t.Fatal("Should fail")
			}
		})
	}
}

// bookParser sets the default value of key:
// * "chapter" to 1
// * "section" to "A"
// * "page" to 2,
// * "line" to 3,
var bookParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: book
        type:
          list:
            elementType:
              map:
                fields:
                - name: chapter
                  type:
                    scalar: numeric
                - name: section
                  type:
                    scalar: string
                - name: sentences
                  type:
                    list:
                      elementType:
                        map:
                          fields:
                          - name: page
                            type:
                              scalar: numeric
                          - name: line
                            type:
                              scalar: numeric
                          - name: text
                            type:
                              scalar: string
                      elementRelationship: associative
                      keys:
                      - page
                      - line
                      defaultedKeys:
                      - fieldName: page
                        defaultValue: "2"
                      - fieldName: line
                        defaultValue: "3"
            elementRelationship: associative
            keys:
            - chapter
            - section
            defaultedKeys:
            - fieldName: chapter
              defaultValue: "1"
            - fieldName: section
              defaultValue: "\"A\""
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestDefaultKeysNested(t *testing.T) {
	tests := map[string]TestCase{
		"apply_missing_every_key_nested": {
			Ops: []Operation{
				Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						book:
						- sentences:
						  - text: blah
					`,
				},
			},
			Object: `
				book:
				- sentences:
				  - text: blah
			`,
			Managed: fieldpath.ManagedFields{
				"default": fieldpath.NewVersionedSet(
					_NS(
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
							"text",
						),
					),
					"v1",
					false,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(bookParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}
