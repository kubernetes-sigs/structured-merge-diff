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

package merge_test

import (
	"reflect"
	"testing"

	"sigs.k8s.io/structured-merge-diff/fieldpath"
	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

var setFieldsParser = func() *typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: sets
  struct:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		panic(err)
	}
	return parser.Type("sets")
}()

// Run apply twice with different objects, you own everything
// and the object looks exactly like the last one applied.
func TestApplyApplySet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- c`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}

	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("a")),
				_P("list", _SV("b")),
				_P("list", _SV("c")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		},
	}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Apply an object, controller updates a different field, apply again,
// you own the field you applied, controller owns their own, no conflicts.
func TestApplyUpdateApplySet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- c`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err = state.Update(config, "controller")
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- aprime
- c
- cprime`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- aprime
- b
- c
- cprime
- d`)
	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}

	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}
	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("a")),
				_P("list", _SV("aprime")),
				_P("list", _SV("c")),
				_P("list", _SV("cprime")),
			),
			APIVersion: "v1",
		},
		"controller": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("b")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		}}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Apply an object, controller adds new fields, apply again with some
// controller fields. you don't get a conflict, you don't own the
// fields. order is constant.
func TestApplyUpdateApplySimilarFieldsSet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- c`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err = state.Update(config, "controller")
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Failed to pply: %v", err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}

	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}
	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("a")),
				_P("list", _SV("c")),
			),
			APIVersion: "v1",
		},
		"controller": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("b")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		}}
	t.Skip("This test is wrong, user shouldn't own b")
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Run apply twice with different objects, you own what you apply and
// the fields you don't specify anymore are removed.
func TestApplyApplyRemovedSet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- b
- d`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	t.Skip("Test doesn't work because items should be removed.")
	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}

	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("b")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		},
	}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Run apply twice with different order, no problem.
func TestApplyApplyDifferentOrderSet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- d
- c
- b
- a`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}

	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list", _SV("a")),
				_P("list", _SV("b")),
				_P("list", _SV("c")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		},
	}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Run apply, controller changes order, apply again and get conflict.
// Force and your order is restored.
func TestApplyUpdateApplyDifferentOrderSet(t *testing.T) {
	state := &State{
		Updater: &merge.Updater{Converter: dummyConverter{}},
		Parser:  setFieldsParser,
	}

	config := typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- d
- c
- b
- a`)
	err = state.Update(config, "controller")
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- d
- b
- c`)
	err = state.Apply(config, "default", false)
	want := merge.Conflicts{
		merge.Conflict{Manager: "controller", Path: _P("list")},
	}
	t.Skip("We don't create conflict on list ordering yet.")
	if got := err; !reflect.DeepEqual(err, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Force-apply
	err = state.Apply(config, "default", true)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	comparison, err := state.CompareLive(config)
	if err != nil {
		t.Fatalf("Failed to compare live with config: %v", err)
	}
	if !comparison.IsSame() {
		t.Fatalf("Expected live and config to be the same: %v", comparison)
	}

	wanted := fieldpath.ManagedFields{
		"default": &fieldpath.VersionedSet{
			Set: _NS(
				_P("list"),
				_P("list", _SV("a")),
				_P("list", _SV("b")),
				_P("list", _SV("c")),
				_P("list", _SV("d")),
			),
			APIVersion: "v1",
		},
	}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}
