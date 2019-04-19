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

package typed

import (
	"fmt"
	"reflect"

	"sigs.k8s.io/structured-merge-diff/schema"
)

func completeKeys(w *mergingWalker) error {
	atom, found := w.schema.Resolve(w.typeRef)
	if !found {
		panic(fmt.Sprintf("Unable to resolve schema in complete keys: %v/%v", w.schema, w.typeRef))
	}
	// Multi-keys can only be in assoc. lists, and the list must not have been removed by the defaulter.
	if atom.List == nil || atom.List.ElementRelationship != schema.Associative || len(atom.List.Keys) <= 1 || w.lhs == nil || w.lhs.ListValue == nil || w.rhs == nil || w.rhs.ListValue == nil {
		return nil
	}

	undefaultedList := w.lhs.ListValue
	defaultedList := w.rhs.ListValue

	changed := true
	matchedLHS := map[int]bool{}
	matchedRHS := map[int]bool{}
	for changed && len(matchedLHS) < len(undefaultedList.Items) {
		changed = false
		for i, lhs := range undefaultedList.Items {
			if lhs.MapValue == nil {
				return fmt.Errorf("expected a map or struct but got: %v", lhs)
			}
			if matchedLHS[i] {
				continue
			}
			possibleMatches := []int{}
			for j, rhs := range defaultedList.Items {
				if rhs.MapValue == nil {
					continue
				}
				if matchedRHS[j] {
					continue
				}
				possibleMatch := true
				for _, key := range atom.List.Keys {
					if fieldLHS, ok := lhs.MapValue.Get(key); ok {
						if fieldRHS, ok := rhs.MapValue.Get(key); ok {
							if !reflect.DeepEqual(fieldLHS, fieldRHS) {
								possibleMatch = false
							}
						}
					}
				}
				if possibleMatch {
					possibleMatches = append(possibleMatches, j)
				}
			}
			if len(possibleMatches) == 1 {
				changed = true
				matchedLHS[i] = true
				matchedRHS[possibleMatches[0]] = true
				for _, key := range atom.List.Keys {
					if _, ok := lhs.MapValue.Get(key); !ok {
						if fieldRHS, ok := defaultedList.Items[possibleMatches[0]].MapValue.Get(key); ok {
							lhs.MapValue.Set(key, fieldRHS.Value)
						}
					}
				}
			}
		}
	}

	return nil
}
