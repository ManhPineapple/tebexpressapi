package customer

import (
	"net/http"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const CustomerBasePath = "/v1/customer"

func CustomerRoutes(l *zap.SugaredLogger, r *redis.Client, createLabel *createlabel.CreateLabel,
	calculatePrice *calculate.CalculatePrice, um *sqlmanager.UserManager,
	sm *sqlmanager.SettingManager, srm *sqlmanager.ServiceManager, pm *sqlmanager.PackageManager,
	bm *sqlmanager.BillManager, trm *sqlmanager.TransactionManager,
	stm *sqlmanager.StateManager, whm *sqlmanager.WareHouseManager, tm *sqlmanager.TrackingManager,
	am *sqlmanager.AnalyticsManager, csm *sqlmanager.CustomerShipmentManager) httputil.Routes {
	s3 := storage.NewAmazonS3(nil)
	alert := alert.NewAlert()

	authHandler := NewAuthHandler(l, r, um)
	userHandler := NewUserHandler(l, r, um, sm, srm)
	configHandler := NewConfigHandler(l)
	packageHandler := NewPackageHandler(l, r, s3, alert, createLabel, calculatePrice, pm, um, stm, whm, srm, bm, sm, tm)
	billHandler := NewBillHandler(l, s3, bm, um)
	// productHandler := NewProductHandler(l, s3, bm, um)
	transactionHandler := NewTransactionHandler(l, r, trm, um)
	serviceHandler := NewServiceHandler(l, um, srm)
	uploadandler := NewUploadHandler(l, s3)
	analyticHandler := NewAnalyticHandler(l, r, am, pm)
	shipmentHandler := NewShipmentHandler(l, r, s3, alert, createLabel, calculatePrice, csm, bm, srm, um, sm, pm, whm, srm)

	return httputil.Routes{
		httputil.Route{
			Name:     "Sign In",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/auth/sign-in",
			Handler:  authHandler.SignIn(),
		},
		httputil.Route{
			Name:     "Sign Up",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/auth/sign-up",
			Handler:  authHandler.SignUp(),
		},
		
		httputil.Route{
			Name:     "Get User",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/users",
			Handler:  userHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get User",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/users-china",
			Handler:  userHandler.GetChina(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get User Token",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/users/token",
			Handler:  userHandler.Token(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Reset Token",
			Method:   http.MethodPut,
			BasePath: CustomerBasePath,
			Pattern:  "/users/token",
			Handler:  userHandler.Reset(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Fetch config Request",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/configs",
			Handler:  configHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Package",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages",
			Handler:  packageHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create package",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/create",
			Handler:  packageHandler.Create(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count List Packages",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/count",
			Handler:  packageHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Package Detail",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/:package_id",
			Handler:  packageHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Cancel packages",
			Method:   http.MethodPut,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/cancel",
			Handler:  packageHandler.Cancel(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Package Return",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/return",
			Handler:  packageHandler.Return(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count List Packages",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/return/count",
			Handler:  packageHandler.CountReturn(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Track package codes logs",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/logs",
			Handler:  packageHandler.Log(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count Track package codes logs",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/logs/count",
			Handler:  packageHandler.LogCount(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Package Holding",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/holding",
			Handler:  packageHandler.Holding(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Package Holding",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/holdingChina",
			Handler:  packageHandler.HoldingChina(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Method:   http.MethodPut,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/:package_id",
			Handler:  packageHandler.Update(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count List Packages holding",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/holding/count",
			Handler:  packageHandler.CountHolding(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Export package",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/export",
			Handler:  packageHandler.Export(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Import package",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/import",
			Handler:  packageHandler.Import(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Import package FBA",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/import/fba",
			Handler:  packageHandler.ImportFBA(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Process package",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/packages/process",
			Handler:  packageHandler.Process(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		// httputil.Route{
		// 	Name:     "Get List Product By User ID",
		// 	Method:   http.MethodGet,
		// 	BasePath: CustomerBasePath,
		// 	Pattern:  "/products",
		// 	Handler: productHandler.GetListProduct(),
		// 	AuthInfo: &auth.AuthInfo{
		// 		Enable: true,
		// 		UserRoles: map[string]bool{
		// 			constant.UserRoleAdmin:            true,
		// 			constant.UserRoleCustomer:         true,
		// 			constant.UserRoleAccountant:       true,
		// 			constant.UserRoleSupport:          true,
		// 			constant.UserRoleSupportLeader:    true,
		// 			constant.UserRoleWarehouse:        true,
		// 			constant.UserRolerBusinessManager: true,
		// 			constant.UserRolerShipPartner:     true,
		// 			constant.UserRoleSale:             true,
		// 			constant.UserRoleSaleOperation:    true,
		// 		},
		// 	},
		// },
		httputil.Route{
			Name:     "Get Bill count",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/count",
			Handler:  billHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Bill list",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/list",
			Handler:  billHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Bill list",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/listChina",
			Handler:  billHandler.ListChina(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Bill Detail",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/:code",
			Handler:  billHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get invoice",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/invoice/:code",
			Handler:  billHandler.Invoice(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get invoices",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/invoices",
			Handler:  billHandler.Invoices(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get bill Fees",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/fees/:bill_code",
			Handler:  billHandler.BillFees(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get bill packages",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/bills/packages/:bill_code",
			Handler:  billHandler.BillPackage(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Transactions",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions",
			Handler:  transactionHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		
		httputil.Route{
			Name:     "Get Transactions China",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/china",
			Handler:  transactionHandler.ListChina(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count Transactions",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/count",
			Handler:  transactionHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create Topup",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/top-up",
			Handler:  transactionHandler.TopUp(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create transaction",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions",
			Handler:  transactionHandler.Create(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Rate Exchange",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/rate-exchange",
			Handler:  transactionHandler.GetRate(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Update Topup",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/top-up/update/:id",
			Handler:  transactionHandler.UpdateTopup(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},

		httputil.Route{
			Name:     "Update Topup",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/transactions/top-up/update-china/:id",
			Handler:  transactionHandler.UpdateTopupChina(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},

		httputil.Route{
			Name:     "Get List Service",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/services",
			Handler:  serviceHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Download file api",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/uploads/file-export/download",
			Handler:  uploadandler.DownloadLabel(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Dashborad analytics",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/analytics",
			Handler:  analyticHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "count shipment",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/count",
			Handler:  shipmentHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get shipment",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments",
			Handler:  shipmentHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "fulfill shipment",
			Method:   http.MethodPost,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/fulfill/:id",
			Handler:  shipmentHandler.Fullfill(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "detail shipment",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/:shipment_id",
			Handler:  shipmentHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "cancel shipment",
			Method:   http.MethodPut,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/cancel/:shipment_id",
			Handler:  shipmentHandler.Cancel(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "fetch shipment item",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/items/:shipment_id",
			Handler:  shipmentHandler.Items(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
		httputil.Route{
			Name:     "detail shipment",
			Method:   http.MethodGet,
			BasePath: CustomerBasePath,
			Pattern:  "/shipments/items/count/:shipment_id",
			Handler:  shipmentHandler.ItemsCount(),
			AuthInfo: &auth.AuthInfo{
				Enable:     true,
				IsCustomer: true,
				UserRoles: map[string]bool{
					constant.UserRoleCustomer: true,
				},
			},
		},
	}
}
