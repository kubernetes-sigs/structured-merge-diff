/*
Copyright 2026 The Kubernetes Authors.
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
	"testing"

	. "sigs.k8s.io/structured-merge-diff/v7/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v7/typed"
)

// webhookParser models the ValidatingWebhookConfiguration shape from the
// PR #336 review: an associative list of webhooks keyed by name, where
// sideEffects and clientConfig are required and clientConfig is a map whose
// leaves (caBundle, service) are individually optional. This lets one manager
// own the required structural fields (full) while another injects only the
// nested optional caBundle (partial).
var webhookParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: webhooks
        type:
          namedType: webhookList
- name: webhookList
  list:
    elementType:
      namedType: webhook
    elementRelationship: associative
    keys:
    - name
- name: webhook
  map:
    fields:
    - name: name
      type:
        scalar: string
      required: true
    - name: sideEffects
      type:
        scalar: string
      required: true
    - name: clientConfig
      type:
        namedType: clientConfig
      required: true
- name: clientConfig
  map:
    fields:
    - name: caBundle
      type:
        scalar: string
    - name: service
      type:
        scalar: string
`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("type")}
}()

// noRequiredFieldsParser is an associative list whose element type declares no
// required fields beyond the key, so every claiming manager is vacuously a
// full item manager.
var noRequiredFieldsParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: list
        type:
          namedType: associativeList
- name: associativeList
  list:
    elementType:
      namedType: elem
    elementRelationship: associative
    keys:
    - name
- name: elem
  map:
    fields:
    - name: name
      type:
        scalar: string
    - name: a
      type:
        scalar: string
    - name: b
      type:
        scalar: string
`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("type")}
}()

// nestedRequiredFieldParser has an element whose only required non-key field is
// nested (conn.endpoint), plus an optional top-level field (caBundle). This
// isolates the two cases a purely structural (direct-member) heuristic gets
// wrong: a manager owning only conn.endpoint is full (no top-level member yet
// owns the required field), and a manager owning only caBundle is partial (has
// a top-level member but owns no required field).
var nestedRequiredFieldParser = func() Parser {
	parser, err := typed.NewParser(`types:
- name: type
  map:
    fields:
      - name: list
        type:
          namedType: associativeList
- name: associativeList
  list:
    elementType:
      namedType: elem
    elementRelationship: associative
    keys:
    - name
- name: elem
  map:
    fields:
    - name: name
      type:
        scalar: string
      required: true
    - name: conn
      type:
        namedType: conn
      required: true
    - name: caBundle
      type:
        scalar: string
- name: conn
  map:
    fields:
    - name: endpoint
      type:
        scalar: string
      required: true
`)
	if err != nil {
		panic(err)
	}
	return SameVersionParser{T: parser.Type("type")}
}()

// TestRequiredItemManager verifies the full-vs-partial item manager semantics:
// an associative list item is deleted when its last full manager (one owning
// all required fields of the element type) drops it, and a partial manager
// (owning only optional/nested fields) can neither delete the item nor block
// its deletion.
// Regression test for https://github.com/kubernetes-sigs/structured-merge-diff/issues/335
func TestRequiredItemManager(t *testing.T) {
	webhookTests := map[string]TestCase{
		"full_manager_can_delete_item": {
			// controller owns the required fields (sideEffects, and
			// clientConfig via clientConfig.service) → full. injector owns
			// only the optional nested clientConfig.caBundle → partial.
			// When controller drops the webhook, it is deleted because the
			// only remaining manager is partial.
			Ops: []Operation{
				Apply{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				ForceApply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  clientConfig:
						    caBundle: "ZmFrZQ=="
					`,
				},
				Apply{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
			},
			Object: `
				webhooks:
			`,
			APIVersion: "v1",
		},
		"full_manager_removes_own_item_keeping_another_managers_item": {
			// controller-a fully owns webhook "a"; controller-b fully owns a
			// different webhook "b". When controller-a drops "a", only "a" is
			// removed — "b" is untouched because controller-a never owned it.
			Ops: []Operation{
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: a
						  sideEffects: "None"
						  clientConfig:
						    service: "svc-a"
					`,
				},
				Apply{
					Manager:    "controller-b",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: b
						  sideEffects: "None"
						  clientConfig:
						    service: "svc-b"
					`,
				},
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
			},
			Object: `
				webhooks:
				- name: b
				  sideEffects: "None"
				  clientConfig:
				    service: "svc-b"
			`,
			APIVersion: "v1",
		},
		"partial_manager_cannot_delete_item": {
			// Same setup, but the partial injector drops the webhook while
			// the full controller remains. The webhook survives; only the
			// injector's caBundle is released.
			Ops: []Operation{
				Apply{
					Manager:    "controller",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				ForceApply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  clientConfig:
						    caBundle: "ZmFrZQ=="
					`,
				},
				Apply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
			},
			Object: `
				webhooks:
				- name: hook
				  sideEffects: "None"
				  clientConfig:
				    service: "svc"
			`,
			APIVersion: "v1",
		},
		"full_manager_drop_keeps_item_if_another_full_remains": {
			// Two full managers share ownership of the required fields.
			// When one drops the webhook, it survives because the other
			// full manager still owns all required fields.
			Ops: []Operation{
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				Apply{
					Manager:    "controller-b",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
			},
			Object: `
				webhooks:
				- name: hook
				  sideEffects: "None"
				  clientConfig:
				    service: "svc"
			`,
			APIVersion: "v1",
		},
		"last_full_manager_drop_deletes_item": {
			// Both full managers drop the webhook; with no full manager
			// left, the item is deleted.
			Ops: []Operation{
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				Apply{
					Manager:    "controller-b",
					APIVersion: "v1",
					Object: `
						webhooks:
						- name: hook
						  sideEffects: "None"
						  clientConfig:
						    service: "svc"
					`,
				},
				Apply{
					Manager:    "controller-a",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
				Apply{
					Manager:    "controller-b",
					APIVersion: "v1",
					Object: `
						webhooks:
					`,
				},
			},
			Object: `
				webhooks:
			`,
			APIVersion: "v1",
		},
	}

	for name, test := range webhookTests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(webhookParser); err != nil {
				t.Fatal(err)
			}
		})
	}

	noRequiredTests := map[string]TestCase{
		"no_required_fields_is_a_noop": {
			// With no required fields, every manager is vacuously full, so
			// the classifier strips nothing and the item survives while
			// apply-two still claims it even after apply-one drops it —
			// identical to the behaviour before this change.
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  a: "1"
					`,
				},
				ForceApply{
					Manager:    "apply-two",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  b: "2"
					`,
				},
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
					`,
				},
			},
			Object: `
				list:
				- name: x
				  b: "2"
			`,
			APIVersion: "v1",
		},
	}

	for name, test := range noRequiredTests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(noRequiredFieldsParser); err != nil {
				t.Fatal(err)
			}
		})
	}

	// These two cases are the ones a structural (direct-member) heuristic
	// classifies incorrectly; required-field ownership gets them right.
	nestedTests := map[string]TestCase{
		"full_owner_of_nested_required_field_preserves_item": {
			// owner is full purely by owning the required nested field
			// conn.endpoint (it has no top-level non-key member). When the
			// partial injector (optional caBundle) drops the item, it must
			// survive because a full manager still owns the required field.
			Ops: []Operation{
				Apply{
					Manager:    "owner",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  conn:
						    endpoint: "https://svc"
					`,
				},
				ForceApply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  caBundle: "ZmFrZQ=="
					`,
				},
				Apply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						list:
					`,
				},
			},
			Object: `
				list:
				- name: x
				  conn:
				    endpoint: "https://svc"
			`,
			APIVersion: "v1",
		},
		"owner_of_only_optional_field_cannot_block_deletion": {
			// injector owns a top-level field (caBundle) but it is optional,
			// so the injector is partial. When the full owner drops the item,
			// it must be deleted even though the injector keeps claiming the
			// optional field.
			Ops: []Operation{
				Apply{
					Manager:    "owner",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  conn:
						    endpoint: "https://svc"
					`,
				},
				ForceApply{
					Manager:    "injector",
					APIVersion: "v1",
					Object: `
						list:
						- name: x
						  caBundle: "ZmFrZQ=="
					`,
				},
				Apply{
					Manager:    "owner",
					APIVersion: "v1",
					Object: `
						list:
					`,
				},
			},
			Object: `
				list:
			`,
			APIVersion: "v1",
		},
	}

	for name, test := range nestedTests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(nestedRequiredFieldParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}
