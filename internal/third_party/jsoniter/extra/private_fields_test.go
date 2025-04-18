package extra

import (
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/structured-merge-diff/v6/internal/third_party/jsoniter"
)

func Test_private_fields(t *testing.T) {
	type TestObject struct {
		field1 string
	}
	SupportPrivateFields()
	should := require.New(t)
	obj := TestObject{}
	should.Nil(jsoniter.UnmarshalFromString(`{"field1":"Hello"}`, &obj))
	should.Equal("Hello", obj.field1)
}
