package constant

const (
	PackageCodeEnable    = 1
	PackageCodeDisable   = 0
	PackageCodeTemp      = 2
	MaxPackageIDGenerate = 99999999

	ExtraFeeStatusDisable = 0
	ExtraFeeStatusEnable  = 1
	ExtraFeeStatusUnPaid  = 10
	ExtraFeeShow          = 1
	ExtraFeeHide          = 0

	PackageStatusCreated              = 1
	PackageStatusCNPurchased          = 3
	PackageStatusPendingPickup        = 2
	PackageStatusPicked               = 10
	PackageStatusWareHouseLabeled     = 11
	PackageStatusWareHouseInContainer = 12
	PackageStatusWareHouseInShipment  = 13
	PackageStatusWareHouseExport      = 14
	PackageStatusImportHub            = 15
	PackageStatusExportHub            = 16
	PackageStatusInTransit            = 30
	PackageStatusDelivered            = 60
	PackageStatusReturned             = 40
	PackageStatusCancelled            = 50
	PackageStatusExpired              = 70
	PackageStatusReship               = 80
	PackageStatusUndelivered          = 90
	PackageStatusArchived             = 55

	PackageUpdateTypeRecipient           = 1
	PackageUpdateTypePhoneNumber         = 2
	PackageUpdateTypeAddress             = 3
	PackageUpdateTypeAddress2            = 13
	PackageUpdateTypeCity                = 4
	PackageUpdateTypeStateCode           = 5
	PackageUpdateTypeZipcode             = 6
	PackageUpdateTypeCountryCode         = 7
	PackageUpdateTypeWeight              = 8
	PackageUpdateTypeVolume              = 9
	PackageUpdateTypeNote                = 10
	PackageUpdateTypeService             = 11
	PackageUpdateTypeDetail              = 12
	PackageUpdateTypeLabel               = 13
	PackageUpdateExtraFeeTypeCovid       = 14
	PackageUpdateExtraFeeTypeOutSize     = 15
	PackageUpdateExtraFeeTypeFixVolume   = 16
	PackageUpdateExtraFeeTypeFixWeight   = 17
	PackageUpdateExtraFeeEditOrder       = 19
	PackageUpdateExtraFeeTypeRefund      = 20
	PackageUpdateExtraFeeTypeOther       = 21
	PackageCheckAddressType              = 22
	PackageIgnoreAddressType             = 23
	PackageUpdateExtraFeeTypeReship      = 24
	PackageUpdateTypeOrderNumber         = 25
	PackageUpdateTypeProduct             = 26
	PackageUpdateExtraFeeTypePeak        = 27
	PackageUpdateExtraFeeTypeCancelLabel = 28
	PackageUpdateExtraFeeTypeOversize    = 29
	PackageUpdateExtraFeeTypeDiscount    = 30
	PackageUpdateAddressExceed           = 31
	PackageUpdateReturnPackage           = 32
	PackageUpdateExtraFeeTypeBattery     = 33
	PackageUpdateExtraFeeTypeInsured     = 34
	PackageUpdateExtraFeeTypeService     = 35
	PackageUpdateTypeCNLabel             = 36
	PackageUpdateExtraFeeCNProduct       = 37
	PackageUpdateExtraFeeCNShipping      = 38
	PackageUpdateExtraFeeCNShippingToVN  = 39
	PackageUpdateExtraFeeCNLabel         = 40

	ExtraFeeTypeCovid                  = 1
	ExtraFeeTypeOutSize                = 2
	ExtraFeeTypeFixVolume              = 3
	ExtraFeeTypeFixWeight              = 4
	ExtraFeeService                    = 6
	ExtraFeeEditOrder                  = 8
	ExtraFeeTypeRefund                 = 9
	ExtraFeeTypeOther                  = 10
	ExtraFeeTypeReship                 = 11
	ExtraFeeTypePeak                   = 12
	ExtraFeeTypeCancelLabel            = 13
	ExtraFeeTypeOversize               = 14
	ExtraFeeTypeDiscount               = 15
	ExtraFeeTypeEditAddress            = 16
	ExtraFeeTypeReturn                 = 17
	ExtraFeeTypeBattery                = 18
	ExtraFeeTypeInsured                = 19
	ExtraFeeTypeAffiliate              = 21
	ExtraFeeTypeService                = 22
	ExtraFeeTypeChinaProductPercentage = 23
	ExtraFeeTypeChinaShipping          = 24
	ExtraFeeTypeChinaProduct           = 25
	ExtraFeeTypeHandling               = 26
	ExtraFeeTypeCNShippingToVN         = 27

	DeliverLogTebexpressCreated      = "LM"
	DeliverLogTebexpressPicked       = "LP"
	DeliverLogTebexpressInTransit    = "Li"
	DeliverLogTebexpressDelivered    = "DE"
	DeliverLogTebexpressReturned     = "LR"
	DeliverLogTebexpressCanceled     = "LC"
	DeliverLogTebexpressWareHouse    = "LW"
	DeliverLogTebexpressCreateManual = "LM"
	DeliverLogLocalLocation          = "Hanoi, VN"
	DeliverLogTebexpressExpired      = "LE"
	DeliverLogTebexpressReship       = "LS"
	DeliverLogTebexpressArchived     = "LA"

	PackageDeliverLogTypeCreated         = 1
	PackageDeliverLogTypePendingPickup   = 2
	PackageDeliverLogTypeRePendingPickup = 3
	PackageDeliverLogTypeInWareHouse     = 10
	PackageDeliverLogTypeWareHouseExport = 14
	PackageDeliverLogTypeInTransit       = 30
	PackageDeliverLogTypeImportHub       = 31
	PackageDeliverLogTypeExportHub       = 32
	PackageDeliverLogTypeDelivered       = 60
	PackageDeliverLogTypeReturned        = 40
	PackageDeliverLogTypeCancelled       = 50
	PackageDeliverLogTypeExpired         = 70
	PackageDeliverLogTypeReship          = 80
	PackageDeliverLogTypeArchived        = 55
	PackageDeliverLogTypeManual          = 65

	PackageCodeStatusActive   = 1
	PackageCodeStatusInActive = 0

	PackageValidAddress   = 1
	PackageInValidAddress = 0

	PackageAlertTypeDisable        = 0
	PackageAlertTypeOverPretransit = 1
	PackageAlertTypeWarehoseReturn = 2
	PackageAlertTypeHubReturn      = 3

	PackageStatusAlertText = "alert"
	PackageBookmaredText   = "bookmarks"

	PackageRefundPending   = 1
	PackageRefundCompleted = 2
	PackageRefundCanceled  = 3
)

var WarehouseStatus = []int64{
	PackageStatusPicked,
	PackageStatusWareHouseLabeled,
	PackageStatusWareHouseInContainer,
	PackageStatusWareHouseInShipment,
	PackageStatusWareHouseExport,
	PackageStatusImportHub,
	PackageStatusExportHub,
	PackageStatusDelivered,
	PackageStatusCancelled,
	PackageDeliverLogTypeInTransit,
	PackageStatusUndelivered,
	PackageStatusReship,
}

var MapTextStatusPackage = map[int]string{
	PackageStatusCreated:              "pending",
	PackageStatusCNPurchased:          "purchased",
	PackageStatusPendingPickup:        "pre-transit",
	PackageStatusPicked:               "picked",
	PackageStatusWareHouseLabeled:     "labeled",
	PackageStatusWareHouseInContainer: "in-container",
	PackageStatusWareHouseInShipment:  "in-shipment",
	PackageStatusWareHouseExport:      "export-warehouse",
	PackageStatusInTransit:            "in-transit",
	PackageStatusImportHub:            "import-hub",
	PackageStatusExportHub:            "export-hub",
	PackageStatusDelivered:            "delivered",
	PackageStatusCancelled:            "canceled",
	PackageStatusExpired:              "expired",
	PackageStatusReship:               "reship",
	PackageStatusUndelivered:          "undelivered",
	PackageStatusArchived:             "archived",
}

var MapTextStatusCustomerPackage = map[int]string{
	PackageStatusCreated:              "pending",
	PackageStatusCNPurchased:          "purchased",
	PackageStatusPendingPickup:        "pre-transit",
	PackageStatusPicked:               "in-transit",
	PackageStatusWareHouseLabeled:     "in-transit",
	PackageStatusWareHouseInContainer: "in-transit",
	PackageStatusWareHouseInShipment:  "in-transit",
	PackageStatusWareHouseExport:      "in-transit",
	PackageStatusInTransit:            "in-transit",
	PackageStatusImportHub:            "in-transit",
	PackageStatusExportHub:            "in-transit",
	PackageStatusDelivered:            "delivered",
	PackageStatusCancelled:            "canceled",
	PackageStatusExpired:              "expired",
	PackageStatusReship:               "in-transit",
	PackageStatusUndelivered:          "undelivered",
	PackageStatusArchived:             "archived",
}

var MapTextStatusAdminPackage = map[int]string{
	PackageStatusCreated:              "pending",
	PackageStatusCNPurchased:          "purchased",
	PackageStatusPendingPickup:        "pre-transit",
	PackageStatusPicked:               "processing",
	PackageStatusWareHouseLabeled:     "processing",
	PackageStatusWareHouseInContainer: "processing",
	PackageStatusWareHouseInShipment:  "processing",
	PackageStatusWareHouseExport:      "processing",
	PackageStatusInTransit:            "in-transit",
	PackageStatusImportHub:            "in-transit",
	PackageStatusExportHub:            "in-transit",
	PackageStatusDelivered:            "delivered",
	PackageStatusCancelled:            "canceled",
	PackageStatusExpired:              "expired",
	PackageStatusUndelivered:          "undelivered",
	PackageStatusReship:               "in-transit",
	PackageStatusArchived:             "archived",
}

var MapTextStatusLog = map[int]string{
	PackageStatusCreated:                 "pending",
	PackageStatusPendingPickup:           "pre-transit",
	PackageDeliverLogTypeReturned:        "pre-transit",
	PackageDeliverLogTypeRePendingPickup: "pre-transit",
	PackageDeliverLogTypeManual:          "in-transit",
	PackageStatusPicked:                  "in-transit",
	PackageStatusWareHouseLabeled:        "in-transit",
	PackageStatusWareHouseInContainer:    "in-transit",
	PackageStatusWareHouseInShipment:     "in-transit",
	PackageStatusWareHouseExport:         "in-transit",
	PackageStatusInTransit:               "in-transit",
	PackageStatusImportHub:               "in-transit",
	PackageStatusExportHub:               "in-transit",
	PackageStatusReship:                  "in-transit",
	PackageStatusDelivered:               "delivered",
	PackageStatusCancelled:               "canceled",
	PackageStatusExpired:                 "expired",
	PackageStatusArchived:                "archived",
}

var MapIntGroupStatusCustomerPackage = map[string][]int64{
	"pending":     []int64{PackageStatusCreated},
	"purchased":   []int64{PackageStatusCNPurchased},
	"pre-transit": []int64{PackageStatusPendingPickup},
	"picked":      []int64{PackageStatusPicked},
	"in-transit": []int64{
		PackageStatusPicked,
		PackageStatusWareHouseLabeled,
		PackageStatusWareHouseInContainer,
		PackageStatusWareHouseInShipment,
		PackageStatusWareHouseExport,
		PackageStatusInTransit,
		PackageStatusImportHub,
		PackageStatusExportHub,
		PackageStatusReship,
	},
	"delivered":   []int64{PackageStatusDelivered},
	"canceled":    []int64{PackageStatusCancelled},
	"expired":     []int64{PackageStatusExpired},
	"undelivered": []int64{PackageStatusUndelivered},
	"archived":    []int64{PackageStatusArchived},
}

var MapIntGroupStatusAdminPackage = map[string][]int64{
	"pending":     []int64{PackageStatusCreated},
	"purchased":   []int64{PackageStatusCNPurchased},
	"pre-transit": []int64{PackageStatusPendingPickup},
	"picked":      []int64{PackageStatusPicked},
	"processing": []int64{PackageStatusPicked,
		PackageStatusWareHouseLabeled,
		PackageStatusWareHouseInContainer,
		PackageStatusWareHouseInShipment,
		PackageStatusWareHouseExport,
	},
	"in-transit": []int64{
		PackageStatusInTransit,
		PackageStatusImportHub,
		PackageStatusExportHub,
		PackageStatusReship,
	},
	"delivered":   []int64{PackageStatusDelivered},
	"canceled":    []int64{PackageStatusCancelled},
	"expired":     []int64{PackageStatusExpired},
	"undelivered": []int64{PackageStatusUndelivered},
	"archived":    []int64{PackageStatusArchived},
}

var MapTextStatusWarehousePackage = map[int]string{
	PackageStatusPicked:               "Đã lấy",
	PackageStatusWareHouseLabeled:     "Kiểm hàng",
	PackageStatusWareHouseInContainer: "Đóng kiện",
	PackageStatusWareHouseInShipment:  "Đóng lô",
	PackageStatusImportHub:            "Nhập hub",
	PackageStatusExportHub:            "Xuất hub",
	PackageStatusCancelled:            "Hủy",
	PackageDeliverLogTypeInTransit:    "Xuất kho",
	PackageStatusDelivered:            "Giao thành công",
	PackageStatusUndelivered:          "Giao không thành công",
	PackageStatusWareHouseExport:      "Xuất kho",
	PackageStatusReship:               "Reship",
}

var MapStatusDescriptionUS = map[int]string{
	PackageStatusCreated:                 "Shipping label created, awaiting item",
	PackageStatusPendingPickup:           "Shipping label created, awaiting item",
	PackageDeliverLogTypeRePendingPickup: "Shipping label created, awaiting item",
	PackageStatusPicked:                  "Accepted at Processing Center",
	PackageStatusWareHouseExport:         "Departed from Processing Center",
	PackageStatusDelivered:               "Delivered",
	PackageStatusInTransit:               "Arriving at international airport to go abroad",
	PackageStatusCancelled:               "Label canceled",
	PackageStatusReturned:                "Package returned",
	PackageDeliverLogTypeExpired:         "Your Tracking is Expired",
	PackageDeliverLogTypeReship:          "Reship",
	PackageDeliverLogTypeImportHub:       "Accepted at Hub",
	PackageDeliverLogTypeExportHub:       "Departed from Hub",
	PackageStatusArchived:                "Package Archived",
}
