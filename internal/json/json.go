package json

import (
	jsonv1 "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"math"
	"strconv"
)

// decodeOptions are the options used whenever this package parses JSON.
// Invalid UTF-8 is tolerated (the offending bytes are replaced with the
// Unicode replacement character) rather than rejected, matching the
// behaviour of the JSON parser this package used previously.
var allowInvalidUTF8 = jsontext.AllowInvalidUTF8(true)

// allowDuplicates allows tolerating duplicates when decoding or encoding.
var allowDuplicates = jsontext.AllowDuplicateNames(true)

// v1Merging matches json/v1 merging behavior for repeated keys
var v1Merging = jsonv1.MergeWithLegacySemantics(true)

// preserveIntegers decodes a JSON number into an `any` as an int64 when it is an
// integer literal that fits in an int64, and as a float64 otherwise. The default
// decoding to float64 silently rounds integers above 2^53, so two distinct
// int64 values (for example list map keys) could not be told apart after a
// round trip.
var preserveIntegers = json.WithUnmarshalers(json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
	switch dec.PeekKind() {
	case '0':
		val, err := dec.ReadValue()
		if err != nil {
			return err
		}
		if i, err := strconv.ParseInt(string(val), 10, 64); err == nil {
			*v = i
			return nil
		}
		f, err := strconv.ParseFloat(string(val), 64)
		if err != nil && !math.IsInf(f, 0) {
			return err
		}
		*v = f
		return nil
	case '{':
		var m map[string]any
		err := json.UnmarshalDecode(dec, &m)
		*v = m
		return err
	case '[':
		var l []any
		err := json.UnmarshalDecode(dec, &l)
		*v = l
		return err
	case '"':
		var str string
		err := json.UnmarshalDecode(dec, &str)
		*v = str
		return err
	case 't', 'f':
		var b bool
		err := json.UnmarshalDecode(dec, &b)
		*v = b
		return err
	default:
		*v = nil
		return dec.SkipValue()
	}
}))

// ReadValueToAnyMergingDuplicates reads the next value to an `any` type, returning any error.
// Sets options AllowInvalidUTF8=true, AllowDuplicateNames=true, MergeWithLegacySemantics=true.
func ReadValueToAnyMergingDuplicates(dec *jsontext.Decoder) (any, error) {
	data, err := dec.ReadValue()
	if err != nil {
		return nil, err
	}
	return UnmarshalToAnyMergingDuplicates(data)
}

// ReadValueToAnyPreservingIntegers is ReadValueToAnyMergingDuplicates, but integer
// literals that fit in an int64 are returned as int64 instead of float64.
func ReadValueToAnyPreservingIntegers(dec *jsontext.Decoder) (any, error) {
	data, err := dec.ReadValue()
	if err != nil {
		return nil, err
	}
	return UnmarshalToAnyPreservingIntegers(data)
}

// UnmarshalToAnyPreservingIntegers is UnmarshalToAnyMergingDuplicates, but integer
// literals too large for a float64 to hold exactly are returned as int64 instead of
// being rounded. Input without such a number takes the unchanged fast path.
func UnmarshalToAnyPreservingIntegers(data []byte) (any, error) {
	if !hasLongDigitRun(data) {
		return UnmarshalToAnyMergingDuplicates(data)
	}
	var v any
	if err := json.Unmarshal(data, &v, allowInvalidUTF8, preserveIntegers); !errors.Is(err, jsontext.ErrDuplicateName) {
		return v, err
	}
	v = nil
	return v, json.Unmarshal(data, &v, allowInvalidUTF8, allowDuplicates, v1Merging, preserveIntegers)
}

// hasLongDigitRun reports whether data has 16 or more consecutive digits. Every
// integer with 15 digits or fewer is exactly representable as a float64, so only
// longer runs can lose precision.
func hasLongDigitRun(data []byte) bool {
	run := 0
	for _, c := range data {
		if c >= '0' && c <= '9' {
			run++
			if run >= 16 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

// UnmarshalToAnyMergingDuplicates unmarshals data to an `any` type, returning any error.
// Sets options AllowInvalidUTF8=true, AllowDuplicateNames=true, MergeWithLegacySemantics=true.
func UnmarshalToAnyMergingDuplicates(data []byte) (any, error) {
	// try the non-duplicate path first, since we don't expect duplicates and json/v2 has an optimized non-merging version
	var v any
	if err := json.Unmarshal(data, &v, allowInvalidUTF8); !errors.Is(err, jsontext.ErrDuplicateName) {
		return v, err
	}
	// only if we got an ErrDuplicateName should we fall back to the slower unmarshal allowing duplicates and merging
	v = nil
	return v, json.Unmarshal(data, &v, allowInvalidUTF8, allowDuplicates, v1Merging)
}
