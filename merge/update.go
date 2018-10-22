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

	"sigs.k8s.io/structured-merge-diff/typed"
)

// Converter is an interface to the conversion logic. The converter
// needs to be able to convert objects from one version to another.
type Converter interface {
	Convert(object typed.TypedValue, version APIVersion) (typed.TypedValue, error)
}

// Updater is the object used to compute updated FieldSets and also
// merge the object on Apply.
type Updater struct {
	Converter Converter
}

func (s *Updater) update(oldObject, newObject typed.TypedValue, owners Owners, workflow string, force bool) (Owners, error) {
	if owners == nil {
		owners = Owners{}
	}
	conflicts := Owners{}
	type Versioned struct {
		oldObject typed.TypedValue
		newObject typed.TypedValue
	}
	versions := map[APIVersion]Versioned{}

	for owner, ownerSet := range owners {
		if owner == workflow {
			continue
		}
		versioned, ok := versions[ownerSet.APIVersion]
		if !ok {
			var err error
			versioned.oldObject, err = s.Converter.Convert(oldObject, ownerSet.APIVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to convert old object: %v", err)
			}
			versioned.newObject, err = s.Converter.Convert(newObject, ownerSet.APIVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to convert new object: %v", err)
			}
			versions[ownerSet.APIVersion] = versioned
		}
		compare, err := versioned.oldObject.Compare(versioned.newObject)
		if err != nil {
			return nil, fmt.Errorf("failed to compare objects: %v", err)
		}

		conflictSet := ownerSet.Intersection(compare.Modified.Union(compare.Added))
		if !conflictSet.Empty() {
			conflicts[owner] = &VersionedSet{
				Set:        conflictSet,
				APIVersion: ownerSet.APIVersion,
			}
		}
	}

	if !force && len(conflicts) != 0 {
		return nil, ConflictsFromOwners(conflicts)
	}

	for owner, conflictSet := range conflicts {
		owners[owner].Set = owners[owner].Set.Difference(conflictSet.Set)
	}

	return owners, nil
}

// Update is the method you should call once you've merged your final
// object on CREATE/UPDATE/PATCH verbs. newObject must be the object
// that you intend to persist (after applying the patch if this is for a
// PATCH call), and liveObject must be the original object (empty if
// this is a CREATE call).
func (s *Updater) Update(liveObject, newObject typed.TypedValue, owners Owners, owner string) (Owners, error) {
	var err error
	owners, err = s.update(liveObject, newObject, owners, owner, true)
	if err != nil {
		return Owners{}, fmt.Errorf("failed to update owners: %v", err)
	}
	compare, err := liveObject.Compare(newObject)
	if err != nil {
		return Owners{}, fmt.Errorf("failed to compare live and new objects: %v", err)
	}
	owners[owner].Set = owners[owner].Set.Union(compare.Modified).Union(compare.Added).Difference(compare.Removed)
	return owners, nil
}

// Apply should be called when Apply is run, given the current object as
// well as the configuration that is applied. This will merge the object
// and return it.
func (s *Updater) Apply(liveObject, configObject typed.TypedValue, owners Owners, owner string, force bool) (typed.TypedValue, Owners, error) {
	newObject, err := liveObject.Merge(configObject)
	if err != nil {
		return typed.TypedValue{}, Owners{}, fmt.Errorf("failed to merge config: %v", err)
	}
	owners, err = s.update(liveObject, newObject, owners, owner, force)
	if err != nil {
		return typed.TypedValue{}, Owners{}, fmt.Errorf("failed to update owners: %v", err)
	}

	// TODO: Remove unconflicting removed fields

	set, err := configObject.ToFieldSet()
	if err != nil {
		return typed.TypedValue{}, Owners{}, fmt.Errorf("failed to get field set: %v", err)
	}
	owners[owner] = &VersionedSet{
		Set:        set,
		APIVersion: APIVersion("v1"), // TODO: We don't support multiple versions yet.
	}
	return newObject, owners, nil
}
