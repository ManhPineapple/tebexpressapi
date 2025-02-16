package constant

const ContainerCodePrefix = 65
const ContainerWaitingClose = 1
const ContainerClosed = 2
const ContainerIntransit = 5
const ContainerCancelled = 3
const ContainerDelivered = 4
const ContainerImportHub = 6
const ContainerExportHub = 7

const ContainerItemActive = 1
const ContainerItemInactive = 0
const ContainerItemFail = 2

const ContainerMaxLength = 274
const ContainerMaxSize = 400
const ContainerMaxWeight = 70

const ContainerTypeUps = 1
const ContainerTypeManual = 2
const ContainerTypeFedEx = 3

var UPSIgnoreCodeLogs = []string{"OT", "VK", "SX", "MP"}

const PackageMaxWeight = 90          // unit lb
const PackageMaxLengthAndGirth = 130 // unit inch
const PackageMaxLength = 90          // unit inch
const PackageMaxLengthOversize = 100 // unit inch

const PackageMaxDemension = 60
const PackageFBAMaxDimension = 274
const PackageFBAMaxLengthAndGirth = 400
const PackageFBAMaxWeight = 90
const PackageFBAMinWeight = 20
