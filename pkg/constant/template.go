package constant

const TemplateOrderImportActive = 1
const TemplateOrderImportInactive = 0
const TemplateDefault = 1
const TemplateIsNotDefault = 0

var MapFieldDefaultImportPackage map[int]string = map[int]string{
	0:  "Tên người nhận",
	1:  "Số điện thoại người nhận",
	2:  "Địa chỉ nhận",
	3:  "Địa chỉ nhận phụ",
	4:  "Thành phố",
	5:  "Mã vùng",
	6:  "Mã bưu điện",
	7:  "Mã quốc gia",
	8:  "Mã đơn hàng",
	9:  "Chi tiết hàng hóa",
	10: "Trọng lượng",
	11: "Dài",
	12: "Rộng",
	13: "Cao",
	14: "Dịch vụ",
	15: "SKU",
}

var RequireFieldImport map[int]string = map[int]string{
	0:  "Tên người nhận",
	2:  "Địa chỉ nhận",
	4:  "Thành phố",
	5:  "Mã vùng",
	6:  "Mã bưu điện",
	7:  "Mã quốc gia",
	8:  "Mã đơn hàng",
	9:  "Chi tiết hàng hóa",
	10: "Trọng lượng",
	11: "Dài",
	12: "Rộng",
	13: "Cao",
	14: "Dịch vụ",
}
