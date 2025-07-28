package utils

import (
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
)

func GetServiceIDToCalculatePrice(CustomTiktokBarcode *string, service *entity.Service) int64 {
	var serviceIDToCalculatePrice int64
	if CustomTiktokBarcode != nil && *CustomTiktokBarcode != "" {
		if constant.IsPriorityService(service.Code) {
			serviceIDToCalculatePrice = 28 // tiktok priority price
		} else if service.Code == constant.ServiceTebprintHubCode {
			serviceIDToCalculatePrice = 11 // labeled tebprinthub price
		} else {
			serviceIDToCalculatePrice = 25 // tiktok price
		}
	} else {
		serviceIDToCalculatePrice = service.ID
	}

	return serviceIDToCalculatePrice
}
