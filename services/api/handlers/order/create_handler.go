package order

import (
	"fmt"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type OrderCreateHandler struct {
	Logger      *zap.SugaredLogger
	RedisClient *redis.Client

	OrderManager *sqlmanager.OrderManager
}

func NewOrderCreateHandler(l *zap.SugaredLogger, r *redis.Client, om *sqlmanager.OrderManager) *OrderCreateHandler {
	return &OrderCreateHandler{
		Logger:      l,
		RedisClient: r,

		OrderManager: om,
	}
}

type CreateRequest struct {
	Codes []string `json:"codes"`
}

type CreateResponse struct {
	Order *entity.Order `json:"order"`
}

var StatusPreTransit = map[int]bool{
	constant.PackageDeliverLogTypePendingPickup:   true,
	constant.PackageDeliverLogTypeRePendingPickup: true,
	constant.PackageDeliverLogTypeInWareHouse:     true,
	constant.PackageDeliverLogTypeWareHouseExport: true,
}

func (h *OrderCreateHandler) Serve() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		rKey := fmt.Sprintf("%s_%d", constant.RedisKeyCreateOrder, userId)
		val, err := h.RedisClient.SetNX(c, rKey, 1, constant.RedisKeyPackageCheckExistsExp).Result()
		defer h.RedisClient.Del(c, rKey)
		if err != nil {
			h.Logger.Errorf("SetNX Redis : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		}

		if !val {
			h.Logger.Errorf("Update package error : %s", err)
			c.JSON(http.StatusBadRequest, "Vui lòng thử lại sau")
			return
		}

		form := &CreateRequest{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{Error: constant.APIResponseMessageValidateInput})
			return
		}

		codes := []string{}
		messages := []string{}
		for _, code := range form.Codes {
			code = strings.TrimSpace(code)
			if code == "" {
				continue
			}

			if !utils.ValidAlphaNumberWithoutSpace(code) {
				messages = append(messages, fmt.Sprintf("Code %s is invalid", code))
				continue
			}

			codes = append(codes, code)
		}

		if len(messages) > 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: messages,
			})
			return
		}

		if len(codes) < 1 {
			c.JSON(http.StatusBadRequest, "Codes is required")
			return
		}

		packages, err := h.OrderManager.GetListPackageItems(sqlmanager.OrderPackageQueryOption{
			UserID:       userId,
			PackageCodes: codes,
		})
		if err != nil {
			h.Logger.Error("GetListPackageItems: ", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageServerInternalError,
			})
			return
		}

		mapPackages := make(map[string]dto.OrderPackage)
		var packageIDs []int64
		for _, v := range packages {
			mapPackages[v.Code] = v
			packageIDs = append(packageIDs, v.ID)
		}

		for _, code := range codes {
			if mapPackages[code].ID == 0 {
				messages = append(messages, fmt.Sprintf("Code %s is not found", code))
				continue
			}

			if utils.Int64Value(mapPackages[code].OrderID) > 0 {
				messages = append(messages, fmt.Sprintf("Code %s is has make order", code))
				continue
			}

			if mapPackages[code].ServiceCode == constant.ServiceFBACode {
				messages = append(messages, fmt.Sprintf("Service %s is not support", constant.ServiceFBACode))
				continue
			}

			if constant.PackageStatusPendingPickup != mapPackages[code].Status {
				messages = append(messages, fmt.Sprintf("Code %s status is invalid", code))
				continue
			}
		}

		if len(messages) > 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageValidateInput,
				Messages: messages,
			})
			return
		}

		order := &entity.Order{
			UserID: userId,
			Status: constant.OrderStatusInTransit,
		}
		if err := h.OrderManager.Create(order, packageIDs); err != nil {
			h.Logger.Error("Create order: ", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.APIResponseMessageServerInternalError,
				Messages: messages,
			})
			return
		}

		order.Count = int64(len(packageIDs))
		c.JSON(http.StatusOK, CreateResponse{Order: order})
	}
}
