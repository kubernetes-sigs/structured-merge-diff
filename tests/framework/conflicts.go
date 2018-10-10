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

package framework

import "strings"

// Conflict is implementing the error interface providing further details about a merge conflict
// TODO: once we figured out how to do this type it should move out of this package
type Conflict struct {
	Field string
}

func (c Conflict) Error() string {
	return c.Field
}

// NotIn returns true if conflict is not found inside Conflicts
func (c Conflict) NotIn(conflicts Conflicts) bool {
	for _, co := range conflicts {
		if co.Error() == c.Error() {
			return false
		}
	}
	return true
}

// Conflicts is implementing the error interface storing a set of conflicts for then checking expected ones
type Conflicts []Conflict

func (c Conflicts) Error() string {
	var errs []string
	for _, co := range c {
		errs = append(errs, co.Error())
	}
	return strings.Join(errs, ", ")
}

// CheckExpectedConflicts and return unexpected ones
// TODO: decide how we want this to return. either fail fast and return all conflicts directly
// or check/filter all conflicts and only return unexpected ones as error (or as Conflicts).
// Also, do we want to fail if expected conflicts do not occur or should this be a different function like `CheckAllExpectedConflicts` and `CheckAnyExpectedConflicts`?
func CheckExpectedConflicts(err error, expected Conflicts) error {
	conflicts, ok := err.(Conflicts)
	if !ok {
		return err
	}

	// Going with just one method for now until we decide the questions above
	remaining := conflicts[:0]
	for _, conflict := range conflicts {
		if conflict.NotIn(expected) {
			remaining = append(remaining, conflict)
		}
	}

	if len(remaining) < 1 {
		return nil
	}
	return remaining
}
