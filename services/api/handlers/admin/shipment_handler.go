package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/fedex"
	"tebexpressapi/pkg/providers/ups"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/file"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ShipmentHandler struct {
	Logger    *zap.SugaredLogger
	Redis     *redis.Client
	StorageS3 storage.S3

	UPS   *ups.UPS
	FEDEX *fedex.FedEx

	UserManager             *sqlmanager.UserManager
	ShipmentManager         *sqlmanager.ShipmentManager
	ContainerManager        *sqlmanager.ContainerManager
	WareHouseManager        *sqlmanager.WareHouseManager
	TrackingManager         *sqlmanager.TrackingManager
	CustomerShipmentManager *sqlmanager.CustomerShipmentManager
	PackageManager          *sqlmanager.PackageManager
}

type GetListCustomerShipmentHandlerResponse struct {
	Shipments []entity.CustomerShipment `json:"shipments"`
}

type GetListShipmentHandlerResponse struct {
	Shipments []entity.Shipment `json:"shipments"`
}

type CountListShipmentResponse struct {
	Count       int64                     `json:"count"`
	CountStatus []dto.CountStatusShipment `json:"count_status"`
}

type CreateShipmentResponse struct {
	Shipment interface{} `json:"shipment"`
}

type CreateShipmentForm struct {
	WarehouseID int64 `json:"warehouse_id"`
	IsFBA       bool  `json:"is_fba"`
}

type GetDetailShipmentResponse struct {
	Shipment       *entity.Shipment `json:"shipment"`
	Containers     interface{}      `json:"containers"`
	CountContainer interface{}      `json:"count_container"`
}

type ShrinkedContainer struct {
	Code     string            `json:"code,omitempty"`
	Packages []*entity.Package `json:"packages"`
}

type GetDeepDetailShipmentHandlerResponse struct {
	ShipmentId int64               `json:"shipment_id"`
	Containers []ShrinkedContainer `json:"containers"`
}

type CancelShipmentResponse struct {
	Success bool `json:"success"`
}

type AppendContainerToShipmentResponse struct {
	Container *entity.Container `json:"container"`
}
type AppendContainerToShipmentForm struct {
	Search     string `json:"search"`
	Barcode    string `json:"barcode"`
	ShipmentID int64  `json:"shipment_id"`
}

type CancelContainerInShipmentForm struct {
	ContainerID int64 `json:"container_id"`
	ShipmentID  int64 `json:"shipment_id"`
}

type CloseShipmentResponse struct {
	Success bool `json:"success"`
}

type CloseShipmentForm struct {
	UpsAccount    int     `json:"ups_account"`
	FedexAccount  int     `json:"fedex_account"`
	ShipmentValue float64 `json:"shipment_value"`
	ID            int     `json:"id"`
}

type DownloadLabelShipmentReponse struct {
	Url []string `json:"url"`
}

type CountListCustomerShipmentResponse struct {
	Count int64 `json:"count"`
}

type ChangeIntransitShipmentResponse struct {
	Success bool `json:"success"`
}
type GetDetailCustomerShipmentResponse struct {
	Shipment    *entity.CustomerShipment `json:"shipment"`
	Packages    []entity.Package         `json:"packages"`
	TotalAmount float64                  `json:"total_amount"`
}

func NewShipmentHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, um *sqlmanager.UserManager, sm *sqlmanager.ShipmentManager, cm *sqlmanager.ContainerManager, wh *sqlmanager.WareHouseManager, tm *sqlmanager.TrackingManager, csm *sqlmanager.CustomerShipmentManager, pm *sqlmanager.PackageManager) *ShipmentHandler {
	return &ShipmentHandler{
		Logger:    l,
		Redis:     r,
		StorageS3: s3,

		UserManager:             um,
		ShipmentManager:         sm,
		ContainerManager:        cm,
		WareHouseManager:        wh,
		TrackingManager:         tm,
		CustomerShipmentManager: csm,
		PackageManager:          pm,
	}
}

func (h *ShipmentHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.ShipmentQueryOptions{
			Status:      cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:      cast.ToString(c.Request.URL.Query().Get("search")),
			HubID:       cast.ToInt64(c.Request.URL.Query().Get("hubID")),
			WarehouseID: cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
			IsFba:       cast.ToInt(c.Request.URL.Query().Get("fba")),
			Limit:       limit,
			Offset:      offset,
		}

		if opts.IsFba > 0 {
			opts.HubID = 0
		}

		if opts.IsFba == 0 && opts.HubID > 0 {
			opts.IsFba = -1
		}

		if user.Role == constant.UserRoleWarehouse {
			if opts.WarehouseID > 0 && opts.WarehouseID != user.WarehouseID {
				c.JSON(http.StatusOK, GetListShipmentHandlerResponse{make([]entity.Shipment, 0)})
			}
			opts.WarehouseID = user.WarehouseID
		}

		shipments, err := h.ShipmentManager.GetShipments(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetListShipmentHandlerResponse{shipments})
	}
}

func (h *ShipmentHandler) ShipmentCustomerCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		status := strings.TrimSpace(c.Request.URL.Query().Get("status"))
		opts := sqlmanager.CustomerShipmentOption{
			InStatus:  constant.MapIntGroupStatusCustomerPackage[status],
			Keyword:   strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleSale || role == constant.UserRoleSupport {
			userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
			opts.SupportID = userID
		}

		count, err := h.CustomerShipmentManager.CountCustomerShipments(opts)
		if err != nil {
			h.Logger.Errorf("Count list customer shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, CountListCustomerShipmentResponse{count})
	}
}

func (h *ShipmentHandler) DetailShipmentCustomer() gin.HandlerFunc {
	return func(c *gin.Context) {
		shipmentID := cast.ToInt64(c.Param("customer_shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}
		shipment, err := h.CustomerShipmentManager.GetCustomerShipment(sqlmanager.CustomerShipmentOption{ID: shipmentID, UserPreload: true, ExtraFeePreload: true})
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		totalAmount, err := h.CustomerShipmentManager.GetCustomerShipmentPrice(shipment.ID)
		if err != nil {
			h.Logger.Errorf("get total shipment amount:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		status, err := h.CustomerShipmentManager.GetCustomerShipmentStatus(shipment.ID)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		shipment.Status = status

		packages, err := h.PackageManager.GetPackagesInCustomerShipment(sqlmanager.PackageQueryOption{
			CustomerShipmentID: shipmentID,
		})

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetDetailCustomerShipmentResponse{shipment, packages, totalAmount})
	}
}

func (h *ShipmentHandler) DeepDetailShipmentCustomer() gin.HandlerFunc {
	return func(c *gin.Context) {
		shipmentID := cast.ToInt64(c.Param("customer_shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}

		var containers []entity.Container
		containers, err := h.ContainerManager.GetContainers(sqlmanager.ContainerQueryOptions{ShipmentID: shipmentID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Failed to fetch containers!")
			return
		}

		containerIDs := make([]int64, len(containers))
		containerMap := make(map[int64]string)

		for i, container := range containers {
			containerIDs[i] = container.ID
			containerMap[container.ID] = container.Code
		}

		packages, err := h.ContainerManager.GetContainerPackages(containerIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, "Failed to fetch packages!")
			return
		}

		containerPackages := make(map[int64][]*entity.Package)
		for _, pkg := range packages {
			for _, containerID := range containerIDs {
				if _, exists := containerPackages[containerID]; !exists {
					containerPackages[containerID] = []*entity.Package{}
				}
				containerPackages[containerID] = append(containerPackages[containerID], pkg)
			}
		}

		var shrinkedContainers []ShrinkedContainer
		for containerID, code := range containerMap {
			shrinkedContainers = append(shrinkedContainers, ShrinkedContainer{
				Code:     code,
				Packages: containerPackages[containerID],
			})
		}

		c.JSON(http.StatusOK, GetDeepDetailShipmentHandlerResponse{
			ShipmentId: shipmentID,
			Containers: shrinkedContainers,
		})
	}
}

func (h *ShipmentHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		opts := sqlmanager.ShipmentQueryOptions{
			Status:      cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:      cast.ToString(c.Request.URL.Query().Get("search")),
			HubID:       cast.ToInt64(c.Request.URL.Query().Get("hubID")),
			WarehouseID: cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
			IsFba:       cast.ToInt(c.Request.URL.Query().Get("fba")),
		}

		if user.Role == constant.UserRoleWarehouse {
			if opts.WarehouseID > 0 && opts.WarehouseID != user.WarehouseID {
				c.JSON(http.StatusOK, CountListShipmentResponse{0, make([]dto.CountStatusShipment, 0)})
			}
			opts.WarehouseID = user.WarehouseID
		}

		if opts.IsFba > 0 {
			opts.HubID = 0
		}

		if opts.IsFba == 0 && opts.HubID > 0 {
			opts.IsFba = -1
		}

		count, err := h.ShipmentManager.CountShipments(opts)

		if err != nil {
			h.Logger.Errorf("Count list container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		countStatus, err := h.ShipmentManager.CountAllStatusShipment(opts)
		if err != nil {
			h.Logger.Errorf("Count all container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListShipmentResponse{count, countStatus})
	}
}

func (h *ShipmentHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		newForm := CreateShipmentForm{}
		if err := c.ShouldBindJSON(&newForm); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if !newForm.IsFBA && newForm.WarehouseID < 1 {
			c.JSON(http.StatusBadRequest, "Warehouse ID is invalid !")
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if user.ID <= 0 || user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		hubID := utils.Int64(newForm.WarehouseID)
		if newForm.WarehouseID < 1 {
			hubID = nil
		}

		shipment := &entity.Shipment{
			Status:   constant.ShipmentWaitingClose,
			UserID:   userID,
			Quantity: 0,
			HubID:    hubID,
			IsFba:    newForm.IsFBA,
		}

		if user.Role == constant.UserRoleWarehouse {
			shipment.WarehouseID = user.WarehouseID
		} else {
			shipment.WarehouseID = viper.GetInt64("warehouse_vn_default")
		}

		err = h.ShipmentManager.SaveShipment(shipment)
		if err != nil {
			h.Logger.Errorf("Create shipment: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, CreateShipmentResponse{shipment})
	}
}

func (h *ShipmentHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		offset, limit := httputil.GetRequestPaginate(c.Request)
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}

		options := sqlmanager.ShipmentQueryOptions{
			ID:              shipmentID,
			ContainerCode:   cast.ToString(c.Request.URL.Query().Get("search")),
			LimitContainer:  limit,
			OffsetContainer: offset,
			LoadContainer:   false,
			LoadManifest:    true,
		}

		if user.Role == constant.UserRoleWarehouse {
			options.WarehouseID = user.WarehouseID
		}

		shipment, err := h.ShipmentManager.GetShipment(options)

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var containers []struct {
			ID             int64      `json:"id,omitempty"`
			Code           string     `json:"code,omitempty"`
			TrackingNumber string     `json:"tracking_number,omitempty"`
			LabelUrl       string     `json:"label_url,omitempty"`
			CarrierID      int64      `json:"carrier_id,omitempty"`
			Width          float64    `json:"width,omitempty"`
			Height         float64    `json:"height,omitempty"`
			Length         float64    `json:"length,omitempty"`
			MaxWeight      float64    `json:"max_weight,omitempty"`
			Weight         float64    `json:"weight,omitempty"`
			ActualWeight   float64    `json:"actual_weight,omitempty"`
			Status         int        `json:"status,omitempty"`
			Barcode        string     `json:"barcode,omitempty"`
			ShipmentID     *int64     `json:"shipment_id,omitempty"`
			HubID          int64      `json:"hub_id,omitempty"`
			CloseAt        *time.Time `json:"close_at,omitempty"`
			HubExportedAt  *time.Time `json:"hub_exported_at,omitempty"`
			HubImportedAt  *time.Time `json:"hub_imported_at,omitempty"`
			WarehouseID    int64      `json:"warehouse_id,omitempty"`
			Type           int        `json:"type,omitempty"`
			CreatedAt      *time.Time `json:"created_at,omitempty"`

			CountItems int64 `json:"count_items"`
		}
		err = h.ShipmentManager.GetContainerInShipment(sqlmanager.ShipmentQueryOptions{
			ID:              shipmentID,
			LimitContainer:  limit,
			OffsetContainer: offset,
		}, &containers)

		containerCount, err := h.ShipmentManager.CountContainerShipment(options)

		if err != nil {
			h.Logger.Errorf("Count container in shipment error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetDetailShipmentResponse{
			shipment,
			containers,
			containerCount,
		})
	}
}

func (h *ShipmentHandler) RemoveContainerShipment() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := CancelContainerInShipmentForm{}
		decoder := json.NewDecoder(c.Request.Body)

		if err := decoder.Decode(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}
		if form.ContainerID <= 0 || form.ShipmentID <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		shipment, err := h.ShipmentManager.GetShipment(sqlmanager.ShipmentQueryOptions{
			ID: form.ShipmentID,
		})

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if shipment.Status != constant.ShipmentWaitingClose {
			c.JSON(http.StatusBadRequest, "Shipment status is invalid !")
			return
		}

		options := sqlmanager.ContainerQueryOptions{
			ID: form.ContainerID,
		}
		container, err := h.ContainerManager.GetContainer(options)

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if utils.Int64Value(container.ShipmentID) != form.ShipmentID {
			c.JSON(http.StatusForbidden, "Container is not in shipment !")
			return
		}

		shipment.Quantity--
		shipment.UpdatedAt = time.Now()
		container.ShipmentID = nil
		container.UpdatedAt = time.Now()

		if shipment.Quantity < 1 && shipment.IsFba {
			shipment.Address = ""
			shipment.City = ""
			shipment.State = ""
			shipment.Country = ""
			shipment.Zipcode = ""
		}

		err = h.ContainerManager.RemoveContainerShipment(container, shipment)

		if err != nil {
			h.Logger.Errorf("Save container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CancelShipmentResponse{true})
	}
}

func (h *ShipmentHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing shipment id !")
			return
		}

		opts := sqlmanager.ShipmentQueryOptions{
			ID:            shipmentID,
			LoadContainer: true,
		}

		shipment, err := h.ShipmentManager.GetShipment(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if shipment.Status == constant.ShipmentCanceled {
			c.JSON(http.StatusBadRequest, "Lô hàng đã bị cancel !")
			return
		}

		if len(shipment.Containers) > 0 {
			c.JSON(http.StatusForbidden, "Lô hàng đã có kiện hàng không thể hủy !")
			return
		}
		shipment.Status = constant.ShipmentCanceled
		shipment.UpdatedAt = time.Now()

		err = h.ShipmentManager.SaveShipment(shipment)
		if err != nil {
			h.Logger.Errorf("Save shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CancelShipmentResponse{true})
	}
}

func (h *ShipmentHandler) Append() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := AppendContainerToShipmentForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if (len(form.Search) == 0 && len(form.Barcode) == 0) || form.ShipmentID <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		var regex = regexp.MustCompile("^[A-Z0-9]*$")
		if form.Barcode != "" && !regex.MatchString(form.Barcode) {
			c.JSON(http.StatusBadRequest, "Mã barcode sai định dạng")
			return
		}

		options := sqlmanager.ContainerQueryOptions{
			Code:    form.Search,
			Barcode: form.Barcode,
		}

		container, err := h.ContainerManager.GetContainer(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Không tìm thấy kiện hàng !")
			return
		}

		if err != nil {
			h.Logger.Errorf("Get container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if container.Status != constant.ContainerClosed {
			c.JSON(http.StatusBadRequest, "Kiện hàng chưa đóng kiện !")
			return
		}

		if container.ShipmentID != nil {
			c.JSON(http.StatusBadRequest, "Kiện hàng đã được thêm vào lô hàng !")
			return
		}

		shipment, err := h.ShipmentManager.GetShipment(sqlmanager.ShipmentQueryOptions{ID: form.ShipmentID})
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if shipment.Status != constant.ShipmentWaitingClose {
			c.JSON(http.StatusBadRequest, "Shipment status is invalid !")
			return
		}

		if container.IsFba > 0 && !shipment.IsFba {
			c.JSON(http.StatusNotFound, "Lô hàng không đi FBA!")
			return
		}

		if container.IsFba < 1 && shipment.IsFba {
			c.JSON(http.StatusNotFound, "Kiện hàng thường không cho vào được lô hàng FBA")
			return
		}

		if shipment.IsFba {
			firstPackageInContainer, err := h.ContainerManager.GetContainerFirstPackage(container.ID)
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, "Kiện hàng trống!")
				return
			}

			if shipment.Quantity == 0 {
				shipment.Recipient = firstPackageInContainer.Recipient
				shipment.Address = firstPackageInContainer.Address1
				shipment.City = firstPackageInContainer.City
				shipment.Zipcode = firstPackageInContainer.Zipcode
				shipment.State = firstPackageInContainer.StateCode
				shipment.Country = firstPackageInContainer.CountryCode
			} else if !checkEqualAddress(firstPackageInContainer, shipment) {
				c.JSON(http.StatusBadRequest, "Địa chỉ đơn hàng không hợp lệ với địa chỉ lô hàng")
				return
			}
		}

		if !shipment.IsFba && utils.Int64Value(shipment.HubID) != container.HubID {
			c.JSON(http.StatusBadRequest, "Kiện hàng không cùng kho với lô hàng")
			return
		}

		wareHouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
			ID:     shipment.WarehouseID,
			Status: constant.WareHouseStatusActive,
		})

		if err != nil {
			h.Logger.Errorf("Get warehouse error: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if !shipment.IsFba && shipment.WarehouseID != container.WarehouseID {
			c.JSON(http.StatusNotFound, fmt.Sprintf("Kiện hàng không nằm trong kho %v", wareHouse.Name))
			return
		}

		shipment.Quantity++
		shipment.UpdatedAt = time.Now()
		container.ShipmentID = &shipment.ID
		container.UpdatedAt = time.Now()
		err = h.ContainerManager.AppendContainerShipment(container, shipment)

		if err != nil {
			h.Logger.Errorf("Save container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, AppendContainerToShipmentResponse{
			container,
		})
	}
}

func (h *ShipmentHandler) Close() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		form := CloseShipmentForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		shipmentID := cast.ToInt64(form.ID)
		if shipmentID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing shipment id !")
			return
		}

		h.Logger.Info("closeShipment form.UpsAccoun: ", form.UpsAccount)
		switch form.UpsAccount {
		case constant.DefaultAccountUps:
			h.UPS = ups.NewUPS()
			break
		case constant.OptionAccountUps2:
			h.UPS = ups.NewUPS2()
			break
		default:
			c.JSON(http.StatusBadRequest, "Chưa chọn tài khoản UPS")
			return
		}

		opts := sqlmanager.ShipmentQueryOptions{
			ID:                     shipmentID,
			LoadWarehouseContainer: true,
			LoadContainer:          true,
		}

		shipment, err := h.ShipmentManager.GetShipment(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if shipment.Status == constant.ShipmentClosed {
			c.JSON(http.StatusOK, CloseShipmentResponse{true})
			return
		}

		if shipment.Status != constant.ShipmentWaitingClose {
			c.JSON(http.StatusBadRequest, "Lô hàng có trạng thái không hợp lệ")
			return
		}

		h.Logger.Info("closeShipment shipment.Containers: ", len(shipment.Containers))
		if len(shipment.Containers) == 0 {
			c.JSON(http.StatusForbidden, "Lô hàng rỗng không thể đóng !")
			return
		}

		rKey := "list_closing_shipment"
		isClosing, err := h.Redis.SIsMember(c, rKey, shipment.ID).Result()
		h.Logger.Info("closeShipment isClosing: ", isClosing)
		if err != nil {
			h.Logger.Errorf("Get redis error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if isClosing {
			c.JSON(http.StatusBadRequest, "Lô hàng đang xử lý, thử lại sau !")
			return
		}

		err = h.Redis.SAdd(c, rKey, shipment.ID).Err()
		if err != nil {
			h.Logger.Errorf("Save redis error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		defer h.Redis.SRem(c, rKey, shipment.ID)

		var packageIdsInShipment []int64

		for _, container := range shipment.Containers {
			if container.Weight == 0 {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Kiện hàng #%v có khối lượng đang rỗng !", container.ID))
				return
			}

			for _, item := range container.ContainerItems {
				packageIdsInShipment = append(packageIdsInShipment, item.PackageID)
			}
		}
		h.Logger.Info("closeShipment packageIdsInShipment: ", len(packageIdsInShipment))
		h.Logger.Info("closeShipment shipment.IsFba: ", shipment.IsFba)
		var wareHouse *entity.Warehouse = nil
		if shipment.IsFba {
			pkg, err := h.ShipmentManager.GetShipmentFirstPackage(shipmentID)
			if err != nil {
				h.Logger.Errorf("get package %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			wareHouse = &entity.Warehouse{
				Name:    pkg.Recipient,
				Phone:   pkg.PhoneNumber,
				Company: pkg.Company,
				Address: pkg.Address1,
				City:    pkg.City,
				Zipcode: pkg.Zipcode,
				State:   pkg.StateCode,
				Country: pkg.CountryCode,
			}

			if wareHouse.Company == "" {
				wareHouse.Company = wareHouse.Name
			}

		} else {
			wareHouse, err = h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
				ID:   utils.Int64Value(shipment.HubID),
				Type: constant.WareHouseTypeInternational,
			})
			if err != nil {
				h.Logger.Errorf("get warehouse %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

		}
		h.Logger.Info("closeShipment wareHouse: ", wareHouse.ID)
		now := time.Now()
		uContainers := make([]entity.Container, 0)
		fContainers := make([]entity.Container, 0)
		for _, container := range shipment.Containers {
			if container.Type == constant.ContainerTypeUps {
				uContainers = append(uContainers, container)
			}
			if container.Type == constant.ContainerTypeFedEx {
				fContainers = append(fContainers, container)
			}
		}

		h.Logger.Info("closeShipment uContainers: ", len(uContainers))
		h.Logger.Info("closeShipment fContainers: ", len(fContainers))
		if form.ShipmentValue <= 0 && len(fContainers) > 0 {
			c.JSON(http.StatusBadRequest, "Giá trị lô hàng không hợp lệ")
			return
		}

		if wareHouse.Country != "AU" {
			if len(uContainers) > 0 {
				switch len(uContainers) {
				case 1:
					response, err := h.UPS.CreateLabelOnePackage(uContainers, wareHouse)
					if err != nil {
						h.Logger.Error("Error when send request to ups:", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					if response.ShipmentResponse.Response.ResponseStatus.Code != constant.ResponseCodeUPSSuccess {
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					packageResults := response.ShipmentResponse.ShipmentResults.PackageResults

					labelImg := packageResults.ShippingLabel.GraphicImage
					if labelImg == "" {
						c.JSON(http.StatusBadRequest, "Label Image is invalid")
						return
					}

					decode, err := base64.StdEncoding.DecodeString(labelImg)
					if err != nil {
						h.Logger.Error("close shipping container:%v", err)
						c.JSON(http.StatusBadRequest, "Label Image is invalid")
						return
					}

					reader := bytes.NewReader(decode)

					fileName := file.RandomFilename(".gif")
					bucket := viper.GetString("bucket.labels")

					err = h.StorageS3.UploadFile(reader, fileName, bucket, constant.ImageContentTypeGIF)
					if err != nil {
						h.Logger.Error("close shipping container:%v", err)
						c.JSON(http.StatusBadRequest, "Save label failed")
						return
					}
					uContainers[0].TrackingNumber = packageResults.TrackingNumber
					uContainers[0].LabelUrl = fileName
					shipment.Price = cast.ToFloat64(response.ShipmentResponse.ShipmentResults.ShipmentCharges.TotalCharges.MonetaryValue)
					break
				default:
					response, err := h.UPS.CreateLabel(uContainers, wareHouse)

					if err != nil {
						h.Logger.Error("Error when send request to ups: %v", err)
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					if response.ShipmentResponse.Response.ResponseStatus.Code != constant.ResponseCodeUPSSuccess {
						c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
						return
					}

					packageResults := response.ShipmentResponse.ShipmentResults.PackageResults

					for index, result := range packageResults {
						labelImg := result.ShippingLabel.GraphicImage
						if labelImg == "" {
							c.JSON(http.StatusBadRequest, "Label Image is invalid")
							return
						}

						decode, err := base64.StdEncoding.DecodeString(labelImg)
						if err != nil {
							h.Logger.Error("close shipping container:%v", err)
							c.JSON(http.StatusBadRequest, "Label Image is invalid")
							return
						}

						reader := bytes.NewReader(decode)

						fileName := file.RandomFilename(".gif")
						bucket := viper.GetString("bucket.labels")

						err = h.StorageS3.UploadFile(reader, fileName, bucket, constant.ImageContentTypeGIF)
						if err != nil {
							h.Logger.Error("close shipping container:%v", err)
							c.JSON(http.StatusBadRequest, "Save label failed")
							return
						}

						uContainers[index].TrackingNumber = result.TrackingNumber
						uContainers[index].LabelUrl = fileName
					}

					shipment.Price = cast.ToFloat64(response.ShipmentResponse.ShipmentResults.ShipmentCharges.TotalCharges.MonetaryValue)
					break
				}
			}
		}

		if len(fContainers) > 0 {
			sWarehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
				ID: shipment.WarehouseID,
			})
			if err != nil {
				h.Logger.Errorf("get shipment warehouse %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			switch form.FedexAccount {
			case constant.DefaultAccountFedex:
				h.FEDEX = fedex.InitFedEx()
				break
			case constant.OptionAccountFedex:
				h.FEDEX = fedex.InitFedEx2()
				break
			default:
				c.JSON(http.StatusBadRequest, "Chưa chọn tài khoản Fedex")
				return
			}

			response, err := h.FEDEX.CreateLabel(fContainers, wareHouse, sWarehouse, form.ShipmentValue)
			if err != nil {
				h.Logger.Error("Error when send request to fedex: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			pieceResponses := response.Output.TransactionShipments[0].PieceResponses
			for i, res := range pieceResponses {

				labelImg := res.PackageDocuments[0].EncodedLabel
				if labelImg == "" {
					c.JSON(http.StatusBadRequest, "Label Image is invalid")
					return
				}

				decode, err := base64.StdEncoding.DecodeString(labelImg)
				if err != nil {
					h.Logger.Error("close shipping container:%v", err)
					c.JSON(http.StatusBadRequest, "Label Image is invalid")
					return
				}

				reader := bytes.NewReader(decode)

				fileName := file.RandomFilename(".pdf")
				bucket := viper.GetString("bucket.labels")

				err = h.StorageS3.UploadFile(reader, fileName, bucket, "application/pdf")
				if err != nil {
					h.Logger.Error("close shipping container:%v", err)
					c.JSON(http.StatusBadRequest, "Save label failed")
					return
				}

				fContainers[i].TrackingNumber = res.TrackingNumber
				fContainers[i].LabelUrl = fileName

			}
		}

		shipment.Status = constant.ShipmentClosed

		shipment.UpdatedAt = now
		shipment.CloseAt = &now
		uContainers = append(uContainers, fContainers...)
		err = h.ShipmentManager.CloseShipment(shipment, uContainers, userID)
		if err != nil {
			h.Logger.Errorf("Save shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		h.Logger.Info("closeShipment wareHouse.ManifestActive: ", wareHouse.ManifestActive)
		if wareHouse.ManifestActive == constant.WareHouseManifestActive {

			// Đối soát tracking number IB Blue và tạo manifest
			_, err := h.manifest(shipment.ID, packageIdsInShipment, wareHouse)
			h.Logger.Info("closeShipment manifest: ", err)
			if err != nil {
				h.Logger.Errorf("manifests %v", err)
			}

			// for _, m := range manifests {
			// 	h.pushNotiSlack(shipmentID, fmt.Sprintf("Manifest number: %v - Manifest URL: %v", m.ManifestNumber, m.ManifestURL))
			// }
		}

		c.JSON(http.StatusOK, CloseShipmentResponse{true})
	}
}

func (h *ShipmentHandler) Zip() gin.HandlerFunc {
	return func(c *gin.Context) {
		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}

		options := sqlmanager.ShipmentQueryOptions{
			ID:            shipmentID,
			LoadContainer: true,
		}

		shipment, err := h.ShipmentManager.GetShipment(options)

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if len(shipment.Containers) == 0 {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		bucket := viper.GetString("bucket.labels")
		pdf := gofpdf.NewCustom(&gofpdf.InitType{
			UnitStr: "cm",
			Size:    gofpdf.SizeType{Wd: 10, Ht: 15},
		})

		defer pdf.Close()
		var res []string
		for _, container := range shipment.Containers {
			if container.LabelUrl == "" {
				continue
			}
			buf, err := h.StorageS3.ReadFile(container.LabelUrl, bucket)
			if err != nil {
				h.Logger.Errorf("Read file s3 error : %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			ext := file.GetExtensionFromString(container.LabelUrl)
			filePath := fmt.Sprintf("%s_%s%s", container.Code, container.TrackingNumber, ext)
			if ext[1:] == "pdf" {
				res = append(res, container.LabelUrl)
				continue
			}

			imageoption := gofpdf.ImageOptions{ImageType: ext[1:], ReadDpi: false}
			img, err := os.Create(filePath)
			if err != nil {
				h.Logger.Errorf("Create file error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			_, err = io.Copy(img, buf)
			if err != nil {
				h.Logger.Errorf("Coppy buffer file error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			pdf.AddPage()
			pdf.ImageOptions(filePath, 0, 0, 10, 15, false, imageoption, 0, "")

			img.Close()
			os.Remove(filePath)

			continue
		}

		outputFile := fmt.Sprintf("shipment_%d.pdf", shipment.ID)
		err = pdf.OutputFileAndClose(outputFile)

		if err != nil {
			h.Logger.Errorf("Save pdf error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		osfile, err := os.Open(outputFile)
		defer os.Remove(outputFile)
		defer osfile.Close()
		buffer, err := ioutil.ReadFile(outputFile)
		if err != nil {
			h.Logger.Errorf("read file error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		osfile.Read(buffer)
		err = h.StorageS3.UploadFile(bytes.NewBuffer(buffer), outputFile, bucket, "application/pdf")
		if err != nil {
			h.Logger.Error("Error UploadFile to %s S3 %v", outputFile, err)
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		res = append(res, outputFile)
		c.JSON(http.StatusOK, DownloadLabelShipmentReponse{res})
	}
}

func (h *ShipmentHandler) CustomerShipment() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		status := strings.TrimSpace(c.Request.URL.Query().Get("status"))

		opts := sqlmanager.CustomerShipmentOption{
			InStatus:    constant.MapIntGroupStatusCustomerPackage[status],
			Keyword:     strings.TrimSpace(c.Request.URL.Query().Get("keyword")),
			StartDate:   cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:     cast.ToString(c.Request.URL.Query().Get("end_date")),
			UserPreload: true,
			Limit:       limit,
			Offset:      offset,
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleSale || role == constant.UserRoleSupport {
			userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
			opts.SupportID = userID
		}

		shipments, err := h.CustomerShipmentManager.GetCustomerShipments(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list shiment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		ids := []int64{}
		for _, v := range shipments {
			ids = append(ids, v.ID)
		}

		mapPrices, err := h.CustomerShipmentManager.GetCustomerShipmentsPrice(ids)
		if err != nil {
			h.Logger.Errorf("Get price shipments error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for i := range shipments {
			id := shipments[i].ID
			shipments[i].Price = mapPrices[id]
		}

		c.JSON(http.StatusOK, GetListCustomerShipmentHandlerResponse{shipments})
	}
}

func (h *ShipmentHandler) Intransit() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		shipmentID := cast.ToInt64(c.Param("shipment_id"))
		if shipmentID < 1 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}

		options := sqlmanager.ShipmentQueryOptions{
			ID:            shipmentID,
			LoadContainer: true,
			LoadPackage:   true,
		}

		shipment, err := h.ShipmentManager.GetShipment(options)

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if shipment.Status != constant.ShipmentClosed {
			h.Logger.Errorf("Get shipment error : %v", err)
			c.JSON(http.StatusBadRequest, "Shipment status is invalid")
			return
		}

		isValid := true
		var pkgs []entity.Package
		for i, container := range shipment.Containers {
			if container.Status != constant.ContainerClosed {
				isValid = false
			}

			if container.TrackingNumber == "" {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Kiện hàng %s chưa có nhãn kiện", container.Code))
				return
			}

			for _, item := range container.ContainerItems {
				pkgs = append(pkgs, item.Package)
			}

			shipment.Containers[i].Status = constant.ContainerIntransit
		}

		if !isValid {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		shipment.Status = constant.ShipmentIntransit
		err = h.ShipmentManager.IntransitShipment(pkgs, shipment, userID)

		if err != nil {
			h.Logger.Errorf("Change shipment intransit error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, ChangeIntransitShipmentResponse{
			true,
		})
	}
}

func (h *ShipmentHandler) manifest(shipmentID int64, packageIdsInShipment []int64, hub *entity.Warehouse) ([]*entity.Manifest, error) {
	manifestInsert := make([]*entity.Manifest, 0)
	var trackingNumbers []string
	var carrier providers.Carrier
	var manifest *providers.ManifestResponse
	var message string
	carrierMap := make(map[string][]string)  // carrierCode -> trackingNumbers
	carrierUserMap := make(map[string]int64) // carrierCode -> userID
	pkg := make([]int64, 0)

	trackingOptions := sqlmanager.TrackingOption{
		PackageIDs: packageIdsInShipment,
		Status:     constant.TrackingStatusSuccess,
	}

	trackings, err := h.TrackingManager.GetTrackings(trackingOptions)
	if err != nil && err != gorm.ErrRecordNotFound {
		h.Logger.Errorf("Get trackings error %v", err)
		return nil, err
	}

	for _, tracking := range trackings {
		var trackingNumber string
		if hub.Country == "AU" {
			trackingNumber = tracking.ShipmentID
			trackingNumbers = append(trackingNumbers, tracking.ShipmentID)
		} else {
			trackingNumber = tracking.TrackingNumber
			trackingNumbers = append(trackingNumbers, tracking.TrackingNumber)
		}
		pkg = append(pkg, tracking.PackageID)

		if tracking.Carrier == nil || tracking.Package == nil {
			return nil, errors.New("missing carrier or package info")
		}

		carrierCode := tracking.Carrier.Code
		carrierMap[carrierCode] = append(carrierMap[carrierCode], trackingNumber)
		carrierUserMap[carrierCode] = tracking.Package.UserID
	}

	log.Println("closeShipment trackingNumbers: ", trackingNumbers)
	if len(carrierMap) < 1 {
		return manifestInsert, nil
	}

	if hub.Country == "AU" && len(trackingNumbers) > 1000 {
		chunk := 1000
		count := 1
		i := 0
	SplitData:
		data := make([]string, 0)
		dataPkg := make([]int64, 0)
		manifestInsert = make([]*entity.Manifest, 0)
		for i < len(trackingNumbers) {
			data = append(data, trackingNumbers[i])
			dataPkg = append(dataPkg, pkg[i])
			i++
			if i > chunk*count {
				count++
				break
			}
		}

		carrier = providers.NewCarrier(providers.CarrierTypeIBBlue, pkg[i])
		manifest, message, err = h.createManifest(carrier, shipmentID, data, hub)
		if message != "" {
			h.Logger.Errorf("manifests error: %v", message)
			if i < len(trackingNumbers) {
				goto SplitData
			} else {
				return nil, err
			}
		}
		if err != nil {
			h.Logger.Errorf("manifests error: %v", err)
			if i < len(trackingNumbers) {
				goto SplitData
			} else {
				return nil, err
			}
		}

		containerIDs, err := h.TrackingManager.GetContainerPackageInShipment(shipmentID, dataPkg)
		if err != nil {
			h.Logger.Errorf("get container error %v", err)
			return nil, err
		}

		for _, manifestItem := range manifest.Usps {
			path := ""
			if hub.Country != "AU" {
				path, err = h.storeManifest(manifestItem.Base64Manifest, manifestItem.ManifestNumber, shipmentID)
				if err != nil {
					h.Logger.Errorf("store manifest: %v", err)
					return nil, err
				}
			}

			for _, ctn := range containerIDs {
				manifestInsert = append(manifestInsert, &entity.Manifest{
					ContainerID:    utils.Int64(ctn),
					ManifestNumber: manifestItem.ManifestNumber,
					ManifestURL:    path,
				})
			}

			for _, id := range dataPkg {
				manifestInsert = append(manifestInsert, &entity.Manifest{
					PackageID:      utils.Int64(id),
					ManifestNumber: manifestItem.ManifestNumber,
					ManifestURL:    path,
				})
			}

			manifestInsert = append(manifestInsert, &entity.Manifest{
				ShipmentID:     utils.Int64(shipmentID),
				ManifestNumber: manifestItem.ManifestNumber,
				ManifestURL:    path,
			})

			err = h.TrackingManager.CreateManifest(manifestInsert)
			if err != nil {
				h.Logger.Errorf("create manifest error %v", err)
				return nil, err
			}
		}

		if i < len(trackingNumbers) {
			goto SplitData
		}
	} else {
		for carrierCode, trackingNumbers := range carrierMap {
			userID := carrierUserMap[carrierCode]
			carrier := providers.NewCarrier(carrierCode, userID)
			if carrier == nil {
				return nil, fmt.Errorf("failed to create carrier for code %s", carrierCode)
			}

			manifest, message, err := h.createManifest(carrier, shipmentID, trackingNumbers, hub)
			if err != nil {
				return nil, fmt.Errorf("create manifest error: %w", err)
			}

			if message != "" {
				return nil, errors.New(message)
			}

			containerIDs, err := h.TrackingManager.GetContainerInShipment(shipmentID)
			if err != nil {
				h.Logger.Errorf("get container error %v", err)
				return nil, err
			}

			for _, manifestItem := range manifest.Usps {
				manifestInsert := make([]*entity.Manifest, 0)
				path := ""
				if hub.Country != "AU" {
					path, err = h.storeManifest(manifestItem.Base64Manifest, manifestItem.ManifestNumber, shipmentID)
					if err != nil {
						h.Logger.Errorf("store manifest: %v", err)
						return nil, err
					}
				}

				for _, ctn := range containerIDs {
					manifestInsert = append(manifestInsert, &entity.Manifest{
						ContainerID:    utils.Int64(ctn),
						ManifestNumber: manifestItem.ManifestNumber,
						ManifestURL:    path,
					})
				}

				for _, id := range pkg {
					manifestInsert = append(manifestInsert, &entity.Manifest{
						PackageID:      utils.Int64(id),
						ManifestNumber: manifestItem.ManifestNumber,
						ManifestURL:    path,
					})
				}

				manifestInsert = append(manifestInsert, &entity.Manifest{
					ShipmentID:     utils.Int64(shipmentID),
					ManifestNumber: manifestItem.ManifestNumber,
					ManifestURL:    path,
				})
				err = h.TrackingManager.CreateManifestWithTx(manifestInsert)
				if err != nil {
					h.Logger.Errorf("create manifest error %v", err)
					return nil, err
				}
			}
		}
	}

	return manifestInsert, nil
}

func (h *ShipmentHandler) storeManifest(url, manifest_number string, shipmentID int64) (string, error) {
	bucket := viper.GetString("bucket.labels")
	today := time.Now().Format("2006-01-02")
	filepath := fmt.Sprintf("manifest/%s/%d/%s.png", today, shipmentID, manifest_number)

	if strings.HasPrefix(strings.ToLower(url), "http") {
		// Check for PDF file
		if strings.HasSuffix(strings.ToLower(url), ".pdf") {
			filepath = strings.Replace(filepath, ".png", ".pdf", 1)

			res, err := http.Get(url)
			if err != nil {
				return "", err
			}
			defer res.Body.Close()

			err = h.StorageS3.UploadFile(res.Body, filepath, bucket, "application/pdf")
			if err != nil {
				h.Logger.Error("upload PDF manifest failed: %v", err)
				return "", err
			}
			return filepath, nil
		}

		// Otherwise, treat as image
		res, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

		im, _, err := image.Decode(res.Body)
		if err != nil {
			h.Logger.Error("decode image label: %v", err)
			return "", err
		}

		src := imaging.Resize(im, 400, 0, imaging.Box)

		var buf bytes.Buffer
		err = imaging.Encode(&buf, src, imaging.PNG)
		if err != nil {
			h.Logger.Error("resize label: %v", err)
			return "", err
		}

		err = h.StorageS3.UploadFile(&buf, filepath, bucket, constant.ImageContentTypePNG)
		if err != nil {
			h.Logger.Error("upload image label: %v", err)
			return "", err
		}

		return filepath, nil
	}

	// Base64-encoded image case
	decode, err := base64.StdEncoding.DecodeString(url)
	if err != nil {
		h.Logger.Error("decode base64 label: %v", err)
		return "", err
	}

	reader := bytes.NewReader(decode)

	err = h.StorageS3.UploadFile(reader, filepath, bucket, constant.ImageContentTypePNG)
	if err != nil {
		h.Logger.Error("upload base64 label: %v", err)
		return "", err
	}

	return filepath, nil
}

func (h *ShipmentHandler) createManifest(carrier providers.Carrier, shipmentID int64, trackingNumbers []string, warehouse *entity.Warehouse) (*providers.ManifestResponse, string, error) {
	body := providers.ManifestRequest{
		ShipmentID:      shipmentID,
		TrackingNumbers: trackingNumbers,
		Line1:           warehouse.Address,
		City:            warehouse.City,
		State:           warehouse.State,
		Zip:             warehouse.Zipcode,
	}

	a, _ := json.Marshal(body)
	log.Println("closeShipment createManifest: ", string(a))

	res, errString, err := carrier.CreateManifest(body)
	if err != nil {
		return nil, "", err
	}

	if errString != "" {
		return nil, errString, nil
	}

	return res, "", nil
}

func checkEqualAddress(v1 *entity.Package, v2 *entity.Shipment) bool {
	if !strings.EqualFold(v1.Address1, v2.Address) {
		return false
	}

	if !strings.EqualFold(v1.City, v2.City) {
		return false
	}

	if !strings.EqualFold(v1.StateCode, v2.State) {
		return false
	}

	if !strings.EqualFold(v1.CountryCode, v2.Country) {
		return false
	}

	if !strings.EqualFold(v1.Zipcode, v2.Zipcode) {
		return false
	}

	return true
}
