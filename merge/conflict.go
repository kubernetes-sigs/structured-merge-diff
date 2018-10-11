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

package merge

import (
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
)

// Conflict is a conflict on a specific field with the current owner of
// that field. It does implement the error interface so that it can be
// used as an error.
type Conflict struct {
	Owner string
	Path  fieldpath.Path
}

// Conflict is an error.
var _ error = Conflict{}

// Error formats the conflict as an error.
func (c Conflict) Error() string {
	return fmt.Sprintf("conflict with %q: %v", c.Owner, c.Path)
}

// Conflicts accumulates multiple conflicts and aggregate them by owners.
type Conflicts []Conflict

var _ error = Conflicts{}

// Error prints the list of conflicts, grouped by sorted owners.
func (conflicts Conflicts) Error() string {
	if len(conflicts) == 1 {
		return conflicts[0].Error()
	}

	m := map[string][]fieldpath.Path{}
	for _, conflict := range conflicts {
		m[conflict.Owner] = append(m[conflict.Owner], conflict.Path)
	}

	owners := []string{}
	for owner := range m {
		owners = append(owners, owner)
	}

	// Print conflicts by sorted owners.
	sort.Strings(owners)

	messages := []string{}
	for _, owner := range owners {
		messages = append(messages, fmt.Sprintf("conflicts with %q:", owner))
		for _, path := range m[owner] {
			messages = append(messages, fmt.Sprintf("- %v", path))
		}
	}
	return strings.Join(messages, "\n")
}

// NewFromSets creates a list of conflicts error from a map of owner to set of fields.
func NewFromSets(sets map[string]*fieldpath.Set) Conflicts {
	conflicts := []Conflict{}

	for owner, set := range sets {
		set.Iterate(func(p fieldpath.Path) {
			conflicts = append(conflicts, Conflict{
				Owner: owner,
				Path:  p,
			})
		})
	}

	return conflicts
}
