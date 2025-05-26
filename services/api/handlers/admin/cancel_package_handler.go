package admin

import (
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type CancelPackageResponse struct {
	Success bool `json:"success"`
}

type CancelPackageForm struct {
	IDS []int64 `json:"ids"`
}

var statusHasRefund = map[int]bool{
	constant.PackageStatusPicked:               true,
	constant.PackageStatusWareHouseLabeled:     true,
	constant.PackageStatusWareHouseInContainer: true,
	constant.PackageStatusWareHouseInShipment:  true,
	constant.PackageStatusWareHouseExport:      true,
}

func (h *PackageHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		form := &CancelPackageForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(form.IDS) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			IDs: form.IDS,
		})
		if err == gorm.ErrRecordNotFound || len(pkgs) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkgsCodes, err := h.PackageManager.GetPackagesCode(sqlmanager.PackageCodeQueryOption{
			IDs: form.IDS,
		})

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Code Detail %v", err, pkgsCodes)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapCodes := make(map[int64]string)
		for _, v := range pkgsCodes {
			mapCodes[v.ID] = v.Code
		}

		allCanceled := true
		logs := []entity.PackageDeliverLog{}
		IDs := make([]int64, 0)

		var refunds []entity.PackageRefund
		var nowRefunds []entity.Package

		for i, pkg := range pkgs {
			if pkg.Status == constant.PackageStatusCancelled || pkg.Status == constant.PackageStatusArchived {
				continue
			} else {
				allCanceled = false
			}

			if pkg.Status == constant.PackageStatusDelivered || pkg.Status == constant.PackageStatusExpired {
				c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
				return
			}

			if pkg.CustomTiktokBarcode != nil && *pkg.CustomTiktokBarcode != "" {
				nowRefunds = append(nowRefunds, pkg)
				pkgs[i].Status = constant.PackageStatusCancelled
				pkgs[i].OrderID = nil
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressCanceled,
					Type:      constant.PackageDeliverLogTypeCancelled,
					UserID:    &userID,
				})
				continue
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Service %s không được hỗ trợ", pkg.Service.Name))
				return
			}

			if statusHasRefund[pkg.Status] && pkg.Service.Code != constant.ServiceINUSCode && pkg.Service.Code != constant.ServiceUS48Code && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceAUCode && pkg.Service.Code != constant.ServiceEUCode && pkg.Service.Code != constant.ServiceAUFCode {
				extraFee, err := h.PackageManager.GetTotalExtrafee(pkg.ID)
				if err != nil {
					h.Logger.Errorf("get package extra fee total, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				oldPackageRefunds, err := h.PackageManager.GetListPackagesRefundByPackageID(pkg.ID, constant.PackageRefundPending)
				if err != nil {
					h.Logger.Errorf("get package refunds, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				var amountRefunds float64 = 0
				if len(oldPackageRefunds) > 0 {
					for _, v := range oldPackageRefunds {
						amountRefunds += v.Amount
					}
				}

				amount := pkg.ShippingFee + extraFee - amountRefunds
				if amount > 0 {
					refunds = append(refunds, entity.PackageRefund{
						PackageID: pkg.ID,
						Amount:    amount,
						Status:    constant.PackageRefundPending,
					})
				}
			}

			if pkg.Status == constant.PackageStatusPendingPickup {
				if pkg.Tracking != nil && pkg.Tracking.ID > 0 && pkg.Service.Code != constant.ServiceINUSCode && pkg.Service.Code != constant.ServiceUS48Code && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceAUCode && pkg.Service.Code != constant.ServiceEUCode && pkg.Service.Code != constant.ServiceAUFCode {
					extraFee, err := h.PackageManager.GetTotalExtrafee(pkg.ID)
					if err != nil {
						h.Logger.Errorf("get package extra fee total, %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					oldPackageRefunds, err := h.PackageManager.GetListPackagesRefundByPackageID(pkg.ID, constant.PackageRefundPending)
					if err != nil {
						h.Logger.Errorf("get package refunds, %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					var amountRefunds float64 = 0
					if len(oldPackageRefunds) > 0 {
						for _, v := range oldPackageRefunds {
							amountRefunds += v.Amount
						}
					}

					amount := pkg.ShippingFee + extraFee - amountRefunds
					if amount > 0 {
						refunds = append(refunds, entity.PackageRefund{
							PackageID: pkg.ID,
							Amount:    amount,
							Status:    constant.PackageRefundPending,
						})
					}
				} else {
					nowRefunds = append(nowRefunds, pkg)
				}
			}

			IDs = append(IDs, pkg.ID)

			pkgs[i].Alert = constant.PackageAlertTypeDisable

			if pkg.Status == constant.PackageStatusCreated || pkg.Status == constant.PackageStatusCNPurchased {
				pkgs[i].Status = constant.PackageStatusArchived
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressArchived,
					Type:      constant.PackageDeliverLogTypeArchived,
					UserID:    &userID,
				})
			} else {
				pkgs[i].Status = constant.PackageStatusCancelled
				pkgs[i].OrderID = nil
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressCanceled,
					Type:      constant.PackageDeliverLogTypeCancelled,
					UserID:    &userID,
				})
			}
		}

		if allCanceled {
			c.JSON(http.StatusOK, CancelPackageResponse{true})
			return
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			ok, err := h.PackageManager.CheckPermissionUserPackages(userID, IDs)
			if err != nil {
				h.Logger.Errorf("check permission %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if !ok {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}
		}

		containers, err := h.PackageManager.GetContainerPackages(IDs)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get container error %v", err, pkgsCodes)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(containers) > 0 && role != constant.UserRoleAdmin { // admin can force cancel
			c.JSON(http.StatusBadRequest, "Đơn hàng không thể hủy vì đang nằm trong kiện")
			return
		}

		err = h.PackageManager.CancelPackages(pkgs, logs, form.IDS, refunds)
		if err != nil {
			h.Logger.Errorf("Save package error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(nowRefunds) > 0 {
			for _, item := range nowRefunds {
				des := fmt.Sprintf("Hoàn tiền cho đơn #%v", item.ID)
				if item.PackageCode != nil {
					des = fmt.Sprintf("Hoàn tiền cho đơn %v", item.PackageCode.Code)
				}

				err = h.PackageRefund.Handle(c, item.ID, userID, des)
				if err != nil {
					h.Logger.Errorf("json marshal %v", err)
				}
			}
		}

		for _, v := range pkgs {
			if v.Tracking == nil {
				continue
			}

			err = h.ShipmentCancelCarrier.Handle(v.Tracking.TrackingNumber)
			if err != nil {
				h.Logger.Errorf("json marshal %v", err)
			}
		}

		c.JSON(http.StatusOK, CancelPackageResponse{true})
	}
}
