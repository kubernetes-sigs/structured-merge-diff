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

var portListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  struct:
    fields:
      - name: containerPorts
        type:
          list:
            elementType:
              struct:
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
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestDefaultKeysFlat(t *testing.T) {
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
					ExpectError: "failed to fix incomplete multi-keys",
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
					ExpectError: "failed to fix incomplete multi-keys",
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
					ExpectError: "failed to fix incomplete multi-keys",
				},
			},
		},
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
				  protocol: TCP
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "protocol"),
					),
					APIVersion: "v1",
				},
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
				  protocol: TCP
				- port: 80
				  protocol: UDP
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("TCP")), "protocol"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP"))),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP")), "port"),
						_P("containerPorts", _KBF("port", _IV(80), "protocol", _SV("UDP")), "protocol"),
					),
					APIVersion: "v1",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state := NewState(portListParser)
			state.Updater.Defaulter = protocolDefaulter{ParseableType: portListParser}
			if err := test.TestWithState(state); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// protocolDefaulter sets the default value of key "protocol" to "TCP",
// it also sets a default value of an additional field "name" to "default"
type protocolDefaulter struct {
	typed.ParseableType
}

var _ merge.Defaulter = protocolDefaulter{}

// Default implements merge.Defaulter
func (d protocolDefaulter) Default(v *typed.TypedValue) (*typed.TypedValue, error) {
	// make a deep copy of v by serializing and deserializing
	y, err := v.AsValue().ToYAML()
	if err != nil {
		return nil, err
	}
	v2, err := d.ParseableType.FromYAMLUnvalidated(typed.YAMLObject(y))
	if err != nil {
		return nil, err
	}

	// Loop over the elements of containerPorts and default certain fields
	if mapValue := v2.AsValue().MapValue; mapValue != nil {
		if containerPorts, ok := mapValue.Get("containerPorts"); ok {
			if listValue := containerPorts.Value.ListValue; listValue != nil {
				for i := range listValue.Items {
					if item := listValue.Items[i].MapValue; item != nil {
						if _, ok := item.Get("protocol"); !ok {
							item.Set("protocol", _SV("TCP"))
						}
						if _, ok := item.Get("name"); !ok {
							item.Set("name", _SV("default"))
						}
					}
				}
			}
		}
	}

	return v2, nil
}

var bookParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  struct:
    fields:
      - name: book
        type:
          list:
            elementType:
              struct:
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
                        struct:
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
            elementRelationship: associative
            keys:
            - chapter
            - section
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
				- chapter: 1
				  section: A
				  sentences:
				  - page: 2
				    line: 3
				    text: blah
			`,
			Managed: fieldpath.ManagedFields{
				"default": &fieldpath.VersionedSet{
					Set: _NS(
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"chapter",
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"section",
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
							"page",
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
							"line",
						),
						_P(
							"book", _KBF("chapter", _IV(1), "section", _SV("A")),
							"sentences", _KBF("page", _IV(2), "line", _IV(3)),
							"text",
						),
					),
					APIVersion: "v1",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			state := NewState(bookParser)
			state.Updater.Defaulter = nestedDefaulter{ParseableType: bookParser}
			if err := test.TestWithState(state); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// nestedDefaulter sets the default value of key:
// * "chapter" to 1
// * "section" to "A"
// * "page" to 2,
// * "line" to 3,
type nestedDefaulter struct {
	typed.ParseableType
}

var _ merge.Defaulter = nestedDefaulter{}

// Default implements merge.Defaulter
func (d nestedDefaulter) Default(v *typed.TypedValue) (*typed.TypedValue, error) {
	// make a deep copy of v by serializing and deserializing
	y, err := v.AsValue().ToYAML()
	if err != nil {
		return nil, err
	}
	v2, err := d.ParseableType.FromYAMLUnvalidated(typed.YAMLObject(y))
	if err != nil {
		return nil, err
	}

	// Loop over the elements of book and default certain fields
	if mapValue := v2.AsValue().MapValue; mapValue != nil {
		if book, ok := mapValue.Get("book"); ok {
			if bookList := book.Value.ListValue; bookList != nil {
				for _, bookListItem := range bookList.Items {
					if bookListItemMap := bookListItem.MapValue; bookListItemMap != nil {
						if _, ok := bookListItemMap.Get("chapter"); !ok {
							bookListItemMap.Set("chapter", _IV(1))
						}
						if _, ok := bookListItemMap.Get("section"); !ok {
							bookListItemMap.Set("section", _SV("A"))
						}
						if sentences, ok := bookListItemMap.Get("sentences"); ok {
							if sentencesList := sentences.Value.ListValue; sentencesList != nil {
								for _, sentencesListItem := range sentencesList.Items {
									if sentencesListItemMap := sentencesListItem.MapValue; sentencesListItemMap != nil {
										if _, ok := sentencesListItemMap.Get("page"); !ok {
											sentencesListItemMap.Set("page", _IV(2))
										}
										if _, ok := sentencesListItemMap.Get("line"); !ok {
											sentencesListItemMap.Set("line", _IV(3))
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return v2, nil
}
