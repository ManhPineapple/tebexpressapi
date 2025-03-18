package customer

import (
	"math"
	"math/big"
	"net/http"
	"strconv"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	Logger *zap.SugaredLogger

	Redis *redis.Client

	TransactionManager *sqlmanager.TransactionManager
	UserManager        *sqlmanager.UserManager
}

type GetTransactionsResponse struct {
	Balance      interface{} `json:"balance"`
	ProcessMoney interface{} `json:"process_money"`
	Transaction  interface{} `json:"transactions"`
}

type CountTransactionsResponse struct {
	Count int64 `json:"count"`
}

type CreateTopupResponse struct {
	Topup interface{} `json:"topup"`
}

type CreateTransactionResponse struct {
	Success bool `json:"success"`
}

type CreateTransactionForm struct {
	Type          int64   `json:"type"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
}

type GetRateChangeResponse struct {
	UsdToVnd  interface{} `json:"usd_to_vnd"`
	UpdatedAt interface{} `json:"updated_at"`
}
type RateExchange struct {
	UserId    int64     `json:"user_id"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TopupForm struct {
	Amount float64 `json:"amount"`
}

type UpdateTopupResponse struct {
	Success bool `json:"success"`
}

func NewTransactionHandler(l *zap.SugaredLogger, r *redis.Client, trm *sqlmanager.TransactionManager, um *sqlmanager.UserManager) *TransactionHandler {
	return &TransactionHandler{
		Logger: l,
		Redis:  r,

		TransactionManager: trm,
		UserManager:        um,
	}
}

func (h *TransactionHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.TransactionQueryParams{
			Limit:     limit,
			Offset:    offset,
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
			UserID:    userID,
			Type:      cast.ToInt64(c.Request.URL.Query().Get("type")),
		}

		if opts.Type > 0 && opts.Type != constant.TransactionLogTypeTopup && opts.Type != constant.TransactionLogTypePay && opts.Type != constant.TransactionLogTypeRefund {
			c.JSON(http.StatusBadRequest, "Type is invalid.")
			return
		}

		if opts.StartDate != "" {
			if startDate := utils.ParseRawDateTime(opts.StartDate); startDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid start date format")
				return
			}
		}

		if opts.EndDate != "" {
			if endDate := utils.ParseRawDateTime(opts.EndDate); endDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid end date format")
				return
			}
		}

		var transactions []entity.Transaction

		err := h.TransactionManager.GetTransactions(opts, &transactions)

		if err != nil {
			h.Logger.Errorf("Get transaction log error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		balance := user.Balance

		processMoney, err := h.TransactionManager.GetProcessMoney(userID)
		if err != nil {
			h.Logger.Errorf("Get Process Money error : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetTransactionsResponse{Balance: balance, ProcessMoney: processMoney, Transaction: transactions})
	}
}

func (h *TransactionHandler) ListChina() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.TransactionQueryParams{
			Limit:       limit,
			Offset:      offset,
			StartDate:   cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:     cast.ToString(c.Request.URL.Query().Get("end_date")),
			UserID:      userID,
			Type:        cast.ToInt64(c.Request.URL.Query().Get("type")),
			ServiceCode: constant.ServiceCNCode,
		}

		if opts.Type > 0 && opts.Type != constant.TransactionLogTypeTopup && opts.Type != constant.TransactionLogTypePay && opts.Type != constant.TransactionLogTypeRefund {
			c.JSON(http.StatusBadRequest, "Type is invalid.")
			return
		}

		if opts.StartDate != "" {
			if startDate := utils.ParseRawDateTime(opts.StartDate); startDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid start date format")
				return
			}
		}

		if opts.EndDate != "" {
			if endDate := utils.ParseRawDateTime(opts.EndDate); endDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid end date format")
				return
			}
		}

		var transactions []entity.Transaction

		err := h.TransactionManager.GetTransactions(opts, &transactions)
		if err != nil {
			h.Logger.Errorf("Get transaction log error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		balance := user.BalanceChina

		processMoney, err := h.TransactionManager.GetProcessMoney(userID)
		if err != nil {
			h.Logger.Errorf("Get Process Money error : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetTransactionsResponse{Balance: balance, ProcessMoney: processMoney, Transaction: transactions})
	}
}

func (h *TransactionHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		opts := sqlmanager.TransactionQueryParams{
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
			UserID:    userID,
			Type:      cast.ToInt64(c.Request.URL.Query().Get("type")),
		}

		if opts.Type > 0 && opts.Type != constant.TransactionLogTypeTopup && opts.Type != constant.TransactionLogTypePay && opts.Type != constant.TransactionLogTypeRefund {
			c.JSON(http.StatusBadRequest, "Type is invalid.")
			return
		}

		if opts.StartDate != "" {
			if startDate := utils.ParseRawDateTime(opts.StartDate); startDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid start date format")
				return
			}
		}

		if opts.EndDate != "" {
			if endDate := utils.ParseRawDateTime(opts.EndDate); endDate == nil {
				c.JSON(http.StatusBadRequest, "Invalid end date format")
				return
			}
		}

		count, err := h.TransactionManager.CountTransactions(opts)
		if err != nil {
			h.Logger.Error("Error while count tickets, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountTransactionsResponse{count})
	}
}

func (h *TransactionHandler) TopUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		_, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		transaction := &entity.Transaction{
			UserID: userID,
			Type:   constant.TransactionLogTypeTopup,
			Status: constant.TransactionStatusDraft,
		}

		topup, err := h.TransactionManager.CreateTopup(transaction)
		if err != nil {
			h.Logger.Errorf("create topup error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateTopupResponse{Topup: topup})
	}
}

func (h *TransactionHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		form := &CreateTransactionForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Type != constant.TransactionLogTypePayoneer && form.Type != constant.TransactionLogTypePingPong {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if form.TransactionID == "" {
			c.JSON(http.StatusBadRequest, "Transaction ID là bắt buộc !")
			return
		}

		if !utils.ValidAlphaNumberWithoutSpace(form.TransactionID) {
			c.JSON(http.StatusBadRequest, "Nội dung không phù hợp")
			return
		}

		transaction := &entity.Transaction{
			UserID:      userID,
			Type:        form.Type,
			Status:      constant.TransactionStatusProcess,
			Amount:      form.Amount,
			Description: form.TransactionID,
		}

		_, err := h.TransactionManager.CreateTopup(transaction)
		if err != nil {
			h.Logger.Errorf("create topup error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateTransactionResponse{true})
	}
}

// remove redis exchage_rate: all currency in system is $ already
// func (h *TransactionHandler) GetRate() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
// 		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

// 		if role != constant.UserRoleCustomer {
// 			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
// 			return
// 		}

// 		if userID <= 0 {
// 			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
// 			return
// 		}
// 		rate, err := h.Redis.Get(c, constant.RedisKeyRateExChange).Result()
// 		if err != nil && err != redis.Nil {
// 			h.Logger.Errorf("Get Redis Rate Exchange error ", err)
// 			c.JSON(http.StatusBadRequest, "This verification link is either invalid or has expired")
// 			return
// 		}

// 		decode := RateExchange{}
// 		json.Unmarshal([]byte(rate), &decode)
// 		if err != nil {
// 			log.Printf("Error marshaling body: %v", err)
// 			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
// 			return
// 		}

// 		c.JSON(http.StatusOK, GetRateChangeResponse{UsdToVnd: decode.Price, UpdatedAt: decode.UpdatedAt})

// 	}
// }

func (h *TransactionHandler) UpdateTopup() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		TransactionID := cast.ToInt64(c.Param("id"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		formData := TopupForm{}
		if err := c.ShouldBindJSON(&formData); err != nil {
			h.Logger.Warnf("Could not decode, %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if formData.Amount <= 0 {
			c.JSON(http.StatusBadRequest, "Vui lòng nhập số tiền lớn hơn 0")
			return
		}

		transaction, err := h.TransactionManager.GetTransactionByID(TransactionID)
		if transaction.ID <= 0 || err != nil {
			h.Logger.Errorf("get transaction: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if transaction.Status != constant.TransactionStatusDraft {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		transaction.Status = constant.TransactionStatusProcess

		f1 := big.NewFloat(formData.Amount)
		f2 := big.NewFloat(100)
		var f3 big.Float
		f3.Mul(f1, f2)
		s := f3.String()
		result, _ := strconv.ParseFloat(s, 64)
		transaction.Amount = math.Floor(result) / 100

		if err = h.TransactionManager.UpdateTransaction(transaction); err != nil {
			h.Logger.Errorf("update topup error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// remove redis exchage_rate: all currency in system is $ already
		// rate, err := h.Redis.Get(c, constant.RedisKeyRateExChange).Result()
		// if err != nil && err != redis.Nil {
		// 	h.Logger.Errorf("Get Redis Rate Exchange error ", err)
		// 	c.JSON(http.StatusBadRequest, "This verification link is either invalid or has expired")
		// 	return
		// }
		// decode := RateExchange{}
		// json.Unmarshal([]byte(rate), &decode)
		// if err != nil {
		// 	log.Printf("Error marshaling body: %v", err)
		// 	c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
		// 	return
		// }

		c.JSON(http.StatusOK, UpdateTopupResponse{Success: true})
	}
}

func (h *TransactionHandler) UpdateTopupChina() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		TransactionID := cast.ToInt64(c.Param("id"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		formData := TopupForm{}
		if err := c.ShouldBindJSON(&formData); err != nil {
			h.Logger.Warnf("Could not decode, %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if formData.Amount <= 0 {
			c.JSON(http.StatusBadRequest, "Vui lòng nhập số tiền lớn hơn 0")
			return
		}

		transaction, err := h.TransactionManager.GetTransactionByID(TransactionID)
		if transaction.ID <= 0 || err != nil {
			h.Logger.Errorf("get transaction: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if transaction.Status != constant.TransactionStatusDraft {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		transaction.Status = constant.TransactionStatusProcess

		f1 := big.NewFloat(formData.Amount)
		f2 := big.NewFloat(100)
		var f3 big.Float
		f3.Mul(f1, f2)
		s := f3.String()
		result, _ := strconv.ParseFloat(s, 64)
		transaction.AmountChina = math.Floor(result) / 100

		if err = h.TransactionManager.UpdateTransaction(transaction); err != nil {
			h.Logger.Errorf("update topup error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// remove redis exchage_rate: all currency in system is $ already
		// rate, err := h.Redis.Get(c, constant.RedisKeyRateExChange).Result()
		// if err != nil && err != redis.Nil {
		// 	h.Logger.Errorf("Get Redis Rate Exchange error ", err)
		// 	c.JSON(http.StatusBadRequest, "This verification link is either invalid or has expired")
		// 	return
		// }
		// decode := RateExchange{}
		// json.Unmarshal([]byte(rate), &decode)
		// if err != nil {
		// 	log.Printf("Error marshaling body: %v", err)
		// 	c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
		// 	return
		// }

		c.JSON(http.StatusOK, UpdateTopupResponse{Success: true})
	}
}
