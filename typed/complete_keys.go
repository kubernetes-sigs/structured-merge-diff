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
	"strings"

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

	original := toListItems(w.lhs)
	defaulted := toListItems(w.rhs)

	return matchBySpecifiedKeys(original, defaulted, atom.List.Keys)
}

type listItem interface {
	Get(string) (*value.Field, bool)
	Set(string, value.Value)
}

var _ listItem = &value.Map{}

func printItem(item listItem) string {
	return value.Value{MapValue: item.(*value.Map)}.String()
}

type itemSet map[listItem]struct{}

func (items itemSet) String() string {
	s := []string{}
	for item := range items {
		s = append(s, printItem(item))
	}
	return strings.Join(s, "\n")
}

func toListItems(v *value.Value) (mapValues []listItem) {
	if v != nil && v.ListValue != nil {
		for _, item := range v.ListValue.Items {
			if item.MapValue != nil {
				mapValues = append(mapValues, item.MapValue)
			}
		}
	}
	return mapValues
}

// matchBySpecifiedKeys uses key values from fully specified defaulted to fill in all
// unspecified keys in original if possible.
func matchBySpecifiedKeys(original, defaulted []listItem, keys []string) error {
	trie := newKeyTrie(keys)
	trie.addAllPartial(original)
	trie.addAllDefaulted(defaulted)
	for trie.hasMatchablePair() {
		partial, match := trie.nextMatchablePair()
		fillUnspecifiedKeys(partial, match, keys)
	}
	return nil
}

// fillUnspecifiedKeys uses key values from fully specified rhs to fill in
// unspecified keys in lhs.
func fillUnspecifiedKeys(lhs, rhs listItem, keys []string) {
	for _, key := range keys {
		if _, ok := lhs.Get(key); !ok {
			if fieldRHS, ok := rhs.Get(key); ok {
				lhs.Set(key, fieldRHS.Value)
			}
		}
	}
}

// keyTrie is used to quickly look up the pairs of matching items
type keyTrie struct {
	defaulted itemSet
	partial   listItem

	keys []string
	val  map[string]*keyTrie
	skip *keyTrie
	ones itemSet
}

func newKeyTrie(keys []string) *keyTrie {
	return &keyTrie{
		keys: keys,
		val:  map[string]*keyTrie{},
		ones: itemSet{},
	}
}

func (k *keyTrie) hasMatchablePair() bool {
	return len(k.ones) != 0
}

func (k *keyTrie) nextMatchablePair() (listItem, listItem) {
	for one := range k.ones {
		for match := range k.get(one) {
			k.removeDefaulted(match)
			return one, match
		}
	}
	panic("user error, called getMatchablePair without calling hasMatchablePairs first")
	return nil, nil
}

func (k *keyTrie) newSubTrie() *keyTrie {
	keys := k.keys[1:]
	if len(keys) == 0 {
		return &keyTrie{defaulted: itemSet{}, ones: k.ones}
	}
	return &keyTrie{
		keys: keys,
		val:  map[string]*keyTrie{},
		ones: k.ones,
	}
}

func (k *keyTrie) addAllDefaulted(items []listItem) {
	for _, item := range items {
		k.addDefaulted(item)
	}
}

func (k *keyTrie) addDefaulted(item listItem) {
	if len(k.keys) == 0 {
		k.defaulted[item] = struct{}{}
		if len(k.defaulted) == 1 {
			k.ones[k.partial] = struct{}{}
		} else if _, ok := k.ones[k.partial]; ok {
			delete(k.ones, k.partial)
		}
		return
	}
	if f, ok := item.Get(k.keys[0]); ok {
		val := f.Value.String()
		if _, ok := k.val[val]; ok {
			k.val[val].addDefaulted(item)
		}
		if k.skip != nil {
			k.skip.addDefaulted(item)
		}
	}
}

func (k *keyTrie) removeDefaulted(item listItem) {
	if len(k.keys) == 0 {
		delete(k.defaulted, item)
		if len(k.defaulted) == 1 {
			k.ones[k.partial] = struct{}{}
		} else if _, ok := k.ones[k.partial]; ok {
			delete(k.ones, k.partial)
		}
		return
	}
	if f, ok := item.Get(k.keys[0]); ok {
		val := f.Value.String()
		if _, ok := k.val[val]; ok {
			k.val[val].removeDefaulted(item)
		}
		if k.skip != nil {
			k.skip.removeDefaulted(item)
		}
	}
}

func (k *keyTrie) addAllPartial(items []listItem) {
	for _, item := range items {
		k.addPartial(item)
	}
}

func (k *keyTrie) addPartial(item listItem) error {
	if k.partial != nil {
		return fmt.Errorf("indistinguishable partial items: %v and %v", printItem(k.partial), printItem(item))
	}
	if len(k.keys) == 0 {
		k.partial = item
		return nil
	}

	if f, ok := item.Get(k.keys[0]); ok {
		val := f.Value.String()
		if _, ok := k.val[val]; !ok {
			k.val[val] = k.newSubTrie()
		}
		return k.val[val].addPartial(item)
	}

	if k.skip == nil {
		k.skip = k.newSubTrie()
	}
	return k.skip.addPartial(item)
}

func (k *keyTrie) get(item listItem) itemSet {
	if len(k.keys) == 0 {
		return k.defaulted
	}
	if f, ok := item.Get(k.keys[0]); ok {
		val := f.Value.String()
		if _, ok := k.val[val]; !ok {
			return itemSet{}
		}
		return k.val[val].get(item)
	}
	return k.skip.get(item)
}
