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
	"github.com/kubernetes-sigs/structured-merge-diff/fieldpath"
)

type SetUpdater struct {
	converter Converter
}

func (s *SetUpdater) Update(old, nw *Object, workflow string, force bool) (map[string]*VersionedSet, error) {
	conflicts := map[string]*VersionedSet{}
	type Versioned struct {
		old *Object
		nw  *Object
	}
	versions := map[Version]Versioned{}

	owners := nw.OwnerSet()
	for owner, ownerSet := range owners {
		if owner == workflow {
			continue
		}
		versioned, ok := versions[ownerSet.Version]
		if !ok {
			versions[ownerSet.Version] = Versioned{
				old: s.converter.Convert(old, ownerSet.Version),
				nw:  s.converter.Convert(nw, ownerSet.Version),
			}
			versioned = versions[ownerSet.Version]
		}
		changed := fieldpath.SetFromDiff(versioned.old.Value, versioned.nw.Value)

		conflictSet := ownerSet.Intersection(changed)
		if !conflictSet.Empty() {
			conflicts[owner] = &VersionedSet{
				Set:     conflictSet,
				Version: ownerSet.Version,
			}
		}
	}

	if !force && len(conflicts) != 0 {
		return nil, ConflictsToError(conflicts)
	}

	for owner, conflictSet := range conflicts {
		owners[owner].Set = owners[owner].Set.Difference(conflictSet.Set)
	}

	return owners, nil
}

func (s *SetUpdater) NonApply(live, nw *Object, owner string) *Object {
	owners, _ := s.Update(live, nw, owner, true)
	changed := fieldpath.SetFromDiff(live.Value, nw.Value)
	removed := fieldpath.SetFromValue(live.Value).Difference(fieldpath.SetFromValue(nw.Value))
	owners[owner].Set = owners[owner].Set.Union(changed).Difference(removed)
	nw.SetOwnerSet(owners)
	return nw
}

func (s *SetUpdater) Apply(live, config *Object, owner string, force bool) (*Object, error) {
	nw := MergeApplied(live, config)
	owners, err := s.Update(live, nw, owner, force)
	if err != nil {
		return nil, err
	}

	// TODO: Remove unconflicting removed fields
	owners[owner] = &VersionedSet{
		Set:     fieldpath.SetFromValue(config.Value),
		Version: config.GroupVersion(),
	}
	nw.SetOwnerSet(owners)
	return nw, nil
}
