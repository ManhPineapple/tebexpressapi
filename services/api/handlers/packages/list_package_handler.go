package packages

import (
	"net/http"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		offset, limit := httputil.GetRequestPaginateForCustomer(c.Request)

		statusString := cast.ToString(c.Request.URL.Query().Get("status"))

		opts := sqlmanager.PackageQueryOption{
			UserID:     userId,
			Limit:      limit,
			Offset:     offset,
			StatusArr:  constant.MapIntGroupStatusCustomerPackage[statusString],
			AlertValue: cast.ToInt(c.Request.URL.Query().Get("alert")),
		}

		if statusString != "" && len(opts.StatusArr) == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Status is invalid"},
			})
			return
		}

		AlertArr := []int64{constant.PackageAlertTypeOverPretransit, constant.PackageAlertTypeHubReturn, constant.PackageAlertTypeWarehoseReturn}
		if opts.AlertValue != 0 && !utils.ContainsNumber(AlertArr, cast.ToInt64(opts.AlertValue)) {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Alert is invalid"},
			})
			return
		}

		stringIds := strings.TrimSpace(c.Request.URL.Query().Get("ids"))
		if stringIds != "" {
			ids := strings.Split(stringIds, ",")
			var IDs []int64
			for _, id := range ids {
				ID := cast.ToInt64(id)
				if ID < 1 {
					continue
				}

				IDs = append(IDs, ID)
			}

			opts.IDs = IDs
		}

		serviceCode := cast.ToString(c.Request.URL.Query().Get("service"))
		if serviceCode != "" {
			opts.ServiceCode = serviceCode
		}

		packages, err := h.PackageManager.GetPackagesForCustomer(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list package error: %v", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		results := dto.TransformPackagesCustomerArray(packages)
		statusTextPending := constant.MapTextStatusCustomerPackage[constant.PackageStatusCreated]

		peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra peak fee: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

			return
		}

		for i := range results {
			urlLabel, err := h.StorageS3.PreAssign(results[i].Label, viper.GetString("bucket.labels"))
			if err != nil {
				h.Logger.Error("PreAssign: ", err)
				continue
			}

			results[i].Tracking.LabelURL = urlLabel
			if results[i].Status != statusTextPending || (results[i].IsPackageExceed && results[i].ShippingFee == 0) {
				continue
			}

			amount := calculate.PeakFee(results[i].Weight)
			if amount > 0 {
				results[i].ExtraFees = append(results[i].ExtraFees, entity.ExtraFeeCustom{
					ExtraFeeType: peakFee.Name,
					Amount:       amount,
					Description:  peakFee.Name,
				})
			}
			results[i].TotalCost = utils.Ceil(results[i].TotalCost+amount, 2)
		}

		c.JSON(http.StatusOK, ListPackageResponse{Packages: results})
	}
}
