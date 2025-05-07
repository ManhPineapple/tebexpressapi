package constant

const (
	ServiceFBACode   = "FBA"
	ServiceUS48Code  = "US48"
	ServiceINUSCode  = "INUS"
	ServiceACTUSCode = "ACTUS"
	ServiceAUCode    = "A"
	ServiceAUFCode   = "AUB"
	ServiceUSCode    = "US"
	ServiceEUCode    = "EU"
	ServiceLABELCode = "LABEL"
	ServiceNDCode    = "ND"

	ServiceStandardCode        = "S"
	ServiceExpressCode         = "E"
	ServiceCNCode              = "CN"
	ServiceTiktokCode          = "T"
	ServiceWarehouseCode       = "WS"
	ServiceExpressPriorityCode = "EP"
	ServiceTiktokPriorityCode  = "TP"

	MaxWeightOz = 704

	ServiceSizeSmall     = 0
	ServiceSizeSmallCode = "smallsize"
	ServiceSizeLarge     = 450
	ServiceSizeLargeCode = "largesize"
	ServiceSizeOver      = 3000
	ServiceSizeOverCode  = "oversize"

	LimitExtraFee1                 = 55
	LimitExtraFee2                 = 76
	DomesticCarrierServiceGround   = "ground"
	DomesticCarrierServicePriority = "priority"
)

var priorityServiceSet = map[string]struct{}{
	ServiceExpressPriorityCode: {},
	ServiceTiktokPriorityCode:  {},
}

func IsPriorityService(code string) bool {
	_, ok := priorityServiceSet[code]
	return ok
}
