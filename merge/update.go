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

	"sigs.k8s.io/structured-merge-diff/v7/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v7/schema"
	"sigs.k8s.io/structured-merge-diff/v7/typed"
	"sigs.k8s.io/structured-merge-diff/v7/value"
)

// Converter is an interface to the conversion logic. The converter
// needs to be able to convert objects from one version to another.
type Converter interface {
	Convert(object *typed.TypedValue, version fieldpath.APIVersion) (*typed.TypedValue, error)
	IsMissingVersionError(error) bool
}

// UpdateBuilder allows you to create a new Updater by exposing all of
// the options and setting them once.
type UpdaterBuilder struct {
	Converter    Converter
	IgnoreFilter map[fieldpath.APIVersion]fieldpath.Filter

	// IgnoredFields provides a set of fields to ignore for each
	IgnoredFields map[fieldpath.APIVersion]*fieldpath.Set

	// Stop comparing the new object with old object after applying.
	// This was initially used to avoid spurious etcd update, but
	// since that's vastly inefficient, we've come-up with a better
	// way of doing that. Create this flag to stop it.
	// Comparing has become more expensive too now that we're not using
	// `Compare` but `value.Equals` so this gives an option to avoid it.
	ReturnInputOnNoop bool
}

func (u *UpdaterBuilder) BuildUpdater() *Updater {
	return &Updater{
		Converter:         u.Converter,
		IgnoreFilter:      u.IgnoreFilter,
		IgnoredFields:     u.IgnoredFields,
		returnInputOnNoop: u.ReturnInputOnNoop,
	}
}

// Updater is the object used to compute updated FieldSets and also
// merge the object on Apply.
type Updater struct {
	// Deprecated: This will eventually become private.
	Converter Converter

	// Deprecated: This will eventually become private.
	IgnoredFields map[fieldpath.APIVersion]*fieldpath.Set

	// Deprecated: This will eventually become private.
	IgnoreFilter map[fieldpath.APIVersion]fieldpath.Filter

	returnInputOnNoop bool
}

func (s *Updater) update(oldObject, newObject *typed.TypedValue, version fieldpath.APIVersion, managers fieldpath.ManagedFields, workflow string, force bool) (fieldpath.ManagedFields, *typed.Comparison, error) {
	conflicts := fieldpath.ManagedFields{}
	removed := fieldpath.ManagedFields{}
	compare, err := oldObject.Compare(newObject)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compare objects: %v", err)
	}

	var versions map[fieldpath.APIVersion]*typed.Comparison

	if s.IgnoredFields != nil && s.IgnoreFilter != nil {
		return nil, nil, fmt.Errorf("IgnoreFilter and IgnoreFilter may not both be set")
	}
	if s.IgnoredFields != nil {
		versions = map[fieldpath.APIVersion]*typed.Comparison{
			version: compare.ExcludeFields(s.IgnoredFields[version]),
		}
	} else {
		versions = map[fieldpath.APIVersion]*typed.Comparison{
			version: compare.FilterFields(s.IgnoreFilter[version]),
		}
	}

	for manager, managerSet := range managers {
		if manager == workflow {
			continue
		}
		compare, ok := versions[managerSet.APIVersion()]
		if !ok {
			var err error
			versionedOldObject, err := s.Converter.Convert(oldObject, managerSet.APIVersion())
			if err != nil {
				if s.Converter.IsMissingVersionError(err) {
					delete(managers, manager)
					continue
				}
				return nil, nil, fmt.Errorf("failed to convert old object: %v", err)
			}
			versionedNewObject, err := s.Converter.Convert(newObject, managerSet.APIVersion())
			if err != nil {
				if s.Converter.IsMissingVersionError(err) {
					delete(managers, manager)
					continue
				}
				return nil, nil, fmt.Errorf("failed to convert new object: %v", err)
			}
			compare, err = versionedOldObject.Compare(versionedNewObject)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to compare objects: %v", err)
			}

			if s.IgnoredFields != nil {
				versions[managerSet.APIVersion()] = compare.ExcludeFields(s.IgnoredFields[managerSet.APIVersion()])
			} else {
				versions[managerSet.APIVersion()] = compare.FilterFields(s.IgnoreFilter[managerSet.APIVersion()])
			}
		}

		conflictSet := managerSet.Set().Intersection(compare.Modified.Union(compare.Added))
		if !conflictSet.Empty() {
			conflicts[manager] = fieldpath.NewVersionedSet(conflictSet, managerSet.APIVersion(), false)
		}

		if !compare.Removed.Empty() {
			removed[manager] = fieldpath.NewVersionedSet(compare.Removed, managerSet.APIVersion(), false)
		}
	}

	if !force && len(conflicts) != 0 {
		return nil, nil, ConflictsFromManagers(conflicts)
	}

	for manager, conflictSet := range conflicts {
		managers[manager] = fieldpath.NewVersionedSet(managers[manager].Set().Difference(conflictSet.Set()), managers[manager].APIVersion(), managers[manager].Applied())
	}

	for manager, removedSet := range removed {
		managers[manager] = fieldpath.NewVersionedSet(managers[manager].Set().Difference(removedSet.Set()), managers[manager].APIVersion(), managers[manager].Applied())
	}

	for manager := range managers {
		if managers[manager].Set().Empty() {
			delete(managers, manager)
		}
	}

	return managers, compare, nil
}

// Update is the method you should call once you've merged your final
// object on CREATE/UPDATE/PATCH verbs. newObject must be the object
// that you intend to persist (after applying the patch if this is for a
// PATCH call), and liveObject must be the original object (empty if
// this is a CREATE call).
func (s *Updater) Update(liveObject, newObject *typed.TypedValue, version fieldpath.APIVersion, managers fieldpath.ManagedFields, manager string) (*typed.TypedValue, fieldpath.ManagedFields, error) {
	var err error
	managers, err = s.reconcileManagedFieldsWithSchemaChanges(liveObject, managers)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, err
	}
	managers, compare, err := s.update(liveObject, newObject, version, managers, manager, true)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, err
	}
	if _, ok := managers[manager]; !ok {
		managers[manager] = fieldpath.NewVersionedSet(fieldpath.NewSet(), version, false)
	}
	set := managers[manager].Set().Difference(compare.Removed).Union(compare.Modified).Union(compare.Added)

	if s.IgnoredFields != nil && s.IgnoreFilter != nil {
		return nil, nil, fmt.Errorf("IgnoreFilter and IgnoreFilter may not both be set")
	}
	var ignoreFilter fieldpath.Filter
	if s.IgnoredFields != nil {
		ignoreFilter = fieldpath.NewExcludeSetFilter(s.IgnoredFields[version])
	} else {
		ignoreFilter = s.IgnoreFilter[version]
	}
	if ignoreFilter != nil {
		set = ignoreFilter.Filter(set)
	}

	managers[manager] = fieldpath.NewVersionedSet(
		set,
		version,
		false,
	)
	if managers[manager].Set().Empty() {
		delete(managers, manager)
	}
	return newObject, managers, nil
}

// Apply should be called when Apply is run, given the current object as
// well as the configuration that is applied. This will merge the object
// and return it.
func (s *Updater) Apply(liveObject, configObject *typed.TypedValue, version fieldpath.APIVersion, managers fieldpath.ManagedFields, manager string, force bool) (*typed.TypedValue, fieldpath.ManagedFields, error) {
	var err error
	managers, err = s.reconcileManagedFieldsWithSchemaChanges(liveObject, managers)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, err
	}
	newObject, err := liveObject.Merge(configObject)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, fmt.Errorf("failed to merge config: %v", err)
	}
	lastSet := managers[manager]
	set, err := configObject.ToFieldSet()
	if err != nil {
		return nil, fieldpath.ManagedFields{}, fmt.Errorf("failed to get field set: %v", err)
	}

	if s.IgnoredFields != nil && s.IgnoreFilter != nil {
		return nil, nil, fmt.Errorf("IgnoreFilter and IgnoreFilter may not both be set")
	}
	var ignoreFilter fieldpath.Filter
	if s.IgnoredFields != nil {
		ignoreFilter = fieldpath.NewExcludeSetFilter(s.IgnoredFields[version])
	} else {
		ignoreFilter = s.IgnoreFilter[version]
	}
	if ignoreFilter != nil {
		set = ignoreFilter.Filter(set)
	}
	managers[manager] = fieldpath.NewVersionedSet(set, version, true)
	newObject, err = s.prune(newObject, managers, manager, lastSet)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, fmt.Errorf("failed to prune fields: %v", err)
	}
	managers, _, err = s.update(liveObject, newObject, version, managers, manager, force)
	if err != nil {
		return nil, fieldpath.ManagedFields{}, err
	}
	if !s.returnInputOnNoop && value.EqualsUsing(value.NewFreelistAllocator(), liveObject.AsValue(), newObject.AsValue()) {
		newObject = nil
	}
	return newObject, managers, nil
}

// prune will remove a field, list or map item, iff:
// * applyingManager applied it last time
// * applyingManager didn't apply it this time
// * no other applier claims to manage it
func (s *Updater) prune(merged *typed.TypedValue, managers fieldpath.ManagedFields, applyingManager string, lastSet fieldpath.VersionedSet) (*typed.TypedValue, error) {
	if lastSet == nil || lastSet.Set().Empty() {
		return merged, nil
	}
	version := lastSet.APIVersion()
	convertedMerged, err := s.Converter.Convert(merged, version)
	if err != nil {
		if s.Converter.IsMissingVersionError(err) {
			return merged, nil
		}
		return nil, fmt.Errorf("failed to convert merged object to last applied version: %v", err)
	}

	sc, tr := convertedMerged.Schema(), convertedMerged.TypeRef()
	pruned := convertedMerged.RemoveItems(lastSet.Set().EnsureNamedFieldsAreMembers(sc, tr))
	pruned, err = s.addBackOwnedItems(convertedMerged, pruned, version, managers, applyingManager)
	if err != nil {
		return nil, fmt.Errorf("failed add back owned items: %v", err)
	}
	pruned, err = s.addBackDanglingItems(convertedMerged, pruned, lastSet)
	if err != nil {
		return nil, fmt.Errorf("failed add back dangling items: %v", err)
	}
	return s.Converter.Convert(pruned, managers[applyingManager].APIVersion())
}

// addBackOwnedItems adds back any fields, list and map items that were removed by prune,
// but other appliers or updaters (or the current applier's new config) claim to own.
func (s *Updater) addBackOwnedItems(merged, pruned *typed.TypedValue, prunedVersion fieldpath.APIVersion, managedFields fieldpath.ManagedFields, applyingManager string) (*typed.TypedValue, error) {
	var err error
	// Cache of the root type per API version, so merged is converted at
	// most once per version to resolve manager sets against their schema.
	// A missing entry means not yet computed; a zero TypeRef means the
	// version could not be resolved (stripping is then skipped).
	typeByVersion := map[fieldpath.APIVersion]schema.TypeRef{}
	rootType := func(version fieldpath.APIVersion) (schema.TypeRef, bool) {
		if tr, cached := typeByVersion[version]; cached {
			return tr, tr != schema.TypeRef{}
		}
		var tr schema.TypeRef
		if converted, cerr := s.Converter.Convert(merged, version); cerr == nil {
			tr = converted.TypeRef()
		}
		typeByVersion[version] = tr
		return tr, tr != schema.TypeRef{}
	}

	managedAtVersion := map[fieldpath.APIVersion]*fieldpath.Set{}
	for manager, managerSet := range managedFields {
		set := managerSet.Set()
		// Strip auto-inserted list item keys from non-applying appliers
		// that are only partial managers of the item (they do not own all
		// required fields of the item's element type). The field set and
		// the schema must be resolved at the same API version. If the
		// version cannot be resolved, the set is left untouched (keys are
		// kept, the safe non-deleting behavior).
		if manager != applyingManager && managerSet.Applied() {
			if tr, ok := rootType(managerSet.APIVersion()); ok {
				set = removePartialManagerKeys(set, merged.Schema(), tr)
			}
		}
		if _, ok := managedAtVersion[managerSet.APIVersion()]; !ok {
			managedAtVersion[managerSet.APIVersion()] = fieldpath.NewSet()
		}
		managedAtVersion[managerSet.APIVersion()] = managedAtVersion[managerSet.APIVersion()].Union(set)
	}
	// Add back owned items at pruned version first to avoid conversion failure
	// caused by pruned fields which are required for conversion.
	if managed, ok := managedAtVersion[prunedVersion]; ok {
		merged, pruned, err = s.addBackOwnedItemsForVersion(merged, pruned, prunedVersion, managed)
		if err != nil {
			return nil, err
		}
		delete(managedAtVersion, prunedVersion)
	}
	for version, managed := range managedAtVersion {
		merged, pruned, err = s.addBackOwnedItemsForVersion(merged, pruned, version, managed)
		if err != nil {
			return nil, err
		}
	}
	return pruned, nil
}

// addBackOwnedItemsForVersion adds back any fields, list and map items that were removed by prune with specific managed field path at a version.
// It is an extracted sub-function from addBackOwnedItems for code reuse.
func (s *Updater) addBackOwnedItemsForVersion(merged, pruned *typed.TypedValue, version fieldpath.APIVersion, managed *fieldpath.Set) (*typed.TypedValue, *typed.TypedValue, error) {
	var err error
	merged, err = s.Converter.Convert(merged, version)
	if err != nil {
		if s.Converter.IsMissingVersionError(err) {
			return merged, pruned, nil
		}
		return nil, nil, fmt.Errorf("failed to convert merged object at version %v: %v", version, err)
	}
	pruned, err = s.Converter.Convert(pruned, version)
	if err != nil {
		if s.Converter.IsMissingVersionError(err) {
			return merged, pruned, nil
		}
		return nil, nil, fmt.Errorf("failed to convert pruned object at version %v: %v", version, err)
	}
	mergedSet, err := merged.ToFieldSet()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create field set from merged object at version %v: %v", version, err)
	}
	prunedSet, err := pruned.ToFieldSet()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create field set from pruned object at version %v: %v", version, err)
	}
	sc, tr := merged.Schema(), merged.TypeRef()
	pruned = merged.RemoveItems(mergedSet.EnsureNamedFieldsAreMembers(sc, tr).Difference(prunedSet.EnsureNamedFieldsAreMembers(sc, tr).Union(managed.EnsureNamedFieldsAreMembers(sc, tr))))
	return merged, pruned, nil
}

// addBackDanglingItems makes sure that the fields list and map items removed by prune were
// previously owned by the currently applying manager. This will add back fields list and map items
// that are unowned or that are owned by Updaters and shouldn't be removed.
func (s *Updater) addBackDanglingItems(merged, pruned *typed.TypedValue, lastSet fieldpath.VersionedSet) (*typed.TypedValue, error) {
	convertedPruned, err := s.Converter.Convert(pruned, lastSet.APIVersion())
	if err != nil {
		if s.Converter.IsMissingVersionError(err) {
			return merged, nil
		}
		return nil, fmt.Errorf("failed to convert pruned object to last applied version: %v", err)
	}
	prunedSet, err := convertedPruned.ToFieldSet()
	if err != nil {
		return nil, fmt.Errorf("failed to create field set from pruned object in last applied version: %v", err)
	}
	mergedSet, err := merged.ToFieldSet()
	if err != nil {
		return nil, fmt.Errorf("failed to create field set from merged object in last applied version: %v", err)
	}
	sc, tr := merged.Schema(), merged.TypeRef()
	prunedSet = prunedSet.EnsureNamedFieldsAreMembers(sc, tr)
	mergedSet = mergedSet.EnsureNamedFieldsAreMembers(sc, tr)
	last := lastSet.Set().EnsureNamedFieldsAreMembers(sc, tr)
	return merged.RemoveItems(mergedSet.Difference(prunedSet).Intersection(last)), nil
}

// reconcileManagedFieldsWithSchemaChanges reconciles the managed fields with any changes to the
// object's schema since the managed fields were written.
//
// Supports:
// - changing types from atomic to granular
// - changing types from granular to atomic
func (s *Updater) reconcileManagedFieldsWithSchemaChanges(liveObject *typed.TypedValue, managers fieldpath.ManagedFields) (fieldpath.ManagedFields, error) {
	result := fieldpath.ManagedFields{}
	for manager, versionedSet := range managers {
		tv, err := s.Converter.Convert(liveObject, versionedSet.APIVersion())
		if s.Converter.IsMissingVersionError(err) { // okay to skip, obsolete versions will be deleted automatically anyway
			continue
		}
		if err != nil {
			return nil, err
		}
		reconciled, err := typed.ReconcileFieldSetWithSchema(versionedSet.Set(), tv)
		if err != nil {
			return nil, err
		}
		if reconciled != nil {
			result[manager] = fieldpath.NewVersionedSet(reconciled, versionedSet.APIVersion(), versionedSet.Applied())
		} else {
			result[manager] = versionedSet
		}
	}
	return result, nil
}

// removePartialManagerKeys walks s alongside the schema rooted at tr (which
// must be the API version of s) and strips auto-inserted list item key
// members for items whose element type has required fields that this manager
// does not fully own. Such a manager is a partial item manager and must not
// prevent a full item manager (one that owns all required fields) from
// deleting the item.
//
// Full vs partial is defined by required-field ownership: full = owns every
// required field of the item's element type; partial = owns only a subset of
// required fields, or only optional/nested fields. An element type with no
// required fields leaves every manager vacuously full, so such items' keys
// are always preserved (an item survives while any manager claims it). When
// a type cannot be resolved, keys are kept — the safe, non-deleting default.
func removePartialManagerKeys(s *fieldpath.Set, sc *schema.Schema, tr schema.TypeRef) *fieldpath.Set {
	a, _ := sc.Resolve(tr)
	result := fieldpath.NewSet()
	s.Members.Iterate(func(pe fieldpath.PathElement) {
		if pe.Key == nil {
			result.Members.Insert(pe)
			return
		}
		// pe is a list item key member. Keep it iff this manager owns all
		// required fields of the list's element type (i.e. is a full item
		// manager). ownsAllRequiredFields returns true for non-map or
		// unresolvable element types, so such keys are always kept.
		keep := true
		if a.List != nil {
			if child, ok := s.Children.Get(pe); ok {
				keep = ownsAllRequiredFields(child, sc, a.List.ElementType)
			}
		}
		if keep {
			result.Members.Insert(pe)
		}
	})
	s.Children.Iterate(func(pe fieldpath.PathElement) {
		child, _ := s.Children.Get(pe)
		cleaned := removePartialManagerKeys(child, sc, childTypeRef(sc, a, pe))
		if !cleaned.Empty() {
			*result.Children.Descend(pe) = *cleaned
		}
	})
	return result
}

// childTypeRef returns the type of the subtree reached by descending pe
// from a node whose resolved atom is a. A zero TypeRef is returned when
// the type cannot be determined (callers treat that as "keep as-is").
func childTypeRef(sc *schema.Schema, a schema.Atom, pe fieldpath.PathElement) schema.TypeRef {
	switch {
	case a.Map != nil && pe.FieldName != nil:
		if sf, ok := a.Map.FindField(*pe.FieldName); ok {
			return sf.Type
		}
		return a.Map.ElementType
	case a.List != nil && pe.Key != nil:
		return a.List.ElementType
	default:
		return schema.TypeRef{}
	}
}

// ownsAllRequiredFields reports whether itemSet (a manager's owned fields
// for a single map item) covers every required field of the map type tr.
// A required scalar or list field must be owned as a member. A required map
// field is owned either as an (empty) member or by owning a subtree that
// covers its own required subfields.
func ownsAllRequiredFields(itemSet *fieldpath.Set, sc *schema.Schema, tr schema.TypeRef) bool {
	a, ok := sc.Resolve(tr)
	if !ok || a.Map == nil {
		return true
	}
	for _, f := range a.Map.Fields {
		if !f.Required {
			continue
		}
		pe := fieldpath.PathElement{FieldName: &f.Name}
		if itemSet.Members.Has(pe) {
			continue
		}
		if fa, ok := sc.Resolve(f.Type); ok && fa.Map != nil {
			if child, ok := itemSet.Children.Get(pe); ok && ownsAllRequiredFields(child, sc, f.Type) {
				continue
			}
		}
		return false
	}
	return true
}
