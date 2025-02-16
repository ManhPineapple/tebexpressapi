package packages

import (
	"encoding/json"
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
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageParseRequestBody,
			})
		}

		var cancelMaxAmount float64 = constant.DefaultCancelMaxAMount
		if user.UserInfo != nil {
			cancelMaxAmount = user.UserInfo.CancelMaxAmount
		}

		form := &CancelForm{}
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(form); err != nil {
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
			h.Logger.Errorf("Get list package error %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		ids := []int64{}
		for i := range pkgs {
			ids = append(ids, pkgs[i].ID)
		}

		refunds := []entity.Package{}
		packageRefunds := []entity.PackageRefund{}
		logs := []entity.PackageDeliverLog{}
		var totalCancelAmount float64 = 0

		rKeyCancel := fmt.Sprintf("%s_%d", "cancel_amount", user.ID)
		cancelAmount, err := h.Redis.Get(c, rKeyCancel).Float64()
		if err != nil && err != redis.Nil {
			h.Logger.Errorf("get redis : %s", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})

			return
		}

		for i, pkg := range pkgs {
			if pkg.UserID != user.ID {
				c.JSON(http.StatusForbidden, httputil.ErrorResponse{
					Error: constant.APIResponseMessagePermissionDenied,
				})

				return
			}

			// thêm service inus và us48 k support qua api do darius k có api cancel
			// if pkg.Service.Code == constant.ServiceFBACode || pkg.Service.Code == constant.ServiceINUSCode || pkg.Service.Code == constant.ServiceUS48Code {
			if pkg.Service.Code == constant.ServiceFBACode {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{fmt.Sprintf("Service %s is not supported", pkg.Service.Name)},
				})

				return
			}

			if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusPendingPickup {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    constant.APIResponseMessageValidateInput,
					Messages: []string{"Package status is invalid"},
				})

				return
			}

			// check maximum number of cancel per day
			if pkg.Status == constant.PackageStatusPendingPickup {
				if cancelAmount > cancelMaxAmount {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error: "Account exceeds order creation limit",
					})

					return
				}

				extraFee, err := h.PackageManager.GetTotalExtrafee(pkg.ID)
				if err != nil {
					h.Logger.Errorf("get package extra fee total, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

					return
				}

				// package has tracking
				// inus and us48 and actus and au and eu are not refunded when cancel
				if pkg.Service.Code != constant.ServiceINUSCode && pkg.Service.Code != constant.ServiceUS48Code && pkg.Service.Code != constant.ServiceAUCode && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceEUCode && pkg.Service.Code != constant.ServiceAUFCode {
					if pkg.Tracking != nil && pkg.Tracking.ID > 0 {
						oldPackageRefunds, err := h.PackageManager.GetListPackagesRefundByPackageID(pkg.ID, constant.PackageRefundPending)
						if err != nil {
							h.Logger.Errorf("get package refunds, %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)

							return
						}

						var amountRefunds float64 = 0
						h.Logger.Info("oldPackageRefunds: ", len(oldPackageRefunds))
						if len(oldPackageRefunds) > 0 {
							for _, v := range oldPackageRefunds {
								amountRefunds += v.Amount
							}
						}

						amount := pkg.ShippingFee + extraFee - amountRefunds
						h.Logger.Infof("amount: %v - %v - %v - %v", pkg.ShippingFee, extraFee, amountRefunds, amount)
						if amount > 0 {
							packageRefunds = append(packageRefunds, entity.PackageRefund{
								PackageID: pkg.ID,
								Amount:    amount,
								Status:    constant.PackageRefundPending,
							})
						}

						totalCancelAmount += amount
					} else {
						h.Logger.Info("refunds: ", pkg.ID)
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
					UserID:    &user.ID,
				})
			} else {
				pkgs[i].Status = constant.PackageStatusCancelled
				pkgs[i].OrderID = nil
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressCanceled,
					Type:      constant.PackageDeliverLogTypeCancelled,
					UserID:    &user.ID,
				})
			}
		}

		err = h.PackageManager.CancelPackages(pkgs, logs, ids, packageRefunds)
		if err != nil {
			h.Logger.Errorf("Save package error %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{Error: constant.APIResponseMessageServerInternalError})

			return
		}

		h.Logger.Info("refunds: ", len(refunds))
		if len(refunds) > 0 {
			for _, item := range refunds {
				des := fmt.Sprintf("Hoàn tiền cho đơn #%v", item.ID)
				if item.PackageCode != nil {
					des = fmt.Sprintf("Hoàn tiền cho đơn %v", item.PackageCode.Code)
				}

				err = h.ShipmentRefund.Handle(c, item.ID, userId, des)
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
