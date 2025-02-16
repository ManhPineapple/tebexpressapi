package order

import (
	"net/http"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const OrderBasePath = "/v1/orders"

func OrderRoutes(l *zap.SugaredLogger, au *auth.Auth, r *redis.Client, om *sqlmanager.OrderManager) httputil.Routes {
	orderCreateHandler := NewOrderCreateHandler(l, r, om)

	return httputil.Routes{
		httputil.Route{
			Name:        "order create",
			Method:      http.MethodPost,
			BasePath:    OrderBasePath,
			Pattern:     "",
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Handler:     orderCreateHandler.Serve(),
		},
	}
}
