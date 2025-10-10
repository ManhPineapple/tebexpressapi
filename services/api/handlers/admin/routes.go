package admin

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

const AdminBasePath = "/v1/admin"

func AdminRoutes(l *zap.SugaredLogger, au *auth.Auth, r *redis.Client,
	calculatePrice *calculate.CalculatePrice, createLabel *createlabel.CreateLabel,
	um *sqlmanager.UserManager, pm *sqlmanager.PackageManager, wh *sqlmanager.WareHouseManager,
	sm *sqlmanager.ServiceManager, bm *sqlmanager.BillManager, cim *sqlmanager.CheckinManager,
	csm *sqlmanager.CustomerShipmentManager, tm *sqlmanager.TrackingManager,
	cm *sqlmanager.ContainerManager, shm *sqlmanager.ShipmentManager, tsm *sqlmanager.TransactionManager,
	stm *sqlmanager.StateManager, setm *sqlmanager.SettingManager, prm *sqlmanager.PromotionManager) httputil.Routes {

	s3 := storage.NewAmazonS3(nil)
	alert := alert.NewAlert()

	userHandler := NewUserHandler(l, um)
	authHandler := NewAuthHandler(l, um)
	packageHandler := NewPackageHandler(l, r, s3, calculatePrice, createLabel, um, pm, tm, wh, stm, sm, bm, setm, alert)
	warehouseHandler := NewWarehouseHandler(l, r, s3, calculatePrice, createLabel, wh, pm, um, cim, csm, bm, sm, tm)
	serviceHandler := NewServiceHandler(l, r, sm)
	billHandler := NewBillHandler(l, bm, um, pm)
	checkInHandler := NewCheckInHandler(l, cim, um)
	exportHandler := NewExportHandler(l, s3, pm, shm)
	containerHandler := NewContainerHandler(l, s3, um, wh, cm, pm, tm)
	shipmentHandler := NewShipmentHandler(l, r, s3, um, shm, cm, wh, tm, csm, pm)
	transactionHandler := NewTransactionHandler(l, tsm, um)
	promotionHandler := NewPromotionManager(l, r, s3, prm, sm, um)
	hubHandler := NewHubHandler(l, pm, um, bm, sm)

	return httputil.Routes{
		httputil.Route{
			Name:     "Get User",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/users",
			Handler:  userHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleWarehouse:        true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleHub:              true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "List User",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/users/all",
			Handler:  userHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Count List Users",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/users/all/count",
			Handler:  userHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Filter by role",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/users/role",
			Handler:  userHandler.Role(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Create user",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/users",
			Handler:  userHandler.Create(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRolerBusinessManager: true,
				},
			},
		},
		httputil.Route{
			Name:     "Update user",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/users/:id",
			Handler:  userHandler.Update(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRolerBusinessManager: true,
				},
			},
		},
		httputil.Route{
			Name:     "Update Status User",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/users/update-status",
			Handler:  userHandler.UpdateStatus(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
				},
			},
		},
		httputil.Route{
			Name:     "Update User Info",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/users/info/:user_id",
			Handler:  userHandler.Info(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
		httputil.Route{
			Name:     "Sign In",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/auth/sign-in",
			Handler:  authHandler.SignIn(),
		},
		httputil.Route{
			Name:     "List Package",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/packages",
			Handler:  packageHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Count Package",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/packages/count",
			Handler:  packageHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleSupport:          true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Check Package In Warehouse",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/warehouses/packages/:package_id/accept",
			Handler:  warehouseHandler.Accept(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
					constant.UserRoleCustomer:  true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Package Detail",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/packages/:id",
			Handler:  packageHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleSupport:          true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Package Detail By Code",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/packages/code/:code",
			Handler:  packageHandler.DetailByCode(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Update package",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/packages/:package_id",
			Handler:  packageHandler.Update(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Cancel packages",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/packages/cancel",
			Handler:  packageHandler.Cancel(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "reship estimate cost",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/packages/reship/estimate-cost/:package_id",
			Handler:  packageHandler.EstimateCostReship(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Import tracking",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/packages/import/tracking",
			Handler:  packageHandler.ImportTracking(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
				},
			},
		},
		httputil.Route{
			Name:     "Process CN Package",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/packages/process_cn",
			Handler:  packageHandler.ProcessCNPackage(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleWarehouse:        true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Update Tiktok package's weight",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/packages/tiktok_weight/:package_id",
			Handler:  packageHandler.ForceUpdateTiktokWeight(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSupport:          true,
				},
			},
		},
		httputil.Route{
			Name:     "Update Tiktok package's Label",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/packages/tiktok_label/:package_id",
			Handler:  packageHandler.UpdateTiktokLabelUrl(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSupport:          true,
				},
			},
		},
		httputil.Route{
			Name:     "List Warehouse",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/warehouses",
			Handler:  warehouseHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleWarehouse:        true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Package In Warehouse",
			BasePath: AdminBasePath,
			Method:   http.MethodGet,
			Pattern:  "/warehouses/packages-in-warehouse",
			Handler:  warehouseHandler.ListPackageInWarehouse(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Package In Warehouse",
			BasePath: AdminBasePath,
			Method:   http.MethodGet,
			Pattern:  "/warehouses/packages-in-warehouse/count",
			Handler:  warehouseHandler.CountPackageInWarehouse(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get package",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/warehouses/packages/:code",
			Handler:  warehouseHandler.GetPackage(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "Return Package In Warehouse",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/warehouses/packages/return",
			Handler:  warehouseHandler.Return(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "check-relabel package",
			BasePath: AdminBasePath,
			Method:   http.MethodPost,
			Pattern:  "/warehouses/packages/:package_id/check-relabel",
			Handler:  warehouseHandler.CheckReLabel(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create label",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/warehouses/packages/labels/:package_id",
			Handler:  warehouseHandler.CreateTracking(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Service",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/services/:service_id",
			Handler:  serviceHandler.Get(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:      true,
					constant.UserRoleAccountant: true,
					constant.UserRoleWarehouse:  true,
					constant.UserRoleSupport:    true,
					constant.UserRoleHub:        true,
					constant.UserRoleSale:       true,
				},
			},
		},
		httputil.Route{
			Name:     "Get List Service",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/services",
			Handler:  serviceHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		// httputil.Route{
		// 	Name:     "Get Service Prices",
		// 	Method:   http.MethodGet,
		// 	BasePath: AdminBasePath,
		// 	Pattern:  "/services/rate",
		// 	Handler:  serviceHandler.GetRate(),
		// 	AuthInfo: &auth.AuthInfo{
		// 		Enable: true,
		// 		UserRoles: map[string]bool{
		// 			constant.UserRoleAdmin:      true,
		// 			constant.UserRoleAccountant: true,
		// 		},
		// 	},
		// },
		// httputil.Route{
		// 	Name:     "Update Service Prices",
		// 	Method:   http.MethodPut,
		// 	BasePath: AdminBasePath,
		// 	Pattern:  "/services/rate",
		// 	Handler:  serviceHandler.UpdateRate(),
		// 	AuthInfo: &auth.AuthInfo{
		// 		Enable: true,
		// 		UserRoles: map[string]bool{
		// 			constant.UserRoleAdmin:      true,
		// 			constant.UserRoleAccountant: true,
		// 		},
		// 	},
		// },
		httputil.Route{
			Name:     "Update Service Prices",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/services/prices",
			Handler:  serviceHandler.UpdatePrices(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get list extra fee type",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/bills/extra-fee-types",
			Handler:  billHandler.FeeTypes(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Bill List",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/bills",
			Handler:  billHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSale:             true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Bill Detail",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/bills/:bill_code",
			Handler:  billHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Bill Fee",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/bills/fee/:bill_code",
			Handler:  billHandler.DetailFee(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Bill Count",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/bills/count",
			Handler:  billHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSale:             true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
				},
			},
		},
		httputil.Route{
			Name:     "Create extra fee",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/bills/extra-fee",
			Handler:  billHandler.ExtraFee(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Fetch Checkin Request",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/checkins",
			Handler:  checkInHandler.Fetch(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "Close Checkin Request",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/checkins/close",
			Handler:  checkInHandler.Close(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:     true,
					constant.UserRoleWarehouse: true,
				},
			},
		},
		httputil.Route{
			Name:     "Download file api",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/uploads/file-export/download",
			Handler:  exportHandler.DownloadLabel(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRoleHub:              true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Import tracking",
			Method:   http.MethodPost,
			BasePath: "/v1/export",
			Pattern:  "/shipment/packages/list",
			Handler:  exportHandler.ExportPackage(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Admin export shipment",
			Method:   http.MethodPost,
			BasePath: "/v1/export",
			Pattern:  "/shipment/shipments",
			Handler:  exportHandler.ExportShipment(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
				},
			},
		},
		httputil.Route{
			Name:     "Count list container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers/count",
			Handler:  containerHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get list container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers",
			Handler:  containerHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create a container",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/containers",
			Handler:  containerHandler.Create(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Detail Container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers/:container_code",
			Handler:  containerHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get list boxes container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers/box",
			Handler:  containerHandler.Box(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Append Package to Container",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/containers/append",
			Handler:  containerHandler.Append(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Cancel Package From Container",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/containers/remove",
			Handler:  containerHandler.Remove(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Manifest Container",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/containers/manifest/:container_id",
			Handler:  containerHandler.Manifest(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get ManifestUrl Container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers/manifest/:container_id",
			Handler:  containerHandler.GetManifestByContainerID(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Close Container",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/containers/close/:container_id",
			Handler:  containerHandler.Close(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Reopen a container",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/containers/open/:container_id",
			Handler:  containerHandler.Open(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Cancel container",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/containers/cancel/:container_id",
			Handler:  containerHandler.Cancel(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get history container",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/containers/history/:container_id",
			Handler:  containerHandler.History(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Update a container",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/containers/:container_id",
			Handler:  containerHandler.Update(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get shipment",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments",
			Handler:  shipmentHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "count customer shipment",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/customer/count",
			Handler:  shipmentHandler.ShipmentCustomerCount(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Get customer shipment detail",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/customer/:customer_shipment_id",
			Handler:  shipmentHandler.DetailShipmentCustomer(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Get containers and packages of shipment by id",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/deep-detail/:customer_shipment_id",
			Handler:  shipmentHandler.DeepDetailShipmentCustomer(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "count shipment",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/count",
			Handler:  shipmentHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Create shipment",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/create",
			Handler:  shipmentHandler.Create(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get shipment by id",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/:shipment_id",
			Handler:  shipmentHandler.Detail(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleAccountant:    true,
					constant.UserRoleHub:           true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
					constant.UserRoleSaleOperation: true,
				},
			},
		},
		httputil.Route{
			Name:     "Cancel a shipment",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/cancel/:shipment_id",
			Handler:  shipmentHandler.Cancel(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Search and append container to shipment",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/append",
			Handler:  shipmentHandler.Append(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Remove a container in shipment",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/cancel-container",
			Handler:  shipmentHandler.RemoveContainerShipment(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Close a shipment",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/close",
			Handler:  shipmentHandler.Close(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get zip label from shipment",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/zip/:shipment_id",
			Handler:  shipmentHandler.Zip(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get customer shipment ",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/customer",
			Handler:  shipmentHandler.CustomerShipment(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleWarehouse:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
					constant.UserRoleSaleOperation:    true,
				},
			},
		},
		httputil.Route{
			Name:     "Intransit shipment",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/shipments/intransit/:shipment_id",
			Handler:  shipmentHandler.Intransit(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
				},
			},
		},
		httputil.Route{
			Name:     "List Transaction logs",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/transactions",
			Handler:  transactionHandler.List(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Get Transaction logs count",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/transactions/count",
			Handler:  transactionHandler.Count(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRoleSupport:          true,
					constant.UserRolerBusinessManager: true,
					constant.UserRolerShipPartner:     true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Change status transaction log",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/transactions/status",
			Handler:  transactionHandler.ChangeStatus(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleAccountant:       true,
					constant.UserRolerBusinessManager: true,
				},
			},
		},
		httputil.Route{
			Name:     "Scan Return Package",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/hubs/package/return",
			Handler:  hubHandler.Return(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
					constant.UserRoleHub:   true,
				},
			},
		},
		httputil.Route{
			Name:     "import container event",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/containers/events/import",
			Handler:  containerHandler.ImportEvent(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
				},
			},
		},
		httputil.Route{
			Name:     "make container event",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/containers/events",
			Handler:  containerHandler.CreateEvent(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:         true,
					constant.UserRoleWarehouse:     true,
					constant.UserRoleSupportLeader: true,
					constant.UserRolerShipPartner:  true,
				},
			},
		},

		httputil.Route{
			Name:     "Create promotions",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/promotions",
			Handler:  promotionHandler.CreatePromotion(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
				},
			},
		},
		httputil.Route{
			Name:     "Fetch promotions",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/promotions",
			Handler:  promotionHandler.GetPromotions(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Count promotions",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/count",
			Handler:  promotionHandler.CountPromotion(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Update promotion",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/:id",
			Handler:  promotionHandler.UpdatePromotion(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleMarketing:        true,
					constant.UserRolerBusinessManager: true,
				},
			},
		},
		httputil.Route{
			Name:     "Append user to promotion",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/append",
			Handler:  promotionHandler.AppendUserToPromotion(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Get promotion users",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/:id/users",
			Handler:  promotionHandler.GetPromotionUser(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin:            true,
					constant.UserRoleMarketing:        true,
					constant.UserRoleSupport:          true,
					constant.UserRoleSupportLeader:    true,
					constant.UserRolerBusinessManager: true,
					constant.UserRoleSale:             true,
				},
			},
		},
		httputil.Route{
			Name:     "Create setting point",
			Method:   http.MethodPost,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/setting-point",
			Handler:  promotionHandler.CreateSettingPoint(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
		httputil.Route{
			Name:     "Get list setting point",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/setting-point",
			Handler:  promotionHandler.GetListSettingPoint(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
		httputil.Route{
			Name:     "Count list setting point",
			Method:   http.MethodGet,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/setting-point/count",
			Handler:  promotionHandler.CountListSettingPoint(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
		httputil.Route{
			Name:     "Update setting point",
			Method:   http.MethodPut,
			BasePath: AdminBasePath,
			Pattern:  "/promotions/setting-point",
			Handler:  promotionHandler.UpdateSettingPoint(),
			AuthInfo: &auth.AuthInfo{
				Enable: true,
				UserRoles: map[string]bool{
					constant.UserRoleAdmin: true,
				},
			},
		},
	}
}
