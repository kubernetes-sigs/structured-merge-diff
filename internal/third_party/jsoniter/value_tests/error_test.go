package test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/structured-merge-diff/v6/internal/third_party/jsoniter"
)

func Test_errorInput(t *testing.T) {
	for _, testCase := range unmarshalCases {
		if testCase.obj != nil {
			continue
		}
		valType := reflect.TypeOf(testCase.ptr).Elem()
		t.Run(valType.String(), func(t *testing.T) {
			for _, data := range []string{
				`x`,
				`n`,
				`nul`,
				`{x}`,
				`{"x"}`,
				`{"x": "y"x}`,
				`{"x": "y"`,
				`{"x": "y", "a"}`,
				`[`,
				`[{"x": "y"}`,
			} {
				ptrVal := reflect.New(valType)
				ptr := ptrVal.Interface()
				err := jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal([]byte(data), ptr)
				require.Error(t, err, "on input %q", data)
			}
		})
	}
}
