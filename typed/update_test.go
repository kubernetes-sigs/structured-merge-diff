package typed_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/internal/fixture"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
)

var updateParser = func() fixture.Parser {
	parser, err := typed.NewParser(`
types:
- name: nestedOptionalFields
  map:
    fields:
    - name: nestedList
      type:
        list:
          elementRelationship: associative
          keys:
          - name
          elementType:
            map:
                fields:
                - name: name
                  type:
                    scalar: string
                - name: value
                  type:
                    scalar: numeric                            
    - name: nestedMap
      type:
        map:
          elementType:
            scalar: numeric
    - name: nested
      type:
        map:
          fields:
            - name: numeric
              type:
                scalar: numeric
            - name: string
              type:
                scalar: string
`)
	if err != nil {
		panic(err)
	}
	return fixture.SameVersionParser{T: parser.Type("nestedOptionalFields")}
}()

func TestUpdate(t *testing.T) {
	tests := map[string]fixture.TestCase{
		"delete_nested_fields_struct": {
			Ops: []fixture.Operation{
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nested:
                            numeric: 1
                            string: my string
                    `,
				},
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nested: {}
                    `,
				},
			},
			APIVersion: `v1`,
			Object:     `{nested: {}}`,
		},
		"delete_nested_fields_list": {
			Ops: []fixture.Operation{
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nestedList:
                            - name: first
                            - name: second
                    `,
				},
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nestedList: []
                    `,
				},
			},
			APIVersion: `v1`,
			Object:     `{nestedList: []}`,
		},
		"delete_nested_fields_map": {
			Ops: []fixture.Operation{
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nestedMap:
                            first: 1
                            second: 2
                            
                    `,
				},
				fixture.Apply{
					Manager:    "default",
					APIVersion: "v1",
					Object: `
                        nestedMap: {}
                    `,
				},
			},
			APIVersion: `v1`,
			Object:     `{nestedMap: {}}`,
		},
	}

	for name, tc := range tests {
		tc2 := tc
		t.Run(name, func(t *testing.T) {
			typed.REMOVEKEEPEMPTYCOLLECTIONS = true
			if err := tc.Test(updateParser); err != nil {
				t.Fatal(err)
			}
		})

		t.Run(name+"Nil", func(t *testing.T) {
			typed.REMOVEKEEPEMPTYCOLLECTIONS = false
			if err := tc2.Test(updateParser); err != nil {
				t.Fatal(err)
			}
		})
	}

}
