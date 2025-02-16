package admin

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"tebexpressapi/pkg/calculate"
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

type ServiceHandler struct {
	Logger *zap.SugaredLogger
	Redis  *redis.Client

	ServiceManager *sqlmanager.ServiceManager
}

type RateExchangeResponse struct {
	RateExchange RateExchange `json:"rate_exchange"`
}

type RateExchange struct {
	UserId    int64     `json:"user_id"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GetListServiceResponse struct {
	Services []*entity.Service `json:"services"`
}

type formUpdateRateExchange struct {
	Price float64 `json:"price"`
}

type updateRateExchangeResponse struct {
	Success bool `json:"success"`
}

type Price struct {
	ServiceID int64   `json:"service_id"`
	PriceID   int64   `json:"price_id"`
	Weight    float64 `json:"weight"`
	Price     float64 `json:"price"`
}

type formUpdatePrice struct {
	Prices []Price `json:"prices"`
}

type updatePriceResponse struct {
	Success bool `json:"success"`
}

type GetServiceResponse struct {
	Service *entity.Service `json:"service"`
}

func NewServiceHandler(l *zap.SugaredLogger, r *redis.Client, sm *sqlmanager.ServiceManager) *ServiceHandler {
	return &ServiceHandler{
		Logger: l,
		Redis:  r,

		ServiceManager: sm,
	}
}

func (h *ServiceHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := cast.ToInt64(c.Param("service_id"))

		service, err := h.ServiceManager.GetServiceByID(id)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, "Không tìm thấy dịch vụ vận chuyển")
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		result := &GetServiceResponse{Service: service}
		c.JSON(http.StatusOK, result)
	}
}

func (h *ServiceHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		search := c.Request.URL.Query().Get("search")
		hasPrice := strings.TrimSpace(c.Request.URL.Query().Get("has_price"))

		opt := sqlmanager.ServiceQueryOption{
			Status:     constant.StatusActive,
			Limit:      limit,
			Offset:     offset,
			Search:     search,
			IsHasPrice: hasPrice == "yes",
		}
		services, err := h.ServiceManager.GetServices(opt)
		if err != nil && err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		result := &GetListServiceResponse{Services: services}
		c.JSON(http.StatusOK, result)
	}
}

func (h *ServiceHandler) GetRate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		rate, err := h.Redis.Get(c, constant.RedisKeyRateExChange).Result()
		if err != nil && err != redis.Nil {
			h.Logger.Errorf("Get Redis Rate Exchange error ", err)
			c.JSON(http.StatusBadRequest, "This verification link is either invalid or has expired")
			return
		}

		if rate != "" {
			decode := RateExchange{}
			json.Unmarshal([]byte(rate), &decode)
			if err != nil {
				log.Printf("Error marshaling body: %v", err)
				c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
				return
			}

			c.JSON(http.StatusOK, RateExchangeResponse{
				RateExchange: decode,
			})

			return
		}

		data := RateExchange{
			Price:     0,
			UserId:    userID,
			UpdatedAt: time.Now(),
		}

		encode, err := json.Marshal(data)
		if err != nil {
			log.Printf("Error marshaling body: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		err = h.Redis.Set(c, constant.RedisKeyRateExChange, encode, -1).Err()
		if err != nil {
			h.Logger.Errorf("Set Redis error : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		}

		c.JSON(http.StatusOK, RateExchangeResponse{
			RateExchange: data,
		})
	}
}

func (h *ServiceHandler) UpdateRate() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}
		form := &formUpdateRateExchange{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Price < 0 {
			c.JSON(http.StatusForbidden, "Giá nhập vào phải lớn hơn 0")
			return
		}
		data := RateExchange{
			Price:     form.Price,
			UserId:    userID,
			UpdatedAt: time.Now(),
		}
		encode, err := json.Marshal(data)
		if err != nil {
			h.Logger.Errorf("Error marshaling body: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		log := &entity.ExchangeRateLog{
			UserID: userID,
			Rate:   form.Price,
		}

		if err := h.ServiceManager.CreateExchangeRateLog(log); err != nil {
			h.Logger.Errorf("save log exchange rate: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		err = h.Redis.Set(c, constant.RedisKeyRateExChange, encode, 0).Err()
		if err != nil {
			h.Logger.Errorf("set redis: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, updateRateExchangeResponse{true})
	}
}

func (h *ServiceHandler) UpdatePrices() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := &formUpdatePrice{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		msg, err := h.validate(form.Prices)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if msg != "" {
			c.JSON(http.StatusBadRequest, msg)
			return
		}

		prices := make(map[int64]map[string]interface{}, len(form.Prices))
		for _, v := range form.Prices {
			prices[v.PriceID] = map[string]interface{}{
				"price": utils.Round(v.Price, 2),
			}
		}

		err = h.ServiceManager.UpdatePrices(prices)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		h.Redis.Del(c, calculate.RedisKey)

		c.JSON(http.StatusOK, updatePriceResponse{true})
	}
}

func (h *ServiceHandler) validate(prices []Price) (string, error) {
	for _, price := range prices {
		if price.PriceID < 0 {
			return "Price ID is required", nil
		}

		if price.Weight < 0 {
			return "Weight is required", nil
		}
	}

	return "", nil
}
