package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/httputil"
	models_dto "tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/array"
	"tebexpressapi/pkg/utils/dbgorm"
	"tebexpressapi/pkg/utils/file"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
	"gorm.io/gorm"
)

type ContainerHandler struct {
	Logger *zap.SugaredLogger

	S3 storage.S3

	UserManager      *sqlmanager.UserManager
	ContainerManager *sqlmanager.ContainerManager
	WareHouseManager *sqlmanager.WareHouseManager
	PackageManager   *sqlmanager.PackageManager
	TrackingManager  *sqlmanager.TrackingManager
}

type CountListContainerResponse struct {
	Count       int64                      `json:"count"`
	CountStatus []dto.CountStatusContainer `json:"count_status"`
}

type GetListContainerResponse struct {
	Containers []ListContainerDTO `json:"containers"`
}

type ListContainerDTO struct {
	entity.Container
	CountItem     int     `json:"count_item"`
	PackageWeight float64 `json:"package_weight"`
}

type CreateContainerForm struct {
	WarehouseID    int64                  `json:"warehouse_id"`
	Type           constant.ContainerType `json:"type"`
	TrackingNumber string                 `json:"tracking_number"`
	FbaType        constant.FbaType       `json:"fba_type"`
}

type CreateContainerResponse struct {
	Container interface{} `json:"container"`
}

type GetDetailContainerResponse struct {
	Container      *entity.Container                `json:"container"`
	Packages       []models_dto.PackageContainerDTO `json:"packages"`
	ContainerCount int                              `json:"container_count"`
	CountItem      int                              `json:"count_item"`
}

type FetchContainerBoxResponse struct {
	Boxes []entity.ContainerBox `json:"boxes"`
}

type AppendPackageToContainerForm struct {
	Search         string `json:"search"`
	TrackingNumber string `json:"tracking_number"`
	ContainerID    int64  `json:"container_id"`
}

type AppendPackageToContainerResponse struct {
	Success bool `json:"success"`
}

type RemovePackageFromContainerResponse struct {
	Success bool `json:"success"`
}

type CancelPackageInContainerForm struct {
	ContainerID int64   `json:"container_id"`
	PackageIDs  []int64 `json:"package_ids"`
}

type CloseContainerRequest struct {
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	Length         float64 `json:"length"`
	Weight         float64 `json:"weight"`
	ActualWeight   float64 `json:"actual_weight"`
	TrackingNumber string  `json:"tracking_number"`
}

type CancelContainerResponse struct {
	Success bool `json:"success"`
}

type ManifestContainerResponse struct {
	ManifestUrl []string `json:"manifest_url"`
}

type CloseContainerResponse struct {
	Success bool `json:"success"`
}

type FetchHistoryContainerResponse struct {
	Logs []entity.ContainerDeliverLog `json:"logs"`
}

type UpdateContainerForm struct {
	TrackingNumber string `json:"tracking_number"`
}

type UpdateContainerResponse struct {
	Success bool `json:"success"`
}

type ReOpenContainerResponse struct {
	Success bool `json:"success"`
}

type ImportEventHandler struct {
	S3               storage.S3
	ContainerManager *sqlmanager.ContainerManager
}

type importEventError struct {
	Line     int      `json:"line"`
	Code     string   `json:"code"`
	Messages []string `json:"errors"`
}

type importEventResponse struct {
	Success bool `json:"success"`
}

type rowEvent struct {
	Line        int
	Code        string    `json:"code"`
	Location    string    `json:"location"`
	Date        string    `json:"date"`
	Time        string    `json:"time"`
	ShipTime    time.Time `json:"ship_time"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

var allowExtensions = map[string]bool{
	".csv":  true,
	".xlsx": true,
	".xlsm": true,
	".xls":  true,
}

type formEvent struct {
	Datetime    time.Time `json:"datetime"`
	Location    string    `json:"location"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
}

type createEventResponse struct {
	Success bool `json:"success"`
}

func NewContainerHandler(l *zap.SugaredLogger, s3 storage.S3, um *sqlmanager.UserManager, wh *sqlmanager.WareHouseManager, cm *sqlmanager.ContainerManager, pm *sqlmanager.PackageManager, tm *sqlmanager.TrackingManager) *ContainerHandler {
	return &ContainerHandler{
		Logger: l,

		S3: s3,

		UserManager:      um,
		ContainerManager: cm,
		WareHouseManager: wh,
		PackageManager:   pm,
		TrackingManager:  tm,
	}
}

func (h *ContainerHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		opts := sqlmanager.ContainerQueryOptions{
			Status:          cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:          cast.ToString(c.Request.URL.Query().Get("search")),
			HubID:           cast.ToInt64(c.Request.URL.Query().Get("warehouse")),
			IgnorePerload:   true,
			NotInShipment:   cast.ToBool(c.Request.URL.Query().Get("not_in_shipment")),
			Type:            cast.ToInt(c.Request.URL.Query().Get("type")),
			FbaType:         cast.ToInt(c.Request.URL.Query().Get("fba_type")),
			IsWarningWeight: cast.ToBool(c.Request.URL.Query().Get("is_warning")),
			WarehouseID:     cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
		}

		if strings.Contains(opts.Search, "C") {
			opts.Search = strings.ReplaceAll(opts.Search, "C", "")
		}

		if user.Role == constant.UserRoleWarehouse {
			if opts.WarehouseID > 0 && opts.WarehouseID != user.WarehouseID {
				c.JSON(http.StatusOK, CountListContainerResponse{0, make([]dto.CountStatusContainer, 0)})
			}
			opts.WarehouseID = user.WarehouseID
		}

		if opts.FbaType > 0 {
			opts.HubID = 0
		}

		if opts.FbaType == 0 && opts.HubID > 0 {
			opts.FbaType = -1
		}

		count, err := h.ContainerManager.CountContainers(opts)
		if err != nil {
			h.Logger.Errorf("Count list container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		countStatus, err := h.ContainerManager.CountAllStatusContainers(opts)
		if err != nil {
			h.Logger.Errorf("Count status container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListContainerResponse{count, countStatus})
	}
}

func (h *ContainerHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.ContainerQueryOptions{
			Status:          cast.ToInt(c.Request.URL.Query().Get("status")),
			Search:          cast.ToString(c.Request.URL.Query().Get("search")),
			HubID:           cast.ToInt64(c.Request.URL.Query().Get("warehouse")),
			NotInShipment:   cast.ToBool(c.Request.URL.Query().Get("not_in_shipment")),
			Type:            cast.ToInt(c.Request.URL.Query().Get("type")),
			FbaType:         cast.ToInt(c.Request.URL.Query().Get("fba_type")),
			IsWarningWeight: cast.ToBool(c.Request.URL.Query().Get("is_warning")),
			WarehouseID:     cast.ToInt64(c.Request.URL.Query().Get("warehouse_id")),
			Limit:           limit,
			Offset:          offset,
		}

		if user.Role == constant.UserRoleWarehouse {
			if opts.WarehouseID > 0 && opts.WarehouseID != user.WarehouseID {
				c.JSON(http.StatusOK, GetListContainerResponse{make([]ListContainerDTO, 0)})
			}
			opts.WarehouseID = user.WarehouseID
		}

		if opts.FbaType > 0 {
			opts.HubID = 0
		}

		if opts.FbaType == 0 && opts.HubID > 0 {
			opts.FbaType = -1
		}

		containers, err := h.ContainerManager.GetContainers(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list containers error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		listContainers := make([]ListContainerDTO, 0)

		if len(containers) > 0 {
			for _, container := range containers {
				packageWeight := float64(0)
				items := container.ContainerItems
				for _, item := range items {
					packageWeight += item.Package.ActualWeight
				}
				countItem := len(container.ContainerItems)
				container.ContainerItems = nil
				listContainers = append(listContainers, ListContainerDTO{container, countItem, packageWeight})
			}
		}

		c.JSON(http.StatusOK, GetListContainerResponse{listContainers})
	}
}

func (h *ContainerHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		newForm := CreateContainerForm{}
		if err := c.ShouldBindJSON(&newForm); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if newForm.WarehouseID < 1 && newForm.FbaType == 0 {
			c.JSON(http.StatusBadRequest, "Chưa chọn kho !")
			return
		}

		if newForm.Type < 1 {
			c.JSON(http.StatusBadRequest, "Chưa chọn loại kiện hàng!")
			return
		}

		if newForm.Type != constant.ContainerTypeUps && newForm.Type != constant.ContainerTypeManual && newForm.Type != constant.ContainerTypeFedEx {
			c.JSON(http.StatusBadRequest, "Loại kiện hàng không hợp lệ!")
			return
		}

		var wareHouse *entity.Warehouse
		var container *entity.Container
		if newForm.FbaType == 0 {
			wareHouse, err = h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
				ID: newForm.WarehouseID,
			})

			if err != nil {
				h.Logger.Errorf("Get warehouse error: %s", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			container = &entity.Container{
				HubID:          newForm.WarehouseID,
				Type:           newForm.Type,
				TrackingNumber: string_util.StripTags(newForm.TrackingNumber),
				Status:         constant.ContainerWaitingClose,
			}
		} else {
			container = &entity.Container{
				FbaType:        newForm.FbaType,
				Status:         constant.ContainerWaitingClose,
				Type:           newForm.Type,
				TrackingNumber: string_util.StripTags(newForm.TrackingNumber),
			}
		}
		if user.Role == constant.UserRoleWarehouse {
			container.WarehouseID = user.WarehouseID
		} else {
			container.WarehouseID = viper.GetInt64("warehouse_vn_default")
		}
		container.WarehouseID = viper.GetInt64("warehouse_vn_default")
		tx, err := h.ContainerManager.CreateContainer(container)
		if err != nil {
			h.Logger.Errorf("Create container: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var Code string
		if newForm.FbaType == 0 {
			Code = fmt.Sprintf("%s%d%09d%s", wareHouse.State, constant.ContainerCodePrefix, container.ID, wareHouse.Country)
		} else {
			Code = fmt.Sprintf("%s%d%09d", "FBA", constant.ContainerCodePrefix, container.ID)
		}

		// filePath, err := h.genBarcode(Code)
		// if err != nil {
		// 	tx.Rollback()
		// 	h.Logger.Errorf("Gen barcode container error: %s", err)
		// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		// 	return
		// }

		// container.Barcode = filePath
		container.Code = Code
		err = h.ContainerManager.SaveContainerWithTransaction(tx, container)
		if err != nil {
			h.Logger.Errorf("Create container: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateContainerResponse{container})
	}
}

func (h *ContainerHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		containerCode := cast.ToString(c.Param("container_code"))
		isFail := cast.ToBool(c.Request.URL.Query().Get("isFail"))
		offset, limit := httputil.GetRequestPaginate(c.Request)
		if len(containerCode) == 0 {
			c.JSON(http.StatusBadRequest, "Shipment ID is invalid !")
			return
		}

		var keywordSearch = cast.ToString(c.Request.URL.Query().Get("search"))
		var packagesInContainer []entity.Package
		var pkgs []models_dto.PackageContainerDTO
		var packageIDs []int64

		options := sqlmanager.ContainerQueryOptions{
			Code:          containerCode,
			OffsetPackage: offset,
			LimitPackage:  limit,
			LoadPackage:   true,
		}

		if user.Role == constant.UserRoleWarehouse {
			options.WarehouseID = user.WarehouseID
		}

		container, err := h.ContainerManager.GetContainer(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		containerItems := make([]*entity.ContainerItem, 0)
		var countItem int64
		if isFail {
			containerItems, err = h.ContainerManager.GetAllItemFailContainer(container.ID)
		} else {
			containerItems, err = h.ContainerManager.GetAllItemContainer(container.ID)
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("GetContainerItems error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if isFail {
			countItem, err = h.ContainerManager.CountContainerFailItems(container.ID)
		} else {
			countItem, err = h.ContainerManager.CountContainerAllItems(container.ID)
		}

		if err != nil {
			h.Logger.Errorf("Count container error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		count, err := h.ContainerManager.CountContainerItems(container.ID)
		if err != nil {
			h.Logger.Errorf("Count container error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapStatus := make(map[int64]*entity.ContainerItem)
		for _, item := range containerItems {
			packageIDs = append(packageIDs, item.PackageID)
			mapStatus[item.PackageID] = item
		}

		opts := sqlmanager.PackageQueryOption{
			Code:   keywordSearch,
			IDs:    packageIDs,
			Limit:  limit,
			Offset: offset,
		}

		if len(containerItems) <= 0 {
			goto breakPackagesInContainer
		}

		packagesInContainer, err = h.PackageManager.GetPackagesContainer(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Packages error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for _, pkg := range packagesInContainer {
			pkgs = append(pkgs, models_dto.PackageContainerDTO{
				Package:     pkg,
				ItemStatus:  mapStatus[pkg.ID].Status,
				Description: mapStatus[pkg.ID].Description,
			})
		}

	breakPackagesInContainer:
		c.JSON(http.StatusOK, GetDetailContainerResponse{
			container,
			pkgs,
			cast.ToInt(count),
			cast.ToInt(countItem),
		})
	}
}

func (h *ContainerHandler) Append() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := AppendPackageToContainerForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if (len(form.Search) == 0 && len(form.TrackingNumber) == 0) || form.ContainerID <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		var packageResult *entity.Package
		var err error

		if form.Search != "" || form.TrackingNumber != "" {
			searchValue := form.Search
			if searchValue == "" {
				searchValue = form.TrackingNumber
			}

			packageResult, err = h.PackageManager.GetPackageByCodeOrTracking(searchValue)

			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, constant.MessageNotFound)
				return
			}

			if err != nil {
				h.Logger.Errorf("Get package by code or tracking error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		optionsContainer := sqlmanager.ContainerQueryOptions{
			ID: form.ContainerID,
		}

		container, err := h.ContainerManager.GetContainer(optionsContainer)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		isFail := false
		description := ""
		wh, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
			ID: container.HubID,
		})

		if err != nil {
			h.Logger.Errorf("Get warehouse error: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		//isAuProcess := wh.Country == "AU" && packageResult.CountryCode == "AU" && packageResult.Tracking == nil
		//isPassAu := true
		if wh.Country == "AU" {
			countItems, err := h.ContainerManager.CountContainerItems(container.ID)
			if err != nil {
				h.Logger.Errorf("Log count container:, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			h.Logger.Info("countItems: ", countItems)
			if countItems >= 333 {
				//isPassAu = false
				isFail = true
				description = "Kiện hàng đi au không thể chứa quá 333 đơn hàng"
			}
		}

		h.Logger.Info("aac: ", packageResult.WarehouseID > 0 && packageResult.WarehouseID != container.WarehouseID && !(container.FbaType > 0) && packageResult.Service.Code != constant.ServiceFBACode && packageResult.Service.Code != constant.ServiceFastFBACode)
		if packageResult.WarehouseID > 0 && packageResult.WarehouseID != container.WarehouseID && !(container.FbaType > 0) && packageResult.Service.Code != constant.ServiceFBACode && packageResult.Service.Code != constant.ServiceFastFBACode {
			isFail = true
			wareHouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: packageResult.WarehouseID})
			if err != nil {
				h.Logger.Errorf("Get warehouse error:, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			description = fmt.Sprintf("Đơn hàng đang nằm trong kho %v", wareHouse.Name)
		}

		if container.Status != constant.ContainerWaitingClose {
			isFail = true
			description = "Kiện hàng có trạng thái không hợp lệ"
		}

		if packageResult.Tracking == nil {
			if packageResult.Service.Code != constant.ServiceFBACode && packageResult.Service.Code != constant.ServiceFastFBACode {
				isFail = true
				description = "Đơn hàng chưa có tracking number"
			}
		} else if utils.Int64Value(packageResult.Tracking.HubID) != container.HubID && !(container.FbaType > 0) &&
			!(packageResult.CustomTiktokBarcode != nil && *packageResult.CustomTiktokBarcode != "") {
			// isFail = true
			description = "Đơn hàng không cùng kho với kiện hàng"
		}

		if packageResult.FbaContainerType > 0 && packageResult.FbaContainerType != container.Type {
			isFail = true
			description = "Đơn hàng không cùng dịch vụ ship (UPS/Fedex)"
		}

		if (container.FbaType > 0) && !((packageResult.Service.Code == constant.ServiceFastFBACode && container.FbaType == constant.FbaTypeFast) || (packageResult.Service.Code == constant.ServiceFBACode && container.FbaType == constant.FbaTypeStandard)) {
			isFail = true
			description = "Đơn hàng FBA không cùng tốc độ ship (Fast/Standard)"
		}

		h.Logger.Info("aac Status: ", packageResult.Status, constant.PackageStatusWareHouseLabeled)
		if packageResult.Status != constant.PackageStatusWareHouseLabeled && packageResult.Status != constant.PackageStatusPicked {
			wL := array.SliceStringToSliceInt(strings.Split(viper.GetString("white_list.warehouse"), ","))
			isSkip := false
			if len(wL) > 0 {
				for _, id := range wL {
					if id == packageResult.UserID && packageResult.Status == constant.PackageStatusPendingPickup {
						isSkip = true
					}
				}
			}
			if !isSkip {
				isFail = true
				description = "Đơn hàng có trạng thái không hợp lệ"
			}
		}

		if ((container.FbaType > 0) && packageResult.Service.Code != constant.ServiceFBACode && packageResult.Service.Code != constant.ServiceFastFBACode) || (!(container.FbaType > 0) && (packageResult.Service.Code == constant.ServiceFBACode || packageResult.Service.Code == constant.ServiceFastFBACode)) {
			isFail = true
			description = "Đơn hàng không cùng service kiện"
		}

		itemInContainer, err := h.ContainerManager.GetContainerByItem(packageResult.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Container By Item error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if itemInContainer != nil && itemInContainer.ContainerID == container.ID {
			c.JSON(http.StatusOK, AppendPackageToContainerResponse{true})
			return
		}

		if itemInContainer != nil && itemInContainer.ContainerID != container.ID && itemInContainer.ID != 0 {
			isFail = true
			description = fmt.Sprintf("Đơn hàng đã được thêm vào kiện %v trước đó", itemInContainer.ContainerID)
		}
		// if isAuProcess {
		// 	isFail = false
		// }

		h.Logger.Info("isFail: ", isFail)
		if isFail {
			err = h.ContainerManager.SaveFailPackageContainer(container, packageResult, description)

			if err != nil {
				h.Logger.Errorf("SaveContainerAndPackage error:, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		} else {
			var dlogs []entity.PackageDeliverLog

			if packageResult.Status == constant.PackageStatusPendingPickup {
				userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
				wareHouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{ID: packageResult.WarehouseID})
				if err != nil {
					h.Logger.Errorf("Get warehouse error:, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				dlogs = append(dlogs, entity.PackageDeliverLog{
					PackageID: packageResult.ID,
					Location:  fmt.Sprintf("%s, %s", wareHouse.City, wareHouse.Country),
					Status:    constant.DeliverLogTebexpressPicked,
					Type:      constant.PackageDeliverLogTypeInWareHouse,
					UserID:    &userID,
					Model: dbgorm.Model{
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				})
			}

			err = h.ContainerManager.SaveContainerAndPackage(container, packageResult, dlogs)
			if err != nil {
				h.Logger.Errorf("SaveContainerAndPackage error:, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		c.JSON(http.StatusOK, AppendPackageToContainerResponse{
			true,
		})
	}
}

func (h *ContainerHandler) Box() gin.HandlerFunc {
	return func(c *gin.Context) {
		boxes, err := h.ContainerManager.GetContainerBoxes(sqlmanager.ContainerBoxQueryOptions{})
		if err != nil {
			h.Logger.Errorf("Fetch shipping boxes: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, FetchContainerBoxResponse{boxes})
	}
}

func (h *ContainerHandler) Remove() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := CancelPackageInContainerForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}
		if form.ContainerID <= 0 || len(form.PackageIDs) <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
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

		if container.Status == constant.ContainerClosed {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		optionsGetPackage := sqlmanager.PackageQueryOption{
			IDs: form.PackageIDs,
		}

		packages, err := h.PackageManager.GetPackages(optionsGetPackage)
		if err == gorm.ErrRecordNotFound || len(packages) <= 0 {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		containerItems, err := h.ContainerManager.GetAllItemContainer(container.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("GetContainerItems error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		mapItem := make(map[int64]*entity.ContainerItem, 0)
		for _, pkg := range packages {
			var exists = false
			for _, item := range containerItems {
				mapItem[item.PackageID] = item
				if pkg.ID == item.PackageID && (item.Status == constant.ContainerItemActive || item.Status == constant.ContainerItemFail) {
					exists = true
				}
			}
			if !exists {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng %v không tồn tại trong kiện", pkg.PackageCode.Code))
				return
			}
		}

		if err != nil {
			h.Logger.Errorf("Get packages error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		err = h.ContainerManager.CancelPackagesInContainer(container, packages, mapItem)
		if err != nil {
			h.Logger.Errorf("CancelPackagesInContainer error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, RemovePackageFromContainerResponse{true})
	}
}

func (h *ContainerHandler) Manifest() gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing container ID")
			return
		}

		container, err := h.ContainerManager.GetContainer(sqlmanager.ContainerQueryOptions{ID: containerID})
		if err != nil {
			h.Logger.Errorf("get container error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if container == nil {
			c.JSON(http.StatusNotFound, constant.APIResponseMessageValidateInput)
			return
		}

		containerItems, err := h.ContainerManager.GetAllItemContainer(container.ID)
		if err != nil {
			h.Logger.Errorf("get container items error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var packageIDs []int64
		for _, item := range containerItems {
			packageIDs = append(packageIDs, item.PackageID)
		}

		warehouseID := utils.Int64Value(&container.HubID)
		wareHouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
			ID:   warehouseID,
			Type: constant.WareHouseTypeInternational,
		})
		if err != nil {
			h.Logger.Errorf("get warehouse error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if wareHouse.ManifestActive != constant.WareHouseManifestActive {
			c.JSON(http.StatusBadRequest, "This warehouse can't create manifest")
			return
		}

		path, err := h.manifestByContainer(container.ID, packageIDs, wareHouse)
		if err != nil {
			h.Logger.Errorf("create manifest by container error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, ManifestContainerResponse{
			ManifestUrl: path,
		})
	}
}

func (h *ContainerHandler) GetManifestByContainerID() gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Invalid container ID")
			return
		}

		manifests, err := h.TrackingManager.GetManifestsByContainerID(containerID)
		if err != nil {
			h.Logger.Errorf("get manifests by containerID error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(manifests) == 0 {
			c.JSON(http.StatusNotFound, "No manifest found for this container")
			return
		}

		var urls []string
		for _, m := range manifests {
			if m.ManifestURL != "" {
				urls = append(urls, m.ManifestURL)
			}
		}

		c.JSON(http.StatusOK, ManifestContainerResponse{
			ManifestUrl: urls,
		})
	}
}

func (h *ContainerHandler) Close() gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing shipment id !")
			return
		}

		form := CloseContainerRequest{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}
		length, height, width := ParseVolumes(form.Length, form.Height, form.Width)

		if length > constant.ContainerMaxLength {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Cạnh dài nhất của kiện hàng vượt quá giới hạn chiều dài tối đa là %v cm", constant.ContainerMaxLength))
			return
		}

		if length+2*(width+height) > constant.ContainerMaxSize {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Kích thước: Dài + (Cao + Rộng)*2 tối đa là %v cm", constant.ContainerMaxSize))
			return
		}

		if form.Weight <= 0 {
			c.JSON(http.StatusBadRequest, "Trọng lượng phải lớn hơn 0")
			return
		}

		if form.Weight > constant.ContainerMaxWeight {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Trọng lượng tối đa là %v kg", constant.ContainerMaxWeight))
			return
		}

		opts := sqlmanager.ContainerQueryOptions{
			ID:          containerID,
			LoadPackage: true,
		}

		container, err := h.ContainerManager.GetContainer(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if container.Status == constant.ContainerCancelled {
			c.JSON(http.StatusBadRequest, "Kiện hàng đã hủy trước đó")
			return
		}

		if container.Status == constant.ContainerClosed {
			c.JSON(http.StatusBadRequest, "Kiện hàng đã đóng trước đó")
			return
		}

		containerItems, err := h.ContainerManager.GetAllItemContainer(container.ID)

		var sItems []*entity.ContainerItem
		for _, item := range containerItems {
			if item.Status == constant.ContainerItemFail {
				c.JSON(http.StatusBadRequest, "Kiện có đơn hàng quét thất bại")
				return
			}
			sItems = append(sItems, item)
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("GetContainerItems error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(sItems) <= 0 {
			c.JSON(http.StatusBadRequest, "Không thể đóng kiện hàng rỗng")
			return
		}

		container.Status = constant.ContainerClosed
		container.UpdatedAt = time.Now()
		container.Weight = form.Weight
		container.ActualWeight = form.ActualWeight
		container.Width = width
		container.Length = length
		container.Height = height
		container.TrackingNumber = form.TrackingNumber

		err = h.ContainerManager.CloseContainer(container)
		if err != nil {
			h.Logger.Errorf("Save shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CloseContainerResponse{true})
	}
}

func (h *ContainerHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing container id !")
			return
		}

		opts := sqlmanager.ContainerQueryOptions{
			ID:          containerID,
			LoadPackage: true,
		}

		container, err := h.ContainerManager.GetContainer(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get shipment error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if container.Status == constant.ContainerCancelled {
			c.JSON(http.StatusBadRequest, "Kiện hàng đã hủy trước đó")
			return
		}

		containerItems, err := h.ContainerManager.GetContainerItems(container.ID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("GetContainerItems error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(container.Packages) > 0 || len(containerItems) > 0 {
			c.JSON(http.StatusForbidden, "Không thể hủy kiện đang chứa đơn hàng")
			return
		}

		container.Status = constant.ContainerCancelled
		container.UpdatedAt = time.Now()

		err = h.ContainerManager.SaveContainer(container)
		if err != nil {
			h.Logger.Errorf("Save container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CancelContainerResponse{true})
	}
}

func (h *ContainerHandler) History() gin.HandlerFunc {
	return func(c *gin.Context) {
		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing container id !")
			return
		}
		logs, err := h.ContainerManager.GetContainerLogs(sqlmanager.ContainerQueryOptions{ID: containerID})
		if err != nil {
			h.Logger.Errorf("Get logs container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, FetchHistoryContainerResponse{logs})
	}
}

func (h *ContainerHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := UpdateContainerForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		var regex = regexp.MustCompile("^[A-Z0-9]*$")
		if form.TrackingNumber != "" && !regex.MatchString(form.TrackingNumber) {
			c.JSON(http.StatusBadRequest, "Tracking ups sai định dạng")
			return
		}

		containerID := cast.ToInt64(c.Param("container_id"))
		if containerID <= 0 {
			c.JSON(http.StatusBadRequest, "Missing container id !")
			return
		}
		opts := sqlmanager.ContainerQueryOptions{
			ID: containerID,
		}

		container, err := h.ContainerManager.GetContainer(opts)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if container.Type != constant.ContainerTypeManual {
			c.JSON(http.StatusBadRequest, constant.MessagePermissionDenied)
			return
		}

		validStatus := []int64{constant.ContainerClosed}
		if !utils.ContainsNumber(validStatus, int64(container.Status)) {
			c.JSON(http.StatusBadRequest, "Trạng thái container không hợp lệ")
			return
		}

		container.TrackingNumber = form.TrackingNumber
		container.UpdatedAt = time.Now()
		err = h.ContainerManager.UpdateContainer(container)

		if err != nil {
			h.Logger.Errorf("Update container error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdateContainerResponse{true})
	}
}

func (h *ContainerHandler) Open() gin.HandlerFunc {
	return func(c *gin.Context) {
		ContainerID := cast.ToInt64(c.Param("container_id"))
		if ContainerID < 1 {
			c.JSON(http.StatusBadRequest, "Container ID is invalid !")
			return
		}

		options := sqlmanager.ContainerQueryOptions{
			ID: ContainerID,
		}

		container, err := h.ContainerManager.GetContainer(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Container error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if container.Status != constant.ContainerClosed {
			c.JSON(http.StatusBadRequest, "Kiện hàng có trạng thái không hợp lệ !")
			return
		}
		if container.ShipmentID != nil {
			c.JSON(http.StatusBadRequest, "Kiện hàng đã cho vào lô !")
			return
		}
		container.Width = 0
		container.Height = 0
		container.Length = 0
		container.Weight = 0
		container.ActualWeight = 0
		container.Status = constant.ContainerWaitingClose
		container.TrackingNumber = ""

		err = h.ContainerManager.SaveContainer(container)
		if err != nil {
			h.Logger.Errorf("Update container error : %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, ReOpenContainerResponse{true})
	}
}

func (h *ContainerHandler) CreateEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		form := &formEvent{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Errorf("parse request body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Datetime.IsZero() {
			c.JSON(http.StatusBadRequest, "Thời gian là bắt buộc")
			return
		}

		form.Location = strings.TrimSpace(form.Location)
		if form.Location == "" {
			c.JSON(http.StatusBadRequest, "Địa diểm kiện hàng là bắt buộc")
			return
		}

		form.Code = strings.TrimSpace(form.Code)
		if form.Code == "" {
			c.JSON(http.StatusBadRequest, "Mã kiện hàng là bắt buộc")
			return
		}

		form.Description = strings.TrimSpace(form.Description)
		if form.Description == "" {
			c.JSON(http.StatusBadRequest, "Nội dung là bắt buộc")
			return
		}

		container, err := h.ContainerManager.GetContainer(sqlmanager.ContainerQueryOptions{Code: form.Code})
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, "Mã kiện hàng không hợp lệ")
			return
		}

		if err != nil {
			h.Logger.Errorf("db fetch container: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		log := &entity.ContainerDeliverLog{
			ContainerID: container.ID,
			Location:    form.Location,
			ShipTime:    form.Datetime,
			Description: form.Description,
			Status:      "I",
			Code:        "",
		}

		olds, err := h.ContainerManager.GetDeliverLogs(container.ID)
		if err != nil {
			h.Logger.Errorf("db fetch container logs: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for _, v := range olds {
			if log.Code == v.Code &&
				v.Status == log.Status &&
				v.ShipTime.Equal(log.ShipTime) &&
				v.Location == log.Location &&
				v.Description == log.Description {
				c.JSON(http.StatusBadRequest, "Event đã tồn tại")
				return
			}
		}

		if err := h.ContainerManager.CreateDeliverLog(log, false); err != nil {
			h.Logger.Errorf("get detail track ups: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, createEventResponse{true})
	}
}

func (h *ContainerHandler) ImportEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		ff, fh, err := c.Request.FormFile("file")
		if err != nil {
			h.Logger.Error("Can't access file: ", err)
			c.JSON(http.StatusBadRequest, "Can't access file.")
			return
		}
		defer ff.Close()

		ext := strings.ToLower(file.GetExtensionFromString(fh.Filename))
		mineType := fh.Header.Get("Content-Type")
		if !allowExtensions[ext] {
			h.Logger.Infof("File mine type: %v", mineType)
			c.JSON(http.StatusBadRequest, "File không đúng định dạng (chỉ chấp nhận csv or excel)")
			return
		}

		buf, err := io.ReadAll(ff)
		if err != nil {
			c.JSON(http.StatusBadRequest, "Error retrieving the file.")
			return
		}

		codes, items, importErrors, err := h.parse(bytes.NewBuffer(buf), ext)
		if err != nil {
			c.JSON(http.StatusBadRequest, "File lỗi không đọc được")
			return
		}

		if len(importErrors) > 0 {
			c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": constant.MessageValidateInput,
				"errors":  importErrors,
			})
			return
		}

		if len(codes) < 1 || len(items) < 1 {
			c.JSON(http.StatusBadRequest, "File không có dữ liệu")
			return
		}

		if len(items) > 200 {
			c.JSON(http.StatusBadRequest, "Bạn chỉ được import tối đa 200 dòng")
			return
		}

		containers, err := h.ContainerManager.GetContainers(sqlmanager.ContainerQueryOptions{Codes: codes, HasHistory: true})
		if err != nil {
			h.Logger.Errorf("Import tracking failed: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapExists := make(map[string]entity.Container)
		for _, v := range containers {
			mapExists[v.Code] = v
		}

		logs := []entity.ContainerDeliverLog{}
		now := time.Now()

	loop:
		for _, v := range items {
			if mapExists[v.Code].ID < 1 {
				importErrors = append(importErrors, importEventError{
					Line:     v.Line,
					Code:     v.Code,
					Messages: []string{fmt.Sprintf("Mã kiện hàng %v không hợp lệ", v.Code)},
				})

				continue
			}

			log := entity.ContainerDeliverLog{
				Model: dbgorm.Model{
					CreatedAt: now,
					UpdatedAt: now,
				},
				ContainerID: mapExists[v.Code].ID,
				Location:    v.Location,
				ShipTime:    v.ShipTime,
				Status:      "I",
				Description: v.Description,
			}

			olds := mapExists[v.Code].ContainerHistory
			for _, v := range olds {
				if v.Status == log.Status &&
					v.ShipTime.Equal(log.ShipTime) &&
					v.Location == log.Location &&
					v.Description == log.Description &&
					v.ContainerID == log.ContainerID {
					continue loop
				}
			}

			for _, v := range logs {
				if v.Status == log.Status &&
					v.ShipTime.Equal(log.ShipTime) &&
					v.Location == log.Location &&
					v.Description == log.Description &&
					v.ContainerID == log.ContainerID {
					continue loop
				}
			}

			logs = append(logs, log)
		}

		if len(importErrors) > 0 {
			c.JSON(http.StatusBadRequest, map[string]interface{}{
				"message": constant.MessageValidateInput,
				"errors":  importErrors,
			})
			return
		}

		if len(logs) > 0 {
			if err := h.ContainerManager.CreateDeliverLogs(logs, false, false); err != nil {
				h.Logger.Errorf("Import tracking failed: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		c.JSON(http.StatusOK, importEventResponse{true})
	}
}

func (h *ContainerHandler) parse(buf *bytes.Buffer, ext string) ([]string, []rowEvent, []importEventError, error) {
	if ext == ".csv" {
		return h.parseCsv(buf)
	}

	return h.parseExcel(buf)
}

func (h *ContainerHandler) parseCsv(buf *bytes.Buffer) ([]string, []rowEvent, []importEventError, error) {
	r2 := csv.NewReader(buf)

	columnCode := -1
	columnDate := -1
	columnTime := -1
	columnLocation := -1
	columnDescription := -1

	var codes = []string{} // order ids
	var items = []rowEvent{}
	var arrErrors = make([]importEventError, 0)

	// Iterate through the records
	var line int = 0
	for {
		line = line + 1
		record, err := r2.Read()

		if err == io.EOF {
			if line == 1 {
				arrErrors = append(arrErrors, importEventError{
					Line:     line,
					Messages: []string{"File is not empty required"},
				})
			}

			break
		}

		if err != nil {
			return codes, items, arrErrors, err
		}

		messages := make([]string, 0)

		// Skip header
		if line == 1 {
			for i, h := range record {
				hn := strings.ToLower(strings.TrimSpace(string_util.TrapBOM(h)))
				switch hn {
				case "code":
					columnCode = i
				case "date":
					columnDate = i
				case "time":
					columnTime = i
				case "location":
					columnLocation = i
				case "description":
					columnDescription = i
				}
			}

			if columnCode == -1 {
				messages = append(messages, "Missing colum CODE")
			}

			if columnDate == -1 {
				messages = append(messages, "Missing colum DATE")
			}

			if columnTime == -1 {
				messages = append(messages, "Missing colum TIME")
			}

			if columnLocation == -1 {
				messages = append(messages, "Missing colum LOCATION")
			}

			if columnDescription == -1 {
				messages = append(messages, "Missing colum DESCRIPTION")
			}

			if len(messages) > 0 {
				arrErrors = append(arrErrors, importEventError{
					Line:     0,
					Code:     "",
					Messages: messages,
				})

				return codes, items, arrErrors, nil
			}

			continue
		}

		event := rowEvent{
			Line:   line,
			Status: "I",
		}

		for i, v := range record {
			s := strings.TrimSpace(v)

			switch i {
			case columnCode:
				event.Code = s
			case columnDate:
				event.Date = s
			case columnTime:
				event.Time = s
			case columnLocation:
				event.Location = s
			case columnDescription:
				event.Description = s
			}
		}

		if len(event.Time) == 5 {
			event.Time = event.Time + ":00"
		}

		if len(event.Time) > 8 {
			event.Time = event.Time[:8]
		}

		messages = h.valid(event)

		dt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", event.Date, event.Time))
		if err != nil {
			dt, err = time.Parse("02/01/2006 15:04:05", fmt.Sprintf("%s %s", event.Date, event.Time))
			if err != nil {
				h.Logger.Errorf("parse time: %v", err)
				messages = append(messages, "Ngày giờ không hợp lệ (VD ngày: 2023-02-14 hoặc 14/02/2023, giờ: 15:04)")
			}
		}

		event.ShipTime = dt.Add(-7 * time.Hour)

		if len(messages) > 0 {
			arrErrors = append(arrErrors, importEventError{
				Line:     line,
				Code:     event.Code,
				Messages: messages,
			})
			continue
		}

		codes = append(codes, event.Code)
		items = append(items, event)
	}

	return codes, items, arrErrors, nil
}

func (h *ContainerHandler) parseExcel(buf *bytes.Buffer) ([]string, []rowEvent, []importEventError, error) {
	var codes = []string{}
	var items = []rowEvent{}
	var arrErrors = make([]importEventError, 0)

	file, err := excelize.OpenReader(buf)
	if err != nil {
		return codes, items, arrErrors, err
	}

	rows, err := file.GetRows("Template")
	if err != nil {
		return codes, items, arrErrors, err
	}

	if len(rows) < 2 {
		return codes, items, arrErrors, errors.New("file is invalid")
	}

	columnCode := -1
	columnDate := -1
	columnTime := -1
	columnLocation := -1
	columnDescription := -1

	headers := rows[0]
	for i, h := range headers {
		hn := strings.ToLower(strings.TrimSpace(string_util.TrapBOM(h)))
		switch hn {
		case "code":
			columnCode = i
		case "date":
			columnDate = i
		case "time":
			columnTime = i
		case "location":
			columnLocation = i
		case "description":
			columnDescription = i
		}
	}

	messages := []string{}
	if columnCode == -1 {
		messages = append(messages, "Missing colum CODE")
	}

	if columnDate == -1 {
		messages = append(messages, "Missing colum DATE")
	}

	if columnTime == -1 {
		messages = append(messages, "Missing colum TIME")
	}

	if columnLocation == -1 {
		messages = append(messages, "Missing colum LOCATION")
	}

	if columnDescription == -1 {
		messages = append(messages, "Missing colum DESCRIPTION")
	}

	if len(messages) > 0 {
		arrErrors = append(arrErrors, importEventError{
			Line:     0,
			Code:     "",
			Messages: messages,
		})

		return codes, items, arrErrors, nil
	}

	data := rows[1:]

	// Iterate through the records
	for n, row := range data {
		line := n + 2
		event := rowEvent{
			Line:   line,
			Status: "I",
		}

		for i, v := range row {
			s := strings.TrimSpace(v)

			switch i {
			case columnCode:
				event.Code = s
			case columnDate:
				event.Date = s
			case columnTime:
				event.Time = s
			case columnLocation:
				event.Location = s
			case columnDescription:
				event.Description = s
			}
		}

		if len(event.Time) == 5 {
			event.Time = event.Time + ":00"
		}

		if len(event.Time) > 8 {
			event.Time = event.Time[:8]
		}

		messages := h.valid(event)
		dt, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", event.Date, event.Time))
		if err != nil {
			dt, err = time.Parse("02/01/2006 15:04:05", fmt.Sprintf("%s %s", event.Date, event.Time))
			if err != nil {
				h.Logger.Errorf("parse time: %v", err)
				messages = append(messages, "Ngày giờ không hợp lệ (VD ngày: 2023-02-14 hoặc 14/02/2023, giờ: 15:04)")
			}
		}

		event.ShipTime = dt.Add(-7 * time.Hour)

		if len(messages) > 0 {
			arrErrors = append(arrErrors, importEventError{
				Line:     line,
				Code:     event.Code,
				Messages: messages,
			})
			continue
		}

		codes = append(codes, event.Code)
		items = append(items, event)
	}

	return codes, items, arrErrors, nil
}

func (h *ContainerHandler) valid(ev rowEvent) []string {
	messages := []string{}

	if ev.Code == "" {
		messages = append(messages, "Mã kiện hàng là bắt buộc")
	}

	if ev.Date == "" {
		messages = append(messages, "Ngày là bắt buộc")
	}

	if ev.Time == "" {
		messages = append(messages, "Giờ là bắt buộc")
	}

	if ev.Location == "" {
		messages = append(messages, "Địa chỉ là bắt buộc")
	}

	if len(ev.Location) > 100 {
		messages = append(messages, "Địa chỉ không quá 100 kí tự")
	}

	if ev.Description == "" {
		messages = append(messages, "Nội dung là bắt buộc")
	}

	if len(ev.Description) > 150 {
		messages = append(messages, "Nội dung không quá 150 kí tự")
	}

	return messages
}

func (h *ContainerHandler) genBarcode(text string) (string, error) {
	bc, err := code128.Encode(text)
	if err != nil {
		return "", err
	}

	scaled, err := barcode.Scale(bc, 400, 100)
	if err != nil {
		return "", err
	}
	img := h.subtitleBarcode(scaled)
	filePath := file.RandomFilename(".png")
	f, _ := os.Create(filePath)
	defer f.Close()

	png.Encode(f, img)
	fileOpen, _ := os.Open(filePath)
	defer fileOpen.Close()

	bucket := viper.GetString("bucket.labels")
	err = h.S3.UploadFile(fileOpen, filePath, bucket, "")
	if err != nil {
		return "", err
	}
	return filePath, nil
}

func (h *ContainerHandler) subtitleBarcode(bc barcode.Barcode) image.Image {
	fontFace := inconsolata.Bold8x16
	fontColor := color.RGBA{0, 0, 0, 255}
	margin := 5 // Space between barcode and text

	// Get the bounds of the string
	bounds, _ := font.BoundString(fontFace, bc.Content())

	widthTxt := int((bounds.Max.X - bounds.Min.X) / 64)
	heightTxt := int((bounds.Max.Y - bounds.Min.Y) / 64)

	// calc width and height
	width := widthTxt
	if bc.Bounds().Dx() > width {
		width = bc.Bounds().Dx()
	}
	height := heightTxt + bc.Bounds().Dy() + margin

	// create result img
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// draw the barcode
	draw.Draw(img, image.Rect(0, 0, bc.Bounds().Dx(), bc.Bounds().Dy()), bc, bc.Bounds().Min, draw.Over)

	// TextPt
	offsetY := bc.Bounds().Dy() + margin - int(bounds.Min.Y/64)
	offsetX := (width - widthTxt) / 2

	point := fixed.Point26_6{
		X: fixed.Int26_6(offsetX * 64),
		Y: fixed.Int26_6(offsetY * 64),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(fontColor),
		Face: fontFace,
		Dot:  point,
	}
	d.DrawString(bc.Content())
	return img
}

func (h *ContainerHandler) manifestByContainer(containerId int64, packageIdsInContainer []int64, hub *entity.Warehouse) ([]string, error) {
	carrierMap := make(map[string][]string)
	carrierPkgIdMap := make(map[string][]int64)
	carrierUserMap := make(map[string]int64)
	var pathArr []string

	trackingOptions := sqlmanager.TrackingOption{
		PackageIDs: packageIdsInContainer,
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
		} else {
			trackingNumber = tracking.TrackingNumber
		}

		if tracking.Carrier == nil || tracking.Package == nil {
			return nil, errors.New("missing carrier or package info")
		}

		carrierCode := tracking.Carrier.Code
		if carrierCode != providers.CarrierTypeTiktok {
			carrierMap[carrierCode] = append(carrierMap[carrierCode], trackingNumber)
			carrierPkgIdMap[carrierCode] = append(carrierPkgIdMap[carrierCode], tracking.PackageID) // NEW
			carrierUserMap[carrierCode] = tracking.Package.UserID
		}
	}

	if len(carrierMap) < 1 {
		return nil, errors.New("no tracking found")
	}

	for carrierCode, trackingNumbers := range carrierMap {
		userID := carrierUserMap[carrierCode]
		carrier := providers.NewCarrier(carrierCode, userID)
		if carrier == nil {
			return nil, fmt.Errorf("failed to create carrier for code %s", carrierCode)
		}

		// check tracking was be manifested automatically by ibblue, rare case
		if carrierCode == providers.CarrierTypeIBBlue {
			if ibblueCarrier, ok := carrier.(*providers.IBBlueCarrier); ok {
				base64Manifests, notManifestedTracks, err := ibblueCarrier.GetManifestByTrackingNumber(trackingNumbers)
				// only need logging
				if err != nil {
					h.Logger.Errorf("failed to get manifest: %w", err)
				}

				// Store successful manifests
				for index, base64Str := range base64Manifests {
					manifestId := fmt.Sprintf("IBBLUE-MANIFEST-%d", index)
					path, err := h.storeManifest(base64Str, manifestId, containerId)
					if err != nil {
						return nil, fmt.Errorf("store manifest error: %w", err)
					}
					pathArr = append(pathArr, path)

					manifestInsert := make([]*entity.Manifest, 0)
					manifestInsert = append(manifestInsert, &entity.Manifest{
						ContainerID:    utils.Int64(containerId),
						ManifestNumber: manifestId,
						ManifestURL:    path,
					})

					err = h.TrackingManager.CreateManifestWithTx(manifestInsert)
					if err != nil {
						return nil, fmt.Errorf("create manifest tx error: %w", err)
					}
				}

				// Only try to create new manifest for failed ones
				if len(notManifestedTracks) == 0 {
					continue
				}
				trackingNumbers = notManifestedTracks
			} else {
				return nil, fmt.Errorf("carrier is not IBBlueCarrier")
			}
		}

		manifest, message, err := h.createManifest(carrier, containerId, trackingNumbers, hub)
		if message != "" || err != nil {
			h.Logger.Errorf("gen manifest error %v", message)
			continue
		}

		for _, manifestItem := range manifest.Usps {
			path := ""
			manifestInsert := make([]*entity.Manifest, 0)
			if hub.Country != "AU" {
				path, err = h.storeManifest(manifestItem.Base64Manifest, manifestItem.ManifestNumber, containerId)
				if err != nil {
					h.Logger.Errorf("store manifest: %v", err)
					return nil, err
				}
			}

			pathArr = append(pathArr, path)

			for _, id := range carrierPkgIdMap[carrierCode] {
				manifestInsert = append(manifestInsert, &entity.Manifest{
					PackageID:      utils.Int64(id),
					ManifestNumber: manifestItem.ManifestNumber,
					ManifestURL:    path,
				})
			}

			manifestInsert = append(manifestInsert, &entity.Manifest{
				ContainerID:    utils.Int64(containerId),
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

	return pathArr, nil
}

func (h *ContainerHandler) storeManifest(url, manifest_number string, shipmentID int64) (string, error) {
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

			err = h.S3.UploadFile(res.Body, filepath, bucket, "application/pdf")
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

		contentType := res.Header.Get("Content-Type")

		switch {
		case strings.Contains(contentType, "pdf"):
			// It's a PDF
			filepath = strings.Replace(filepath, ".png", ".pdf", 1)
			err = h.S3.UploadFile(res.Body, filepath, bucket, constant.ImageContentTypePDF)
			if err != nil {
				h.Logger.Error("upload PDF manifest failed: %v", err)
				return "", err
			}
			return filepath, nil

		case strings.HasPrefix(contentType, "image/"):
			// It's an image
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

			err = h.S3.UploadFile(&buf, filepath, bucket, constant.ImageContentTypePNG)
			if err != nil {
				h.Logger.Error("upload image label: %v", err)
				return "", err
			}
			return filepath, nil

		default:
			// Unknown or unsupported type
			bodyPeek := make([]byte, 256)
			res.Body.Read(bodyPeek) // Peek first few bytes for debugging
			h.Logger.Errorf("unsupported content type: %s (body starts with: %s)", contentType, string(bodyPeek))
			return "", fmt.Errorf("unsupported content type: %s", contentType)
		}
	}

	// Base64-encoded image case
	decode, err := base64.StdEncoding.DecodeString(url)
	if err != nil {
		h.Logger.Error("decode base64 label: %v", err)
		return "", err
	}

	isPDF := bytes.HasPrefix(decode, []byte("%PDF"))
	reader := bytes.NewReader(decode)

	if isPDF {
		filepath = strings.Replace(filepath, ".png", ".pdf", 1)
		err = h.S3.UploadFile(reader, filepath, bucket, "application/pdf")
		if err != nil {
			h.Logger.Errorf("upload base64 PDF manifest failed: %v", err)
			return "", err
		}
	} else {
		err = h.S3.UploadFile(reader, filepath, bucket, constant.ImageContentTypePNG)
		if err != nil {
			h.Logger.Error("upload base64 label: %v", err)
			return "", err
		}
	}
	return filepath, nil
}

func (h *ContainerHandler) createManifest(carrier providers.Carrier, shipmentID int64, trackingNumbers []string, warehouse *entity.Warehouse) (*providers.ManifestResponse, string, error) {
	body := providers.ManifestRequest{
		ShipmentID:      shipmentID,
		TrackingNumbers: trackingNumbers,
		Name:            warehouse.Name,
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

func ParseVolumes(l, h, w float64) (length, height, width float64) {
	length = l
	height = h
	width = w

	if length < height {
		height, length = length, height
	}

	if width < height {
		height, width = width, height
	}

	if length < width {
		width, length = length, width
	}

	return
}
