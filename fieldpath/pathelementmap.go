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
	"sigs.k8s.io/structured-merge-diff/v4/treeset"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

// PathElementValueMap is a map from PathElement to value.Value.
//
// TODO(apelisse): We have multiple very similar implementation of this
// for PathElementSet and SetNodeMap, so we could probably share the
// code.
type PathElementValueMap struct {
	members []pathElementValue
	set     *treeset.IntWithComparator
}

func (s *PathElementValueMap) Compare(i, j int) int {
	return s.members[i].PathElement.Compare(s.members[j].PathElement)
}

func (s *PathElementValueMap) CompareNew(k interface{}, i int) int {
	return k.(*PathElement).Compare(s.members[i].PathElement)
}

func MakePathElementValueMap(size int) *PathElementValueMap {
	r := &PathElementValueMap{
		members: make([]pathElementValue, 0, size),
	}
	r.set = treeset.NewIntWithComparator(0, r)
	return r
}

type pathElementValue struct {
	PathElement PathElement
	Value       value.Value
}

// Insert adds the pathelement and associated value in the map.
func (s *PathElementValueMap) Insert(pe PathElement, v value.Value) {
	s.members = append(s.members, pathElementValue{PathElement: pe, Value: v})
	if created := s.set.Insert(len(s.members) - 1); !created {
		// if nothing is created, undo the appending
		s.members = s.members[:len(s.members)-1]
	}
}

// Get retrieves the value associated with the given PathElement from the map.
// (nil, false) is returned if there is no such PathElement.
func (s *PathElementValueMap) Get(pe PathElement) (value.Value, bool) {
	v, ok := s.set.Find(&pe)
	if !ok {
		return nil, false
	}
	return s.members[v].Value, true
}
