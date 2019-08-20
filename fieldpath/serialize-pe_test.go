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

import "testing"

func TestPathElementRoundTrip(t *testing.T) {
	tests := []string{
		`i:0`,
		`i:1234`,
		`f:`,
		`f:spec`,
		`f:more-complicated-string`,
		`k:{"name":"my-container"}`,
		`k:{"port":"8080","protocol":"TCP"}`,
		`k:{"optionalField":null}`,
		`k:{"jsonField":{"A":1,"B":null,"C":"D","E":{"F":"G"}}}`,
		`k:{"listField":["1","2","3"]}`,
		`v:null`,
		`v:"some-string"`,
		`v:1234`,
		`v:{"some":"json"}`,
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			pe, err := DeserializePathElement(test)
			if err != nil {
				t.Fatalf("Failed to create path element: %v", err)
			}
			output, err := SerializePathElement(pe)
			if err != nil {
				t.Fatalf("Failed to create string from path element: %v", err)
			}
			if test != output {
				t.Fatalf("Expected round-trip:\ninput: %v\noutput: %v", test, output)
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

func TestV2Number(t *testing.T) {
	for _, vt := range []v2ValueType{vtField, vtValue, vtKey, vtIndex} {
		for _, et := range []v2EntryType{etSelf, etChildren, etBoth} {
			n := v2CombineTypes(et, vt)
			eb, vb := v2SplitTypes(n)
			if eb != et || vb != vt {
				t.Errorf("(%v,%v) -> %v -> (%v,%v)", et, vt, n, eb, vb)
			}
		}
	}
}
