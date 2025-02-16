package constant

var MapTypeAuditLogByExtraFeeTypeID = map[int64]int{
	ExtraFeeTypeCovid:       PackageUpdateExtraFeeTypeCovid,
	ExtraFeeTypeOutSize:     PackageUpdateExtraFeeTypeOutSize,
	ExtraFeeTypeFixVolume:   PackageUpdateTypeVolume,
	ExtraFeeTypeFixWeight:   PackageUpdateTypeWeight,
	ExtraFeeService:         PackageUpdateTypeService,
	ExtraFeeEditOrder:       PackageUpdateExtraFeeEditOrder,
	ExtraFeeTypeRefund:      PackageUpdateExtraFeeTypeRefund,
	ExtraFeeTypeOther:       PackageUpdateExtraFeeTypeOther,
	ExtraFeeTypeReship:      PackageUpdateExtraFeeTypeReship,
	ExtraFeeTypePeak:        PackageUpdateExtraFeeTypePeak,
	ExtraFeeTypeCancelLabel: PackageUpdateExtraFeeTypeCancelLabel,
	ExtraFeeTypeOversize:    PackageUpdateExtraFeeTypeOversize,
	ExtraFeeTypeDiscount:    PackageUpdateExtraFeeTypeDiscount,
	ExtraFeeTypeEditAddress: PackageUpdateAddressExceed,
	ExtraFeeTypeReturn:      PackageUpdateReturnPackage,
	ExtraFeeTypeBattery:     PackageUpdateExtraFeeTypeBattery,
	ExtraFeeTypeInsured:     PackageUpdateExtraFeeTypeInsured,
	ExtraFeeTypeService:     PackageUpdateExtraFeeTypeService,
}
