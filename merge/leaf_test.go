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

var leafFieldsParser = func() *typed.Parser {
	parser, err := typed.NewParser(`types:
- name: leafFields
  struct:
    fields:
    - name: numeric
      type:
        scalar: numeric
    - name: string
      type:
        scalar: string
    - name: bool
      type:
        scalar: boolean`)
	if err != nil {
		panic(err)
	}
	return parser
}()

// Run apply twice with different objects, you own everything
// and the object looks exactly like the last one applied.
func TestApplyApplyLeaf(t *testing.T) {
	state := &State{
		Updater:  &merge.Updater{},
		Parser:   leafFieldsParser,
		Typename: "leafFields",
	}

	config := typed.YAMLObject(`
numeric: 1
string: "string"`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
numeric: 2
string: "string"
bool: false`)
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
				_P("numeric"), _P("string"), _P("bool"),
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
func TestApplyUpdateApplyLeaf(t *testing.T) {
	state := &State{
		Updater:  &merge.Updater{Converter: dummyConverter{}},
		Parser:   leafFieldsParser,
		Typename: "leafFields",
	}

	config := typed.YAMLObject(`
numeric: 1
string: "string"`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// Controller updates the value of "bool", doesn't change
	// anything else.
	config = typed.YAMLObject(`
numeric: 1
string: "string"
bool: true`)
	err = state.Update(config, "controller")
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// User applies a different string and different value
	config = typed.YAMLObject(`
numeric: 2
string: "new string"`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
numeric: 2
string: "new string"
bool: true`)
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
				_P("numeric"), _P("string"),
			),
			APIVersion: "v1",
		},
		"controller": &fieldpath.VersionedSet{
			Set: _NS(
				_P("bool"),
			),
			APIVersion: "v1",
		}}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Apply an object, controller updates some of your fields, apply again,
// you get a conflict, apply force, it gets resolved.
func TestApplyUpdateApplyWithConflictsLeaf(t *testing.T) {
	state := &State{
		Updater:  &merge.Updater{Converter: dummyConverter{}},
		Parser:   leafFieldsParser,
		Typename: "leafFields",
	}

	config := typed.YAMLObject(`
numeric: 1
string: "string"`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// Controller updates the value of "bool" and "string"
	config = typed.YAMLObject(`
numeric: 1
string: "controller string"
bool: true`)
	err = state.Update(config, "controller")
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// User applies a different string and different value,
	// they should get a conflict on string.
	config = typed.YAMLObject(`
numeric: 2
string: "user string"`)
	err = state.Apply(config, "default", false)
	want := merge.Conflicts{
		merge.Conflict{Manager: "controller", Path: _P("string")},
	}
	if got := err; !reflect.DeepEqual(err, want) {
		t.Fatalf("want %v, got %v", want, got)
	}

	// Apply force, they shouldn't get any conflict.
	err = state.Apply(config, "default", true)
	if err != nil {
		t.Fatalf("Failed to force-apply: %v", err)
	}

	config = typed.YAMLObject(`
numeric: 2
string: "user string"
bool: true`)
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
				_P("numeric"), _P("string"),
			),
			APIVersion: "v1",
		},
		"controller": &fieldpath.VersionedSet{
			Set: _NS(
				_P("bool"),
			),
			APIVersion: "v1",
		}}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}

// Run apply twice with different objects, you own what you apply and
// the fields you don't specify anymore are dangling.
func TestApplyApplyDanglingLeaf(t *testing.T) {
	state := &State{
		Updater:  &merge.Updater{},
		Parser:   leafFieldsParser,
		Typename: "leafFields",
	}

	config := typed.YAMLObject(`
numeric: 1
string: "string"
bool: true`)
	err := state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
string: "updated string"`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
numeric: 1
string: "updated string"
bool: true`)
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
				_P("string"),
			),
			APIVersion: "v1",
		},
	}
	if diff := state.Managers.Difference(wanted); len(diff) != 0 {
		t.Fatalf("Expected Managers to be %v, got %v", wanted, state.Managers)
	}
}
