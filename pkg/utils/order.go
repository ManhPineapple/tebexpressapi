package utils

import (
	"strings"
	"tebexpressapi/pkg/utils/array"

	"github.com/spf13/viper"
)

func IsCustomerBlacklist(customerID int64) bool {
	blacklist := strings.TrimSpace(viper.GetString("blacklist.users"))
	if blacklist == "" {
		return false
	}

	ignoreUsers := array.SliceStringToSliceInt(strings.Split(blacklist, ","))
	return array.InArray(ignoreUsers, customerID)
}

func IsStateWhiteList(customerID int64) bool {
	blacklist := strings.TrimSpace(viper.GetString("white_list.state"))
	if blacklist == "" {
		return false
	}

	ignoreUsers := array.SliceStringToSliceInt(strings.Split(blacklist, ","))
	return array.InArray(ignoreUsers, customerID)
}
