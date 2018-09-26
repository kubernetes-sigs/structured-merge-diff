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

	"github.com/kubernetes-sigs/structured-merge-diff/value"
)

func TestSet(t *testing.T) {
	s1 := &Set{}
	s1.Insert(MakePathOrDie("foo", 0, "bar", "baz"))
	s1.Insert(MakePathOrDie("foo", 0, "bar"))
	s1.Insert(MakePathOrDie("foo", 0))
	s1.Insert(MakePathOrDie("foo", 1, "bar", "baz"))
	s1.Insert(MakePathOrDie("foo", 1, "bar"))
	s1.Insert(MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))))
	s1.Insert(MakePathOrDie("qux", KeyByFields("name", value.StringValue("first")), "bar"))
	s1.Insert(MakePathOrDie("qux", KeyByFields("name", value.StringValue("second")), "bar"))

	table := []struct {
		set              *Set
		check            Path
		expectMembership bool
	}{
		{s1, MakePathOrDie("qux", KeyByFields("name", value.StringValue("second"))), false},
		{s1, MakePathOrDie("qux", KeyByFields("name", value.StringValue("second")), "bar"), true},
		{s1, MakePathOrDie("qux", KeyByFields("name", value.StringValue("first"))), true},
		{s1, MakePathOrDie("xuq", KeyByFields("name", value.StringValue("first"))), false},
		{s1, MakePathOrDie("foo", 0), true},
		{s1, MakePathOrDie("foo", 0, "bar"), true},
		{s1, MakePathOrDie("foo", 0, "bar", "baz"), true},
		{s1, MakePathOrDie("foo", 1), false},
		{s1, MakePathOrDie("foo", 1, "bar"), true},
		{s1, MakePathOrDie("foo", 1, "bar", "baz"), true},
	}

	for _, tt := range table {
		got := tt.set.Has(tt.check)
		if e, a := tt.expectMembership, got; e != a {
			t.Errorf("%v: wanted %v, got %v", tt.check.String(), e, a)
		}
	}
}
