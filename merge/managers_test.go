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

package merge_test

import (
	"fmt"
	"reflect"
	"testing"

	"sigs.k8s.io/structured-merge-diff/merge"
)

func TestManagersDifference(t *testing.T) {
	tests := []struct {
		name string
		lhs  merge.Managers
		rhs  merge.Managers
		out  merge.Managers
	}{
		{
			name: "Empty sets",
			out:  merge.Managers{},
		},
		{
			name: "Empty RHS",
			lhs: merge.Managers{
				"default": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
			out: merge.Managers{
				"default": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
		},
		{
			name: "Empty LHS",
			rhs: merge.Managers{
				"default": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
			out: merge.Managers{
				"default": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
		},
		{
			name: "Different managers",
			lhs: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
			rhs: merge.Managers{
				"two": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
			out: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
				"two": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
		},
		{
			name: "Same manager, different version",
			lhs: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("integer")),
					APIVersion: "v1",
				},
			},
			rhs: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v2",
				},
			},
			out: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string"), _P("bool")),
					APIVersion: "v2",
				},
			},
		},
		{
			name: "Set difference",
			lhs: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("string")),
					APIVersion: "v1",
				},
			},
			rhs: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("string"), _P("bool")),
					APIVersion: "v1",
				},
			},
			out: merge.Managers{
				"one": &merge.VersionedSet{
					Set:        _NS(_P("numeric"), _P("bool")),
					APIVersion: "v1",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf(test.name), func(t *testing.T) {
			want := test.out
			got := test.lhs.Difference(test.rhs)
			if !reflect.DeepEqual(want, got) {
				t.Errorf("want %v, got %v", want, got)
			}
		})
	}
}
