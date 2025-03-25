package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/config"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/logger"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"

	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tebexpressapi/services/api/handlers/admin"
	"tebexpressapi/services/api/handlers/customer"
	"tebexpressapi/services/api/handlers/order"
	"tebexpressapi/services/api/handlers/packages"
	"tebexpressapi/services/api/handlers/prices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type api struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	logger     *zap.SugaredLogger

	server *http.Server

	redisConn *redis.Client
	mysqlConn *gorm.DB
}

func NewApi(configFile string) *api {
	app := &api{}

	ctx, cancelFunc := context.WithCancel(context.Background())
	app.ctx = ctx
	app.cancelFunc = cancelFunc
	app.logger = logger.InitLoggerRotate()

	if len(configFile) > 0 {
		configFiles := strings.Split(configFile, ";")

		// read by input file
		err := config.ReadConfigByFiles("toml", configFiles)
		if err != nil {
			app.logger.Panic("initConfig: Could not load conf: ", err)
		}
		app.logger.Info("init config")
	}

	mysqlConn, err := storage.NewMysqlConnection(storage.DefaultMysqlFromConfig(nil))
	if err != nil {
		app.logger.Panic(err)
	}

	redisConn, err := storage.NewRedisConnection(storage.DefaultRedisFromConfig(nil))
	if err != nil {
		app.logger.Panic(err)
	}

	app.redisConn = redisConn
	app.mysqlConn = mysqlConn
	auth := auth.Init(app.mysqlConn)
	orderManager := sqlmanager.NewOrderManager(app.mysqlConn)
	userManager := sqlmanager.NewUserManager(app.mysqlConn)
	serviceManager := sqlmanager.NewServiceManager(app.mysqlConn)
	packageManager := sqlmanager.NewPackageManager(app.mysqlConn)
	billManager := sqlmanager.NewBillManager(app.mysqlConn)
	stateManager := sqlmanager.NewStateManager(app.mysqlConn)
	productManager := sqlmanager.NewProductManager(app.mysqlConn)
	warehouseManager := sqlmanager.NewWareHouseManager(app.mysqlConn)
	settingManager := sqlmanager.NewSettingManager(app.mysqlConn)
	promotionManager := sqlmanager.NewPromotionManager(app.mysqlConn)
	trackingManager := sqlmanager.NewTrackingManager(app.mysqlConn)
	checkInManager := sqlmanager.NewCheckinManager(app.mysqlConn)
	customShipmentManager := sqlmanager.NewCustomerShipmentManager(app.mysqlConn)
	containerManager := sqlmanager.NewContainerManager(app.mysqlConn)
	shipmentManager := sqlmanager.NewShipmentManager(app.mysqlConn)
	transactionManager := sqlmanager.NewTransactionManager(app.mysqlConn)
	analyticsManager := sqlmanager.NewAnalyticsManager(app.mysqlConn)

	calculatePrice := calculate.New(app.mysqlConn, app.redisConn)
	createLabel := createlabel.Init(app.mysqlConn, app.redisConn)

	routes := httputil.Routes{
		httputil.Route{
			Name:     "healthz",
			Method:   http.MethodGet,
			BasePath: "",
			Pattern:  "",
			Handler:  healthz(),
		},
	}

	routes = append(routes, order.OrderRoutes(app.logger, auth, app.redisConn, orderManager)...)
	routes = append(routes, packages.PackageRoutes(app.logger, auth, app.redisConn, app.mysqlConn, calculatePrice, createLabel, userManager, serviceManager,
		packageManager, billManager, stateManager, productManager, warehouseManager, settingManager, promotionManager, trackingManager)...)
	routes = append(routes, prices.PriceRoutes(app.logger, auth, app.redisConn, app.mysqlConn, calculatePrice, serviceManager, stateManager, warehouseManager)...)
	routes = append(routes, admin.AdminRoutes(app.logger, auth, app.redisConn, calculatePrice, createLabel,
		userManager, packageManager, warehouseManager,
		serviceManager, billManager, checkInManager, customShipmentManager, trackingManager, containerManager, shipmentManager, transactionManager, stateManager, settingManager)...)
	routes = append(routes, customer.CustomerRoutes(app.logger, app.redisConn, createLabel, calculatePrice, userManager, settingManager, serviceManager, packageManager, billManager, transactionManager, stateManager, productManager, warehouseManager, trackingManager, analyticsManager, customShipmentManager)...)

	r := gin.Default()
	r.Use(httputil.CORSMiddleware())

	for _, route := range routes {
		handlers := make([]gin.HandlerFunc, 0)
		if route.AuthInfo != nil && route.AuthInfo.Enable {
			if route.AuthInfo.IsCustomer {
				handlers = append(handlers, auth.AuthenticationCustomerVerify(route.AuthInfo))
			} else {
				handlers = append(handlers, auth.AuthenticationVerify(route.AuthInfo))
			}
		}

		handlers = append(handlers, route.Middlewares...)
		handlers = append(handlers, route.Handler)
		r.Handle(route.Method, route.BasePath+route.Pattern, handlers...)
	}

	app.server = &http.Server{
		Addr:         fmt.Sprintf(":%v", viper.GetString("api.port")),
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
		Handler:      r,
	}

	return app
}

func (h *api) Start() {
	if err := h.server.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
		log.Printf("listen: %s\n", err)
	}

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	keepRunning := true
	for keepRunning {
		select {
		case <-h.ctx.Done():
			h.logger.Info("terminating: context cancelled")
			h.cancelFunc()
			keepRunning = false
		case <-sigterm:
			h.logger.Info("terminating: via signal")
			keepRunning = false
		}
	}
}

func (h *api) Stop() {
	if err := h.server.Shutdown(h.ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	mysql, _ := h.mysqlConn.DB()
	mysql.Close()
	_ = h.redisConn.Close()

	logger.CloseRotate(h.logger)
	log.Println("Server exiting")
}

func healthz() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(200, "Ananbay API!!!")
	}
}
