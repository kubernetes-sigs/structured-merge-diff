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
	"sort"

	"sigs.k8s.io/structured-merge-diff/schema"
	"sigs.k8s.io/structured-merge-diff/value"
)

func completeKeys(w *mergingWalker) error {
	atom, found := w.schema.Resolve(w.typeRef)
	if !found {
		panic(fmt.Sprintf("Unable to resolve schema in complete keys: %v/%v", w.schema, w.typeRef))
	}
	// defaulted keys can only be in a keyed associative lists
	if atom.List == nil || atom.List.ElementRelationship != schema.Associative || len(atom.List.Keys) == 0 {
		return nil
	}

	original := getItems(w.lhs)
	defaulted := getItems(w.rhs)

	return matchBySpecifiedKeys(original, defaulted, atom.List.Keys)
}

func getItems(v *value.Value) (mapValues []*value.Map) {
	if v != nil && v.ListValue != nil {
		for _, item := range v.ListValue.Items {
			if item.MapValue != nil {
				mapValues = append(mapValues, item.MapValue)
			}
		}
	}
	return mapValues
}

// matchBySpecifiedKeys uses key values from fully specified rhs to fill in all
// unspecified keys in lhs if possible.
// TODO: Use a trie on keys instead of an n^2 loop.
func matchBySpecifiedKeys(original, defaulted []*value.Map, keys []string) error {
	sortPartialItems(original, keys)
	matched := map[*value.Map]bool{}
	for _, lhs := range original {
		for _, rhs := range defaulted {
			// match each rhs item at most once
			if matched[rhs] {
				continue
			}

			// if we found a match for lhs, fill in the missing keys
			if isMatch(lhs, rhs, keys) {
				matched[rhs] = true
				fillUnspecifiedKeys(lhs, rhs, keys)
				break
			}
		}
	}
	return nil
}

// sortPartialItems sorts a slice of list items by the number of keys specified,
// in descending order (most completely specified first).
func sortPartialItems(original []*value.Map, keys []string) {
	sort.Slice(original, func(i, j int) bool {
		iKeys, jKeys := 0, 0
		for _, key := range keys {
			if _, ok := original[i].Get(key); ok {
				iKeys++
			}
			if _, ok := original[j].Get(key); ok {
				jKeys++
			}
		}
		return iKeys > jKeys
	})
}

// isMatch checking if all key values present in lhs match the values in rhs
func isMatch(lhs, rhs *value.Map, keys []string) bool {
	for _, key := range keys {
		if fieldLHS, ok := lhs.Get(key); ok {
			if fieldRHS, ok := rhs.Get(key); ok {
				if !reflect.DeepEqual(fieldLHS, fieldRHS) {
					return false
				}
			}
		}
	}
	return true
}

// fillUnspecifiedKeys uses key values from fully specified rhs to fill in
// unspecified keys in lhs.
func fillUnspecifiedKeys(lhs, rhs *value.Map, keys []string) {
	for _, key := range keys {
		if _, ok := lhs.Get(key); !ok {
			if fieldRHS, ok := rhs.Get(key); ok {
				lhs.Set(key, fieldRHS.Value)
			}
		}
	}
}
