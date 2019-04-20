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

package typed_test

import (
	"testing"

	. "sigs.k8s.io/structured-merge-diff/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/typed"
)

var keyedListParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: type
  struct:
    fields:
    - name: list
      type:
        namedType: keyedList
- name: keyedList
  list:
    elementType:
      struct:
        fields:
        - name: a
          type:
            scalar: string
        - name: b
          type:
            scalar: string
        - name: c
          type:
            scalar: string
        - name: d
          type:
            scalar: string
        - name: value
          type:
            scalar: string
    elementRelationship: associative
    keys: ["a", "b", "c", "d"]
`)
	if err != nil {
		panic(err)
	}
	return parser.Type("type")
}()

func TestCompleteKeys(t *testing.T) {
	tests := []struct {
		name      string
		original  typed.YAMLObject
		defaulted typed.YAMLObject
		out       typed.YAMLObject
	}{
		{
			name: "all keys specified",
			original: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
			`,
			defaulted: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
			`,
			out: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
			`,
		},
		{
			name: "no keys specified",
			original: `
				list:
				- {}
			`,
			defaulted: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
			`,
			out: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
			`,
		},
		{
			name: "all combinations of keys specified",
			original: `
				list:
				list:
				- {value: "0"}
				- {d: "1", value: "1"}
				- {c: "1", value: "2"}
				- {c: "1", d: "1", value: "3"}
				- {b: "1", value: "4"}
				- {b: "1", d: "1", value: "5"}
				- {b: "1", c: "1", value: "6"}
				- {b: "1", c: "1", d: "1", value: "7"}
				- {a: "1", value: "8"}
				- {a: "1", d: "1", value: "9"}
				- {a: "1", c: "1", value: "10"}
				- {a: "1", c: "1", d: "1", value: "11"}
				- {a: "1", b: "1", value: "12"}
				- {a: "1", b: "1", d: "1", value: "13"}
				- {a: "1", b: "1", c: "1", value: "14"}
				- {a: "1", b: "1", c: "1", d: "1", value: "15"}
			`,
			defaulted: `
				list:
				- {a: "0", b: "0", c: "0", d: "0"}
				- {a: "0", b: "0", c: "0", d: "1"}
				- {a: "0", b: "0", c: "1", d: "0"}
				- {a: "0", b: "0", c: "1", d: "1"}
				- {a: "0", b: "1", c: "0", d: "0"}
				- {a: "0", b: "1", c: "0", d: "1"}
				- {a: "0", b: "1", c: "1", d: "0"}
				- {a: "0", b: "1", c: "1", d: "1"}
				- {a: "1", b: "0", c: "0", d: "0"}
				- {a: "1", b: "0", c: "0", d: "1"}
				- {a: "1", b: "0", c: "1", d: "0"}
				- {a: "1", b: "0", c: "1", d: "1"}
				- {a: "1", b: "1", c: "0", d: "0"}
				- {a: "1", b: "1", c: "0", d: "1"}
				- {a: "1", b: "1", c: "1", d: "0"}
				- {a: "1", b: "1", c: "1", d: "1"}
			`,
			out: `
				list:
				- {a: "0", b: "0", c: "0", d: "0", value: "0"}
				- {a: "0", b: "0", c: "0", d: "1", value: "1"}
				- {a: "0", b: "0", c: "1", d: "0", value: "2"}
				- {a: "0", b: "0", c: "1", d: "1", value: "3"}
				- {a: "0", b: "1", c: "0", d: "0", value: "4"}
				- {a: "0", b: "1", c: "0", d: "1", value: "5"}
				- {a: "0", b: "1", c: "1", d: "0", value: "6"}
				- {a: "0", b: "1", c: "1", d: "1", value: "7"}
				- {a: "1", b: "0", c: "0", d: "0", value: "8"}
				- {a: "1", b: "0", c: "0", d: "1", value: "9"}
				- {a: "1", b: "0", c: "1", d: "0", value: "10"}
				- {a: "1", b: "0", c: "1", d: "1", value: "11"}
				- {a: "1", b: "1", c: "0", d: "0", value: "12"}
				- {a: "1", b: "1", c: "0", d: "1", value: "13"}
				- {a: "1", b: "1", c: "1", d: "0", value: "14"}
				- {a: "1", b: "1", c: "1", d: "1", value: "15"}
			`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original, err := keyedListParser.FromYAMLUnvalidated(FixTabsOrDie(test.original))
			if err != nil {
				t.Fatalf("Failed to parse original object: %v", err)
			}
			defaulted, err := keyedListParser.FromYAML(FixTabsOrDie(test.defaulted))
			if err != nil {
				t.Fatalf("failed to parse defaulted object: %v", err)
			}
			out, err := keyedListParser.FromYAML(FixTabsOrDie(test.out))
			if err != nil {
				t.Fatalf("failed to parse out object: %v", err)
			}
			got, err := original.CompleteKeys(defaulted)
			if err != nil {
				t.Fatalf("failed to complete keys: %v", err)
			}
			comparison, err := out.Compare(got)
			if err != nil {
				t.Fatalf("failed to compare result and expected: %v", err)
			}
			if !comparison.IsSame() {
				t.Errorf("Result is different from expected:\n%v", comparison)
			}
		})
	}
}
