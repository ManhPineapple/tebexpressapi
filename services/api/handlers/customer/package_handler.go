package customer

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"tebexpressapi/pkg/alert"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/createlabel"
	"tebexpressapi/pkg/dto"
	"tebexpressapi/pkg/helpers/authhelper"
	hpkg "tebexpressapi/pkg/helpers/packages"
	"tebexpressapi/pkg/httputil"
	dto_models "tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/order"
	packageutils "tebexpressapi/pkg/package_utils"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ups"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"tebexpressapi/pkg/utils/file"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PackageHandler struct {
	Logger  *zap.SugaredLogger
	Redis   *redis.Client
	LocalS3 storage.S3

	UPS                        *ups.UPS
	USStates                   []*entity.State
	CalculatePrice             *calculate.CalculatePrice
	CreateLabel                *createlabel.CreateLabel
	EstimateExceedPackage      *packageutils.EstimateExceedPackage
	EstimateCost               *packageutils.EstimateCost
	ShipmentCreateLabelHandler *packageutils.CreateLabelHandler
	PackageRefund              *packageutils.PackageRefund
	ShipmentCancelCarrier      *packageutils.ShipmentCancelCarrier

	PartnerMap map[string]int64

	PackageManager   *sqlmanager.PackageManager
	UserManager      *sqlmanager.UserManager
	StateManager     *sqlmanager.StateManager
	ServiceManager   *sqlmanager.ServiceManager
	BillManager      *sqlmanager.BillManager
	WareHouseManager *sqlmanager.WareHouseManager
	ProductManager   *sqlmanager.ProductManager
	TrackingManager  *sqlmanager.TrackingManager
}

type GetListPackagesResponse struct {
	Packages []PackagesDTO `json:"packages"`
}

type CancelPackageResponse struct {
	Success bool `json:"success"`
}

type CancelPackageForm struct {
	IDS []int64 `json:"ids"`
}

type PackagesDTO struct {
	dbgorm.Model
	Code                 string     `json:"code"`
	Company              string     `json:"company"`
	PhoneNumber          string     `json:"phone_number"`
	Address1             string     `json:"address_1" gorm:"column:address_1"`
	Address2             string     `json:"address_2" gorm:"column:address_2"`
	City                 string     `json:"city"`
	StateCode            string     `json:"state_code"`
	Zipcode              string     `json:"zipcode"`
	CountryCode          string     `json:"country_code"`
	OrderNumber          string     `json:"order_number"`
	Status               int        `json:"status,omitempty"`
	StatusString         string     `gorm:"-" json:"status_string"`
	ShippingFee          float64    `json:"shipping_fee"`
	ServiceName          string     `json:"service_name"`
	TrackingNumber       string     `json:"tracking_number"`
	Label                string     `json:"label"`
	ValidateAddress      int        `json:"validate_address"`
	Alert                int64      `json:"alert"`
	Weight               float64    `json:"weight"`
	IsPackageExceed      bool       `json:"is_package_exceed"`
	AcceptedAt           *time.Time `json:"accepted_at"`
	DeliveredAt          *time.Time `json:"delivered_at"`
	IsBookmark           bool       `json:"is_bookmark"`
	EstimateDeliveryRate string     `json:"estimate_delivery_rate"`

	CustomTiktokBarcode string `json:"custom_tiktok_barcode"`
	IsEarlyScan         bool   `json:"is_early_scan,omitempty"`

	PackageName       string  `json:"package_name"`
	PackageQuantity   int64   `json:"package_quantity"`
	TotalProductPrice float64 `json:"product_price"`
}

type UpdatePackageResponse struct {
	Package     *dto.PackageDetailDTO `json:"package"`
	DeliverLogs interface{}           `json:"deliver_logs"`
	ExtraFree   interface{}           `json:"extra_fee"`
	Success     bool                  `json:"success"`
}

type CountListPackageReturnResponse struct {
	Count int64 `json:"count"`
}
type CountListPackagesResponse struct {
	Count         int64                          `json:"count"`
	CountBookmark int64                          `json:"count_bookmark"`
	StatusCount   []dto.CountStatusStringPackage `json:"status_count"`
}

type TrackingListShippingCode struct {
	Codes []string `json:"codes"`
}

type TrackingListShippingCodeResponse struct {
	Packages []dto.TrackingDTO `json:"packages"`
	Logs     interface{}       `json:"logs"`
}

type CountPackagesHoldingResponse struct {
	Count int64 `json:"count"`
}

type CountTrackingListShippingCodeResponse struct {
	Count       int64                          `json:"count"`
	StatusCount []dto.CountStatusStringPackage `json:"status_count"`
}

type GetPackageDetailResponse struct {
	Package       *dto.PackageDetailDTO `json:"package"`
	DeliverLogs   interface{}           `json:"deliver_logs"`
	AuditLogs     interface{}           `json:"audit_logs"`
	ExtraFree     interface{}           `json:"extra_fee"`
	PackageRefund interface{}           `json:"package_refund"`
}

type PackageRefundDTO struct {
	dbgorm.Model
	PackageID int64   `json:"package_id"`
	Amount    float64 `json:"amount"`
	Status    int     `json:"status"`
	Code      string  `json:"code"`
}

type GetListPackagesHoldingResponse struct {
	PackagesHolding []PackageRefundDTO `json:"package_holding"`
	Day             int                `json:"day"`
}

type ImportPackageError struct {
	Line     int64    `json:"line"`
	Value    []string `json:"value"`
	Messages []string `json:"messages"`
}

type ImportPackageResponse struct {
	Errors       []ImportPackageError `json:"errors"`
	Total        int64                `json:"total"`
	ImportSucess int64                `json:"import_sucess"`
}

type ProcessPackageForm struct {
	Ids []int64 `json:"ids"`
}

type ProcessPackageResponse struct {
	Success        bool `json:"success"`
	PromotionLabel bool `json:"promotion_label"`
}

type ImportFbaPackageError struct {
	Line     int64    `json:"line"`
	Value    []string `json:"value"`
	Messages []string `json:"messages"`
}

type ImportFbaPackageResponse struct {
	Errors       []ImportPackageError `json:"errors"`
	Total        int64                `json:"total"`
	ImportSucess int64                `json:"import_sucess"`
}

type GetListPackageReturnResponse struct {
	Packages interface{} `json:"packages"`
}

type ExportPackageForm struct {
	IDS       []int64  `json:"ids"`
	StartDate string   `json:"start_date"`
	EndDate   string   `json:"end_date"`
	Search    string   `json:"search"`
	SearchBy  string   `json:"search_by"`
	StatusArr []string `json:"status_arr"`
	Alert     int      `json:"alert"`
}
type ExportPackageReponse struct {
	Download string `json:"download"`
}

type CreatePackageResponse struct {
	ID int64 `json:"id"`
}

var ErrorMaxWeight = errors.New("Max shipment customer weight")

func NewPackageHandler(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, alert alert.Alert, createLabel *createlabel.CreateLabel,
	calculatePrice *calculate.CalculatePrice, pm *sqlmanager.PackageManager,
	um *sqlmanager.UserManager, sm *sqlmanager.StateManager, whm *sqlmanager.WareHouseManager,
	srm *sqlmanager.ServiceManager, bm *sqlmanager.BillManager, stm *sqlmanager.SettingManager, tm *sqlmanager.TrackingManager, prm *sqlmanager.ProductManager) *PackageHandler {
	usStates, err := sm.GetStates(sqlmanager.StateOption{
		Countries: []string{"US", "AU"},
	})

	if err != nil {
		l.Panic(err)
	}

	return &PackageHandler{
		Logger:  l,
		Redis:   r,
		LocalS3: s3,

		USStates:                   usStates,
		CalculatePrice:             calculatePrice,
		CreateLabel:                createLabel,
		EstimateExceedPackage:      packageutils.NewEstimateExceedPackage(l, pm, whm, srm, calculatePrice, createLabel),
		EstimateCost:               packageutils.NewEstimateCost(l, pm, whm, srm, createLabel),
		ShipmentCreateLabelHandler: packageutils.NewCreateLabelHandler(l, r, s3, stm, pm, bm, um, whm, srm, createLabel, alert),
		PackageRefund:              packageutils.NewPackageRefund(l, pm, bm),
		ShipmentCancelCarrier:      packageutils.NewShipmentCancelCarrier(l, pm, tm),

		PartnerMap: entity.PartnerMap,

		PackageManager:   pm,
		UserManager:      um,
		StateManager:     sm,
		ServiceManager:   srm,
		BillManager:      bm,
		WareHouseManager: whm,
		ProductManager:   prm,
		TrackingManager:  tm,
	}
}

func (h *PackageHandler) UploadImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, gin.H{"error": constant.MessagePermissionDenied})
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": constant.MessagePermissionDenied})
			return
		}

		file, header, err := c.Request.FormFile("image")
		if err != nil {
			h.Logger.Errorf("Failed to retrieve image file: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve image file"})
			return
		}
		defer file.Close()

		fileBytes, err := io.ReadAll(file)
		if err != nil {
			h.Logger.Errorf("Failed to read image file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image file"})
			return
		}

		fileExt := filepath.Ext(header.Filename)
		filePath := fmt.Sprintf("cn_invoices/%d-%d%s", userID, time.Now().Unix(), fileExt)
		bucketName := viper.GetString("bucket.cn_product_invoice")

		err = h.LocalS3.UploadFile(bytes.NewReader(fileBytes), filePath, bucketName, "image/jpeg")
		if err != nil {
			h.Logger.Errorf("Failed to upload image to S3: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image to S3"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"url": filePath})
	}
}

func (h *PackageHandler) Create() gin.HandlerFunc {
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

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		validator := order.MakeValidator(h.StateManager).SetLang("EN")
		// Validate form
		form, err := validator.Decode(c.Request, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.APIResponseMessageValidateInput,
			})

			return
		}

		existedPackages, _ := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			OrderNumber:     form.OrderNumber,
			UserID:          userID,
			IgnoreStatusArr: []int64{constant.PackageStatusArchived, constant.PackageStatusCancelled},
		})

		if len(existedPackages) > 0 {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã đơn hàng %s đã tồn tại.", form.OrderNumber))
			return
		}

		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
		if form.Service == "" {
			form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
		}
		if form.Service == "" {
			c.JSON(http.StatusBadRequest, "Dịch vụ không được trống")
			return
		}
		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
			return
		}

		if service.Country != form.Country {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, form.Country))
			return
		}

		if user.PartnerID != 0 && service.PartnerID != user.PartnerID {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, form.Country))
			return
		}

		if service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không được hỗ trợ", service.Name))
			return
		}

		if service != nil {
			form.ServiceCode = service.Code
		}
		if form != nil {
			// Đơn CN có label tiktok riêng, không cần validate
			if form.CustomTiktokBarcode != "" && form.ServiceCode == constant.ServiceCNCode {
				form.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.OrderNumber)
				if form.OrderNumber == "" {
					c.JSON(http.StatusBadRequest, "Mã đơn hàng không để trống")
					return
				}

				if len(form.OrderNumber) > 200 {
					c.JSON(http.StatusBadRequest, "Mã đơn hàng không được vượt quá 200 ký tự")
					return
				}

				form.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Detail)
				if form.Detail == "" {
					c.JSON(http.StatusBadRequest, "Chi tiết sản phẩm không để trống")
					return
				}

				if len(form.Detail) > 1000 {
					c.JSON(http.StatusBadRequest, "Chi tiết sản phẩm không được vượt quá 1000 ký tự")
					return
				}

			} else if form.ServiceCode == constant.ServiceTiktokCode || form.CustomTiktokBarcode != "" {
				validator.ValidateTiktokPkg(form)
			} else if form.ServiceCode == constant.ServiceCNCode {
				validator.ValidateChinaPackage(form)
			} else {
				validator.Validate(form)
				validator.ValidateVolumes(form)
				validator.ValidateFbaServicePackage(form)
			}
		}

		if messages := validator.Errors(); len(messages) > 0 {
			c.JSON(http.StatusBadRequest, messages[0])
			return
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		form.Weight = math.Ceil(form.Weight*100) / 100
		form.Length = math.Ceil(form.Length*100) / 100
		form.Width = math.Ceil(form.Width*100) / 100
		form.Height = math.Ceil(form.Height*100) / 100

		hasProducts := []*entity.PackageProducts{}
		for _, packageProduct := range form.PackageProducts {
			product, err := h.ProductManager.GetProductByID(packageProduct.ProductID)
			if err != nil {
				c.JSON(http.StatusBadRequest, "Sản phẩm không tồn tại")
				return
			}

			if packageProduct.Quantity > product.Stock {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Sản phẩm %s không đủ hàng trong kho", product.SKU))
				return
			}

			hasProducts = append(hasProducts, &entity.PackageProducts{
				ProductID: packageProduct.ProductID,
				Status:    constant.PackageProductsStatusActive,
				Quantity:  packageProduct.Quantity,
				Product:   product,
			})
		}

		sp := &entity.Package{
			OrderNumber:       form.OrderNumber,
			Detail:            form.Detail,
			Recipient:         form.Recipient,
			PhoneNumber:       form.Phone,
			Address1:          form.Address1,
			Address2:          form.Address2,
			City:              form.City,
			StateCode:         form.State,
			Zipcode:           form.Zipcode,
			CountryCode:       form.Country,
			Weight:            form.Weight,
			Length:            form.Length,
			Width:             form.Width,
			Height:            form.Height,
			IncludeBattery:    form.IncludeBattery,
			ValidateAddress:   constant.PackageValidAddress,
			UserID:            userID,
			ServiceID:         service.ID,
			Status:            constant.PackageStatusCreated,
			PackageProducts:   hasProducts,
			Service:           service,
			PartnerID:         user.PartnerID,
			PackageName:       form.PackageName,
			PackageQuantity:   form.PackageQuantity,
			TotalProductPrice: form.TotalProductPrice,
		}

		if service.Code == constant.ServiceCNCode {
			sp.CNIsPurchased = form.CNIsPurchased
			if form.CNIsPurchased == false {
				sp.CNProductLink = form.CNProductLink
				sp.CNProductPrice = form.CNProductPrice
			} else if form.CNIsPurchased == true {
				sp.CNShippingFee = form.CNShippingFee
				sp.Status = constant.PackageStatusCNPurchased
			}

			sp.CustomCNBarcode = form.CustomCNBarcode
			sp.CNNote = form.CNNote
			sp.CNProductImage = form.CNProductImage
			if form.ImageUpload != "" {
				sp.CNInvoiceImage = form.ImageUpload
			}
		}

		sp.IsEarlyScan = form.IsEarlyScan
		if service.Code == constant.ServiceTiktokCode || form.CustomTiktokBarcode != "" {
			sp.CustomTiktokBarcode = &form.CustomTiktokBarcode

			driveRegex := regexp.MustCompile(`drive\.google\.com/file/d/([^/]+)/`)
			matches := driveRegex.FindStringSubmatch(form.CustomTiktokBarcode)
			if len(matches) > 1 {
				fileID := matches[1]
				form.CustomTiktokBarcode = fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
			}
			filePath, err := order.StoreLabelS3(h.LocalS3, form.CustomTiktokBarcode, "pdf", fmt.Sprintf("tiktok_%s_%03d", sp.OrderNumber, rand.Intn(1000)))
			if err != nil {
				h.Logger.Errorf("Failed upload to S3: %s", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			sp.Label = filePath
		}

		if service.Code == constant.ServiceWarehouseCode {
			sp.Label = form.ImageUpload
		}

		var isErrorEsPrice bool
		var serviceIDToCalculatePrice int64
		if form.CustomTiktokBarcode != "" {
			if constant.IsPriorityService(service.Code) {
				serviceIDToCalculatePrice = 28 // tiktok priority price
			} else {
				serviceIDToCalculatePrice = 25 // tiktok price
			}
		} else {
			serviceIDToCalculatePrice = service.ID
		}
		price, priceOutSize, err := h.CalculatePrice.Price3(c, userID, serviceIDToCalculatePrice, user.Class, form.Weight, form.Length, form.Height, form.Width, form.Country)
		if err == calculate.ErrorNotService {
			if service.Code != constant.ServiceCNCode {
				c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
				return
			} else {
				err = nil
			}
		}

		if service.Code == constant.ServiceLABELCode {
			carrier := providers.NewCarrier(service.DomesticCarrier.Code, userID)
			if carrier == nil {
				c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
					Error:    "Bad request",
					Messages: []string{"The service code is invalid"},
				})

				return
			}

			h.Logger.Info("LABEL CODE: ", price)
			cost, err := h.PackageEstimateCost(c, carrier, sp)
			if err == nil {
				price = cost + price/100*cost
			}
			h.Logger.Info("LABEL CODE: ", price, cost, err)
		}

		// if form.Country == "AU" || service.Code == constant.ServiceFBACode {
		if err == calculate.ErrorMaxWeight {
			msg := "Trọng lượng cho phép vượt quá giới hạn"
			if price > 0 {
				msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
			}

			c.JSON(http.StatusBadRequest, msg)
			return
		}

		if err == calculate.ErrorMaxVolume {
			msg := "Kích thước vượt quá giới hạn cho phép"
			if price > 0 {
				msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
			}

			c.JSON(http.StatusBadRequest, msg)
			return
		}

		sp.ShippingFee = price

		if err != nil && err != calculate.ErrorMaxWeight && err != calculate.ErrorMaxVolume {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if priceOutSize > 0 {
			sp.ExtraFee = []entity.ExtraFee{{Amount: priceOutSize, ExtraFeeTypeID: constant.ExtraFeeTypeOutSize}}
		}

		if !isErrorEsPrice {
			fees, err := h.CalculatePrice.PromotionExtras(sp, sp.ExtraFee, price)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if len(fees) > 0 {
				if sp.ExtraFee == nil {
					sp.ExtraFee = []entity.ExtraFee{}
				}

				sp.ExtraFee = append(sp.ExtraFee, fees...)
			}
		}

		if form.IsTradeMark {
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         1,
				ExtraFeeTypeID: constant.ExtraFeeTypeTradeMark,
			})
		}

		if sp.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
			})
		}

		isInsured, insuredFee, err := h.CalculatePrice.PromotionInsured(sp.UserID)
		if err != nil {
			h.Logger.Errorf("promotion insured: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if isInsured {
			sp.IsInsured = true
			if insuredFee > 0 {
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         insuredFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeInsured,
				})
			}
		}
		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         extraFeeService,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
			})
		}

		if sp.Service.Code == constant.ServiceCNCode {
			if sp.Status == constant.PackageStatusCreated && sp.CNProductPrice != 0 {
				var cnPricePercentage float64
				if sp.CNProductPrice > 200 {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_high_price_percentage")
				} else {
					cnPricePercentage = viper.GetFloat64("extra_fees.cn_low_price_percentage")
				}
				minProxyPrice := viper.GetFloat64("extra_fees.cn_min_proxy_buying_fee")
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         math.Max(sp.CNProductPrice*cnPricePercentage, minProxyPrice),
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaProductPercentage,
				})

				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         sp.CNProductPrice,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaProduct,
				})

			} else if sp.Status == constant.PackageStatusCNPurchased && sp.CNShippingFee != 0 {
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         sp.CNShippingFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeChinaShipping,
				})
			}

			defaultCNShippingFeeToVN := viper.GetFloat64("extra_fees.default_cn_ship_to_vn_fee")
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         defaultCNShippingFeeToVN,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeCNShippingToVN,
			})
			defaultCNHandlingFee := viper.GetFloat64("extra_fees.default_cn_handling_fee") // Phí handling + active tracking
			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         defaultCNHandlingFee,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
			})
		}

		if sp.Service.Code == constant.ServiceWarehouseCode {
			defaultWsHandlingFee := viper.GetFloat64("extra_fees.default_ws_handling_fee")

			if form.IsTiktokWarehouse {
				price = defaultWsHandlingFee // dùng label riêng sẽ bỏ qua phí tạo tracking ibblue
				sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
					Amount:         defaultWsHandlingFee,
					PackageID:      utils.Int64(sp.ID),
					ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
				})
			}
			if user.Balance < price {
				c.JSON(http.StatusBadRequest, "Tài khoản của quý khách không đủ tiền, vui lòng nạp thêm tiền.")
				return
			}
		}

		tiktokEarlyScanFee := viper.GetFloat64("extra_fees.default_tiktok_early_scan_fee")
		if sp.IsEarlyScan {
			price += tiktokEarlyScanFee

			sp.ExtraFee = append(sp.ExtraFee, entity.ExtraFee{
				Amount:         tiktokEarlyScanFee,
				PackageID:      utils.Int64(sp.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeEarlyScanTiktok,
			})
		}

		packageIDsCreated, err := h.PackageManager.CreatePackages([]*entity.Package{sp}, userID)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
				return
			}
			var mysqlErr *mysql.MySQLError
			if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
				c.JSON(http.StatusBadRequest, "Mã vạch đã tồn tại")
				return
			}

			h.Logger.Error("Error create shipping package", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(hasProducts) > 0 {
			for _, packageProduct := range hasProducts {
				err := h.ProductManager.AdjustStock(packageProduct.ProductID, packageProduct.PackageID, -packageProduct.Quantity)
				if err != nil {
					h.Logger.Errorf("Error when get product: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				}
			}
		}

		if len(packageIDsCreated) <= 0 {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		options := sqlmanager.PackageQueryOption{ID: packageIDsCreated[0]}
		packageCreated, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		//push estimate queue
		if isErrorEsPrice {
			err = h.EstimateExceedPackage.Handle(c, packageCreated, user)
			if err != nil {
				h.Logger.Error("Error publish message queue check package address: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		c.JSON(http.StatusOK, CreatePackageResponse{ID: packageCreated.ID})
	}
}

func (h *PackageHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opts := sqlmanager.PackageQueryOption{
			UserID:          userID,
			Limit:           limit,
			Offset:          offset,
			Code:            c.Request.URL.Query().Get("code"),
			CStatus:         []int{constant.PackageCodeEnable, constant.PackageCodeDisable, constant.PackageCodeTemp},
			SearchBy:        cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:          cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue:      cast.ToInt(c.Request.URL.Query().Get("alert")),
			IsBookmark:      cast.ToBool(c.Request.URL.Query().Get("is_bookmark")),
			ExceptFba:       true,
			IsPreloadRefund: true,
		}

		statusString := cast.ToString(c.Request.URL.Query().Get("status"))
		statusArr := strings.Split(cast.ToString(c.Request.URL.Query().Get("status_arr")), ",")

		statusArr = append(statusArr, statusString)
		if statusString == constant.PackageStatusAlertText {
			opts.QueryAlert = true
		} else {
			for _, status := range statusArr {
				opts.StatusArr = append(opts.StatusArr, constant.MapIntGroupStatusCustomerPackage[status]...)
			}
		}

		if opts.AlertValue != constant.PackageAlertTypeDisable && !utils.ContainsNumber([]int64{constant.PackageAlertTypeOverPretransit, constant.PackageAlertTypeWarehoseReturn, constant.PackageAlertTypeHubReturn}, cast.ToInt64(opts.AlertValue)) {
			c.JSON(http.StatusBadRequest, "Invalid alert value")
		}

		if statusString != "" && len(opts.StatusArr) == 0 && !opts.QueryAlert && !opts.IsBookmark {
			c.JSON(http.StatusBadRequest, "Invalid status")
			return
		}

		serviceCode := cast.ToString(c.Request.URL.Query().Get("service"))
		if serviceCode != "" {
			opts.ServiceCode = serviceCode
		} else {
			opts.IgnoreServiceCodes = []string{constant.ServiceCNCode}
		}

		opts.HasTiktokLabel = cast.ToBool(c.Request.URL.Query().Get("has_tiktok_label"))
		opts.IsEarlyScan = cast.ToBool(c.Request.URL.Query().Get("is_early_scan"))

		startDate := strings.TrimSpace(c.Request.URL.Query().Get("start_date"))
		if startDate != "" {
			if dt := utils.ParseRawDateTime(startDate); dt == nil {
				c.JSON(http.StatusBadRequest, "Invalid start date format")
				return
			}

			byDate := strings.TrimSpace(c.Request.URL.Query().Get("by_date"))
			if byDate == "accept" {
				opts.InWarehouseStartDate = startDate
			} else {
				opts.StartDate = startDate
			}
		}

		endDate := strings.TrimSpace(c.Request.URL.Query().Get("end_date"))
		if endDate != "" {
			if dt := utils.ParseRawDateTime(endDate); dt == nil {
				c.JSON(http.StatusBadRequest, "Invalid end date format")
				return
			}

			byDate := strings.TrimSpace(c.Request.URL.Query().Get("by_date"))
			if byDate == "accept" {
				opts.InWarehouseEndDate = endDate
			} else {
				opts.EndDate = endDate
			}
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		opts.TrackingStatus = constant.TrackingStatusSuccess
		packages, err := h.PackageManager.GetPackages(opts)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mRate := make(map[string]string)
		packagesDTOs := []PackagesDTO{}
		for i, Package := range packages {
			pRefunds := Package.PackageRefunds
			isNotRefund := false
			for _, refund := range pRefunds {
				if refund.Status != constant.PackageRefundCompleted {
					isNotRefund = true
				}
			}
			var extraFee float64 = 0
			for _, fee := range Package.ExtraFee {
				if fee.ExtraFeeTypeID == constant.ExtraFeeTypeCancelLabel && isNotRefund {
					continue
				}
				extraFee += fee.Amount
			}
			packages[i].PackageRefunds = nil
			packages[i].ShippingFee = packages[i].ShippingFee + extraFee
			packages[i].StatusString = constant.MapTextStatusCustomerPackage[packages[i].Status]

			newPackage := PackagesDTO{}

			if err := httputil.Transform(packages[i], &newPackage); err != nil {
				h.Logger.Errorf("transform package : %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if packages[i].Service != nil {
				newPackage.ServiceName = packages[i].Service.Name
			}
			if packages[i].PackageCode != nil && packages[i].PackageCode.Status != constant.PackageCodeTemp {
				newPackage.Code = packages[i].PackageCode.Code
			}
			if packages[i].Tracking != nil {
				newPackage.Label = packages[i].Tracking.LabelURL
				newPackage.TrackingNumber = packages[i].Tracking.TrackingNumber
			} else {
				newPackage.Label = packages[i].Label
			}

			if Package.Status == constant.PackageStatusCreated {
				newPackage.Code = ""
				newPackage.TrackingNumber = ""
			}

			if Package.DeliveredAt != nil {
				var key string
				if Package.CheckinWarehouseAt != nil {
					timeSince := utils.Floor(Package.DeliveredAt.Sub(cast.ToTime(Package.CheckinWarehouseAt)).Hours()/24, 0)
					if timeSince < 1 {
						key = fmt.Sprintf("%v H", utils.Floor(Package.DeliveredAt.Sub(cast.ToTime(Package.CheckinWarehouseAt)).Hours(), 0))
					} else {
						key = fmt.Sprintf("%v D", timeSince)
					}
				} else {
					timeSince := utils.Floor(Package.DeliveredAt.Sub(cast.ToTime(Package.CreatedAt)).Hours()/24, 0)
					if timeSince < 1 {
						key = fmt.Sprintf("%v H", utils.Floor(Package.DeliveredAt.Sub(cast.ToTime(Package.CreatedAt)).Hours(), 0))
					} else {
						key = fmt.Sprintf("%v D", timeSince)
					}
				}

				if mRate[key] != "" {
					newPackage.EstimateDeliveryRate = mRate[key]
				}

			}

			newPackage.AcceptedAt = Package.CheckinWarehouseAt
			newPackage.DeliveredAt = Package.DeliveredAt
			newPackage.IsBookmark = Package.IsBookmark
			if Package.CustomTiktokBarcode != nil {
				newPackage.CustomTiktokBarcode = *Package.CustomTiktokBarcode
			}
			newPackage.IsEarlyScan = Package.IsEarlyScan
			newPackage.PackageName = Package.PackageName
			newPackage.PackageQuantity = Package.PackageQuantity
			newPackage.TotalProductPrice = Package.TotalProductPrice

			packagesDTOs = append(packagesDTOs, newPackage)
		}

		c.JSON(http.StatusOK, GetListPackagesResponse{packagesDTOs})
	}
}

func (h *PackageHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		opts := sqlmanager.PackageQueryOption{
			UserID:     userID,
			Code:       c.Request.URL.Query().Get("code"),
			SearchBy:   cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: cast.ToInt(c.Request.URL.Query().Get("alert")),
			IsBookmark: cast.ToBool(c.Request.URL.Query().Get("is_bookmark")),
			ExceptFba:  true,
		}

		serviceCode := cast.ToString(c.Request.URL.Query().Get("service"))
		if serviceCode != "" {
			opts.ServiceCode = serviceCode
		}

		statusString := cast.ToString(c.Request.URL.Query().Get("status"))
		statusArr := strings.Split(cast.ToString(c.Request.URL.Query().Get("status_arr")), ",")

		if statusString == constant.PackageStatusAlertText {
			opts.QueryAlert = true
		} else {
			opts.StatusArr = append(opts.StatusArr, constant.MapIntGroupStatusCustomerPackage[statusString]...)
			for _, status := range statusArr {
				opts.StatusArr = append(opts.StatusArr, constant.MapIntGroupStatusCustomerPackage[status]...)
			}
		}

		if opts.AlertValue != constant.PackageAlertTypeDisable && !utils.ContainsNumber([]int64{constant.PackageAlertTypeOverPretransit, constant.PackageAlertTypeWarehoseReturn, constant.PackageAlertTypeHubReturn}, cast.ToInt64(opts.AlertValue)) {
			c.JSON(http.StatusBadRequest, "Invalid alert value")
		}

		if statusString != "" && len(opts.StatusArr) == 0 && !opts.QueryAlert && !opts.IsBookmark {
			c.JSON(http.StatusBadRequest, "Invalid status")
			return
		}

		startDate := strings.TrimSpace(c.Request.URL.Query().Get("start_date"))
		if startDate != "" {
			if dt := utils.ParseRawDateTime(startDate); dt == nil {
				c.JSON(http.StatusBadRequest, "Invalid start date format")
				return
			}

			byDate := strings.TrimSpace(c.Request.URL.Query().Get("by_date"))
			if byDate == "accept" {
				opts.InWarehouseStartDate = startDate
			} else {
				opts.StartDate = startDate
			}
		}

		endDate := strings.TrimSpace(c.Request.URL.Query().Get("end_date"))
		if endDate != "" {
			if dt := utils.ParseRawDateTime(endDate); dt == nil {
				c.JSON(http.StatusBadRequest, "Invalid end date format")
				return
			}

			byDate := strings.TrimSpace(c.Request.URL.Query().Get("by_date"))
			if byDate == "accept" {
				opts.InWarehouseEndDate = endDate
			} else {
				opts.EndDate = endDate
			}
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		Count, err := h.PackageManager.CountPackages(opts)

		if err != nil {
			h.Logger.Errorf("Count package error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		optsCountAll := sqlmanager.PackageQueryOption{
			UserID:     userID,
			StartDate:  cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:    cast.ToString(c.Request.URL.Query().Get("end_date")),
			Code:       c.Request.URL.Query().Get("code"),
			SearchBy:   cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: cast.ToInt(c.Request.URL.Query().Get("alert")),
			IsBookmark: cast.ToBool(c.Request.URL.Query().Get("is_bookmark")),
			ExceptFba:  true,
		}

		if serviceCode != "" {
			optsCountAll.ServiceCode = serviceCode
		}
		for _, status := range statusArr {
			optsCountAll.StatusArr = append(optsCountAll.StatusArr, constant.MapIntGroupStatusCustomerPackage[status]...)
		}

		var count int64

		countStatus, err := h.PackageManager.CountAllStatusPackages(optsCountAll)
		if err != nil {
			h.Logger.Errorf("Count all status  error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		optsCountAlert := sqlmanager.PackageQueryOption{
			UserID:     userID,
			StartDate:  cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:    cast.ToString(c.Request.URL.Query().Get("end_date")),
			Code:       c.Request.URL.Query().Get("code"),
			QueryAlert: true,
			SearchBy:   cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: cast.ToInt(c.Request.URL.Query().Get("alert")),
			IsBookmark: cast.ToBool(c.Request.URL.Query().Get("is_bookmark")),
			ExceptFba:  true,
		}
		for _, status := range statusArr {
			optsCountAlert.StatusArr = append(optsCountAlert.StatusArr, constant.MapIntGroupStatusCustomerPackage[status]...)
		}

		if serviceCode != "" {
			optsCountAlert.ServiceCode = serviceCode
		}
		countAlert, err := h.PackageManager.CountPackages(optsCountAlert)
		if err != nil {
			h.Logger.Errorf("Count all status  error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var countStatusString = make([]dto.CountStatusStringPackage, 0)

		for _, statusInt := range countStatus {

			statusConverted := constant.MapTextStatusCustomerPackage[statusInt.Status]
			if statusConverted == "" {
				continue
			}

			var existing = false
			var index int
			for i, statusString := range countStatusString {
				if statusConverted == statusString.Status {
					existing = true
					index = i
				}
			}

			if !existing {
				countStatusString = append(countStatusString, dto.CountStatusStringPackage{
					Status: statusConverted,
					Count:  statusInt.Count,
				})
			} else {
				countStatusString[index].Count = countStatusString[index].Count + statusInt.Count
			}

			count = count + statusInt.Count
		}

		countStatusString = append(countStatusString, dto.CountStatusStringPackage{
			Status: constant.PackageStatusAlertText,
			Count:  countAlert,
		})

		opts = sqlmanager.PackageQueryOption{
			UserID:     userID,
			Code:       c.Request.URL.Query().Get("code"),
			SearchBy:   cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: cast.ToInt(c.Request.URL.Query().Get("alert")),
			ExceptFba:  true,
			IsBookmark: true,
		}

		if serviceCode != "" {
			opts.ServiceCode = serviceCode
		}

		bmCount, err := h.PackageManager.CountPackages(opts)

		if err != nil {
			h.Logger.Errorf("Count package error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListPackagesResponse{Count, bmCount, countStatusString})
	}
}

func (h *PackageHandler) Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		formData := &TrackingListShippingCode{}
		if err := c.ShouldBindJSON(formData); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(formData.Codes) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		if len(formData.Codes) > 50 {
			c.JSON(http.StatusBadRequest, "Số lượng mã tracking không vượt quá 50")
			return
		}

		packages, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			CodeAndTracking: formData.Codes,
		})
		if err == gorm.ErrRecordNotFound || len(packages) == 0 {
			c.JSON(http.StatusBadRequest, "Không tìm thấy")
			return
		}
		if err != nil {
			h.Logger.Errorf("Get packages error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkgs := []dto.TrackingDTO{}

		var ids []int64
		for i := range packages {
			packages[i].StatusString = constant.MapTextStatusCustomerPackage[packages[i].Status]

			var tracking *dto.TrackingInfoDTO
			var packageCode *dto.PackageCodeInfoDTO
			var deliveredAt, checkInWarehouseAt *time.Time
			if packages[i].Tracking != nil {
				tracking = &dto.TrackingInfoDTO{
					PackageID:      packages[i].Tracking.PackageID,
					TrackingNumber: packages[i].Tracking.TrackingNumber,
				}
			}

			if packages[i].PackageCode != nil {
				packageCode = &dto.PackageCodeInfoDTO{
					Code: packages[i].PackageCode.Code,
				}
			}

			if packages[i].DeliveredAt != nil {
				deliveredAt = packages[i].DeliveredAt
			}

			if packages[i].CheckinWarehouseAt != nil {
				checkInWarehouseAt = packages[i].CheckinWarehouseAt
			}

			pkgs = append(pkgs, dto.TrackingDTO{
				ID:                 packages[i].ID,
				Tracking:           tracking,
				CountryCode:        packages[i].CountryCode,
				PackageCode:        packageCode,
				StatusString:       packages[i].StatusString,
				DeliveredAt:        deliveredAt,
				CheckinWarehouseAt: checkInWarehouseAt,
				Alert:              packages[i].Alert,
			})

			ids = append(ids, packages[i].ID)
		}

		logs, err := h.PackageManager.GetDeliverPackageLogsByIDs(ids)
		if err == gorm.ErrRecordNotFound || len(logs) == 0 {
			c.JSON(http.StatusBadRequest, "Không tìm thấy")
			return
		}
		if err != nil {
			h.Logger.Errorf("Get deliver logs error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		logs = dto.Transfomer(logs)
		c.JSON(http.StatusOK, TrackingListShippingCodeResponse{Packages: pkgs, Logs: logs})
	}
}

func (h *PackageHandler) LogCount() gin.HandlerFunc {
	return func(c *gin.Context) {
		formData := &TrackingListShippingCode{}
		if err := c.ShouldBindJSON(formData); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(formData.Codes) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		if len(formData.Codes) > 50 {
			c.JSON(http.StatusBadRequest, "Số lượng mã tracking không vượt quá 50")
			return
		}

		opts := sqlmanager.PackageQueryOption{
			CodeAndTracking: formData.Codes,
		}

		Count, err := h.PackageManager.CountPackages(opts)

		if err != nil {
			h.Logger.Errorf("Count package error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var count int64

		countStatus, err := h.PackageManager.CountAllStatusPackages(opts)
		if err != nil {
			h.Logger.Errorf("Count all status  error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var countStatusString = make([]dto.CountStatusStringPackage, 0)

		for _, statusInt := range countStatus {

			statusConverted := constant.MapTextStatusCustomerPackage[statusInt.Status]
			if statusConverted == "" {
				continue
			}

			var existing = false
			var index int
			for i, statusString := range countStatusString {
				if statusConverted == statusString.Status {
					existing = true
					index = i
				}
			}

			if !existing {
				countStatusString = append(countStatusString, dto.CountStatusStringPackage{
					Status: statusConverted,
					Count:  statusInt.Count,
				})
			} else {
				countStatusString[index].Count = countStatusString[index].Count + statusInt.Count
			}

			count = count + statusInt.Count
		}

		c.JSON(http.StatusOK, CountTrackingListShippingCodeResponse{Count, countStatusString})
	}
}

func (h *PackageHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		packageID := cast.ToInt64(c.Param("package_id"))
		if packageID < 1 {
			c.JSON(http.StatusBadRequest, "Missing package id")
			return
		}

		options := sqlmanager.PackageQueryOption{
			ID:             packageID,
			TrackingStatus: constant.TrackingStatusSuccess,
		}

		packages, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if userID != packages.UserID {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		opt := sqlmanager.ProductQueryOption{
			UserID: userID,
			Status: constant.StatusActive,
		}

		products, err := h.ProductManager.GetProducts(opt)
		if err != nil && err == gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var mapProducts = make(map[int64]*entity.Product)

		for _, product := range products {
			mapProducts[product.ID] = product
		}

		packageDTO := &dto.PackageDetailDTO{}
		if err := httputil.Transform(packages, packageDTO); err != nil {
			h.Logger.Errorf("transform package detail error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if packageDTO.ActualWeight < packageDTO.Weight {
			packageDTO.ActualWeight = packageDTO.Weight
		}

		if packageDTO.ActualHeight*packageDTO.ActualWidth*packageDTO.ActualLength < packageDTO.Height*packageDTO.Width*packageDTO.Length {
			packageDTO.ActualHeight = packageDTO.Height
			packageDTO.ActualWidth = packageDTO.Width
			packageDTO.ActualLength = packageDTO.Length
		}

		packageDTO.ID = packages.ID
		packageDTO.StatusString = constant.MapTextStatusCustomerPackage[packages.Status]
		packageDTO.Status = 0
		packageDTO.IncludeBattery = packages.IncludeBattery
		packageDTO.ServiceName = packages.Service.Name
		packageDTO.ServiceCode = packages.Service.Code

		if packages.Service.Code == constant.ServiceCNCode {
			packageDTO.CNIsPurchased = packages.CNIsPurchased
			packageDTO.CustomCNBarcode = packages.CustomCNBarcode
			packageDTO.CNProductLink = packages.CNProductLink
			packageDTO.CNProductPrice = packages.CNProductPrice
			packageDTO.CNInvoiceImage = packages.CNInvoiceImage
			packageDTO.CNShippingFee = packages.CNShippingFee
			packageDTO.CNProductImage = packages.CNProductImage
			packageDTO.CNNote = packages.CNNote
		}
		if packages.Service.Code == constant.ServiceTiktokCode || packages.CustomTiktokBarcode != nil {
			packageDTO.CustomTiktokBarcode = *packages.CustomTiktokBarcode
			packageDTO.IsEarlyScan = packages.IsEarlyScan
		}

		packageDTO.IsInsured = packages.IsInsured
		packageDTO.IsBookmark = packages.IsBookmark
		if packages.Tracking != nil {
			packageDTO.Label = packages.Tracking.LabelURL
			packageDTO.TrackingNumber = packages.Tracking.TrackingNumber
		}
		if packages.PackageCode != nil && packages.PackageCode.Status != constant.PackageCodeTemp && packages.Status != constant.PackageStatusArchived {
			packageDTO.CodePackage = packages.PackageCode.Code
		}

		for i := range packages.PackageProducts {
			if mapProducts[packageDTO.PackageProducts[i].ProductID] != nil {
				packageDTO.PackageProducts[i].Name = mapProducts[packageDTO.PackageProducts[i].ProductID].Name
				packageDTO.PackageProducts[i].SKU = mapProducts[packageDTO.PackageProducts[i].ProductID].SKU
			}
		}

		deliverLogs, err := h.PackageManager.GetDeliverLogsByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		auditLogs, err := h.PackageManager.GetAuditLogsByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Audit Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		extraFee, err := h.PackageManager.GetExtraFeeByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		refundConfigDay := viper.GetInt("package.day_refund_expire_pending")
		pkgRefund, err := h.PackageManager.GetPackakgeRefundByPackageID(packageDTO.ID)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if err != gorm.ErrRecordNotFound {
			temp := pkgRefund.CreatedAt.AddDate(0, 0, refundConfigDay)
			packageDTO.EstimateDateProcess = &temp
		}

		packagesDTOs := []PackageRefundDTO{}

		if packageDTO.Status != constant.PackageStatusCancelled || packageDTO.Status != constant.PackageStatusExpired {
			opts := sqlmanager.PackageQueryOption{
				UserID: userID,
				// Status: constant.PackageRefundPending,
				ID: packageDTO.ID,
			}

			pkgRefunds, err := h.PackageManager.GetListPackagesRefund(opts)

			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("Get package holding error, %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			for i, _ := range pkgRefunds {
				packageRefund := PackageRefundDTO{}

				if err := httputil.Transform(pkgRefunds[i], &packageRefund); err != nil {
					h.Logger.Errorf("transform package : %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				packageRefund.Code = pkgRefunds[i].Package.PackageCode.Code
				packageRefund.CreatedAt = pkgRefunds[i].CreatedAt.AddDate(0, 0, refundConfigDay)
				packagesDTOs = append(packagesDTOs, packageRefund)

			}
		}

		deliverLogs = dto.Transfomer2(deliverLogs)

		inStatusEstimateDelivery := map[int]bool{
			constant.PackageStatusPicked:               true,
			constant.PackageStatusWareHouseLabeled:     true,
			constant.PackageStatusWareHouseInContainer: true,
			constant.PackageStatusWareHouseInShipment:  true,
			constant.PackageStatusWareHouseExport:      true,
			constant.PackageStatusImportHub:            true,
			constant.PackageStatusExportHub:            true,
			constant.PackageStatusInTransit:            true,
		}
		if inStatusEstimateDelivery[packages.Status] {
			weight, length, height, width := packages.Weight, packages.Length, packages.Height, packages.Width
			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.ActualWeight != 0 {
				weight = packages.ActualWeight
			}

			if packages.CountryCode != "US" {
				estimate, err := hpkg.EstimateDelivery(c, h.Redis, h.CreateLabel, packages.CountryCode, 0, weight, length, height, width, packages.ServiceID)
				if err != nil {
					h.Logger.Error(err)
				}

				packageDTO.EstimateDelivery = estimate
			} else if packages.Tracking != nil {
				estimate, err := hpkg.EstimateDelivery(c, h.Redis, h.CreateLabel, packages.CountryCode, packages.Tracking.Zone, weight, length, height, width, packages.ServiceID)
				if err != nil {
					h.Logger.Error(err)
				}

				packageDTO.EstimateDelivery = estimate
			}
		}

		c.JSON(http.StatusOK, GetPackageDetailResponse{Package: packageDTO, DeliverLogs: deliverLogs, AuditLogs: auditLogs, ExtraFree: extraFee, PackageRefund: packagesDTOs})
	}
}

func (h *PackageHandler) Holding() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opts := sqlmanager.PackageQueryOption{
			UserID:    userID,
			Limit:     limit,
			Offset:    offset,
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
			Search:    cast.ToString(c.Request.URL.Query().Get("search")),
			Status:    constant.PackageRefundPending,
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

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		packages, err := h.PackageManager.GetListPackagesRefund(opts)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get package holding error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		packagesDTOs := []PackageRefundDTO{}
		for i, _ := range packages {
			packageRefund := PackageRefundDTO{}

			if err := httputil.Transform(packages[i], &packageRefund); err != nil {
				h.Logger.Errorf("transform package : %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			packageRefund.Code = packages[i].Package.PackageCode.Code
			packagesDTOs = append(packagesDTOs, packageRefund)

		}

		day := viper.GetInt("package.day_refund_expire_pending")

		c.JSON(http.StatusOK, GetListPackagesHoldingResponse{packagesDTOs, day})
	}
}

func (h *PackageHandler) CountHolding() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		opts := sqlmanager.PackageQueryOption{
			UserID:    userID,
			StartDate: cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:   cast.ToString(c.Request.URL.Query().Get("end_date")),
			Search:    cast.ToString(c.Request.URL.Query().Get("search")),
			Status:    constant.PackageRefundPending,
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

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		Count, err := h.PackageManager.CountPackagesRefund(opts)

		if err != nil {
			h.Logger.Errorf("Count package error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountPackagesHoldingResponse{Count})
	}
}

func (h *PackageHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		packageID := cast.ToInt64(c.Param("package_id"))

		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if packageID < 1 {
			c.JSON(http.StatusBadRequest, "Invalid order id")
			return
		}

		rKey := fmt.Sprintf("%s_%d", constant.RedisKeyPackageCheckExists, packageID)

		val, err := h.Redis.SetNX(c, rKey, "value", constant.RedisKeyPackageCheckExistsExp).Result()

		if err != nil {
			h.Logger.Errorf("SetNX Redis : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
		}

		if !val {
			h.Logger.Errorf("Update package error : %s", err)
			c.JSON(http.StatusBadRequest, "Vui lòng thử lại sau")
			return
		}

		defer h.Redis.Del(c, rKey)

		currentPackage, err := h.PackageManager.GetPackageByPackageID(packageID)

		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if currentPackage.UserID != userID {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		if currentPackage.Status != constant.PackageStatusCreated && currentPackage.Status != constant.PackageStatusCNPurchased {
			c.JSON(http.StatusBadRequest, "Khách hàng chỉ được sửa đơn ở trạng thái tạo mới")
			return
		}

		if currentPackage.Service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không được hỗ trợ", currentPackage.Service.Name))
			return
		}

		validator := order.MakeValidator(h.StateManager)
		form, err := validator.Decode(c.Request, true)
		if err != nil {
			h.Logger.Errorf("validator fail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.Service)
		if form.Service == "" {
			form.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(form.ServiceCode)
		}
		if form.Service == "" {
			c.JSON(http.StatusBadRequest, "Dịch vụ không được trống")
			return
		}
		service, err := h.ServiceManager.GetServiceByCode(form.Service)
		if err != nil {
			c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
			return
		}
		form.ServiceCode = service.Code
		if service.Country != form.Country {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, form.Country))
			return
		}

		if service.Code == constant.ServiceFBACode {
			c.JSON(http.StatusBadRequest, fmt.Sprintf("Dịch vụ %s không được hỗ trợ", service.Name))
			return
		}

		if form != nil {
			if form.ServiceCode == constant.ServiceCNCode {
				validator.ValidateChinaPackage(form)
			} else {
				validator.Validate(form)
				validator.ValidateVolumes(form)
				validator.ValidateFbaServicePackage(form)
			}
		}

		if messages := validator.Errors(); len(messages) > 0 {
			c.JSON(http.StatusBadRequest, messages[0])
			return
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		form.Weight = utils.Ceil(form.Weight, 2)
		form.Length = utils.Ceil(form.Length, 2)
		form.Width = utils.Ceil(form.Width, 2)
		form.Height = utils.Ceil(form.Height, 2)

		//update package audit log

		var logs []entity.PackageAuditLog
		mapchange := make(map[string]interface{})

		var hasUpdatePrice bool

		if form.Recipient != currentPackage.Recipient {
			mapchange["recipient"] = form.Recipient
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Recipient,
				Value:    form.Recipient,
				Type:     constant.PackageUpdateTypeRecipient,
			})
		}

		if form.Phone != currentPackage.PhoneNumber {
			mapchange["phone_number"] = form.Phone
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.PhoneNumber,
				Value:    form.Phone,
				Type:     constant.PackageUpdateTypePhoneNumber,
			})
		}

		if form.Address1 != currentPackage.Address1 {
			mapchange["address_1"] = form.Address1
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address1,
				Value:    form.Address1,
				Type:     constant.PackageUpdateTypeAddress,
			})
		}

		if form.Address2 != currentPackage.Address2 {
			mapchange["address_2"] = form.Address2
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Address2,
				Value:    form.Address2,
				Type:     constant.PackageUpdateTypeAddress2,
			})
		}

		if form.City != currentPackage.City {
			mapchange["city"] = form.City
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.City,
				Value:    form.City,
				Type:     constant.PackageUpdateTypeCity,
			})
		}

		if form.State != currentPackage.StateCode {
			mapchange["state_code"] = form.State
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.StateCode,
				Value:    form.State,
				Type:     constant.PackageUpdateTypeStateCode,
			})
		}

		if form.Zipcode != currentPackage.Zipcode {
			mapchange["zipcode"] = form.Zipcode
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Zipcode,
				Value:    form.Zipcode,
				Type:     constant.PackageUpdateTypeZipcode,
			})
		}

		if form.Country != currentPackage.CountryCode {
			hasUpdatePrice = true
			mapchange["country_code"] = form.Country
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.CountryCode,
				Value:    form.Country,
				Type:     constant.PackageUpdateTypeCountryCode,
			})
		}

		if fmt.Sprintf("%.2f", form.Weight) != fmt.Sprintf("%.2f", currentPackage.Weight) {
			hasUpdatePrice = true
			mapchange["weight"] = form.Weight
			logs = append(logs, entity.PackageAuditLog{
				OldValue: fmt.Sprintf("%.2f", currentPackage.Weight),
				Value:    fmt.Sprintf("%.2f", form.Weight),
				Type:     constant.PackageUpdateTypeWeight,
			})
		}

		if service.ID != currentPackage.ServiceID {
			hasUpdatePrice = true
			mapchange["service_id"] = service.ID

			var newLog = entity.PackageAuditLog{
				OldValue: "",
				Value:    service.Name,
				Type:     constant.PackageUpdateTypeService,
			}

			if currentPackage.Service != nil {
				newLog.OldValue = currentPackage.Service.Name
			}
			logs = append(logs, newLog)
		}

		if currentPackage.Service.Code == constant.ServiceCNCode {
			// Update when pending
			if form.CNProductLink != currentPackage.CNProductLink {
				mapchange["cn_product_link"] = form.CNProductLink
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNProductLink,
					Value:    form.CNProductLink,
					Type:     constant.PackageUpdateTypeCNLabel,
				})
			}

			if form.CNProductPrice != currentPackage.CNProductPrice {
				mapchange["cn_product_price"] = form.CNProductPrice
				newLog := entity.PackageAuditLog{
					OldValue: strconv.FormatFloat(currentPackage.CNProductPrice, 'f', -1, 64),
					Value:    strconv.FormatFloat(form.CNProductPrice, 'f', -1, 64),
					Type:     constant.PackageUpdateExtraFeeCNProduct,
				}
				logs = append(logs, newLog)

				err := h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, []entity.PackageAuditLog{newLog})
				if err != nil {
					h.Logger.Errorf("Update cn extra fee err: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}

			// Update when purchased
			if form.CNShippingFee != currentPackage.CNShippingFee {
				mapchange["cn_shipping_fee"] = form.CNShippingFee
				newLog := entity.PackageAuditLog{
					OldValue: strconv.FormatFloat(currentPackage.CNShippingFee, 'f', -1, 64),
					Value:    strconv.FormatFloat(form.CNShippingFee, 'f', -1, 64),
					Type:     constant.PackageUpdateExtraFeeCNShipping,
				}
				logs = append(logs, newLog)

				err := h.PackageManager.UpdateExtraFee(currentPackage.ID, 0, userID, []entity.PackageAuditLog{newLog})
				if err != nil {
					h.Logger.Errorf("Update cn extra fee err: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}

			if form.CustomCNBarcode != currentPackage.CustomCNBarcode {
				mapchange["custom_cn_barcode"] = form.CustomCNBarcode
				var oldValue string
				if currentPackage.CustomCNBarcode != nil {
					oldValue = *currentPackage.CustomCNBarcode
				}

				var newValue string
				if form.CustomCNBarcode != nil {
					newValue = *form.CustomCNBarcode
				}

				logs = append(logs, entity.PackageAuditLog{
					OldValue: oldValue,
					Value:    newValue,
					Type:     constant.PackageUpdateTypeCNLabel,
				})
			}

			if form.ImageUpload != "" {
				mapchange["cn_invoice_image"] = form.ImageUpload
			}

			if form.CNNote != currentPackage.CNNote {
				mapchange["cn_note"] = form.CNNote
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNNote,
					Value:    form.CNNote,
					Type:     constant.PackageUpdateTypeNote,
				})
			}

			if form.CNProductImage != currentPackage.CNProductImage {
				mapchange["cn_product_image"] = form.CNProductImage
				logs = append(logs, entity.PackageAuditLog{
					OldValue: currentPackage.CNProductImage,
					Value:    form.CNProductImage,
					Type:     constant.PackageUpdateTypeProduct,
				})
			}
		}

		if form.OrderNumber == "" {
			form.OrderNumber = currentPackage.OrderNumber
		}

		if form.OrderNumber != currentPackage.OrderNumber {
			mapchange["order_number"] = form.OrderNumber
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.OrderNumber,
				Value:    form.OrderNumber,
				Type:     constant.PackageUpdateTypeOrderNumber,
			})
		}

		if form.Detail != currentPackage.Detail {
			mapchange["detail"] = form.Detail
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentPackage.Detail,
				Value:    form.Detail,
				Type:     constant.PackageUpdateTypeDetail,
			})
		}

		if form.Length != currentPackage.Length || form.Width != currentPackage.Width || form.Height != currentPackage.Height {
			hasUpdatePrice = true

			currentVolume := cast.ToString(currentPackage.Length) + "x" + cast.ToString(currentPackage.Width) + "x" + cast.ToString(currentPackage.Height)
			updateVolume := cast.ToString(form.Length) + "x" + cast.ToString(form.Width) + "x" + cast.ToString(form.Height)

			mapchange["length"] = form.Length
			mapchange["width"] = form.Width
			mapchange["height"] = form.Height
			logs = append(logs, entity.PackageAuditLog{
				OldValue: currentVolume,
				Value:    updateVolume,
				Type:     constant.PackageUpdateTypeVolume,
			})
		}

		mapchange["include_battery"] = form.IncludeBattery

		if form.PackageName != currentPackage.PackageName {
			mapchange["package_name"] = form.PackageName
		}

		if form.PackageQuantity != currentPackage.PackageQuantity {
			mapchange["package_quantity"] = form.PackageQuantity
		}

		if form.TotalProductPrice != currentPackage.TotalProductPrice {
			mapchange["total_product_price"] = form.TotalProductPrice
		}

		var price float64 = 0
		var priceOutSize float64 = 0
		var isPackageExceed bool
		var isErrorEsPrice bool

		var serviceIDToCalculatePrice int64
		if currentPackage.CustomTiktokBarcode != nil && *currentPackage.CustomTiktokBarcode != "" {
			if constant.IsPriorityService(service.Code) {
				serviceIDToCalculatePrice = 28 // tiktok priority price
			} else {
				serviceIDToCalculatePrice = 25 // tiktok price
			}
		} else {
			serviceIDToCalculatePrice = service.ID
		}
		if hasUpdatePrice || currentPackage.IsPackageExceed {
			price, priceOutSize, err = h.CalculatePrice.Price3(c, userID, serviceIDToCalculatePrice, user.Class, form.Weight, form.Length, form.Height, form.Width, form.Country)
			if err == calculate.ErrorNotService {
				if service.Code != constant.ServiceCNCode {
					c.JSON(http.StatusBadRequest, "Dịch vụ không hợp lệ")
					return
				} else {
					err = nil
				}
			}

			if service.Code == constant.ServiceLABELCode {
				carrier := providers.NewCarrier(service.DomesticCarrier.Code, userID)
				if carrier == nil {
					c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
						Error:    "Bad request",
						Messages: []string{"The service code is invalid"},
					})

					return
				}

				h.Logger.Info("LABEL CODE: ", price)
				cost, err := h.PackageEstimateCost(c, carrier, currentPackage)
				if err == nil {
					price = cost + price/100*cost
				}
				h.Logger.Info("LABEL CODE: ", price, cost, err)
			}

			// if form.Country == "AU" || service.Code == constant.ServiceFBACode {
			if err == calculate.ErrorMaxWeight {
				msg := "Trọng lượng cho phép vượt quá giới hạn"
				if price > 0 {
					msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if err == calculate.ErrorMaxVolume {
				msg := "Kích thước vượt quá giới hạn cho phép"
				if price > 0 {
					msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(price)-1)
				}

				c.JSON(http.StatusBadRequest, msg)
				return
			}

			if err != nil && err != calculate.ErrorMaxVolume && err != calculate.ErrorMaxWeight {
				h.Logger.Errorf("parse body: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if (price != currentPackage.ShippingFee) || isErrorEsPrice {
			if (service.Code == constant.ServiceCNCode && hasUpdatePrice) || price > 0 {
				mapchange["shipping_fee"] = price
				mapchange["is_package_exceed"] = isPackageExceed
			}
		}

		if len(mapchange) == 0 {
			c.JSON(http.StatusOK, UpdatePackageResponse{Success: true})
			return
		}

		currentPackage.Weight = form.Weight
		currentPackage.Length = form.Length
		currentPackage.Width = form.Width
		currentPackage.Height = form.Height
		currentPackage.ActualWeight = form.Weight
		currentPackage.ActualLength = form.Length
		currentPackage.ActualWidth = form.Width
		currentPackage.ActualHeight = form.Height

		var fees []entity.ExtraFee
		var cPrice float64 = price
		if !isErrorEsPrice {
			if !currentPackage.IsPackageExceed && !hasUpdatePrice {
				cPrice = currentPackage.ShippingFee
			}
			fees, err = h.CalculatePrice.PromotionExtras(currentPackage, currentPackage.ExtraFee, cPrice)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if form.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			fees = append(fees, entity.ExtraFee{
				PackageID:      &currentPackage.ID,
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(form.Width, form.Height, form.Length, *service)
		if extraFeeService > 0 {
			fees = append(fees, entity.ExtraFee{
				Amount:         extraFeeService,
				PackageID:      utils.Int64(currentPackage.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}

		err = h.PackageManager.SaveUpdatePackage2(currentPackage.ID, userID, mapchange, logs, priceOutSize, fees, currentPackage.Status)
		if err != nil {
			errDetail := strings.Split(cast.ToString(err), ":")
			if errDetail[1] == " Incorrect string value" {
				c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
				return
			}
			h.Logger.Errorf("update change package: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		options := sqlmanager.PackageQueryOption{
			ID: packageID,
		}

		packages, err := h.PackageManager.GetPackageDetail(options)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}
		packageDTO := &dto.PackageDetailDTO{}
		if err := httputil.Transform(packages, packageDTO); err != nil {
			h.Logger.Errorf("transform package detail error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		packageDTO.ID = packages.ID
		packageDTO.StatusString = constant.MapTextStatusCustomerPackage[packages.Status]
		packageDTO.Status = 0
		packageDTO.ServiceName = packages.Service.Name
		if packages.Tracking != nil {
			packageDTO.TrackingNumber = packages.Tracking.TrackingNumber
		}
		if packages.PackageCode != nil {
			packageDTO.CodePackage = packages.PackageCode.Code
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		deliverLogs, err := h.PackageManager.GetDeliverLogsByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		extraFee, err := h.PackageManager.GetExtraFeeByPkgID(packageID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Deliver Logs %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdatePackageResponse{Package: packageDTO, DeliverLogs: deliverLogs, ExtraFree: extraFee})
	}
}

func (h *PackageHandler) Import() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}
		package_type := c.Query("package_type")

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		file, handlerFile, err := c.Request.FormFile("file")
		if err != nil {
			h.Logger.Error("Can't access file: ", err)
			c.JSON(http.StatusBadRequest, "Can't access file.")
			return
		}
		defer file.Close()

		if !constant.AllowExtensionFileImportPackage[handlerFile.Header.Get("Content-Type")] {
			h.Logger.Error("Format file is not allow: ", handlerFile.Header.Get("Content-Type"))
			c.JSON(http.StatusBadRequest, "File upload sai định dạng.")
			return
		}

		f, err := handlerFile.Open()
		if err != nil {
			h.Logger.Errorf("Error get list state US: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapStates := make(map[string]*entity.State)
		for _, state := range h.USStates {
			key := fmt.Sprintf("%s-%s", state.Country, state.Code)
			mapStates[key] = state

			key = fmt.Sprintf("%s-%s", state.Country, strings.ToUpper(state.Name))
			mapStates[key] = state
		}

		var template *entity.ImportPackageTemplate
		var (
			isValidColumn bool
			packages      []*entity.Package
			importErrors  []ImportPackageError
			total         int64
		)
		if package_type == "CN" {
			isValidColumn, packages, importErrors, total, err = h.ImportChinaPackageXlsx(c, f, user, mapStates, template)
		} else {
			isValidColumn, packages, importErrors, total, err = h.ImportPackageXlsx(c, f, user, mapStates, template)
		}

		if !isValidColumn {
			c.JSON(http.StatusBadRequest, "File upload sai format!")
			return
		}
		if err != nil {
			h.Logger.Error("Error while import package: ", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		//var packageIDsCreated []int64
		for i, pkg := range packages {
			existedPackages, _ := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
				OrderNumber:     pkg.OrderNumber,
				UserID:          userID,
				IgnoreStatusArr: []int64{constant.PackageStatusArchived, constant.PackageStatusCancelled},
			})

			if len(existedPackages) > 0 {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã đơn hàng %s đã tồn tại.", pkg.OrderNumber))
				return
			}

			if packages[i].Service.Code == constant.ServiceLABELCode {
				carrier := providers.NewCarrier(packages[i].Service.DomesticCarrier.Code, user.ID)
				if carrier == nil {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				h.Logger.Info("LABEL CODE: ", packages[i].ShippingFee)
				cost, err := h.PackageEstimateCost(c, carrier, packages[i])
				if err == nil {
					packages[i].ShippingFee = cost + packages[i].ShippingFee/100*cost
				}
				h.Logger.Info("LABEL CODE: ", packages[i].ShippingFee, cost, err)
			}
		}

		if len(packages) > 0 {

			_, err := h.PackageManager.CreatePackages(packages, userID)
			if err != nil {
				errDetail := strings.Split(cast.ToString(err), ":")
				if errDetail[1] == " Incorrect string value" {
					c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
					return
				}

				if strings.Contains(errDetail[1], "Duplicate entry") {
					c.JSON(http.StatusInternalServerError, "Mã nhãn hoặc SKU trùng")
					return
				}

				h.Logger.Error("Error create shipping package: ", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		for _, pkg := range packages {
			if pkg.IsPackageExceed {
				err := h.EstimateExceedPackage.Handle(c, pkg, user)
				if err != nil {
					h.Logger.Error("Error publish message queue check package address: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}
		}

		c.JSON(http.StatusOK, ImportPackageResponse{importErrors, total, cast.ToInt64(len(packages))})
	}
}

func (h *PackageHandler) Export() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		exportForm := &ExportPackageForm{}
		if err := c.ShouldBindJSON(exportForm); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		opts := sqlmanager.PackageQueryOption{
			IDs:        exportForm.IDS,
			UserID:     userID,
			Search:     exportForm.Search,
			SearchBy:   exportForm.SearchBy,
			StartDate:  exportForm.StartDate,
			EndDate:    exportForm.EndDate,
			AlertValue: exportForm.Alert,
			Preload: []string{
				"Service",
				"ExtraFee",
			},
			TrackingStatus: constant.TrackingStatusSuccess,
		}

		for _, status := range exportForm.StatusArr {
			opts.StatusArr = append(opts.StatusArr, constant.MapIntGroupStatusCustomerPackage[status]...)
		}

		packages, err := h.PackageManager.GetPackages(opts)

		if err != nil {
			h.Logger.Errorf("Get list package error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		filePath, err := h.exportCsvPackageXlsx(packages, userID)
		if err != nil {
			h.Logger.Errorf("Error while export csv file %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, ExportPackageReponse{Download: filePath})
	}
}

func (h *PackageHandler) ImportFBA() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusForbidden, "User id required")
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			h.Logger.Errorf("get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		file, handlerFile, err := c.Request.FormFile("file")
		if err != nil {
			h.Logger.Error("Can't access file: ", err)
			c.JSON(http.StatusBadRequest, "Can't access file.")
			return
		}
		defer file.Close()

		if !constant.AllowExtensionFileImportPackage[handlerFile.Header.Get("Content-Type")] {
			h.Logger.Error("Format file is not allow: ", handlerFile.Header.Get("Content-Type"))
			c.JSON(http.StatusBadRequest, "File upload sai định dạng.")
			return
		}

		f, err := handlerFile.Open()
		if err != nil {
			h.Logger.Error("Cant Open file: ", err)
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		usStates, err := h.StateManager.GetStates(sqlmanager.StateOption{
			Countries: []string{"US", "AU"},
		})

		if err != nil {
			h.Logger.Errorf("Error get list state US: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapStates := make(map[string]*entity.State)
		for _, state := range usStates {
			key := fmt.Sprintf("%s-%s", state.Country, state.Code)
			mapStates[key] = state

			key = fmt.Sprintf("%s-%s", state.Country, strings.ToUpper(state.Name))
			mapStates[key] = state
		}

		var template *entity.ImportPackageTemplate

		_, msg, packages, importErrors, total, totalWeight, ratePrice, err := h.ImportFBAPackageXlsx(c, f, user, mapStates, template)
		if msg != "" {
			c.JSON(http.StatusBadRequest, msg)
			return
		}

		if err != nil {
			h.Logger.Error("Error while import package: ", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(packages) > 0 && len(importErrors) == 0 {
			// extras, err := order.FBARateExtras(c, h.UPS, packages, h.Redis)
			// if err != nil {
			// 	h.Logger.Errorf("estimate cost %v", err)
			// 	c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			// 	return
			// }

			for _, pkgs := range packages {
				_, err = h.PackageManager.CreatePackageFBA(pkgs, totalWeight, ratePrice, nil)
				if err != nil {
					errDetail := strings.Split(cast.ToString(err), ":")
					if errDetail[1] == " Incorrect string value" {
						c.JSON(http.StatusInternalServerError, "Kí tự không hợp lệ")
						return
					}
					h.Logger.Error("Error create shipping package: ", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
			}
		}

		c.JSON(http.StatusOK, ImportFbaPackageResponse{importErrors, total, cast.ToInt64(len(packages))})
	}
}

func (h *PackageHandler) Process() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		formData := &ProcessPackageForm{}
		if err := c.ShouldBindJSON(formData); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(formData.Ids) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		// check account package cancel exeeds from user info
		msg, err := authhelper.CheckCancelLimit2(c, userID, h.UserManager, h.Redis)
		if err != nil {
			h.Logger.Error("Get order detail: ", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if msg != "" {
			c.JSON(http.StatusBadRequest, "Tài khoản vượt quá hạn mức tạo đơn hàng")
			return
		}

		// get packages from ids
		pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			IDs: formData.Ids,
		})
		if err == gorm.ErrRecordNotFound || len(pkgs) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		rkey := "package_call_label"
		pkgIDs := []int64{}
		tiktokPkgs := []entity.Package{}
		fbaPkgIDs := []int64{}
		for _, pkg := range pkgs {
			if pkg.UserID != userID {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}

			// check package is created / purchased
			if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusCNPurchased {
				c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
				return
			}

			if pkg.Service.Code == constant.ServiceCNCode {
				c.JSON(http.StatusBadRequest, "Gói hàng Trung Quốc chỉ có thể tạo tracking bởi nhân viên hỗ trợ.")
				return
			}

			// check package is exceed
			if pkg.IsPackageExceed && pkg.ShippingFee == 0 {
				c.JSON(http.StatusBadRequest, "Đơn hàng quá cỡ đang được tính giá.Thử lại sau !")
				return
			}

			if pkg.PackageCode != nil && pkg.PackageCode.Status == constant.PackageCodeDisable {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Mã vận đơn %s đã bị hủy", pkg.PackageCode.Code))
				return
			}

			if pkg.ValidateAddress != constant.PackageValidAddress {
				c.JSON(http.StatusBadRequest, "Địa chỉ không hợp lệ")
				return
			}

			if pkg.Service.Code == constant.ServiceCNCode && pkg.CustomCNBarcode == nil {
				c.JSON(http.StatusBadRequest, "Đơn hàng CN Exclusive cần bổ sung mã vạch tự tạo, hãy cập nhật thông tin đơn hàng")
				return
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn %s không được hỗ trợ", constant.ServiceFBACode))
				return
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				fbaPkgIDs = append(fbaPkgIDs, pkg.ID)
			} else if pkg.Service.Code == constant.ServiceTiktokCode || pkg.CustomTiktokBarcode != nil {
				tiktokPkgs = append(tiktokPkgs, pkg)
			} else {
				pkgIDs = append(pkgIDs, pkg.ID)
			}

			if pkg.Service.Code != constant.ServiceFBACode {
				isCallLabel, err := h.Redis.SIsMember(c, rkey, pkg.ID).Result()
				if err != nil {
					c.JSON(http.StatusBadRequest, constant.MessageServerInternalError)
					return
				}

				if isCallLabel {
					c.JSON(http.StatusBadRequest, fmt.Sprintf("Đơn hàng #%s đang được tạo mã tracking", pkg.OrderNumber))
					return
				}
			}
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// create bill
		bill, err := h.BillManager.GetOrCreateNowBill(user.ID)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.APIResponseMessageServerInternalError)
			return
		}

		peakFee, err := h.BillManager.GetExtraFeeTypeByID(constant.ExtraFeeTypePeak)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get extra peak fee: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var shippingFee float64 = 0
		for i, pkg := range pkgs {
			if peakFee != nil {
				amount := calculate.PeakFee(pkg.Weight)
				if amount > 0 {
					pkgs[i].ExtraFee = append(pkgs[i].ExtraFee, entity.ExtraFee{
						Model: dbgorm.Model{
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						BillID:         utils.Int64(bill.ID),
						PackageID:      utils.Int64(pkg.ID),
						ExtraFeeTypeID: peakFee.ID,
						Description:    peakFee.Name,
						Amount:         amount,
						Status:         constant.ExtraFeeStatusEnable,
					})
				}
			}

			var extraFee float64 = 0
			for _, fee := range pkgs[i].ExtraFee {
				extraFee += fee.Amount
			}

			shippingFee += pkgs[i].ShippingFee + extraFee
		}

		shippingFee = utils.ToFixed(shippingFee, 2)
		// check user balance is greater than shipping fee
		isPackageCN := pkgs[0].Service.Code == constant.ServiceCNCode
		if isPackageCN {
			if user.Balance < shippingFee {
				c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
				return
			}
		} else {
			if user.Balance < shippingFee && (user.UserInfo == nil || user.UserInfo.DebtMaxAmount <= 0) {
				c.JSON(http.StatusInternalServerError, "Số dư ví không đủ. Vui lòng nạp thêm")
				return
			}

			if user.Balance-shippingFee < 0 && user.UserInfo != nil && user.UserInfo.DebtMaxAmount > 0 {
				if err != nil && err != gorm.ErrRecordNotFound {
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				if user.Balance < 0 && user.UserInfo.DebtTime != nil && user.UserInfo.DebtTime.AddDate(0, 0, user.UserInfo.DebtMaxDay).Before(time.Now()) {
					c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá thời hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
					return
				}
				if math.Abs(user.Balance-shippingFee) > user.UserInfo.DebtMaxAmount {
					c.JSON(http.StatusInternalServerError, "Tài khoản của bạn đã nợ quá giới hạn cho phép. Vui lòng nạp thêm tiền để tiếp tục sử dụng dịch vụ")
					return
				}
			}
		}

		if len(fbaPkgIDs) > 0 {
			opt := sqlmanager.CreateBillOption{
				Packages:    pkgs,
				BillID:      bill.ID,
				ShippingFee: shippingFee,
				UserID:      userID,
			}

			_, err = h.BillManager.CreateBill(opt, user, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			err = h.EstimateCost.Handle(c, fbaPkgIDs, 0)
			if err != nil {
				log.Printf("Error publish message queue shipment estimate cost: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			err = h.ShipmentCreateLabelHandler.Handle(c, fbaPkgIDs, false, false, 0)
			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if len(pkgIDs) > 0 {
			for _, id := range pkgIDs {
				_ = h.Redis.SAdd(c, rkey, id).Err()
			}

			err = h.ShipmentCreateLabelHandler.Handle(c, pkgIDs, true, isPackageCN, 0)
			if err != nil {
				h.Logger.Error("Error publish message queue shipment-create-label: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		if len(tiktokPkgs) > 0 {
			for _, pkg := range tiktokPkgs {
				if pkg.Service.Code != constant.ServiceTiktokCode && pkg.CustomTiktokBarcode == nil {
					h.Logger.Errorf("Package's service is not Tiktok")
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}

				trackingNumber, mapRecipientChange, err := utils.GetNslogOcrOutput(*pkg.CustomTiktokBarcode)

				err = h.PackageManager.SaveUpdatePackage2(
					pkg.ID,
					userID,
					mapRecipientChange,
					[]entity.PackageAuditLog{},
					0,
					[]entity.ExtraFee{},
					pkg.Status,
				)
				if err != nil {
					h.Logger.Errorf("Update packages error: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				// create bill
				err, packageCodes := h.PackageManager.CreatePackageCodes([]entity.Package{pkg})
				if err != nil {
					h.Logger.Errorf("Error create package code: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				pkg.PackageCode = packageCodes[0]

				price := pkg.ShippingFee
				for _, fee := range pkg.ExtraFee {
					price += fee.Amount
				}
				billID, err := h.BillManager.GetOrCreateNowBillID(pkg.UserID)
				opt := sqlmanager.CreateBillOption{
					Packages:    []entity.Package{pkg},
					BillID:      billID,
					ShippingFee: price,
					UserID:      pkg.UserID,
				}

				user, err := h.UserManager.GetUserByID(pkg.UserID)
				if err != nil {
					h.Logger.Errorf("Error get user: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				var refundCoupon *entity.ExtraFee
				_, err = h.BillManager.CreateBillWithLabelPromotion(opt, user, refundCoupon, 0)
				if err != nil {
					h.Logger.Errorf("Error create bill: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				texasWarehouse, err := h.WareHouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
					Status: 1,
					State:  "TX",
				})

				trackings := []entity.Tracking{{
					PackageID:      pkg.ID,
					TrackingNumber: trackingNumber,
					LabelURL:       pkg.Label,
					CarrierID:      5, //hard-coded
					Status:         constant.TrackingStatusSuccess,
					Weight:         pkg.Weight,
					Width:          pkg.Width,
					Length:         pkg.Length,
					Height:         pkg.Height,
					ShipmentCost:   pkg.ShippingFee,
					HubID:          &texasWarehouse.ID,
					UserID:         userID,
					CarrierService: "FirstClass",
				}}

				err = h.TrackingManager.CreateTrackingLabeled(trackings)
			}
		}

		c.JSON(http.StatusOK, ProcessPackageResponse{Success: true, PromotionLabel: false})
	}
}

func (h *PackageHandler) Cancel() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		form := &CancelPackageForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if len(form.IDS) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		var cancelMaxAmount float64 = constant.DefaultCancelMaxAMount
		userInfo, err := h.UserManager.GetUserInfoByUserID(userID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get user info %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if err != gorm.ErrRecordNotFound {
			cancelMaxAmount = userInfo.CancelMaxAmount
		}

		rKeyCancel := fmt.Sprintf("%s_%d", "cancel_amount", userID)
		cancelAmount, err := h.Redis.Get(c, rKeyCancel).Float64()
		if err != nil && err != redis.Nil {
			h.Logger.Errorf("get redis : %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		pkgs, err := h.PackageManager.GetPackages(sqlmanager.PackageQueryOption{
			IDs: form.IDS,
		})
		if err == gorm.ErrRecordNotFound || len(pkgs) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get Package Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		refunds := []entity.Package{}
		packageRefunds := []entity.PackageRefund{}
		logs := []entity.PackageDeliverLog{}
		var totalCancelAmount float64 = 0

		for i, pkg := range pkgs {
			if pkg.UserID != userID {
				c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
				return
			}

			if pkg.CustomTiktokBarcode != nil && *pkg.CustomTiktokBarcode != "" {
				refunds = append(refunds, pkg)
				pkgs[i].Status = constant.PackageStatusCancelled
				pkgs[i].OrderID = nil
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressCanceled,
					Type:      constant.PackageDeliverLogTypeCancelled,
					UserID:    &userID,
				})
				continue
			}

			if pkg.Service.Code == constant.ServiceFBACode {
				c.JSON(http.StatusBadRequest, fmt.Sprintf("Service %s không được hỗ trợ", pkg.Service.Name))
				return
			}

			if pkg.Status != constant.PackageStatusCreated && pkg.Status != constant.PackageStatusPendingPickup && pkg.Status != constant.PackageStatusCNPurchased {
				c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
				return
			}

			if pkg.Status == constant.PackageStatusPendingPickup {
				if cancelAmount > cancelMaxAmount {
					c.JSON(http.StatusBadRequest, "Tài khoản vượt quá hạn mức tạo đơn hàng")
					return
				}

				extraFee, err := h.PackageManager.GetTotalExtrafee(pkg.ID)
				if err != nil {
					h.Logger.Errorf("get package extra fee total, %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				// inus and us48 and au and eu are not refunded when cancel
				if pkg.Service.Code != constant.ServiceINUSCode && pkg.Service.Code != constant.ServiceUS48Code && pkg.Service.Code != constant.ServiceACTUSCode && pkg.Service.Code != constant.ServiceAUCode && pkg.Service.Code != constant.ServiceEUCode && pkg.Service.Code != constant.ServiceAUFCode {
					if pkg.Tracking != nil && pkg.Tracking.ID > 0 {
						oldPackageRefunds, err := h.PackageManager.GetListPackagesRefundByPackageID(pkg.ID, constant.PackageRefundPending)
						if err != nil {
							h.Logger.Errorf("get package refunds, %v", err)
							c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
							return
						}

						var amountRefunds float64 = 0
						if len(oldPackageRefunds) > 0 {
							for _, v := range oldPackageRefunds {
								amountRefunds += v.Amount
							}
						}

						amount := pkg.ShippingFee + extraFee - amountRefunds
						if amount > 0 {
							packageRefunds = append(packageRefunds, entity.PackageRefund{
								PackageID: pkg.ID,
								Amount:    amount,
								Status:    constant.PackageRefundPending,
							})
						}

						totalCancelAmount += amount
					} else {
						refunds = append(refunds, pkg)
						totalCancelAmount = totalCancelAmount + pkg.ShippingFee + extraFee
					}
				}
			}

			pkgs[i].Alert = constant.PackageAlertTypeDisable
			if pkg.Status == constant.PackageStatusCreated {
				pkgs[i].Status = constant.PackageStatusArchived
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressArchived,
					Type:      constant.PackageDeliverLogTypeArchived,
					UserID:    &userID,
				})
			} else {
				pkgs[i].Status = constant.PackageStatusCancelled
				pkgs[i].OrderID = nil
				logs = append(logs, entity.PackageDeliverLog{
					PackageID: pkg.ID,
					Status:    constant.DeliverLogTebexpressCanceled,
					Type:      constant.PackageDeliverLogTypeCancelled,
					UserID:    &userID,
				})
			}

			for _, packageProduct := range pkg.PackageProducts {
				err := h.ProductManager.AdjustStock(packageProduct.ProductID, packageProduct.PackageID, packageProduct.Quantity)
				if err != nil {
					h.Logger.Errorf("Error when get product: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				}
			}
		}

		err = h.PackageManager.CancelPackages(pkgs, logs, form.IDS, packageRefunds)
		if err != nil {
			h.Logger.Errorf("Save package error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		h.Logger.Info("refunds: ", len(refunds))
		if len(refunds) > 0 {
			for _, item := range refunds {
				des := fmt.Sprintf("Hoàn tiền cho đơn #%v", item.ID)
				if item.PackageCode != nil {
					des = fmt.Sprintf("Hoàn tiền cho đơn %v", item.PackageCode.Code)
				}

				err = h.PackageRefund.Handle(c, item.ID, userID, des)
				if err != nil {
					h.Logger.Errorf("json marshal %v", err)
				}
			}
		}

		for _, v := range pkgs {
			if v.Tracking == nil {
				continue
			}

			err = h.ShipmentCancelCarrier.Handle(v.Tracking.TrackingNumber)
			if err != nil {
				h.Logger.Errorf("json marshal %v", err)
			}
		}

		amount := cancelAmount + totalCancelAmount
		if amount > 0 {
			t := time.Now().Add(7 * time.Hour)
			end, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s 23:59:59", t.Format("2006-01-02")))
			if err != nil {
				h.Logger.Errorf("time.parse: %s", err)
			}

			expiration := end.Sub(t)
			err = h.Redis.Set(c, rKeyCancel, cancelAmount+totalCancelAmount, expiration).Err()
			if err != nil {
				h.Logger.Errorf("get redis : %s", err)
			}
		}

		c.JSON(http.StatusOK, CancelPackageResponse{true})
	}
}

func (h *PackageHandler) Return() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opts := sqlmanager.PackageQueryOption{
			Limit:      limit,
			Offset:     offset,
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: constant.PackageAlertTypeHubReturn,
			StartDate:  cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:    cast.ToString(c.Request.URL.Query().Get("end_date")),
			UserID:     userID,
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		packages := []dto_models.InfoPackageReturnDTO{}
		err := h.PackageManager.GetPackagesReturn(opts, &packages)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get packages %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetListPackageReturnResponse{packages})
	}
}

func (h *PackageHandler) CountReturn() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
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

		opts := sqlmanager.PackageQueryOption{
			SearchBy:   cast.ToString(c.Request.URL.Query().Get("search_by")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			AlertValue: constant.PackageAlertTypeHubReturn,
			UserID:     userID,
			StartDate:  cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:    cast.ToString(c.Request.URL.Query().Get("end_date")),
			Code:       c.Request.URL.Query().Get("code"),
		}

		if opts.SearchBy != "" && !utils.ValidSlug(opts.SearchBy) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		countAlert, err := h.PackageManager.CountPackagesReturn(opts)
		if err != nil {
			h.Logger.Errorf("Count all status  error, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListPackageReturnResponse{countAlert})
	}
}

func (h *PackageHandler) ImportPackageXlsx(c context.Context, file io.Reader, user *entity.User, mapStates map[string]*entity.State, template *entity.ImportPackageTemplate) (isValidColumn bool, packages []*entity.Package, importErrors []ImportPackageError, total int64, err error) {
	packages = make([]*entity.Package, 0)
	importErrors = make([]ImportPackageError, 0)
	isValidColumn = true
	// header := []string{}

	isInsured, insuredFee, err := h.CalculatePrice.PromotionInsured(user.ID)
	if err != nil {
		return true, packages, importErrors, total, err
	}

	defer func() {
		if r := recover(); r != nil {
			h.Logger.Errorf("Error when process excel file: %v", r)
			isValidColumn = false
		}

		// if len(header) < 15 || len(header) > 16 {
		// 	isValidColumn = false

		// }
	}()

	h.Logger.Info("template", template)

	columnReceiveName := 0
	columnReceivePhone := 1
	columnReceiveAddress1 := 2
	columnReceiveAddress2 := 3
	columnCity := 4
	columnStateCode := 5
	columnZipcode := 6
	columnCountry := 7
	columnSKU := 8
	columnDetail := 9
	columnWeight := 10
	columnLength := 11
	columnWidth := 12
	columnHeight := 13
	columnService := 14
	columnIsTradeMark := 15
	columnBattery := 16
	columnCustomTiktokBarcode := 17
	columnIsEarlyScan := 18
	columnPackageName := 19
	columnPackageQuantity := 20
	columnTotalProductPrice := 21
	var total_column = 22

	f, err := excelize.OpenReader(file)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return false, packages, importErrors, total, err
	}

	err = h.CalculatePrice.LoadPrices(c)
	if err != nil {
		h.Logger.Errorf("Load list prices: %v", err)
		return true, packages, importErrors, total, err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return true, packages, importErrors, total, err
	}
	rows = filterEmptyRowsAndColumns(rows)

	services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
		Status: constant.StatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get list service error: %v", err)
		return false, packages, importErrors, total, err
	}

	var mapNameServices, mapCodeServices = make(map[string]*entity.Service), make(map[string]*entity.Service)
	for _, service := range services {
		mapNameServices[strings.ToUpper(service.Name)] = service
		mapCodeServices[strings.ToUpper(service.Code)] = service
	}

	validator := order.MakeValidator(h.StateManager)
	validator.SetStates(mapStates)

	for indexRow, row := range rows {
		h.Logger.Info("len(row) > total_column: ", indexRow, len(row), total_column)
		if len(row) > total_column || len(row) == 0 {
			continue
		} else if indexRow > 0 && len(row) < total_column {
			for i := len(row); i < total_column; i++ {
				row = append(row, "")
			}
		}

		if indexRow == 0 {
			// header = row
			continue
		}

		total++

		data := &order.PackageResource{}
		data.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveName])
		data.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress1])

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		data.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCity])
		data.Country = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCountry])
		data.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnSKU])
		data.State = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnStateCode])
		data.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnZipcode])
		data.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnDetail])

		if columnReceivePhone >= 0 {
			data.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceivePhone])
		}

		data.Weight = cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		data.Length = cast.ToFloat64(strings.TrimSpace(row[columnLength]))
		data.Width = cast.ToFloat64(strings.TrimSpace(row[columnWidth]))
		data.Height = cast.ToFloat64(strings.TrimSpace(row[columnHeight]))
		data.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnService])

		var values []string
		var messages []string

		if columnBattery > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnBattery])
			if strings.ToUpper(value) == "YES" {
				data.IncludeBattery = true
			} else if strings.ToUpper(value) == "NO" {
				data.IncludeBattery = false
			} else if strings.ToUpper(value) != "" {
				values = append(values, value)
				messages = append(messages, "Cột pin không hợp lệ")
			}
		}

		if columnIsTradeMark > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnIsTradeMark])
			if strings.ToUpper(value) == "YES" {
				data.IsTradeMark = true
			} else if strings.ToUpper(value) == "NO" {
				data.IsTradeMark = false
			} else if strings.ToUpper(value) != "" {
				values = append(values, value)
				messages = append(messages, "Cột Hàng Trademark không hợp lệ")
			}
		}

		if columnCustomTiktokBarcode > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCustomTiktokBarcode])
			if value != "" {
				data.CustomTiktokBarcode = value
			} else if data.Service == "Ship By Tiktok" {
				messages = append(messages, "Mã đơn Tiktok không được để trống")
			}
		}

		if columnIsEarlyScan > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnIsEarlyScan])
			if strings.ToUpper(value) == "YES" {
				data.IsEarlyScan = true
			} else if strings.ToUpper(value) == "NO" || strings.ToUpper(value) == "" {
				data.IsEarlyScan = false
			} else if strings.ToUpper(value) != "" {
				values = append(values, value)
				messages = append(messages, "Cột Scan sớm không hợp lệ")
			}
		}

		if columnPackageName > 0 {
			packageName := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnPackageName])
			if packageName == "" {
				messages = append(messages, "Tên sản phẩm không được để trống")
			} else {
				data.PackageName = packageName
			}
		}

		if columnPackageQuantity > 0 {
			packageQuantity := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnPackageQuantity])
			if packageQuantity == "" {
				messages = append(messages, "Số lượng không được để trống")
			} else if quantity, err := strconv.ParseInt(packageQuantity, 10, 64); err != nil {
				messages = append(messages, fmt.Sprintf("Số lượng không hợp lệ: %v", err))
			} else {
				data.PackageQuantity = quantity
			}
		}

		if columnTotalProductPrice > 0 {
			TotalProductPrice := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnTotalProductPrice])
			if TotalProductPrice == "" {
				messages = append(messages, "Giá đơn hàng không được để trống")
			} else if quantity, err := strconv.ParseFloat(TotalProductPrice, 10); err != nil {
				messages = append(messages, fmt.Sprintf("Giá đơn hàng không hợp lệ: %v", err))
			} else {
				data.TotalProductPrice = quantity
			}
		}

		var service *entity.Service
		data.Service = strings.ToUpper(data.Service)
		if mapCodeServices[data.Service] != nil {
			service = mapCodeServices[data.Service]
			data.ServiceCode = service.Code
		} else if mapNameServices[data.Service] != nil {
			service = mapNameServices[data.Service]
			data.ServiceCode = service.Code
		} else {
			values = append(values, data.Service)
			messages = append(messages, "Dịch vụ không hợp lệ")
		}

		validator.Reset()
		if data.Service == "TIKTOK" || service.Code == constant.ServiceTiktokCode || data.CustomTiktokBarcode != "" {
			validator.ValidateTiktokPkg(data)
		} else {
			validator.Validate(data)
		}

		if data != nil {
			validator.ValidateVolumes(data)
			validator.ValidateFbaServicePackage(data)
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("validate: %v", err)
			return isValidColumn, packages, importErrors, total, err
		}

		if errs := validator.Errors(); len(errs) > 0 {
			messages = append(messages, errs...)
			valueErrors := validator.ValueErrors()
			values = append(values, valueErrors...)
		}

		if len(messages) > 0 && (service == nil || service.ID < 0) {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}

			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		if data.Country != "" {
			if (service.Country == constant.EUState && !utils.ContainsString(constant.EUCountries, data.Country)) || (service.Country != constant.EUState && service.Country != data.Country) {
				values = append(values, data.Service)
				messages = append(messages, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, data.Country))
			}
		}

		if service.Code == constant.ServiceFBACode {
			values = append(values, data.Service)
			messages = append(messages, fmt.Sprintf("Dịch vụ %s không được hỗ trợ", service.Name))
		}

		var shippingFee, extraFee float64
		var serviceIDToCalculatePrice int64
		if data.CustomTiktokBarcode != "" {
			if constant.IsPriorityService(service.Code) {
				serviceIDToCalculatePrice = 28 // tiktok priority price
			} else {
				serviceIDToCalculatePrice = 25 // tiktok price
			}
		} else {
			serviceIDToCalculatePrice = service.ID
		}

		shippingFee, extraFee, err = h.CalculatePrice.Price3(c, user.ID, serviceIDToCalculatePrice, user.Class, data.Weight, data.Length, data.Height, data.Width, data.Country)

		if data.Country == "AU" || service.Code == constant.ServiceFBACode || service.Code == constant.ServiceINUSCode || service.Code == constant.ServiceUS48Code || service.Code == constant.ServiceEUCode {
			if err == calculate.ErrorMaxWeight {
				msg := "Trọng lượng cho phép vượt quá giới hạn"
				if shippingFee > 0 {
					msg = fmt.Sprintf("Trọng lượng không được vượt quá %v grams", math.Ceil(shippingFee)-1)
				}

				values = append(values, cast.ToString(data.Weight))
				messages = append(messages, msg)
			}

			if err == calculate.ErrorMaxVolume {
				msg := "Kích thước vượt quá giới hạn cho phép"
				if shippingFee > 0 {
					msg = fmt.Sprintf("Kích thước không hợp lệ (LxHxW/5 <= %v)", math.Ceil(shippingFee)-1)
				}

				values = append(values, fmt.Sprintf("%.2fx%.2fx%.2f", data.Length, data.Height, data.Width))
				messages = append(messages, msg)
			}
		}

		isPackageExceed := false
		if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume {
			isPackageExceed = true
			shippingFee = 0
		} else if err != nil {
			h.Logger.Errorf("get price service: %v", err)
			return isValidColumn, packages, importErrors, total, err
		}

		if len(messages) > 0 {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}

			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		// default package has valid address
		var validateAddress = constant.PackageValidAddress

		// if data.Country != "AU" {
		// 	if service.DomesticCarrier.ID != 0 {
		// 		if service.DomesticCarrier.Code == providers.CarrierTypeIBBlue {
		// 			validateAddress = constant.PackageInValidAddress
		// 		}
		// 	}
		// }

		pkg := &entity.Package{
			OrderNumber:       data.OrderNumber,
			Detail:            data.Detail,
			Recipient:         data.Recipient,
			PhoneNumber:       data.Phone,
			Address1:          data.Address1,
			Address2:          data.Address2,
			City:              data.City,
			StateCode:         data.State,
			Zipcode:           data.Zipcode,
			CountryCode:       data.Country,
			Weight:            data.Weight,
			Length:            data.Length,
			Width:             data.Width,
			Height:            data.Height,
			UserID:            user.ID,
			ServiceID:         service.ID,
			Service:           service,
			IncludeBattery:    data.IncludeBattery,
			Status:            constant.PackageStatusCreated,
			ValidateAddress:   validateAddress,
			IsPackageExceed:   isPackageExceed,
			PackageName:       data.PackageName,
			PackageQuantity:   data.PackageQuantity,
			TotalProductPrice: data.TotalProductPrice,
		}

		var extraFees []entity.ExtraFee
		if !pkg.IsPackageExceed {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         extraFee,
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
			})
		}

		if pkg.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
			})
		}

		if isInsured {
			pkg.IsInsured = true
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         insuredFee,
				ExtraFeeTypeID: constant.ExtraFeeTypeInsured,
			})
		}

		if data.IsTradeMark {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         1,
				ExtraFeeTypeID: constant.ExtraFeeTypeTradeMark,
			})
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(data.Width, data.Height, data.Length, *service)
		if extraFeeService > 0 {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         extraFeeService,
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
			})
		}

		pkg.IsEarlyScan = data.IsEarlyScan
		if service.Code == constant.ServiceTiktokCode || data.CustomTiktokBarcode != "" {
			pkg.CustomTiktokBarcode = &data.CustomTiktokBarcode

			driveRegex := regexp.MustCompile(`drive\.google\.com/file/d/([^/]+)/`)
			matches := driveRegex.FindStringSubmatch(data.CustomTiktokBarcode)
			if len(matches) > 1 {
				fileID := matches[1]
				data.CustomTiktokBarcode = fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
			}
			filePath, err := order.StoreLabelS3(h.LocalS3, data.CustomTiktokBarcode, "pdf", fmt.Sprintf("tiktok_%s_%03d", pkg.OrderNumber, rand.Intn(1000)))
			if err != nil {
				h.Logger.Errorf("Upload s3 when import %s err: %s", pkg.OrderNumber, err)
				return true, nil, importErrors, total, err
			}
			pkg.Label = filePath
		}

		if pkg.IsEarlyScan {
			tiktokEarlyScanFee := viper.GetFloat64("extra_fees.default_tiktok_early_scan_fee")
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         tiktokEarlyScanFee,
				ExtraFeeTypeID: constant.ExtraFeeTypeEarlyScanTiktok,
			})
		}

		pkg.ShippingFee = shippingFee
		pkg.ExtraFee = extraFees

		fees, err := h.CalculatePrice.PromotionExtras(pkg, pkg.ExtraFee, shippingFee)
		if err != nil {
			return true, packages, importErrors, total, err
		}

		if !pkg.IsPackageExceed {
			pkg.ExtraFee = append(pkg.ExtraFee, fees...)
		}

		packages = append(packages, pkg)
	}

	h.Logger.Info("packages: ", len(packages))
	return isValidColumn, packages, importErrors, total, nil
}

func (h *PackageHandler) ImportChinaPackageXlsx(c context.Context, file io.Reader, user *entity.User, mapStates map[string]*entity.State, template *entity.ImportPackageTemplate) (isValidColumn bool, packages []*entity.Package, importErrors []ImportPackageError, total int64, err error) {
	packages = make([]*entity.Package, 0)
	importErrors = make([]ImportPackageError, 0)
	isValidColumn = true

	isInsured, insuredFee, err := h.CalculatePrice.PromotionInsured(user.ID)
	if err != nil {
		return true, packages, importErrors, total, err
	}

	defer func() {
		if r := recover(); r != nil {
			h.Logger.Errorf("Error when process excel file: %v", r)
			isValidColumn = false
		}
	}()

	h.Logger.Info("template", template)

	columnReceiveName := 0
	columnReceivePhone := 1
	columnReceiveAddress1 := 2
	columnReceiveAddress2 := 3
	columnCity := 4
	columnStateCode := 5
	columnZipcode := 6
	columnCountry := 7
	columnSKU := 8
	columnDetail := 9
	columnWeight := 10
	columnLength := 11
	columnWidth := 12
	columnHeight := 13
	columnIsTradeMark := 14
	columnService := 15
	columnIsPurchased := 16
	columnCNProductLink := 17
	columnCNProductImage := 18
	columnCNProductPrice := 19
	columnCNNote := 20
	columnCNShippingToVN := 21
	columnCustomCNBarcode := 22
	columnPackageName := 23
	columnPackageQuantity := 24
	columnTotalProductPrice := 25
	columnCustomTiktokBarcode := 26
	columnIsEarlyScan := 27
	var total_column = 28

	f, err := excelize.OpenReader(file)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return false, packages, importErrors, total, err
	}

	err = h.CalculatePrice.LoadPrices(c)
	if err != nil {
		h.Logger.Errorf("Load list prices: %v", err)
		return true, packages, importErrors, total, err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return true, packages, importErrors, total, err
	}
	rows = filterEmptyRowsAndColumns(rows)

	if err != nil {
		h.Logger.Errorf("Get list service error: %v", err)
		return false, packages, importErrors, total, err
	}

	validator := order.MakeValidator(h.StateManager)
	validator.SetStates(mapStates)

	for indexRow, row := range rows {
		h.Logger.Info("len(row) > total_column: ", indexRow, len(row), total_column)
		if len(row) > total_column || len(row) == 0 {
			continue
		} else if indexRow > 0 && len(row) < total_column {
			for i := len(row); i < total_column; i++ {
				row = append(row, "")
			}
		}

		if indexRow == 0 {
			continue
		}

		total++

		data := &order.PackageResource{}
		data.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveName])
		data.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress1])

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		data.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCity])
		data.Country = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCountry])
		data.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnSKU])
		data.State = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnStateCode])
		data.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnZipcode])
		data.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnDetail])

		if columnReceivePhone >= 0 {
			data.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceivePhone])
		}

		data.Weight = cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		data.Length = cast.ToFloat64(strings.TrimSpace(row[columnLength]))
		data.Width = cast.ToFloat64(strings.TrimSpace(row[columnWidth]))
		data.Height = cast.ToFloat64(strings.TrimSpace(row[columnHeight]))
		data.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnService])

		var values []string
		var messages []string
		if data.Service != constant.ServiceCNCode {
			messages = append(messages, "Dịch vụ chỉ cho phép CN")
		}

		if columnIsTradeMark > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnIsTradeMark])
			if strings.ToUpper(value) == "YES" {
				data.IsTradeMark = true
			} else if strings.ToUpper(value) == "NO" {
				data.IsTradeMark = false
			} else if strings.ToUpper(value) != "" {
				values = append(values, value)
				messages = append(messages, "Cột Hàng Trademark không hợp lệ")
			}
		}

		isPurchasedTextValue := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnIsPurchased])
		if isPurchasedTextValue == "Hàng đã mua" {
			data.CNIsPurchased = true
		} else if isPurchasedTextValue == "Hàng nhờ mua" {
			data.CNIsPurchased = false
		} else {
			messages = append(messages, "Loại dịch vụ CN không hợp lệ")
		}

		data.CNProductLink = cast.ToString(strings.TrimSpace(row[columnCNProductLink]))
		data.CNProductImage = cast.ToString(strings.TrimSpace(row[columnCNProductImage]))
		data.CNProductPrice = cast.ToFloat64(strings.TrimSpace(row[columnCNProductPrice]))
		data.CNNote = cast.ToString(strings.TrimSpace(row[columnCNNote]))
		data.CNShippingFee = cast.ToFloat64(strings.TrimSpace(row[columnCNShippingToVN]))

		customCNBarcodeValue := cast.ToString(strings.TrimSpace(row[columnCustomCNBarcode]))
		if customCNBarcodeValue != "" {
			data.CustomCNBarcode = &customCNBarcodeValue
		} else {
			data.CustomCNBarcode = nil
		}

		if columnPackageName > 0 {
			packageName := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnPackageName])
			if packageName == "" {
				messages = append(messages, "Tên sản phẩm không được để trống")
			} else {
				data.PackageName = packageName
			}
		}

		if columnPackageQuantity > 0 {
			packageQuantity := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnPackageQuantity])
			if packageQuantity == "" {
				messages = append(messages, "Số lượng không được để trống")
			} else if quantity, err := strconv.ParseInt(packageQuantity, 10, 64); err != nil {
				messages = append(messages, fmt.Sprintf("Số lượng không hợp lệ: %v", err))
			} else {
				data.PackageQuantity = quantity
			}
		}

		if columnTotalProductPrice > 0 {
			TotalProductPrice := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnTotalProductPrice])
			if TotalProductPrice == "" {
				messages = append(messages, "Giá đơn hàng không được để trống")
			} else if quantity, err := strconv.ParseFloat(TotalProductPrice, 10); err != nil {
				messages = append(messages, fmt.Sprintf("Giá đơn hàng không hợp lệ: %v", err))
			} else {
				data.TotalProductPrice = quantity
			}
		}

		data.CustomTiktokBarcode = cast.ToString(strings.TrimSpace(row[columnCustomTiktokBarcode]))
		if columnIsEarlyScan > 0 {
			value := string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnIsEarlyScan])
			if strings.ToUpper(value) == "YES" {
				data.IsEarlyScan = true
			} else if strings.ToUpper(value) == "NO" || strings.ToUpper(value) == "" {
				data.IsEarlyScan = false
			} else if strings.ToUpper(value) != "" {
				values = append(values, value)
				messages = append(messages, "Cột Scan sớm không hợp lệ")
			}
		}

		validator.Reset()
		// Nếu đơn CN có Label tiktok, không cần validate
		if data.CustomTiktokBarcode == "" {
			validator.ValidateChinaPackage(data)
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("validate: %v", err)
			return isValidColumn, packages, importErrors, total, err
		}

		if errs := validator.Errors(); len(errs) > 0 {
			messages = append(messages, errs...)
			valueErrors := validator.ValueErrors()
			values = append(values, valueErrors...)
		}

		var shippingFee, extraFee float64
		serviceCN, err := h.ServiceManager.GetServiceByCode(data.Service)
		var serviceIDToCalculatePrice int64
		if data.CustomTiktokBarcode != "" {
			if constant.IsPriorityService(serviceCN.Code) {
				serviceIDToCalculatePrice = 28 // tiktok priority price
			} else {
				serviceIDToCalculatePrice = 25 // tiktok price
			}
		} else {
			serviceIDToCalculatePrice = serviceCN.ID
		}
		shippingFee, extraFee, err = h.CalculatePrice.Price3(c, user.ID, serviceIDToCalculatePrice, user.Class, data.Weight, data.Length, data.Height, data.Width, data.Country)

		isPackageExceed := false
		if err == calculate.ErrorMaxWeight || err == calculate.ErrorMaxVolume {
			isPackageExceed = true
			shippingFee = 0
		} else if err == calculate.ErrorNotService && data.Weight == 0 {
			// Đơn CN có khả năng không nhập cân nặng
		} else if err != nil {
			h.Logger.Errorf("get price service: %v", err)
			return isValidColumn, packages, importErrors, total, err
		}

		if len(messages) > 0 {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}

			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		// default package has valid address
		var validateAddress = constant.PackageValidAddress

		pkg := &entity.Package{
			OrderNumber:       data.OrderNumber,
			Detail:            data.Detail,
			Recipient:         data.Recipient,
			PhoneNumber:       data.Phone,
			Address1:          data.Address1,
			Address2:          data.Address2,
			City:              data.City,
			StateCode:         data.State,
			Zipcode:           data.Zipcode,
			CountryCode:       data.Country,
			Weight:            data.Weight,
			Length:            data.Length,
			Width:             data.Width,
			Height:            data.Height,
			UserID:            user.ID,
			ServiceID:         serviceCN.ID,
			Service:           serviceCN,
			IncludeBattery:    data.IncludeBattery,
			Status:            constant.PackageStatusCreated,
			ValidateAddress:   validateAddress,
			IsPackageExceed:   isPackageExceed,
			PackageName:       data.PackageName,
			PackageQuantity:   data.PackageQuantity,
			TotalProductPrice: data.TotalProductPrice,

			CNIsPurchased:   data.CNIsPurchased,
			CNProductLink:   data.CNProductLink,
			CNProductPrice:  data.CNProductPrice,
			CNProductImage:  data.CNProductImage,
			CNShippingFee:   data.CNShippingFee,
			CustomCNBarcode: data.CustomCNBarcode,
			CNNote:          data.CNNote,

			CustomTiktokBarcode: &data.CustomTiktokBarcode,
			IsEarlyScan:         data.IsEarlyScan,
		}
		var extraFees []entity.ExtraFee

		if data.IsTradeMark {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         1,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeTradeMark,
			})
		}

		pkg.IsEarlyScan = data.IsEarlyScan
		if data.CustomTiktokBarcode != "" {
			pkg.CustomTiktokBarcode = &data.CustomTiktokBarcode

			driveRegex := regexp.MustCompile(`drive\.google\.com/file/d/([^/]+)/`)
			matches := driveRegex.FindStringSubmatch(data.CustomTiktokBarcode)
			if len(matches) > 1 {
				fileID := matches[1]
				data.CustomTiktokBarcode = fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
			}
			filePath, err := order.StoreLabelS3(h.LocalS3, data.CustomTiktokBarcode, "pdf", fmt.Sprintf("tiktok_%s_%03d", pkg.OrderNumber, rand.Intn(1000)))
			if err != nil {
				h.Logger.Errorf("Upload s3 when import %s err: %s", pkg.OrderNumber, err)
				return true, nil, importErrors, total, err
			}
			pkg.Label = filePath
		}

		if pkg.IsEarlyScan {
			tiktokEarlyScanFee := viper.GetFloat64("extra_fees.default_tiktok_early_scan_fee")
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         tiktokEarlyScanFee,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeEarlyScanTiktok,
			})
		}

		if data.CNIsPurchased == false {
			pkg.CNProductLink = data.CNProductLink
			pkg.CNProductPrice = data.CNProductPrice
			pkg.CustomCNBarcode = data.CustomCNBarcode
		} else if data.CNIsPurchased == true {
			pkg.CNShippingFee = data.CNShippingFee
			pkg.CustomCNBarcode = data.CustomCNBarcode

			pkg.Status = constant.PackageStatusCNPurchased
		}

		if pkg.Status == constant.PackageStatusCreated && pkg.CNProductPrice != 0 {
			var cnPricePercentage float64
			if pkg.CNProductPrice > 200 {
				cnPricePercentage = viper.GetFloat64("extra_fees.cn_high_price_percentage")
			} else {
				cnPricePercentage = viper.GetFloat64("extra_fees.cn_low_price_percentage")
			}
			minProxyPrice := viper.GetFloat64("extra_fees.cn_min_proxy_buying_fee")
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         math.Max(pkg.CNProductPrice*cnPricePercentage, minProxyPrice),
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeChinaProductPercentage,
			})

			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         pkg.CNProductPrice,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeChinaProduct,
			})

		} else if pkg.Status == constant.PackageStatusCNPurchased && pkg.CNShippingFee != 0 {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         pkg.CNShippingFee,
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeChinaShipping,
			})
		}

		defaultCNShippingFeeToVN := viper.GetFloat64("extra_fees.default_cn_ship_to_vn_fee")
		extraFees = append(extraFees, entity.ExtraFee{
			Amount:         defaultCNShippingFeeToVN,
			PackageID:      utils.Int64(pkg.ID),
			ExtraFeeTypeID: constant.ExtraFeeTypeCNShippingToVN,
		})
		defaultCNHandlingFee := viper.GetFloat64("extra_fees.default_cn_handling_fee") // Phí handling + active tracking
		extraFees = append(extraFees, entity.ExtraFee{
			Amount:         defaultCNHandlingFee,
			PackageID:      utils.Int64(pkg.ID),
			ExtraFeeTypeID: constant.ExtraFeeTypeHandling,
		})

		if !pkg.IsPackageExceed {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         extraFee,
				ExtraFeeTypeID: constant.ExtraFeeTypeOutSize,
			})
		}

		if pkg.IncludeBattery {
			fee := h.CalculatePrice.GetExtraFeeBaterry()
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         fee,
				ExtraFeeTypeID: constant.ExtraFeeTypeBattery,
			})
		}

		if isInsured {
			pkg.IsInsured = true
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         insuredFee,
				ExtraFeeTypeID: constant.ExtraFeeTypeInsured,
			})
		}

		extraFeeService := h.CalculatePrice.GetServiceExtraFeee(data.Width, data.Height, data.Length, *serviceCN)
		if extraFeeService > 0 {
			extraFees = append(extraFees, entity.ExtraFee{
				Amount:         extraFeeService,
				ExtraFeeTypeID: constant.ExtraFeeTypeService,
			})
		}

		pkg.ShippingFee = shippingFee
		pkg.ExtraFee = extraFees

		fees, err := h.CalculatePrice.PromotionExtras(pkg, pkg.ExtraFee, shippingFee)
		if err != nil {
			return true, packages, importErrors, total, err
		}

		if !pkg.IsPackageExceed {
			pkg.ExtraFee = append(pkg.ExtraFee, fees...)
		}

		packages = append(packages, pkg)
	}

	h.Logger.Info("packages: ", len(packages))
	return isValidColumn, packages, importErrors, total, nil
}

func filterEmptyRowsAndColumns(rows [][]string) [][]string {
	if len(rows) == 0 {
		return rows
	}

	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	nonEmptyCols := make([]bool, colCount)

	for _, row := range rows {
		for colIdx, cell := range row {
			if colIdx < colCount && cell != "" {
				nonEmptyCols[colIdx] = true
			}
		}
	}

	var filtered [][]string
	for _, row := range rows {
		isRowEmpty := true
		for _, cell := range row {
			if cell != "" {
				isRowEmpty = false
				break
			}
		}

		if isRowEmpty {
			continue
		}

		var newRow []string
		for colIdx, cell := range row {
			if colIdx < colCount && nonEmptyCols[colIdx] {
				newRow = append(newRow, cell)
			}
		}

		filtered = append(filtered, newRow)
	}

	return filtered
}

func (h *PackageHandler) ImportFBAPackageXlsx(c context.Context, file io.Reader, user *entity.User, mapStates map[string]*entity.State, template *entity.ImportPackageTemplate) (isValidColumn bool, msg string, packages map[string][]*entity.Package, importErrors []ImportPackageError, total int64, totalWeight, ratePrice float64, err error) {
	packages = make(map[string][]*entity.Package, 0)
	importErrors = make([]ImportPackageError, 0)
	isValidColumn = true

	var service *entity.Service

	defer func() {
		if r := recover(); r != nil {
			h.Logger.Errorf("Error when process excel file: %v", r)
			isValidColumn = false
		}

	}()

	columnReceiveName := 0
	columnReceivePhone := 1
	columnReceiveAddress1 := 2
	columnReceiveAddress2 := 3
	columnCity := 4
	columnStateCode := 5
	columnZipcode := 6
	columnCountry := 7
	columnSKU := 8
	columnDetail := 9
	columnWeight := 10
	columnLength := 11
	columnWidth := 12
	columnHeight := 13
	columnService := 14
	var total_column = 15

	f, err := excelize.OpenReader(file)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
		Status: constant.StatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get list service error: %v", err)
		return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
	}

	var mapNameServices, mapCodeServices = make(map[string]*entity.Service), make(map[string]*entity.Service)
	for _, service := range services {
		mapNameServices[strings.ToUpper(service.Name)] = service
		mapCodeServices[strings.ToUpper(service.Code)] = service
	}

	validator := order.MakeValidator(h.StateManager)
	validator.SetStates(mapStates)

	for indexRow, row := range rows {
		h.Logger.Info("len(row) > 15: ", indexRow, len(row), total_column)
		if len(row) > 15 || len(row) == 0 {
			continue
		} else if indexRow > 0 && len(row) < total_column {
			for i := len(row); i < total_column; i++ {
				row = append(row, "")
			}
		}

		if indexRow == 0 {
			continue
		}

		total++

		data := &order.PackageResource{}
		data.Recipient = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveName])
		data.Address1 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress1])

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		data.City = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCity])
		data.Country = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnCountry])
		data.OrderNumber = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnSKU])
		data.State = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnStateCode])
		data.Zipcode = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnZipcode])
		data.Detail = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnDetail])

		if columnReceivePhone >= 0 {
			data.Phone = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceivePhone])
		}

		if columnReceiveAddress2 >= 0 {
			data.Address2 = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnReceiveAddress2])
		}

		w := cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		w = cast.ToFloat64(int(math.Ceil(w / 1000)))
		data.Weight = w * 1000
		data.Length = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnLength])))))
		data.Width = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnWidth])))))
		data.Height = cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnHeight])))))
		data.Service = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(row[columnService])

		h.Logger.Info("aaa: ", strings.TrimSpace(row[columnWeight]), cast.ToFloat64(int(math.Ceil(cast.ToFloat64(strings.TrimSpace(row[columnWeight]))))))
		var values []string
		var messages []string

		data.Service = strings.ToUpper(data.Service)
		data.ServiceCode = constant.ServiceFBACode
		if mapCodeServices[data.Service] != nil {
			service = mapCodeServices[data.Service]
		} else if mapNameServices[data.Service] != nil {
			service = mapNameServices[data.Service]
		} else {
			values = append(values, data.Service)
			messages = append(messages, "Dịch vụ không hợp lệ")
		}

		if len(messages) > 0 && (service == nil || service.ID < 0) {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}

			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		validator.Reset()
		validator.Validate(data)

		if data != nil {
			validator.ValidateVolumes(data)
			validator.ValidateFbaServicePackage(data)
		}

		if err := validator.Error(); err != nil {
			h.Logger.Errorf("validate: %v", err)
			return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
		}

		if errs := validator.Errors(); len(errs) > 0 {
			messages = append(messages, errs...)
			valueErrors := validator.ValueErrors()
			values = append(values, valueErrors...)
		}

		if service.Country != data.Country {
			values = append(values, data.Service)
			messages = append(messages, fmt.Sprintf("Dịch vụ %s không hỗ trợ %s", service.Name, data.Country))
		}
		if service.Code != constant.ServiceFBACode {
			values = append(values, data.Service)
			messages = append(messages, "Mã dịch vụ không hợp lệ")
		}

		pkg := &entity.Package{
			OrderNumber:     data.OrderNumber,
			Detail:          data.Detail,
			Recipient:       data.Recipient,
			PhoneNumber:     data.Phone,
			Address1:        data.Address1,
			Address2:        data.Address2,
			City:            data.City,
			StateCode:       data.State,
			Zipcode:         data.Zipcode,
			CountryCode:     data.Country,
			Weight:          data.Weight,
			Length:          data.Length,
			Width:           data.Width,
			Height:          data.Height,
			UserID:          user.ID,
			ServiceID:       service.ID,
			Status:          constant.PackageStatusCreated,
			ValidateAddress: constant.PackageValidAddress,
		}

		h.Logger.Infof("fbapkg: %v - %v - %v - %v", pkg.OrderNumber, pkg.Weight, pkg.Height, pkg.Length, pkg.Width)
		key := fmt.Sprintf("%v_%v_%v_%v_%v", pkg.Address1, pkg.City, pkg.StateCode, pkg.Zipcode, pkg.CountryCode)
		if _, ok := packages[key]; !ok {
			packages[key] = make([]*entity.Package, 0)
		}

		// if len(packages) > 0 {
		// 	if pkg.Address1 != packages[0].Address1 {
		// 		values = append(values, pkg.Address1)
		// 		messages = append(messages, fmt.Sprintf("Địa chỉ không trùng nhau"))
		// 	}

		// 	if pkg.City != packages[0].City {
		// 		values = append(values, pkg.City)
		// 		messages = append(messages, fmt.Sprintf("Thành phố không trùng nhau"))
		// 	}

		// 	if pkg.StateCode != packages[0].StateCode {
		// 		values = append(values, pkg.StateCode)
		// 		messages = append(messages, fmt.Sprintf("Mã vùng không trùng nhau"))
		// 	}

		// 	if pkg.Zipcode != packages[0].Zipcode {
		// 		values = append(values, pkg.Zipcode)
		// 		messages = append(messages, fmt.Sprintf("Mã bưu điện không trùng nhau"))
		// 	}

		// 	if pkg.CountryCode != packages[0].CountryCode {
		// 		values = append(values, pkg.CountryCode)
		// 		messages = append(messages, fmt.Sprintf("Mã quốc gia không trùng nhau"))
		// 	}
		// }

		if len(messages) > 0 {
			for i := range messages {
				messages[i] = fmt.Sprintf("%s\n", messages[i])
			}
			importErrors = append(importErrors, ImportPackageError{
				Line:     int64(indexRow) + 1,
				Value:    values,
				Messages: messages,
			})
			continue
		}

		packages[key] = append(packages[key], pkg)
		wp, _ := calculate.CalcFBAPriceWeight(pkg.Weight, pkg.Length, pkg.Height, pkg.Width)
		totalWeight += wp
	}

	if len(packages) > 0 && len(importErrors) < 1 {
		for _, pkgs := range packages {
			var pkgWeights float64
			for _, pkg := range pkgs {
				wp, _ := calculate.CalcFBAPriceWeight(pkg.Weight, pkg.Length, pkg.Height, pkg.Width)
				pkgWeights += wp
			}

			h.Logger.Info("pkgW: ", pkgWeights)
			ratePrice, _, err = h.CalculatePrice.GetRatePriceByTotalWeight(c, pkgWeights, service.ID, user.Class)
			if err != nil {
				return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
			}
		}

		var maxW float64
		ratePrice, maxW, err = h.CalculatePrice.GetRatePriceByTotalWeight(c, totalWeight, service.ID, user.Class)
		h.Logger.Infof("bbb: %v - %v - %v", ratePrice, maxW, totalWeight)
		if err != nil {
			return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
		}
		if ratePrice == 0 {
			if maxW > 0 {
				return isValidColumn, fmt.Sprintf("Khối lượng lô hàng không được vượt quá %.2f kg", maxW/constant.KgToGram), packages, importErrors, total, totalWeight, ratePrice, ErrorMaxWeight
			} else {
				return isValidColumn, "Khối lượng lô hàng không hợp lệ", packages, importErrors, total, totalWeight, ratePrice, ErrorMaxWeight
			}
		} else {
			for i, pkgs := range packages {
				for j := range pkgs {
					shippingFee := h.CalculatePrice.GetFbaPackagePrice(ratePrice, packages[i][j].Weight, packages[i][j].Width, packages[i][j].Length, packages[i][j].Height)
					shippingFee = cast.ToFloat64(int(math.Ceil(shippingFee)))
					packages[i][j].ShippingFee = shippingFee
				}
			}
		}
	}

	return isValidColumn, "", packages, importErrors, total, totalWeight, ratePrice, err
}

func (h *PackageHandler) exportCsvPackageXlsx(packages []entity.Package, userID int64) (string, error) {
	header := []string{
		"Ananbay tracking",
		"Last mile tracking",
		"Chi tiết hàng hóa",
		"Ngày tạo",
		"Tên người nhận",
		"SĐT người nhận",
		"Địa chỉ nhận",
		"Địa chỉ nhận (phụ)",
		"Thành phố",
		"Mã vùng",
		"Mã bưu điện",
		"Mã quốc gia",
		"Mã đơn hàng",
		"Trọng lượng",
		"Dài",
		"Rộng",
		"Cao",
		"Dịch vụ",
		"Tổng cước",
		"Phí giao",
		"Phí phát sinh",
		"Trạng thái",
		"SKU",
	}

	f1 := excelize.NewFile()

	for index, item := range header {
		colStr := fmt.Sprintf("%s1", toCharStr(index+1))
		err := f1.SetCellValue("Sheet1", colStr, item)
		if err != nil {
			h.Logger.Errorf("Write xlsx header error: %v", err)
		}
	}

	fileName := fmt.Sprintf("danh_sach_van_don_%d_%02d_%02d.xlsx", time.Now().Day(), int(time.Now().Month()), time.Now().Year())
	f, err := os.Create(fileName)
	if err != nil {
		h.Logger.Errorf("Create file tmp: %v", err)
		return "", err
	}
	defer f.Close()
	defer os.Remove(fileName)
	writer := csv.NewWriter(f)
	defer writer.Flush()

	for indexRow, Package := range packages {
		if Package.Service == nil {
			fmt.Println(Package.ID)

		}
		var trackingNumber string
		if Package.Tracking != nil {
			trackingNumber = Package.Tracking.TrackingNumber
		}
		tmp := []string{}
		pkgCode := ""
		if Package.PackageCode != nil {
			pkgCode = Package.PackageCode.Code
		}

		if Package.Status == constant.PackageStatusCreated {
			pkgCode = ""
			trackingNumber = ""
		}

		tmp = append(tmp, []string{
			pkgCode,
			trackingNumber,
			Package.Detail,
			cast.ToString(Package.CreatedAt.Add(7 * time.Hour).Format("2006-01-02T15:04:05")),
		}...)

		var extraFee float64
		for _, fee := range Package.ExtraFee {
			extraFee += fee.Amount
		}

		if Package.Status == constant.PackageStatusCreated {
			if Package.ActualWeight > 0 {
				extraFee += calculate.PeakFee(Package.ActualWeight)
			} else {
				extraFee += calculate.PeakFee(Package.Weight)
			}
		}

		sku := "N/A"

		tmp = append(tmp, []string{
			Package.Recipient,
			Package.PhoneNumber,
			Package.Address1,
			Package.Address2,
			Package.City,
			Package.StateCode,
			Package.Zipcode,
			Package.CountryCode,
			Package.OrderNumber,
			cast.ToString(Package.Weight),
			cast.ToString(Package.Length),
			cast.ToString(Package.Width),
			cast.ToString(Package.Height),
			Package.Service.Name,
			fmt.Sprintf("%.2f", Package.ShippingFee+extraFee),
			fmt.Sprintf("%.2f", Package.ShippingFee),
			fmt.Sprintf("%.2f", extraFee),
			cast.ToString(constant.MapTextStatusCustomerPackage[Package.Status]),
			sku,
		}...)

		for indexCol, item := range tmp {
			cellStr := fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)
			err := f1.SetCellValue("Sheet1", cellStr, item)
			if err != nil {
				h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
			}
		}
	}

	if err := f1.SaveAs(fileName); err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}

	filePath := fmt.Sprintf("exports/%s%s", file.MakeUploadUserIDPath(userID, ""), fileName)

	buffer, err := ioutil.ReadFile(fileName)
	if err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}
	f.Read(buffer)

	contentType := constant.ContentTypeXlsxExcel2007Above
	bucketExport := viper.GetString("bucket.export_packages")
	err = h.LocalS3.UploadFile(bytes.NewBuffer(buffer), filePath, bucketExport, contentType)

	if err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}

	return filePath, nil
}

func (h *PackageHandler) PackageEstimateCost(c context.Context, carrier providers.Carrier, pkg *entity.Package) (float64, error) {
	if carrier == nil {
		if pkg.CountryCode == "AU" {
			carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
		} else {
			carrierCode, err := h.CreateLabel.GetCarrierCode(c, *pkg, pkg.UserID, "")
			if err != nil {
				return 0, err
			}

			if carrierCode != "" {
				carrier = providers.NewCarrier(carrierCode, pkg.UserID)
			} else {
				carrier = providers.NewCarrier(pkg.Service.DomesticCarrier.Code, pkg.UserID)
			}
		}
	}

	isWl := utils.IsStateWhiteList(pkg.UserID)
	igState := ""
	if isWl {
		igState = "CA"
	}

	wareHouses, err := h.WareHouseManager.GetWareHouses(sqlmanager.OptionWareHouse{
		Type:        constant.WareHouseTypeInternational,
		IgnoreState: igState,
		Status:      constant.WareHouseStatusActive,
	})

	if err != nil {
		h.Logger.Errorf("Get estimate warehouses error, %v", err)
		return 0, err
	}

	clone := &entity.Package{}
	if err := utils.DeepCopy(pkg, clone); err != nil {
		h.Logger.Errorf("copy deep, %v", err)
		return 0, err
	}

	var pkgWhs []entity.PackageWarehouseCost
	var wg sync.WaitGroup
	var m sync.Mutex
	apiCost := make(map[int64]float64, 0)
	for _, wareHouse := range wareHouses {
		wg.Add(1)

		go func(wareHouse entity.Warehouse) {
			defer wg.Done()

			if wareHouse.Country != pkg.CountryCode {
				return
			}

			cost := entity.PackageWarehouseCost{
				HubID:     wareHouse.ID,
				Warehouse: &wareHouse,
				OrgCost:   0,
			}

			res, msg, err := order.EstimateCost(carrier, clone, wareHouse)
			if msg != "" {
				h.Logger.Errorf("Estimate cost org err: %v", msg)
				return
			}

			if err != nil {
				h.Logger.Errorf("Estimate cost org err: %v", err)
				return
			}

			cost.OrgCost = res.TotalCost + wareHouse.HandlingFee
			cost.Zone = res.Zone

			if wareHouse.Status == constant.WareHouseStatusActive {
				m.Lock()
				pkgWhs = append(pkgWhs, cost)
				apiCost[wareHouse.ID] = res.TotalCost
				m.Unlock()
			}

			err = h.WareHouseManager.CreateEstimateCost(&cost)
			if err != nil {
				h.Logger.Errorf("Create estimate cost err: %v", err)
			}
		}(wareHouse)

	}

	wg.Wait()

	if len(pkgWhs) == 0 {
		return 0, errors.New("can't find lowest cost warehouse")
	}

	minW := pkgWhs[0]

	for _, wh := range pkgWhs {
		if wh.OrgCost < minW.OrgCost {
			minW = wh
		}
	}
	return apiCost[minW.HubID], nil
}
