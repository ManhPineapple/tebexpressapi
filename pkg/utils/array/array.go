package array

import (
	"strconv"

	"github.com/spf13/cast"
)

func UniqueString(intSlice []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func UniqueInt(intSlice []int64) []int64 {
	keys := make(map[int64]bool)
	var list []int64
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func InArrayString(intSlice []string, value string) bool {
	for _, v := range intSlice {
		if v == value {
			return true
		}
	}
	return false
}

func InArray(intSlice []int64, value int64) bool {
	for _, v := range intSlice {
		if v == value {
			return true
		}
	}
	return false
}

func SliceStringToSliceInt(stringSlice []string) []int64 {
	intSlice := make([]int64, 0)

	for _, i := range stringSlice {
		j := cast.ToInt64(i)
		intSlice = append(intSlice, j)
	}

	return intSlice
}

func Int64ToString(intSlice []int64) []string {
	var list []string
	for _, v := range intSlice {
		list = append(list, strconv.FormatInt(v, 10))
	}

	return list
}
