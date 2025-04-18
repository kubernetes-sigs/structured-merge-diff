package any_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/structured-merge-diff/v6/internal/third_party/jsoniter"
)

func Test_read_null_as_any(t *testing.T) {
	should := require.New(t)
	any := jsoniter.Get([]byte(`null`))
	should.Equal(0, any.ToInt())
	should.Equal(float64(0), any.ToFloat64())
	should.Equal("", any.ToString())
	should.False(any.ToBool())
}
