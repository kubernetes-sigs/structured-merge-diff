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

import ()

// Set identifies a set of fields.
type Set struct {
	// Members lists fields that are part of the set.
	// TODO: will be serialized as a list of path elements.
	Members PathElementSet

	// Children lists child fields which themselves have children that are
	// members of the set. Appearance in this list does not imply membership.
	// Note: this is a tree, not an arbitrary graph.
	Children SetNodeMap
}

// Insert adds the field identified by `p` to the set. Important: parent fields
// are NOT added to the set; if that is desired, they must be added separately.
func (s *Set) Insert(p Path) {
	if len(p) == 0 {
		// Zero-length path identifies the entire object; we don't
		// track top-level ownership.
		return
	}
	for {
		if len(p) == 1 {
			s.Members.Insert(p[0])
			return
		}
		s = s.Children.Descend(p[0])
		p = p[1:]
	}
}

// Has returns true if the field referenced by `p` is a member of the set.
func (s *Set) Has(p Path) bool {
	if len(p) == 0 {
		// No one ones "the entire object"
		return false
	}
	for {
		if len(p) == 1 {
			return s.Members.Has(p[0])
		}
		var ok bool
		s, ok = s.Children.Get(p[0])
		if !ok {
			return false
		}
		p = p[1:]
	}
}

// setNode is a pair of PathElement / Set, for the purpose of expressing
// nested set membership.
type setNode struct {
	pathElement PathElement
	set         *Set
}

// SetNodeMap is a map of PathElement to subset.
type SetNodeMap struct {
	members map[string]setNode
}

// Descend adds pe to the set if necessary, returning the associated subset.
func (s *SetNodeMap) Descend(pe PathElement) *Set {
	serialized := pe.String()
	if s.members == nil {
		s.members = map[string]setNode{}
	}
	if n, ok := s.members[serialized]; ok {
		return n.set
	} else {
		ss := &Set{}
		s.members[serialized] = setNode{
			pathElement: pe,
			set:         ss,
		}
		return ss
	}
}

// Get returns (the associated set, true) or (nil, false) if there is none.
func (s *SetNodeMap) Get(pe PathElement) (*Set, bool) {
	if s.members == nil {
		return nil, false
	}
	serialized := pe.String()
	if n, ok := s.members[serialized]; ok {
		return n.set, true
	}
	return nil, false
}
