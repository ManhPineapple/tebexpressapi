package utils

import "time"

// String return a pointer to the string value
func String(v string) *string {
	return &v
}

// StringValue return value of the string pointer of
// "" if the pointer is null
func StringValue(v *string) string {
	if v != nil {
		return *v
	}

	return ""
}

// Int return a pointer to the int value
func Int(v int) *int {
	return &v
}

// IntValue return value of the int pointer of
// 0 if the pointer is null
func IntValue(v *int) int {
	if v != nil {
		return *v
	}

	return 0
}

// Int64 return a pointer to the int64 value
func Int64(v int64) *int64 {
	return &v
}

// Float64 return a pointer to the float64 value
func Float64(v float64) *float64 {
	return &v
}

// Int64Value return value of the int64 pointer of
// 0 if the pointer is null
func Int64Value(v *int64) int64 {
	if v != nil {
		return *v
	}

	return 0
}

// Float64Value return value of the float64 pointer of
// 0 if the pointer is null
func Float64Value(v *float64) float64 {
	if v != nil {
		return *v
	}

	return 0
}

// Int64 return a pointer to the int64 value
func Bool(v bool) *bool {
	return &v
}

// BoolValue return value of the bool pointer of
// 0 if the pointer is null
func BoolValue(v *bool) bool {
	if v != nil {
		return *v
	}

	return false
}

func Time(v *time.Time) time.Time {
	if v != nil {
		return *v
	}

	return time.Now()
}
