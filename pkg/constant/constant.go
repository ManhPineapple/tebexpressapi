package constant

import "time"

const FakeShopId int64 = 169696969691

const ThumbnailSize int = 680
const MaxImageSizeThunbQueue int64 = 10485760
const DirStorage = "/home/storage/"

const MaxNumberSyncOrders int = 50

const NotifyStatusReaded = 1
const NotifyStatusUnread = 0

const DefaultPageNumber = 1
const DefaultPageSize = 50
const DefaultPageLimit = 250

const ProvideStatusActive = "active"
const ProvideStatusDeactive = "deactive"

const ImageContentTypePNG = "image/png"
const ImageContentTypeGIF = "image/gif"
const ImageContentTypePDF = "application/pdf"

const MaxWeightShippingPackage = 450

const NotifyTypeOrder = "order"
const NotifyTypeBilling = "billing"

const LineShipStatusActive = 1
const LineShipStatusDeactive = 0

const StatusActive = 1
const StatusDeactive = 2

const ServiceTypePuclic = 1
const ServiceTypePriority = 2
const ServiceTypePartner = 3

const NotificationNotProcessed = 0
const NotificationStatusSuccess = 1
const NotificationStatusFaild = 2

const EmailNotiNotSend = 0
const EmailNotiSent = 1

var FileCsvFormatAllow = map[string]bool{
	"application/octet-stream":      true,
	"text/csv":                      true,
	"application/csv":               true,
	"application/vnd.ms-excel":      true,
	"text/x-csv":                    true,
	"text/x-comma-separated-values": true,
	"text/comma-separated-values":   true,
}

var FileExcelFormatAllow = map[string]bool{
	"application/vnd.ms-excel":                                          true,
	"application/vnd.ms-excel.sheet.binary.macroEnabled.12":             true,
	"application/vnd.ms-excel.sheet.macroEnabled.12":                    true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	"application/wps-office.xlsx":                                       true,
}

var AllowExtensionFileImportPackage = map[string]bool{
	"text/x-comma-separated-values":                                     true,
	"text/comma-separated-values":                                       true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

const ContentTypeXlsxExcel2007Above = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
const ContentTypeXlsxExcelOld = "application/vnd.ms-excel"
const ContentTypeCsv = "application/csv;charset=utf-8"
const ContentTypePDF = "application/pdf"

const RedisKeyPackageCheckExists = "package_is_exists"
const RedisKeyPackageCheckExistsExp = 1 * time.Minute

const PriceMaximumWeight = 3000.01

const ORDER_DESC = "DESC"
const ORDER_ASC = "ASC"

const RedisKeyCreateOrder = "create_order"

const KgToGram = 1000
