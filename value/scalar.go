/*
Copyright 2019 The Kubernetes Authors.

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

package value

// IsFloat returns true if the value can be converted to a float, false
// otherwise.
func IsFloat(v Value) bool {
	if v == nil {
		return false
	} else if _, ok := v.(float64); ok {
		return true
	} else if _, ok := v.(float32); ok {
		return true
	}
	return false
}

// ValueFloat will convert value into a float64, and panic if it can't.
func ValueFloat(v Value) float64 {
	if f, ok := v.(float32); ok {
		return float64(f)
	}
	return v.(float64)
}

// Compare compares floats. The result will be 0 if lhs==rhs, -1 if f <
// rhs, and +1 if f > rhs.
func FloatCompare(lhs, rhs float64) int {
	if lhs > rhs {
		return 1
	} else if lhs < rhs {
		return -1
	}
	return 0
}

// IsInt returns true if the value can be converted to an integer, false
// otherwise.
func IsInt(v Value) bool {
	if v == nil {
		return false
	} else if _, ok := v.(int); ok {
		return true
	} else if _, ok := v.(int8); ok {
		return true
	} else if _, ok := v.(int16); ok {
		return true
	} else if _, ok := v.(int32); ok {
		return true
	} else if _, ok := v.(int64); ok {
		return true
	}
	return false
}

// ValueInt will convert value into a int64, and panic if it can't.
func ValueInt(v Value) int64 {
	if i, ok := v.(int); ok {
		return int64(i)
	} else if i, ok := v.(int8); ok {
		return int64(i)
	} else if i, ok := v.(int16); ok {
		return int64(i)
	} else if i, ok := v.(int32); ok {
		return int64(i)
	}
	return v.(int64)
}

// IntCompare compares integers. The result will be 0 if i==rhs, -1 if i <
// rhs, and +1 if i > rhs.
func IntCompare(lhs, rhs int64) int {
	if lhs > rhs {
		return 1
	} else if lhs < rhs {
		return -1
	}
	return 0
}

// IsString returns true if the value can be converted to a string, false
// otherwise.
func IsString(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.(string)
	return ok
}

// ValueString will convert value into a string, and panic if it can't.
func ValueString(v Value) string {
	return v.(string)
}

// IsBool returns true if the value can be converted to a boolean, false
// otherwise.
func IsBool(v Value) bool {
	if v == nil {
		return false
	}
	_, ok := v.(bool)
	return ok
}

// ValueBool will convert value into a boolean, and panic if it can't.
func ValueBool(v Value) bool {
	return v.(bool)
}

// Compare compares booleans. The result will be 0 if b==rhs, -1 if b <
// rhs, and +1 if b > rhs.
func BoolCompare(lhs, rhs bool) int {
	if lhs == rhs {
		return 0
	} else if lhs == false {
		return -1
	}
	return 1
}

// IsNull returns true if the value is null, false otherwise.
func IsNull(v Value) bool {
	return v == nil
}
