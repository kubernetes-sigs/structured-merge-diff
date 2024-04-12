package builder

import (
	"io"
	"testing"
)

func TestFastObjParse(t *testing.T) {
	testCases := map[string][]string{
		`{}`:                                   {},
		`{"a": 1, "b": {}}`:                    {`"a"`, `1`, `"b"`, `{}`},
		`{"a": 1, "b": 2}`:                     {`"a"`, `1`, `"b"`, `2`},
		`{"a": 1, "b": 2, "c": 3}`:             {`"a"`, `1`, `"b"`, `2`, `"c"`, `3`},
		`{"a": "1", "b": "2", "c": "3"}`:       {`"a"`, `"1"`, `"b"`, `"2"`, `"c"`, `"3"`},
		`{"a": "1", "b": {"c": 3}}`:            {`"a"`, `"1"`, `"b"`, `{"c": 3}`},
		`{"a": "1", "b": {"c": []}, "d": "4"}`: {`"a"`, `"1"`, `"b"`, `{"c": []}`, `"d"`, `"4"`},
		`{"port":443,"protocol":"tcp"}`:        {`"port"`, `443`, `"protocol"`, `"tcp"`},
	}

	for tc, ans := range testCases {
		tc := tc
		ans := ans
		t.Run(tc, func(t *testing.T) {
			parser := NewFastObjParser([]byte(tc))

			results := []string{}
			for {
				v, err := parser.Parse()
				if err == io.EOF {
					break
				} else if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				results = append(results, string(v))
			}

			if len(results) != len(ans) {
				t.Fatalf("unexpected results: %v", results)
			}

			for i := 0; i < len(results); i++ {
				if results[i] != ans[i] {
					t.Fatalf("unexpected results: got %v, want %v", results, ans)
				}
			}
		})
	}
}
