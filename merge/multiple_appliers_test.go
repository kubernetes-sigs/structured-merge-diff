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
)

func TestMultipleAppliersSetBroken(t *testing.T) {
	tests := map[string]TestCase{
		"remove_one_keep_one": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- a
						- b
						- c
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- c
						- d
					`,
				},
				Apply{
					Manager:    "apply-one",
					APIVersion: "v3",
					Object: `
						list:
						- a
					`,
				},
			},
			Object: `
				list:
				- a
				- c
				- d
			`,
			Managed: fieldpath.ManagedFields{
				"apply-one": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _SV("a")),
					),
					APIVersion: "v3",
				},
				"apply-two": &fieldpath.VersionedSet{
					Set: _NS(
						_P("list", _SV("c")),
						_P("list", _SV("d")),
					),
					APIVersion: "v2",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.Test(setFieldsParser) == nil {
				t.Fatal("Broken test passed")
			}
		})
	}
}
