package typed_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

func TestInvalidOverride(t *testing.T) {
	// Exercises code path for invalidly specifying a scalar type is atomic
	parser, err := typed.NewParser(`
    types:
    - name: type
      map:
        fields:
          - name: field
            type:
              scalar: numeric
              elementRelationship: atomic
      `)

	if err != nil {
		t.Fatal(err)
	}

	sameVersionParser := fixture.SameVersionParser{T: parser.Type("type")}

	test := fixture.TestCase{
		Ops: []fixture.Operation{
			fixture.Apply{
				Manager: "apply_one",
				Object: `
                        field: 1
                    `,
				APIVersion: "v1",
			},
		},
		APIVersion: "v1",
		Error:      "no type found matching: inlined type",
	}

	test.TestOptionCombinations(t, sameVersionParser)
}
