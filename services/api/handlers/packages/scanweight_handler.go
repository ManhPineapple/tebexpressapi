package packages

import (
	"fmt"
	"math"
	"net/http"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type ScanWeightRequest struct {
	OrderNumber string  `json:"order_number"`
	Weight      float64 `json:"weight"`
	Length      float64 `json:"length"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
}

type ScanWeightResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
	ChuteNo string `json:"ChuteNo"`
}

func (h *PackageHandler) ScanWeight() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := &RawScanWeightRequest{}
		if err := c.ShouldBindJSON(raw); err != nil {
			h.Logger.Errorf("Bad body request: %v", err)
			resp := ScanWeightResponse{
				Result:  "false",
				Message: constant.MessageParseRequestBody,
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusBadRequest, resp)
			return
		}

		form := &ScanWeightRequest{
			OrderNumber: raw.TicketsNum,
			Weight:      raw.Weight,
			Length:      raw.Length,
			Width:       raw.Width,
			Height:      raw.Height,
		}

		pkg, err := h.PackageManager.GetPackage(sqlmanager.PackageQueryOption{
			Code:            form.OrderNumber,
			IgnoreStatusArr: []int64{constant.PackageStatusCancelled, constant.PackageStatusArchived},
		})
		if err != nil {
			resp := ScanWeightResponse{
				Result:  "false",
				Message: "Package not found",
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusNotFound, resp)
			return
		}
		if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusPendingPickup {
			resp := ScanWeightResponse{
				Result:  "false",
				Message: "Package status invalid",
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusBadRequest, resp)
			return
		}

		pkgUser, err := h.UserManager.GetUserByID(pkg.UserID)
		if err != nil {
			h.Logger.Errorf("Error when get user: %v", err)
			resp := ScanWeightResponse{
				Result:  "false",
				Message: constant.MessageServerInternalError,
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusInternalServerError, resp)
			return
		}

		now := time.Now()
		pkg.ScanWeightAt = &now
		pkg.Weight = 1000 * form.Weight
		pkg.Width = form.Width
		pkg.Length = form.Length
		pkg.Height = form.Height

		service := pkg.Service
		serviceIDToCalculatePrice := utils.GetServiceIDToCalculatePrice(pkg.CustomTiktokBarcode, service)

		oldPrice := pkg.ShippingFee
		newPrice, _, err := h.CalculatePrice.Price3(c, pkg.UserID, serviceIDToCalculatePrice, pkgUser.Class, pkg.Weight, pkg.Length, pkg.Height, pkg.Width, pkg.CountryCode)
		if err == calculate.ErrorNotService {
			if service.Code != constant.ServiceCNCode {
				resp := ScanWeightResponse{
					Result:  "false",
					Message: "Dịch vụ không hợp lệ",
				}
				h.Logger.Infof("ScanWeight response: %+v", resp)
				c.JSON(http.StatusBadRequest, resp)
				return
			} else {
				err = nil
			}
		}
		priceDiff := newPrice - oldPrice

		if service.Code == constant.ServiceTebprintHubCode && pkg.CustomTiktokBarcode == nil {
			carrier := providers.NewCarrier(service.DomesticCarrier.Code, pkg.UserID)
			if carrier == nil {
				resp := ScanWeightResponse{
					Result:  "false",
					Message: "The service code is invalid",
				}
				h.Logger.Infof("ScanWeight response: %+v", resp)
				c.JSON(http.StatusBadRequest, resp)
				return
			}

			cost, err := h.PackageEstimateCost(c, carrier, &pkg)
			if err == nil {
				const additionalTebprintCost = 0.5
				newPrice = cost + additionalTebprintCost
			}
		}

		if service.Code == constant.ServiceLABELCode {
			carrier := providers.NewCarrier(service.DomesticCarrier.Code, pkg.UserID)
			if carrier == nil {
				resp := ScanWeightResponse{
					Result:  "false",
					Message: "The service code is invalid",
				}
				h.Logger.Infof("ScanWeight response: %+v", resp)
				c.JSON(http.StatusBadRequest, resp)
				return
			}

			h.Logger.Info("LABEL CODE before calc: ", newPrice)
			cost, err := h.PackageEstimateCost(c, carrier, &pkg)
			if err == nil {
				newPrice = cost + newPrice/100*cost
			}
			h.Logger.Info("LABEL CODE after calc: ", newPrice, cost, err)
		}

		if err == calculate.ErrorMaxWeight {
			msg := "Trọng lượng cho phép vượt quá giới hạn"
			if newPrice > 0 {
				msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(newPrice)-1)
			}
			resp := ScanWeightResponse{
				Result:  "false",
				Message: msg,
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusBadRequest, resp)
			return
		}

		if err == calculate.ErrorMaxVolume {
			msg := "Kích thước vượt quá giới hạn cho phép"
			if newPrice > 0 {
				msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(newPrice)-1)
			}
			resp := ScanWeightResponse{
				Result:  "false",
				Message: msg,
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusBadRequest, resp)
			return
		}

		if pkg.Status == constant.PackageStatusPendingPickup { // pretransit package create extrafee
			if (pkg.CustomTiktokBarcode != nil && *pkg.CustomTiktokBarcode != "") || priceDiff > 0 { // label seller khong hoan` tien`
				billID, err := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
				extraFee := &entity.ExtraFee{
					PackageID:      &pkg.ID,
					BillID:         &billID,
					Amount:         priceDiff,
					Description:    fmt.Sprintf("Tổng kết giá theo cân nặng cho đơn %s", pkg.OrderNumber),
					ExtraFeeTypeID: constant.ExtraFeeTypeFixWeight,
					Status:         constant.ExtraFeeStatusEnable,
				}

				adManhPineappleEmail := "manh.tv0911@gmail.com"
				admin, err := h.UserManager.GetUser(sqlmanager.UserQueryOption{
					Email: adManhPineappleEmail,
				})
				err = h.BillManager.CreateExtraFee(extraFee, pkg.UserID, admin.ID)
				if err != nil {
					h.Logger.Errorf("Save extra fee error %v", err)
					resp := ScanWeightResponse{
						Result:  "false",
						Message: constant.MessageServerInternalError,
					}
					h.Logger.Infof("ScanWeight response: %+v", resp)
					c.JSON(http.StatusInternalServerError, resp)
					return
				}
			}
		} else { // pending package update shipping fee
			pkg.ShippingFee = newPrice
		}

		err = h.PackageManager.UpdatePackage(&pkg, pkg.ID)
		if err != nil {
			h.Logger.Errorf("Error when update package: %v", err)
			resp := ScanWeightResponse{
				Result:  "false",
				Message: constant.MessageServerInternalError,
			}
			h.Logger.Infof("ScanWeight response: %+v", resp)
			c.JSON(http.StatusInternalServerError, resp)
			return
		}

		// success
		resp := ScanWeightResponse{
			Result:  "true",
			Message: "Scan weight success",
		}
		h.Logger.Infof("ScanWeight response: %+v", resp)
		c.JSON(http.StatusOK, resp)
	}
}
