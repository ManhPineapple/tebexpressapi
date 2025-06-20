package packages

import (
	"net/http"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/httputil/auth"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const PackageBasePath = "/v1/packages"

func PackageRoutes(l *zap.SugaredLogger, au *auth.Auth, r *redis.Client, mysqlConn *gorm.DB,
	calculatePrice *calculate.CalculatePrice, createLabel *createlabel.CreateLabel,
	um *sqlmanager.UserManager, sm *sqlmanager.ServiceManager,
	pm *sqlmanager.PackageManager, bm *sqlmanager.BillManager,
	stm *sqlmanager.StateManager, prm *sqlmanager.ProductManager, wm *sqlmanager.WareHouseManager,
	sem *sqlmanager.SettingManager, pom *sqlmanager.PromotionManager, tm *sqlmanager.TrackingManager) httputil.Routes {

	packageHandler := &PackageHandler{
		Logger:    l,
		Redis:     r,
		StorageS3: storage.NewAmazonS3(nil),

		CalculatePrice: calculatePrice,
		CreateLabel:    createLabel,

		ShipmentEstimateCost:       packageutils.NewEstimateCost(l, pm, wm, sm, createLabel),
		ShipmentRefund:             packageutils.NewPackageRefund(l, pm, bm),
		ShipmentRefundCarrier:      packageutils.NewShipmentCancelCarrier(l, pm, tm),
		ShipmentCreateLabelHandler: packageutils.NewCreateLabelHandler(l, r, storage.NewAmazonS3(nil), sem, pm, bm, um, wm, sm, createLabel, nil),

		UserManager:      um,
		ServiceManager:   sm,
		PackageManager:   pm,
		BillManager:      bm,
		StateManager:     stm,
		ProductManager:   prm,
		WareHouseManager: wm,
		SettingManager:   sem,
		PromotionManager: pom,
		TrackingManager:  tm,
	}

	return httputil.Routes{
		httputil.Route{
			Name:        "Validate address",
			Method:      http.MethodPost,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/address/validate",
			Handler:     packageHandler.ValidateAddress(),
		},
		httputil.Route{
			Name:        "List packages",
			Method:      http.MethodGet,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "",
			Handler:     packageHandler.List(),
		},
		httputil.Route{
			Name:        "Create a package",
			Method:      http.MethodPost,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "",
			Handler:     packageHandler.Create(),
		},
		httputil.Route{
			Name:        "Update a package",
			Method:      http.MethodPut,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/:id",
			Handler:     packageHandler.Update(),
		},
		httputil.Route{
			Name:        "Package detail",
			Method:      http.MethodGet,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/:id",
			Handler:     packageHandler.Detail(),
		},
		httputil.Route{
			Name:        "Delivery packages",
			Method:      http.MethodPost,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/delivery/:id",
			Handler:     packageHandler.Delivery(),
		},
		httputil.Route{
			Name:        "Cancel package",
			Method:      http.MethodPut,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/cancel",
			Handler:     packageHandler.Cancel(),
		},
		httputil.Route{
			Name:     "Get Package Track",
			Method:   http.MethodGet,
			BasePath: PackageBasePath,
			Pattern:  "/:id/track",
			Handler:  packageHandler.Track(),
		},
		httputil.Route{
			Name:        "Fetch label handler",
			Method:      http.MethodGet,
			BasePath:    PackageBasePath,
			Middlewares: []gin.HandlerFunc{au.VerifyCustomer()},
			Pattern:     "/label/:code",
			Handler:     packageHandler.FetchLabel(),
		},
	}
}
