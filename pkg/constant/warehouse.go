package constant

//const EstimateCostStatusActive = 1
//const EstimateCostStatusInActive = 0

const WareHouseStatusDisable = 0
const WareHouseStatusActive = 1
const WareHouseStatusDeactivate = 2

const WareHouseManifestActive = 1
const WareHouseManifestInActive = 2

const WareHouseTypeInternational = 1

const HubItemFilterStatusPending = 10
const HubItemFilterStatusIn = 20
const HubItemFilterStatusExport = 30
const HubItemFilterStatusReturn = 40
const HubItemFilterStatusReship = 50

const WareHouseTypeHub = 1
const WareHouseTypeWareHouse = 2

var HubItemFilterStatus = []int64{
	HubItemFilterStatusPending,
	HubItemFilterStatusIn,
	HubItemFilterStatusExport,
	HubItemFilterStatusReturn,
	HubItemFilterStatusReship,
}

const ExportHubManifest = 6688
