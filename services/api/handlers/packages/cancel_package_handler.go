package packages

import (
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

func (h *PackageHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		form := &CancelForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
			return
		}

		if len(form.IDs) == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: []string{"List packages is not empty"},
			})

			return
		}

		var cancelMaxAmount float64 = constant.DefaultCancelMaxAMount
		userInfo, err := h.UserManager.GetUserInfoByUserID(userID)
		if err != nil && err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
			return
		}

		if err != gorm.ErrRecordNotFound {
			cancelMaxAmount = userInfo.CancelMaxAmount
		}

		rKeyCancel := fmt.Sprintf("%s_%d", "cancel_amount", userID)
		cancelAmount, err := h.Redis.Get(c, rKeyCancel).Float64()
		if err != nil && err != redis.Nil {
			h.Logger.Errorf("get redis : %s", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
			return
		}

		pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			IDs: form.IDs,
		})
		if err == gorm.ErrRecordNotFound || len(pkgs) == 0 {
			c.JSON(http.StatusNotFound, httputil.ErrorResponse{
				Error: constant.APIResponseMessageNotFound,
			})

			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
			return
		}

		refunds := []entity.Package{}
		packageRefunds := []entity.PackageRefund{}
		logs := []entity.PackageDeliverLog{}
		var totalCancelAmount float64 = 0

		for i, pkg := range pkgs {
			if pkg.UserID != userID {
				c.JSON(http.StatusForbidden, httputil.ErrorResponse{
					Error: constant.APIResponseMessagePermissionDenied,
				})
				return
			}

			if pkg.Service.Code == constant.ServiceFBACode || pkg.Service.Code == constant.ServiceFastFBACode {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error: fmt.Sprintf("Service %s is not support", pkg.Service.Name),
				})
				return
			}

			if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusPendingPickup && pkg.Status != constant.PackageStatusCNPurchased {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error: constant.APIResponseMessageValidateInput,
				})
				return
			}

			if pkg.Status == constant.PackageStatusPendingPickup {
				if pkg.CustomTiktokBarcode != nil && *pkg.CustomTiktokBarcode != "" {
					refunds = append(refunds, pkg)
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

				if cancelAmount > cancelMaxAmount {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error: "Account excess create package limit",
					})
					return
				}

				extraFee, err := h.PackageManager.GetTotalExtrafeeToRefund(pkg.ID)
				if err != nil {
					h.Logger.Errorf("get package extra fee total, %v", err)
					c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
						Error: constant.APIResponseMessageServerInternalError,
					})
					return
				}

				// inus and us48 and au and eu are not refunded when cancel
				if pkg.Service.Code != constant.ServiceINUSCode && pkg.Service.Code != constant.ServiceUS48Code && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceAUCode && pkg.Service.Code != constant.ServiceEUCode && pkg.Service.Code != constant.ServiceAUFCode {
					if pkg.Tracking != nil && pkg.Tracking.ID > 0 {
						oldPackageRefunds, err := h.PackageManager.GetListPackagesRefundByPackageID(pkg.ID, constant.PackageRefundPending)
						if err != nil {
							h.Logger.Errorf("get package refunds, %v", err)
							c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
								Error: constant.APIResponseMessageServerInternalError,
							})
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
							packageRefunds = append(packageRefunds, entity.PackageRefund{
								PackageID: pkg.ID,
								Amount:    amount,
								Status:    constant.PackageRefundPending,
							})
						}

						totalCancelAmount += amount
					} else {
						refunds = append(refunds, pkg)
						totalCancelAmount = totalCancelAmount + pkg.ShippingFee + extraFee
					}
				}
			}

			pkgs[i].Alert = constant.PackageAlertTypeDisable
			if pkg.Status == constant.PackageStatusCreated {
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

			for _, packageProduct := range pkg.PackageProducts {
				err := h.ProductManager.AdjustStock(packageProduct.ProductID, packageProduct.PackageID, packageProduct.Quantity)
				if err != nil {
					h.Logger.Errorf("Error when get product: %v", err)
					c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
						Error: constant.APIResponseMessageServerInternalError,
					})
				}
			}
		}

		err = h.PackageManager.CancelPackages(pkgs, logs, form.IDs, packageRefunds)
		if err != nil {
			h.Logger.Errorf("Save package error %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
			return
		}

		h.Logger.Info("refunds: ", len(refunds))
		if len(refunds) > 0 {
			for _, item := range refunds {
				des := fmt.Sprintf("Hoàn tiền cho đơn #%v", item.ID)
				if item.PackageCode != nil {
					des = fmt.Sprintf("Hoàn tiền cho đơn %v", item.PackageCode.Code)
				}

				err = h.ShipmentRefund.Handle(c, item.ID, userID, des)
				if err != nil {
					h.Logger.Errorf("json marshal %v", err)
				}
			}
		}

		for _, v := range pkgs {
			if v.Tracking == nil {
				continue
			}

			err = h.ShipmentRefundCarrier.Handle(v.Tracking.TrackingNumber)
			if err != nil {
				h.Logger.Errorf("json marshal %v", err)
			}
		}

		amount := cancelAmount + totalCancelAmount
		if amount > 0 {
			t := time.Now().Add(7 * time.Hour)
			end, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 23:59:59", t.Format("2006-01-02")))
			if err != nil {
				h.Logger.Errorf("time.parse: %s", err)
			}

			expiration := end.Sub(t)
			err = h.Redis.Set(c, rKeyCancel, cancelAmount+totalCancelAmount, expiration).Err()
			if err != nil {
				h.Logger.Errorf("get redis : %s", err)
			}
		}

		c.JSON(http.StatusOK, CancelResponse{true})
	}
}
