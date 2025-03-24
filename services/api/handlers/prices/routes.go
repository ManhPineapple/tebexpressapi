package prices

import (
	"net/http"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	PriceBasePath   = "/v1/prices"
	ServiceBasePath = "/v1/services"
)

func PriceRoutes(l *zap.SugaredLogger, au *auth.Auth, r *redis.Client, mysqlConn *gorm.DB, calculatePrice *calculate.CalculatePrice,
	sm *sqlmanager.ServiceManager, stm *sqlmanager.StateManager, wm *sqlmanager.WareHouseManager) httputil.Routes {

	priceHandler := &PriceHandler{
		Logger: l,

		CalculatePrice: calculatePrice,
		PartnerMaps:    entity.PartnerMap,

		ServiceManager:   sm,
		StateManager:     stm,
		WareHouseManager: wm,
	}

	return httputil.Routes{
		httputil.Route{
			Name:        "Get price",
			Method:      http.MethodPost,
			BasePath:    PriceBasePath,
			Middlewares: []gin.HandlerFunc{},
			Pattern:     "",
			Handler:     priceHandler.GetPackagePrice(),
		},
		httputil.Route{
			Name:        "Get services",
			Method:      http.MethodGet,
			BasePath:    ServiceBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "",
			Handler:     priceHandler.GetServices(),
		},
	}
}
