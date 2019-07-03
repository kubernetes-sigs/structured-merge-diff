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

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/value"
)

func TestSetString(t *testing.T) {
	p := MakePathOrDie("foo", PathElement{Key: KeyByFields("name", value.StringValue("first"))})
	s1 := NewSet(p)

	if p.String() != s1.String() {
		t.Errorf("expected single entry set to just call the path's string, but got %s %s", p, s1)
	}
}

func TestSetIterator(t *testing.T) {
	s1 := NewSetAsList(
		MakePathOrDie("foo", 0, "bar", "baz"),
		MakePathOrDie("foo", 0, "bar", "zot"),
		MakePathOrDie("foo", 0, "bar"),
		MakePathOrDie("foo", 0),
		MakePathOrDie("foo", 1, "bar", "baz"),
		MakePathOrDie("foo", 1, "bar"),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("first")), "bar"),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("second")), "bar"),
	)

	s2 := NewSetAsList()

	it := s1.Iterator()
	path := it.Next()
	for path != nil {
		s2.Insert(path)
		path = it.Next()
	}

	if !s1.Equals(s2) {
		t.Errorf("Iterate missed something?\n%v\n\n%v", s1, s2)
	}
}

func TestSetEquals(t *testing.T) {
	table := []struct {
		a     *Set
		b     *Set
		equal bool
	}{
		{
			a:     NewSet(MakePathOrDie("foo")),
			b:     NewSet(MakePathOrDie("bar")),
			equal: false,
		},
		{
			a:     NewSet(MakePathOrDie("foo")),
			b:     NewSet(MakePathOrDie("foo")),
			equal: true,
		},
		{
			a:     NewSet(),
			b:     NewSet(MakePathOrDie(0, "foo")),
			equal: false,
		},
		{
			a:     NewSet(MakePathOrDie(1, "foo")),
			b:     NewSet(MakePathOrDie(0, "foo")),
			equal: false,
		},
		{
			a:     NewSet(MakePathOrDie(1, "foo")),
			b:     NewSet(MakePathOrDie(1, "foo", "bar")),
			equal: false,
		},
		{
			a: NewSet(
				MakePathOrDie(0),
				MakePathOrDie(1),
			),
			b: NewSet(
				MakePathOrDie(1),
				MakePathOrDie(0),
			),
			equal: true,
		},
		{
			a: NewSet(
				MakePathOrDie("foo", 0),
				MakePathOrDie("foo", 1),
			),
			b: NewSet(
				MakePathOrDie("foo", 1),
				MakePathOrDie("foo", 0),
			),
			equal: true,
		},
		{
			a: NewSet(
				MakePathOrDie("foo", 0),
				MakePathOrDie("foo"),
				MakePathOrDie("bar", "baz"),
				MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))),
			),
			b: NewSet(
				MakePathOrDie("foo", 1),
				MakePathOrDie("bar", "baz"),
				MakePathOrDie("bar"),
				MakePathOrDie("qux", KeyByFields("name", value.StringValue("second"))),
			),
			equal: false,
		},
	}

	for _, tt := range table {
		if e, a := tt.equal, tt.a.Equals(tt.b); e != a {
			t.Errorf("expected %v, got %v for:\na=\n%v\nb=\n%v", e, a, tt.a, tt.b)
		}
	}
}

func TestSetUnion(t *testing.T) {
	// Even though this is not a table driven test, since the thing under
	// test is recursive, we should be able to craft a single input that is
	// sufficient to check all code paths.

	s1 := NewSetAsList(
		MakePathOrDie("foo"),
		MakePathOrDie("foo", 0),
		MakePathOrDie("bar", "baz"),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))),
		MakePathOrDie("parent", "child", "grandchild"),
	)

	s2 := NewSetAsList(
		MakePathOrDie("foo", 1),
		MakePathOrDie("bar", "baz"),
		MakePathOrDie("bar"),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("second"))),
		MakePathOrDie("parent", "child"),
	)

	u := NewSetAsList(
		MakePathOrDie("foo"),
		MakePathOrDie("foo", 0),
		MakePathOrDie("foo", 1),
		MakePathOrDie("bar", "baz"),
		MakePathOrDie("bar"),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))),
		MakePathOrDie("qux", KeyByFields("name", value.StringValue("second"))),
		MakePathOrDie("parent", "child"),
		MakePathOrDie("parent", "child", "grandchild"),
	)

	got := Union(s1.Iterator(), s2.Iterator())

	if !got.Equals(u) {
		t.Errorf("union: expected: \n%v\n, got \n%v\n", u, got)
	}
}

func TestSetIntersectionDifference(t *testing.T) {
	// Even though this is not a table driven test, since the thing under
	// test is recursive, we should be able to craft a single input that is
	// sufficient to check all code paths.

	nameFirst := KeyByFields("name", value.StringValue("first"))
	s1 := NewSetAsList(
		MakePathOrDie("a0"),
		MakePathOrDie("a1"),
		MakePathOrDie("foo", 0),
		MakePathOrDie("foo", 1),
		MakePathOrDie("b0", nameFirst),
		MakePathOrDie("b1", nameFirst),
		MakePathOrDie("bar", "c0"),

		MakePathOrDie("cp", nameFirst, "child"),
	)

	s2 := NewSetAsList(
		MakePathOrDie("a1"),
		MakePathOrDie("a2"),
		MakePathOrDie("foo", 1),
		MakePathOrDie("foo", 2),
		MakePathOrDie("b1", nameFirst),
		MakePathOrDie("b2", nameFirst),
		MakePathOrDie("bar", "c2"),

		MakePathOrDie("cp", nameFirst),
	)
	t.Logf("s1:\n%v\n", s1)
	t.Logf("s2:\n%v\n", s2)

	t.Run("intersection", func(t *testing.T) {
		i := NewSetAsList(
			MakePathOrDie("a1"),
			MakePathOrDie("foo", 1),
			MakePathOrDie("b1", nameFirst),
		)

		got := Intersection(s1.Iterator(), s2.Iterator())
		if !got.Equals(i) {
			t.Errorf("expected: \n%v\n, got \n%v\n", i, got)
		}
	})

	t.Run("s1 - s2", func(t *testing.T) {
		sDiffS2 := NewSetAsList(
			MakePathOrDie("a0"),
			MakePathOrDie("foo", 0),
			MakePathOrDie("b0", nameFirst),
			MakePathOrDie("bar", "c0"),
			MakePathOrDie("cp", nameFirst, "child"),
		)

		got := Difference(s1.Iterator(), s2.Iterator())
		if !got.Equals(sDiffS2) {
			t.Errorf("expected: \n%v\n, got \n%v\n", sDiffS2, got)
		}
	})

	t.Run("s2 - s1", func(t *testing.T) {
		s2DiffS := NewSetAsList(
			MakePathOrDie("a2"),
			MakePathOrDie("foo", 2),
			MakePathOrDie("b2", nameFirst),
			MakePathOrDie("bar", "c2"),
			MakePathOrDie("cp", nameFirst),
		)

		got := Difference(s2.Iterator(), s1.Iterator())
		if !got.Equals(s2DiffS) {
			t.Errorf("expected: \n%v\n, got \n%v\n", s2DiffS, got)
		}
	})

	t.Run("intersection (the hard way)", func(t *testing.T) {
		i := NewSetAsList(
			MakePathOrDie("a1"),
			MakePathOrDie("foo", 1),
			MakePathOrDie("b1", nameFirst),
		)

		// We can construct Intersection out of two union and
		// three difference calls.
		u := Union(s1.Iterator(), s2.Iterator())
		t.Logf("s1 u s2:\n%v\n", u)
		notIntersection := Union(Difference(s2.Iterator(), s1.Iterator()).Iterator(), Difference(s1.Iterator(), s2.Iterator()).Iterator())
		t.Logf("s1 !i s2:\n%v\n", notIntersection)
		got := Difference(u.Iterator(), notIntersection.Iterator())
		if !got.Equals(i) {
			t.Errorf("expected: \n%v\n, got \n%v\n", i, got)
		}
	})
}

func TestSetNodeMapIterate(t *testing.T) {
	set := &SetNodeMap{}
	toAdd := 5
	addedElements := make([]string, toAdd)
	for i := 0; i < toAdd; i++ {
		p := i
		pe := PathElement{Index: &p}
		addedElements[i] = pe.String()
		_ = set.Descend(pe)
	}

	iteratedElements := make(map[string]bool, toAdd)
	set.Iterate(func(pe PathElement) {
		iteratedElements[pe.String()] = true
	})

	if len(iteratedElements) != toAdd {
		t.Errorf("expected %v elements to be iterated over, got %v", toAdd, len(iteratedElements))
	}
	for _, pe := range addedElements {
		if _, ok := iteratedElements[pe]; !ok {
			t.Errorf("expected to have iterated over %v, but never did", pe)
		}
	}
}
