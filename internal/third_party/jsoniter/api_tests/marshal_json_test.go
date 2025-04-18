package test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/structured-merge-diff/v6/internal/third_party/jsoniter"
)

type Foo struct {
	Bar interface{}
}

func (f Foo) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(f.Bar)
	return buf.Bytes(), err
}

// Standard Encoder has trailing newline.
func TestEncodeMarshalJSON(t *testing.T) {

	foo := Foo{
		Bar: 123,
	}
	should := require.New(t)
	var buf, stdbuf bytes.Buffer
	enc := jsoniter.ConfigCompatibleWithStandardLibrary.NewEncoder(&buf)
	enc.Encode(foo)
	stdenc := json.NewEncoder(&stdbuf)
	stdenc.Encode(foo)
	should.Equal(stdbuf.Bytes(), buf.Bytes())
}
