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
	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// APIVersion describes the version of an object or of a fieldset.
type APIVersion string

// VersionedSet associates a version to a set.
type VersionedSet struct {
	*fieldpath.Set
	APIVersion APIVersion
}

// Owners is a map from workflow-id to VersionedSet (what they own in
// what version).
type Owners map[string]*VersionedSet

// Updater is the object used to compute updated FieldSets and also
// merge the object on Apply.
type Updater struct{}

// Update is the method you should call once you've merged your final
// object on CREATE/UPDATE/PATCH verbs. newObject must be the object
// that you intend to persist (after applying the patch if this is for a
// PATCH call), and liveObject must be the original object (empty if
// this is a CREATE call).
func (s *Updater) Update(liveObject, newObject typed.TypedValue, owners Owners, owner string) (Owners, error) {
	return Owners{}, nil
}

// Apply should be called when Apply is run, given the current object as
// well as the configuration that is applied. This will merge the object
// and return it.
func (s *Updater) Apply(liveObject, configObject typed.TypedValue, owners Owners, owner string, force bool) (typed.TypedValue, Owners, error) {
	return typed.TypedValue{}, Owners{}, nil
}
