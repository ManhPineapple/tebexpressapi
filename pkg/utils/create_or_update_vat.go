package utils

import (
	"strconv"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"

	"github.com/spf13/viper"
)

type UpdateVatAfterPretransit struct {
	NewExtrafee    *entity.ExtraFee
	BillID         int64
	UpdateByUserID int64
	PaidUserID     int64
}

// CreateOrUpdateVat recalculates VAT fee for a package.
// Case 1: *, nil, nil: CreatePackageCase
// Case 2: nil, *, nil: UpdatePendingPackageCase
// Case 3: nil, nil, *: UpdatePreTransitOrLaterPackageCase
func CreateOrUpdateVat(pkg *entity.Package, billID, createByUserID int64) (*entity.ExtraFee, *entity.PackageAuditLog, *UpdateVatAfterPretransit) {
	vatRate := viper.GetFloat64("extra_fees.vat_ratio")

	var (
		totalFeeBeforeVat = pkg.ShippingFee
		existedVatIndex   = -1
	)

	// Sum all fees except VAT, detect VAT index if exists
	for i, fee := range pkg.ExtraFee {
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeVatTax {
			existedVatIndex = i
		} else {
			totalFeeBeforeVat += fee.Amount
		}
	}
	vatAmount := totalFeeBeforeVat * vatRate

	if existedVatIndex >= 0 {
		if pkg.Status != constant.PackageStatusCreated &&
			pkg.Status != constant.PackageStatusCNPurchased {

			// pretransit or later package case
			pkg.ExtraFee[existedVatIndex].Amount = vatAmount
			return nil, nil, &UpdateVatAfterPretransit{
				UpdateByUserID: createByUserID,
				NewExtrafee:    &pkg.ExtraFee[existedVatIndex],
				BillID:         billID,
				PaidUserID:     pkg.UserID,
			}
		} else {
			// update pending package case
			return nil, &entity.PackageAuditLog{
				OldValue: strconv.FormatFloat(pkg.ExtraFee[existedVatIndex].Amount, 'f', -1, 64),
				Value:    strconv.FormatFloat(vatAmount, 'f', -1, 64),
				Type:     constant.PackageUpdateExtraFeeVatTax,
			}, nil
		}
	} else {
		// create package case, create new extrafee
		return &entity.ExtraFee{
			PackageID:      &pkg.ID,
			ExtraFeeTypeID: constant.ExtraFeeTypeVatTax,
			Amount:         vatAmount,
			Status:         constant.ExtraFeeStatusEnable,
		}, nil, nil
	}
}
