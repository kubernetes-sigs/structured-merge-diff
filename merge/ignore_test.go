package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v3/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v3/internal/fixture"
)

func TestIgnoredFields(t *testing.T) {
	tests := map[string]TestCase{
		"do_not_own_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("string"),
					),
				},
				ExpectState{
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
				},
				ExpectManagedFields{
					Manager: "default",
					Fields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
				},
			},
		},
		"do_not_steal_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
				},
				ExpectState{
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "some string"
					`,
				},
				ExpectManagedFields{
					Manager: "default",
					Fields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
						fieldpath.MakePathOrDie("string"),
					),
				},
				Update{
					Manager:    "default2",
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "no string"
					`,
					IgnoredFields: fieldpath.NewSet(fieldpath.MakePathOrDie("string")),
				},
				ExpectState{
					APIVersion: "v1",
					Object: `
						numeric: 1
						string: "no string"
					`,
				},
				ExpectManagedFields{
					Manager: "default2",
					Fields:  nil,
				},
			},
		},
		"do_not_own_deep_ignored": {
			APIVersion: "v1",
			Ops: []Operation{
				Update{
					Manager:    "default",
					APIVersion: "v1",
					Object:     `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
					IgnoredFields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("obj"),
					),
				},
				ExpectState{
					APIVersion: "v1",
					Object:     `{"numeric": 1, "obj": {"string": "foo", "numeric": 2}}`,
				},
				ExpectManagedFields{
					Manager: "default",
					Fields: fieldpath.NewSet(
						fieldpath.MakePathOrDie("numeric"),
					),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(DeducedParser); err != nil {
				t.Fatal("Should fail:", err)
			}
		})
	}
}
