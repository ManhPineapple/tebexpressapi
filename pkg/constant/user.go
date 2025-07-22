package constant

const TokenActive = 1
const TokenDeactive = 0

const UserClassPublic = 1
const UserClassPriority = 2
const UserClassPartner = 3
const DefaultRefundDay = 14
const DefaultCancelMaxAMount = 1000
const AlertBalanceAmount = 1000

var DefinedPackageValue = map[int64]string{
	1: "Không có nhu cầu thường xuyên",
	2: "Dưới 150 đơn hàng / tháng",
	3: "Từ 150 - dưới 900 đơn hàng / tháng",
	4: "Từ 900 - dưới 3000 đơn hàng / tháng",
	5: "Từ 3000 - dưới 6000 đơn hàng / tháng",
	6: "Từ 6000 đơn hàng trên tháng trở lên",
}

var MapClassUser = map[int64]string{
	UserClassPublic:   "Public",
	UserClassPriority: "Priority",
	UserClassPartner:  "Partner",
}
