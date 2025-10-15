package packages

import (
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		id := cast.ToInt64(c.Param("id"))
		if id < 1 {
			c.JSON(http.StatusUnauthorized, httputil.ErrorResponse{
				Error:   constant.MessageValidateInput,
				Message: "Missing package id",
			})

			return
		}

		opts := sqlmanager.PackageQueryOption{
			ID:     id,
			UserID: userId,
		}

		pkg, err := h.PackageManager.GetPackageDetailForCustomer(opts)
		if err == gorm.ErrRecordNotFound || pkg.ID == 0 {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
				Error: constant.MessageNotFound,
			})

			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		if pkg.UserID != userId {
			c.JSON(http.StatusForbidden, httputil.ErrorResponse{
				Error: constant.MessagePermissionDenied,
			})

			return
		}

		if pkg.Status == constant.PackageStatusArchived || pkg.PCodeStatus == constant.PackageCodeTemp {
			pkg.Code = ""
		}

		urlLabel := ""
		h.Logger.Info("LabelURL: ", pkg.Tracking.LabelURL)
		if pkg.Tracking.LabelURL != "" {
			urlLabel, err = h.StorageS3.PreAssign(pkg.Tracking.LabelURL, viper.GetString("bucket.labels"))
			if err != nil {
				h.Logger.Error("PreAssign: ", err)
				c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
					Error: constant.MessageServerInternalError,
				})

				return
			}
		}

		sp := &OrderDetailCustomer{
			ID:              pkg.ID,
			Code:            pkg.Code,
			Sku:             pkg.OrderNumber,
			Recipient:       pkg.Recipient,
			Company:         pkg.Company,
			PhoneNumber:     pkg.PhoneNumber,
			Address1:        pkg.Address1,
			Address2:        pkg.Address2,
			City:            pkg.City,
			StateCode:       pkg.StateCode,
			Zipcode:         pkg.Zipcode,
			CountryCode:     pkg.CountryCode,
			Detail:          pkg.Detail,
			Weight:          pkg.Weight,
			Width:           pkg.Width,
			Length:          pkg.Length,
			Height:          pkg.Height,
			Status:          constant.MapTextStatusCustomerPackage[pkg.Status],
			UserID:          pkg.UserID,
			UserFullName:    pkg.UserFullName,
			UserEmail:       pkg.UserEmail,
			UserPhoneNumber: pkg.UserPhoneNumber,
			ShippingFee:     pkg.ShippingFee,
			BillCode:        pkg.BillCode,
			ServiceCode:     pkg.ServiceCode,
			CustomCNBarcode: pkg.CustomCNBarcode,
			ExtraFees:       pkg.ExtraFees,
			TotalCost:       pkg.ShippingFee,
			OrderID:         utils.Int64Value(pkg.OrderID),
			IncludeBattery:  pkg.IncludeBattery,
			Tracking: entity.TrackingCustom{
				TrackingNumber:  pkg.Tracking.TrackingNumber,
				LastMileCarrier: pkg.Tracking.LastMileCarrier,
				LabelURL:        urlLabel,
			},
			CreatedAt:         pkg.CreatedAt,
			UpdatedAt:         pkg.UpdatedAt,
			PackageName:       pkg.PackageName,
			PackageQuantity:   pkg.PackageQuantity,
			TotalProductPrice: pkg.TotalProductPrice,
		}

		for i, v := range sp.ExtraFees {
			sp.TotalCost += v.Amount

			if v.Description == "" {
				sp.ExtraFees[i].Description = sp.ExtraFees[i].ExtraFeeType
			}
		}

		sp.TotalCost = utils.ToFixed(sp.TotalCost, 2)

		if pkg.Status == constant.PackageStatusCreated {
			sp.Tracking = entity.TrackingCustom{}
		}

		c.JSON(http.StatusOK, PackageDetailResponse{Package: sp})
	}
}
