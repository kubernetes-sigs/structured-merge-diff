/*
Copyright 2021 The Kubernetes Authors.

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

package treeset

import (
	"math/rand"
)

type IntWithComparator struct {
	Rand *rand.Rand

	nodes      []treeNode
	comparator Comparator
	root       int
}

type treeNode struct {
	Value int

	priority       int
	lChild, rChild int
}

// empty is the special value of the index of an empty tree.
const empty = -1

func NewIntWithComparator(capacity int, comparator Comparator) *IntWithComparator {
	return &IntWithComparator{
		nodes:      make([]treeNode, 0, capacity),
		comparator: comparator,
		root:       empty,
		Rand:       NewRand(),
	}
}

// split divides the given tree into two trees. The first has all values <= key,
// and the remaining goes to the second.
func (t *IntWithComparator) split(u int, key int) (int, int) {
	if u == empty {
		return empty, empty
	}
	cur := t.nodeAt(u)
	if t.comparator.Compare(key, cur.Value) < 0 {
		l, r := t.split(cur.lChild, key)
		cur.lChild = r
		return l, u
	}
	l, r := t.split(cur.rChild, key)
	cur.rChild = l
	return u, r
}

// merge combines two trees into one. All values in u must be equal or less than v.
func (t *IntWithComparator) merge(u, v int) int {
	if u == empty {
		return v
	}
	if v == empty {
		return u
	}
	nU, nV := t.nodeAt(u), t.nodeAt(v)
	if nU.priority > nV.priority {
		nU.rChild = t.merge(nU.rChild, v)
		return u
	}
	nV.lChild = t.merge(u, nV.lChild)
	return v
}

func (t *IntWithComparator) createNode(value int) (index int) {
	t.nodes = append(t.nodes, treeNode{
		Value:    value,
		priority: t.randomPriority(),
		lChild:   -1,
		rChild:   -1,
	})
	return len(t.nodes) - 1
}

func (t *IntWithComparator) findNew(u int, newValue interface{}) (int, bool) {
	if u == empty {
		return 0, false
	}
	if r := t.comparator.CompareNew(newValue, t.nodeAt(u).Value); r < 0 {
		return t.findNew(t.nodeAt(u).lChild, newValue)
	} else if r > 0 {
		return t.findNew(t.nodeAt(u).rChild, newValue)
	}
	return t.nodeAt(u).Value, true
}

func (t *IntWithComparator) findValue(u int, value int) (int, bool) {
	if u == empty {
		return 0, false
	}
	if r := t.comparator.Compare(value, t.nodeAt(u).Value); r < 0 {
		return t.findValue(t.nodeAt(u).lChild, value)
	} else if r > 0 {
		return t.findValue(t.nodeAt(u).rChild, value)
	}
	return t.nodeAt(u).Value, true
}

func (t *IntWithComparator) Insert(value int) (created bool) {
	l, r := t.split(t.root, value)
	if _, ok := t.findValue(l, value); !ok {
		l = t.merge(l, t.createNode(value))
		created = true
	}
	t.root = t.merge(l, r)
	return
}

func (t *IntWithComparator) nodeAt(i int) *treeNode {
	return &t.nodes[i]
}

func (t *IntWithComparator) Find(newValue interface{}) (index int, found bool) {
	return t.findNew(t.root, newValue)
}

func (t *IntWithComparator) Minimal() (int, bool) {
	if t.root == empty {
		return 0, false
	}
	cur := t.root
	for t.nodeAt(cur).lChild != empty {
		cur = t.nodeAt(cur).lChild
	}
	return t.nodeAt(cur).Value, true
}

func (t *IntWithComparator) Maximal() (int, bool) {
	if t.root == empty {
		return 0, false
	}
	cur := t.root
	for t.nodeAt(cur).rChild != empty {
		cur = t.nodeAt(cur).rChild
	}
	return t.nodeAt(cur).Value, true
}

func (t *IntWithComparator) Size() int {
	return len(t.nodes)
}

func (t *IntWithComparator) Iterator() Iterator {
	return newIterator(t)
}

func (t *IntWithComparator) Union(rhs *IntWithComparator) *IntWithComparator {
	if t.Size() > rhs.Size() {
		out := t.Clone()
		for it := rhs.Iterator(); it.HasNext(); {
			v, _ := it.Next()
			out.Insert(v)
		}
		return out
	}
	out := rhs.Clone()
	for it := t.Iterator(); it.HasNext(); {
		v, _ := it.Next()
		out.Insert(v)
	}
	return out
}

func (t *IntWithComparator) UnionMerge(rhs *IntWithComparator) *IntWithComparator {
	out := &IntWithComparator{
		Rand:       NewRand(),
		nodes:      make([]treeNode, len(t.nodes), len(t.nodes)+len(rhs.nodes)),
		comparator: t.comparator,
	}
	copy(out.nodes, t.nodes)
	copy(out.nodes[len(t.nodes):], rhs.nodes)
	for i := range rhs.nodes {
		out.nodes[len(t.nodes)+i].Value += len(t.nodes)
	}
	out.root = out.merge(t.root, rhs.root+len(t.nodes))
	return out
}

func (t *IntWithComparator) Clone() *IntWithComparator {
	r := &IntWithComparator{
		Rand:       NewRand(),
		nodes:      make([]treeNode, len(t.nodes)),
		comparator: t.comparator,
		root:       t.root,
	}
	copy(r.nodes, t.nodes)
	return r
}
