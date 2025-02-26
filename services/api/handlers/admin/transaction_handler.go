package admin

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils/array"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	Logger *zap.SugaredLogger

	UserManager        *sqlmanager.UserManager
	TransactionManager *sqlmanager.TransactionManager
}

type GetListTransactionResponse struct {
	Transactions []entity.Transaction `json:"transactions"`
}

type CounTransactionResponse struct {
	Count int64                        `json:"count"`
	All   []dto.CountStatusTransaction `json:"all"`
}

type ChangeStatusTransactionResponse struct {
	Success bool `json:"success"`
}

type ChangeStatusTransactionForm struct {
	ID     int64   `json:"id"`
	Status int64   `json:"status"`
	Amount float64 `json:"amount"`
	AmountChina float64 `json:"amount_china"`
}

func NewTransactionHandler(l *zap.SugaredLogger, tm *sqlmanager.TransactionManager, um *sqlmanager.UserManager) *TransactionHandler {
	return &TransactionHandler{
		Logger: l,

		TransactionManager: tm,
		UserManager:        um,
	}
}

func (h *TransactionHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		user, error := h.UserManager.GetUserByID(userID)
		if error != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, error)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if user.ID <= 0 || user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if cast.ToBool(c.Request.URL.Query().Get("tester")) {
			ignoreUsers = []int64{}
		}

		options := sqlmanager.TransactionQueryParams{
			Type:           cast.ToInt64(c.Request.URL.Query().Get("type")),
			Status:         cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:         strings.TrimSpace(c.Request.URL.Query().Get("search")),
			SearchBy:       strings.TrimSpace(c.Request.URL.Query().Get("search_by")),
			IsPreloadUser:  true,
			IsPreloadAdmin: true,
			Limit:          limit,
			Offset:         offset,
			IgnoreUsers:    ignoreUsers,
			UserID:         cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			PartnerID:      user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			options.SupportID = user.ID
		}

		var transactions []entity.Transaction
		err := h.TransactionManager.GetTransactions(options, &transactions)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get transaction log error:%v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetListTransactionResponse{transactions})
	}
}

func (h *TransactionHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		user, error := h.UserManager.GetUserByID(userID)
		if error != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, error)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if user.ID <= 0 || user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		options := sqlmanager.TransactionQueryParams{
			Type:      cast.ToInt64(c.Request.URL.Query().Get("type")),
			Status:    cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:    strings.TrimSpace(c.Request.URL.Query().Get("search")),
			SearchBy:  strings.TrimSpace(c.Request.URL.Query().Get("search_by")),
			UserID:    cast.ToInt64(c.Request.URL.Query().Get("user_id")),
			PartnerID: user.PartnerID,
		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			options.SupportID = user.ID
		}

		count, err := h.TransactionManager.CountTransactions(options)
		if err != nil {
			h.Logger.Errorf("Get count transaction log error:%v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var allStatus []dto.CountStatusTransaction
		err = h.TransactionManager.CountAllStatusTransactions(options, &allStatus)
		if err != nil {
			h.Logger.Errorf("Get count transaction log error:%v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CounTransactionResponse{count, allStatus})
	}
}

func (h *TransactionHandler) ChangeStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}
		if role != constant.UserRoleAdmin && role != constant.UserRoleAccountant {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		form := &ChangeStatusTransactionForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Warnf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Status != constant.TransactionStatusFailure && form.Status != constant.TransactionStatusSuccess {
			c.JSON(http.StatusBadRequest, "Status is invalid !")
			return
		}
		if form.ID <= 0 {
			c.JSON(http.StatusBadRequest, "Transaction id is invalid !")
			return
		}

		transaction, err := h.TransactionManager.GetTransactionByID(form.ID)
		fmt.Println("cccccccccccccccc")
		fmt.Println(transaction.AmountChina)
		if err != nil {
			log.Printf("Get transaction logs error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if transaction.Status != constant.TransactionStatusProcess {
			c.JSON(http.StatusBadRequest, "Transaction status is invalid !")
			return
		}

		transaction.AdminID = userID
		transaction.Status = form.Status
		transaction.UpdatedAt = time.Now()
		err = h.TransactionManager.SaveTransaction(transaction)
		if err != nil {
			log.Printf("Save transaction logs error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, ChangeStatusTransactionResponse{true})
	}
}
