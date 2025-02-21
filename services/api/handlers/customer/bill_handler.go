package customer

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/invoice"
	bevn "tebexpressapi/pkg/invoice/font/BeVn"
	bevnbold "tebexpressapi/pkg/invoice/font/BeVnBold"
	bevnitalic "tebexpressapi/pkg/invoice/font/BeeVnItalic"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/file"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BillHandler struct {
	Logger  *zap.SugaredLogger
	LocalS3 storage.S3

	BillManager *sqlmanager.BillManager
	UserManager *sqlmanager.UserManager
	ProductManager *sqlmanager.ProductManager
}

type BillCountReponse struct {
	Count int64 `json:"count"`
}

type ExportBillsForm struct {
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	PackageFee    string `json:"package"`
	ExtraFee      string `json:"extra"`
	CommissionFee string `json:"commission"`
	ExportType    string `json:"export_type"`
}

type BillListReponse struct {
	Bills []entity.Bill `json:"bills"`
}

type ExtraFee struct {
	ID          int64     `json:"id"`
	TypeName    string    `json:"type_name"`
	PackageID   int64     `json:"package_id"`
	ShipmentID  int64     `json:"shipment_id"`
	PackageCode string    `json:"package_code"`
	BillID      *int64    `json:"bill_id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}
type GetBillExtraFeeResponse struct {
	Fees  []ExtraFee `json:"fees"`
	Count int64      `json:"count"`
}

type Package struct {
	ID             int64     `json:"id"`
	Code           string    `json:"code"`
	TrackingNumber string    `json:"tracking_number"`
	ShippingFee    float64   `json:"shipping_fee"`
	Status         int       `json:"status"`
	UpdatedAt      time.Time `json:"update_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type GetBillPackagesResponse struct {
	Packages []Package `json:"packages"`
	Count    int64     `json:"count"`
}

type GetDetailBillResponse struct {
	Bill *entity.Bill `json:"bill"`
}

type GetInvoiceReponse struct {
	Download string `json:"download"`
}

func NewBillHandler(l *zap.SugaredLogger, s3 storage.S3, bm *sqlmanager.BillManager, um *sqlmanager.UserManager) *BillHandler {
	return &BillHandler{
		Logger:  l,
		LocalS3: s3,

		BillManager: bm,
		UserManager: um,
	}
}

func (h *BillHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.BillQueryOption{
			StartDate:          cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:            cast.ToString(c.Request.URL.Query().Get("end_date")),
			Limit:              limit,
			Offset:             offset,
			UserID:             userID,
			Search:             cast.ToString(c.Request.URL.Query().Get("search")),
			HidePreloadUser:    true,
			HidePreloadPackage: true,
		}

		bills, err := h.BillManager.Fetch(opts)
		if err != nil {
			h.Logger.Errorf("get bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, BillListReponse{Bills: bills})
	}
}

func (h *BillHandler) ListChina() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.BillQueryOption{
			StartDate:          cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:            cast.ToString(c.Request.URL.Query().Get("end_date")),
			Limit:              limit,
			Offset:             offset,
			UserID:             userID,
			Search:             cast.ToString(c.Request.URL.Query().Get("search")),
			HidePreloadUser:    true,
			HidePreloadPackage: true,
			ServiceCode:		constant.ServiceCNCode,
		}

		bills, err := h.BillManager.Fetch(opts)
		if err != nil {
			h.Logger.Errorf("get bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, BillListReponse{Bills: bills})
	}
}

func (h *BillHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}
		opts := sqlmanager.BillQueryOption{
			StartDate:  cast.ToString(c.Request.URL.Query().Get("start_date")),
			EndDate:    cast.ToString(c.Request.URL.Query().Get("end_date")),
			Search:     cast.ToString(c.Request.URL.Query().Get("search")),
			UserID:     userID,
			PackageFee: cast.ToString(c.Request.URL.Query().Get("package")),
			ExtraFee:   cast.ToString(c.Request.URL.Query().Get("extra")),
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

		count, err := h.BillManager.Count(opts)
		if err != nil {
			h.Logger.Errorf("count bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, BillCountReponse{Count: count})
	}
}

func (h *BillHandler) BillFees() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opts := sqlmanager.BillFeeQueryOption{
			BillCode: cast.ToString(c.Param("bill_code")),
			UserID:   userID,
			Type:     cast.ToInt(c.Request.URL.Query().Get("type")),
			Limit:    limit,
			Offset:   offset,
		}
		if len(opts.BillCode) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		extraFees, err := h.BillManager.GetBillExtraFee(opts)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var items = []ExtraFee{}
		for _, v := range extraFees {
			item := ExtraFee{
				ID:          v.ID,
				TypeName:    "",
				PackageID:   utils.Int64Value(v.PackageID),
				ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
				PackageCode: "",
				BillID:      v.BillID,
				Amount:      v.Amount,
				Description: v.Description,
				Status:      v.Status,
				UpdatedAt:   v.UpdatedAt,
				CreatedAt:   v.CreatedAt,
			}

			if v.Package != nil && v.Package.PackageCode != nil {
				item.PackageCode = v.Package.PackageCode.Code
			}

			if v.ExtraFeeType != nil {
				item.TypeName = v.ExtraFeeType.Name
				if item.Description == "" {
					item.Description = v.ExtraFeeType.Name
				}
			}

			items = append(items, item)
		}

		Count, err := h.BillManager.CountBillExtraFee(opts)

		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetBillExtraFeeResponse{items, Count})
	}
}

func (h *BillHandler) BillPackage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		offset, limit := httputil.GetRequestPaginate(c.Request)

		code := strings.TrimSpace(c.Param("bill_code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, "Bill code is required")
			return
		}

		opts := sqlmanager.BillPackageQueryOption{
			Select:          "id, package_code_id, shipping_fee, status, updated_at, created_at",
			BillCode:        code,
			UserID:          userID,
			Limit:           limit,
			Offset:          offset,
			PerloadTracking: true,
		}

		packages, err := h.BillManager.GetBillPackages(opts)
		if err != nil {
			h.Logger.Errorf("fetch packages in bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		items := []Package{}
		for _, v := range packages {
			item := Package{
				ID:             v.ID,
				Code:           "",
				TrackingNumber: "",
				ShippingFee:    v.ShippingFee,
				Status:         v.Status,
				UpdatedAt:      v.UpdatedAt,
				CreatedAt:      v.CreatedAt,
			}

			if v.Tracking != nil {
				item.TrackingNumber = v.Tracking.TrackingNumber
			}

			if v.PackageCode != nil {
				item.Code = v.PackageCode.Code
			}

			items = append(items, item)
		}

		count, err := h.BillManager.CountBillItemPackages(opts)
		if err != nil {
			h.Logger.Errorf("count packages in bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetBillPackagesResponse{items, count})
	}
}

func (h *BillHandler) Detail() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		code := strings.TrimSpace(c.Param("code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, "Bill code is required")
			return
		}

		bill, err := h.BillManager.GetBillByCode(code, userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
		}

		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetDetailBillResponse{bill})
	}
}

func (h *BillHandler) Invoice() gin.HandlerFunc {
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

		code := strings.TrimSpace(c.Param("code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, "Bill code is required")
			return
		}

		//get bill
		bill, err := h.BillManager.GetBillByCode(code, userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
		}

		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		currentDate := time.Now().Format("2006-01-02")
		if currentDate == bill.CreatedAt.Format("2006-01-02") {
			c.JSON(http.StatusBadRequest, "Bill in status Created cannot export")
			return
		}
		//get packages
		optsPackage := sqlmanager.BillPackageQueryOption{
			Select:          "id, package_code_id, shipping_fee, status, updated_at, created_at",
			BillCode:        code,
			UserID:          userID,
			PerloadTracking: true,
		}

		packages, err := h.BillManager.GetBillPackages(optsPackage)
		if err != nil {
			h.Logger.Errorf("fetch packages in bill: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		listPackages := []invoice.Package{}
		for _, v := range packages {
			item := invoice.Package{
				ID:             v.ID,
				Code:           "",
				TrackingNumber: "",
				ShippingFee:    v.ShippingFee,
				Status:         v.Status,
				UpdatedAt:      v.UpdatedAt,
				CreatedAt:      v.CreatedAt,
			}

			if v.Tracking != nil {
				item.TrackingNumber = v.Tracking.TrackingNumber
			}

			if v.PackageCode != nil {
				item.Code = v.PackageCode.Code
			}

			listPackages = append(listPackages, item)
		}

		//get extra_fee
		opts := sqlmanager.BillFeeQueryOption{
			BillCode: code,
			UserID:   userID,
		}

		extraFees, err := h.BillManager.GetBillExtraFee(opts)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var extraFeeArr = []invoice.ExtraFee{}
		var totalExtraFee float64

		var refundFeeArr = []invoice.ExtraFee{}
		var totalRefundFee float64

		var comFeeArr = []invoice.ExtraFee{}
		var totalComFee float64

		for _, v := range extraFees {
			item := invoice.ExtraFee{
				ID:          v.ID,
				TypeName:    "",
				PackageID:   utils.Int64Value(v.PackageID),
				ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
				PackageCode: "",
				BillID:      v.BillID,
				Amount:      v.Amount,
				Description: v.Description,
				Status:      v.Status,
				UpdatedAt:   v.UpdatedAt,
				CreatedAt:   v.CreatedAt,
			}

			if v.Package != nil && v.Package.PackageCode != nil {
				item.PackageCode = v.Package.PackageCode.Code
			}

			if v.ExtraFeeType != nil {
				item.TypeName = v.ExtraFeeType.Name
			}

			if v.ExtraFeeTypeID != constant.ExtraFeeTypeDiscount {
				totalExtraFee += item.Amount
				extraFeeArr = append(extraFeeArr, item)
				continue
			}

			if v.Amount >= 0 {
				totalExtraFee += item.Amount
				extraFeeArr = append(extraFeeArr, item)
			} else {
				item.Amount = math.Abs(item.Amount)
				totalRefundFee += item.Amount
				refundFeeArr = append(refundFeeArr, item)
			}
		}

		//get refund_fee
		optsRefund := sqlmanager.BillFeeQueryOption{
			BillCode: code,
			UserID:   userID,
			Type:     constant.ExtraFeeTypeRefund,
		}

		refundFees, err := h.BillManager.GetBillExtraFee(optsRefund)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for _, v := range refundFees {
			item := invoice.ExtraFee{
				ID:          v.ID,
				TypeName:    "",
				PackageID:   utils.Int64Value(v.PackageID),
				ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
				PackageCode: "",
				BillID:      v.BillID,
				Amount:      math.Abs(v.Amount),
				Description: v.Description,
				Status:      v.Status,
				UpdatedAt:   v.UpdatedAt,
				CreatedAt:   v.CreatedAt,
			}

			if v.Package != nil && v.Package.PackageCode != nil {
				item.PackageCode = v.Package.PackageCode.Code
			}

			if v.Coupon != nil {
				item.CouponCode = v.Coupon.Code
			}

			if v.ExtraFeeType != nil {
				item.TypeName = v.ExtraFeeType.Name
			}

			totalRefundFee += item.Amount
			refundFeeArr = append(refundFeeArr, item)
		}

		//get commission fee
		optsCommission := sqlmanager.BillFeeQueryOption{
			BillCode: code,
			UserID:   userID,
			Type:     constant.ExtraFeeTypeAffiliate,
		}

		commissionFees, err := h.BillManager.GetBillExtraFee(optsCommission)
		if err != nil {
			h.Logger.Errorf("Get bill error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for _, v := range commissionFees {
			item := invoice.ExtraFee{
				ID:          v.ID,
				TypeName:    "",
				PackageID:   utils.Int64Value(v.PackageID),
				ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
				PackageCode: "",
				BillID:      v.BillID,
				Amount:      math.Abs(v.Amount),
				Description: v.Description,
				Status:      v.Status,
				UpdatedAt:   v.UpdatedAt,
				CreatedAt:   v.CreatedAt,
			}

			if v.Package != nil && v.Package.PackageCode != nil {
				item.PackageCode = v.Package.PackageCode.Code
			}

			if v.Coupon != nil {
				item.CouponCode = v.Coupon.Code
			}

			if v.ExtraFeeType != nil {
				item.TypeName = v.ExtraFeeType.Name
			}

			totalComFee += item.Amount
			comFeeArr = append(comFeeArr, item)
		}

		var total float64 = bill.ShippingFee + bill.ExtraFee
		data := invoice.Data{
			Email:          user.Email,
			FullName:       user.FullName,
			BillCode:       bill.Code,
			BillDate:       bill.CreatedAt.Add(7 * time.Hour).Format("January 2, 2006"),
			ShippingFee:    utils.FormatNumber(bill.ShippingFee),
			TotalAmount:    fmt.Sprintf("$%s", utils.FormatNumber(utils.ToFixed(math.Abs(total), 2))),
			TotalExtraFee:  utils.FormatNumber(utils.ToFixed(totalExtraFee, 2)),
			TotalRefundFee: utils.FormatNumber(utils.ToFixed(totalRefundFee, 2)),
			TotalComFee:    utils.FormatNumber(utils.ToFixed(totalComFee, 2)),
			Packages:       listPackages,
		}

		if len(extraFeeArr) > 0 {
			data.ExtraFee = extraFeeArr
		}

		if len(refundFeeArr) > 0 {
			data.RefundFee = refundFeeArr
		}

		if len(comFeeArr) > 0 {
			data.ComissionFee = comFeeArr
		}

		if len(listPackages)+len(extraFeeArr)+len(refundFeeArr)+len(comFeeArr) < 1 {
			c.JSON(http.StatusBadRequest, "Hóa đơn này trống")
			return
		}

		if total < 0 {
			data.TotalAmount = fmt.Sprintf("-%s", data.TotalAmount)
		}

		fileName, err := invoice.GeneratePdf(data, bill, user)
		if err != nil {
			h.Logger.Errorf("Create file pdf error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		filePath := fmt.Sprintf("bills/%s%s", file.MakeUploadUserIDPath(user.ID, ""), fileName)
		osfile, err := os.Open(fileName)
		if err != nil {
			h.Logger.Errorf("Open file pdf error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		defer os.Remove(fileName)
		defer osfile.Close()
		buffer, err := ioutil.ReadFile(fileName)
		if err != nil {
			h.Logger.Errorf("read file error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		osfile.Read(buffer)
		contentType := constant.ContentTypePDF
		bucketExport := viper.GetString("bucket.export_billing")
		err = h.LocalS3.UploadFile(bytes.NewBuffer(buffer), filePath, bucketExport, contentType)

		if err != nil {
			h.Logger.Errorf("upload file to S3 error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetInvoiceReponse{Download: filePath})

	}
}

func (h *BillHandler) Invoices() gin.HandlerFunc {
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

		exportForm := &ExportBillsForm{}
		if err := c.ShouldBindJSON(exportForm); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if exportForm.ExportType != constant.ExportTypeDetail && exportForm.ExportType != constant.ExportTypeGeneral {
			c.JSON(http.StatusBadRequest, "Loại hóa đơn export yêu cầu phải chọn !")
			return
		}

		opts := sqlmanager.BillQueryOption{
			StartDate:          exportForm.StartDate,
			EndDate:            exportForm.EndDate,
			UserID:             user.ID,
			HidePreloadUser:    true,
			HidePreloadPackage: true,
		}

		PackageFee := exportForm.PackageFee
		ExtraFee := exportForm.ExtraFee
		CommissionFee := exportForm.CommissionFee

		bills, err := h.BillManager.Fetch(opts)
		if err != nil {
			h.Logger.Errorf("get bills: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(bills) < 1 {
			h.Logger.Errorf("get bills: %v", err)
			c.JSON(http.StatusInternalServerError, "Không có hóa đơn nào ")
			return

		}

		currentDate := time.Now().Format("2006-01-02")

		var datas = []invoice.Data{}
		var total float64
		for _, bill := range bills {
			if currentDate == bill.CreatedAt.Format("2006-01-02") {
				continue
			}
			data, paymentAmount, err := h.GetInvoice(&bill, *user, PackageFee, ExtraFee, CommissionFee)
			if err != nil {
				h.Logger.Errorf("get bill: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if data != nil {
				datas = append(datas, *data)
			}
			total += paymentAmount
		}
		var fileName string
		if exportForm.ExportType == constant.ExportTypeGeneral {
			fileName, err = h.GeneratePdf(datas, *user, opts, total)
			if err != nil {
				h.Logger.Errorf("Create file pdf error %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		} else if exportForm.ExportType == constant.ExportTypeDetail {
			fileName, err = h.exportBillXlsx(datas, opts)
			if err != nil {
				h.Logger.Errorf("Create file excel error %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		filePath := fmt.Sprintf("bills/%s%s", file.MakeUploadUserIDPath(user.ID, ""), fileName)
		osfile, err := os.Open(fileName)
		if err != nil {
			h.Logger.Errorf("Open file pdf error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		defer os.Remove(fileName)
		defer osfile.Close()
		buffer, err := ioutil.ReadFile(fileName)
		if err != nil {
			h.Logger.Errorf("read file error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		osfile.Read(buffer)
		contentType := constant.ContentTypePDF
		if exportForm.ExportType == constant.ExportTypeDetail {
			contentType = constant.ContentTypeXlsxExcel2007Above
		}
		bucketExport := viper.GetString("bucket.export_billing")
		err = h.LocalS3.UploadFile(bytes.NewBuffer(buffer), filePath, bucketExport, contentType)

		if err != nil {
			h.Logger.Errorf("upload file to S3 error: %v/n", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetInvoiceReponse{Download: filePath})
	}
}

func (h *BillHandler) GetInvoice(bill *entity.Bill, user entity.User, PackageFee string, ExtraFee string, CommissionFee string) (*invoice.Data, float64, error) {
	bill, err := h.BillManager.GetBillByCode(bill.Code, bill.UserID)
	if err == gorm.ErrRecordNotFound {
		h.Logger.Errorf("Write xlsx header error: %v", err)
		return nil, 0, err
	}

	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return nil, 0, err
	}

	//get packages
	optsPackage := sqlmanager.BillPackageQueryOption{
		Select:          "id, package_code_id, shipping_fee, status, updated_at, created_at,customer_shipment_id",
		BillCode:        bill.Code,
		UserID:          bill.UserID,
		PerloadTracking: true,
	}

	packages, err := h.BillManager.GetBillPackages(optsPackage)
	if err != nil {
		h.Logger.Errorf("fetch packages in bill: %v", err)
		return nil, 0, err
	}

	listPackages := []invoice.Package{}
	for _, v := range packages {
		item := invoice.Package{
			ID:             v.ID,
			Code:           "",
			TrackingNumber: "",
			ShippingFee:    v.ShippingFee,
			ShipmentID:     utils.Int64Value(v.CustomerShipmentID),
			Status:         v.Status,
			UpdatedAt:      v.UpdatedAt,
			CreatedAt:      v.CreatedAt,
		}

		if v.Tracking != nil {
			item.TrackingNumber = v.Tracking.TrackingNumber
		}

		if v.PackageCode != nil {
			item.Code = v.PackageCode.Code
		}

		listPackages = append(listPackages, item)
	}

	//get extra_fee
	opts := sqlmanager.BillFeeQueryOption{
		BillCode: bill.Code,
		UserID:   bill.UserID,
	}

	extraFees, err := h.BillManager.GetBillExtraFee(opts)
	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return nil, 0, err
	}

	var extraFeeArr = []invoice.ExtraFee{}
	var totalExtraFee float64

	var refundFeeArr = []invoice.ExtraFee{}
	var totalRefundFee float64

	var comFeeArr = []invoice.ExtraFee{}
	var totalComFee float64

	for _, v := range extraFees {
		item := invoice.ExtraFee{
			ID:          v.ID,
			TypeName:    "",
			PackageID:   utils.Int64Value(v.PackageID),
			PackageCode: "",
			BillID:      v.BillID,
			Amount:      v.Amount,
			Description: v.Description,
			Status:      v.Status,
			UpdatedAt:   v.UpdatedAt,
			CreatedAt:   v.CreatedAt,
			ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
		}

		if v.Package != nil && v.Package.PackageCode != nil {
			item.PackageCode = v.Package.PackageCode.Code
		}

		if v.ExtraFeeType != nil {
			item.TypeName = v.ExtraFeeType.Name
			if item.Description == "" {
				item.Description = v.ExtraFeeType.Name
			}
		}

		if v.ExtraFeeTypeID != constant.ExtraFeeTypeDiscount {
			totalExtraFee += item.Amount
			extraFeeArr = append(extraFeeArr, item)
			continue
		}

		if v.Amount >= 0 {
			totalExtraFee += item.Amount
			extraFeeArr = append(extraFeeArr, item)
		} else {
			item.Amount = math.Abs(item.Amount)
			totalRefundFee += item.Amount
			refundFeeArr = append(refundFeeArr, item)
		}
	}

	//get refund_fee
	optsRefund := sqlmanager.BillFeeQueryOption{
		BillCode: bill.Code,
		UserID:   bill.UserID,
		Type:     constant.ExtraFeeTypeRefund,
	}

	refundFees, err := h.BillManager.GetBillExtraFee(optsRefund)
	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return nil, 0, err
	}

	for _, v := range refundFees {
		item := invoice.ExtraFee{
			ID:          v.ID,
			TypeName:    "",
			PackageID:   utils.Int64Value(v.PackageID),
			PackageCode: "",
			BillID:      v.BillID,
			Amount:      math.Abs(v.Amount),
			Description: v.Description,
			Status:      v.Status,
			UpdatedAt:   v.UpdatedAt,
			CreatedAt:   v.CreatedAt,
			ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
		}

		if v.Package != nil && v.Package.PackageCode != nil {
			item.PackageCode = v.Package.PackageCode.Code
		}

		if v.ExtraFeeType != nil {
			item.TypeName = v.ExtraFeeType.Name
			if item.Description == "" {
				item.Description = v.ExtraFeeType.Name
			}
		}

		totalRefundFee += item.Amount
		refundFeeArr = append(refundFeeArr, item)
	}

	//get commission fee
	optsCommission := sqlmanager.BillFeeQueryOption{
		BillCode: bill.Code,
		UserID:   bill.UserID,
		Type:     constant.ExtraFeeTypeAffiliate,
	}

	commissionFees, err := h.BillManager.GetBillExtraFee(optsCommission)
	if err != nil {
		h.Logger.Errorf("Get bill error: %v", err)
		return nil, 0, err
	}

	for _, v := range commissionFees {
		item := invoice.ExtraFee{
			ID:          v.ID,
			TypeName:    "",
			PackageID:   utils.Int64Value(v.PackageID),
			PackageCode: "",
			BillID:      v.BillID,
			Amount:      math.Abs(v.Amount),
			Description: v.Description,
			Status:      v.Status,
			UpdatedAt:   v.UpdatedAt,
			CreatedAt:   v.CreatedAt,
			ShipmentID:  utils.Int64Value(v.CustomerShipmentID),
		}

		if v.Package != nil && v.Package.PackageCode != nil {
			item.PackageCode = v.Package.PackageCode.Code
		}

		if v.ExtraFeeType != nil {
			item.TypeName = v.ExtraFeeType.Name
			if item.Description == "" {
				item.Description = v.ExtraFeeType.Name
			}
		}

		totalComFee += item.Amount
		comFeeArr = append(comFeeArr, item)
	}

	var total float64 = bill.ShippingFee + bill.ExtraFee
	data := invoice.Data{
		Email:          user.Email,
		FullName:       user.FullName,
		BillCode:       bill.Code,
		BillDate:       bill.CreatedAt.Add(7 * time.Hour).Format("02/01/2006"),
		ShippingFee:    utils.FormatNumber(bill.ShippingFee),
		TotalAmount:    fmt.Sprintf("$%s", utils.FormatNumber(utils.ToFixed(math.Abs(total), 2))),
		TotalExtraFee:  utils.FormatNumber(utils.ToFixed(totalExtraFee, 2)),
		TotalRefundFee: utils.FormatNumber(utils.ToFixed(totalRefundFee, 2)),
		TotalComFee:    utils.FormatNumber(utils.ToFixed(totalComFee, 2)),
		Packages:       listPackages,
		ExtraFee:       extraFeeArr,
		RefundFee:      refundFeeArr,
		ComissionFee:   comFeeArr,
	}

	if ExtraFee == "" {
		data.RefundFee = []invoice.ExtraFee{}
		data.ExtraFee = []invoice.ExtraFee{}
	}

	if PackageFee == "" {
		data.Packages = []invoice.Package{}
	}

	if CommissionFee == "" {
		data.ComissionFee = []invoice.ExtraFee{}
	}

	if len(data.Packages)+len(data.ExtraFee)+len(data.RefundFee)+len(data.ComissionFee) < 1 {
		return nil, 0, nil
	}

	if total < 0 {
		data.TotalAmount = fmt.Sprintf("-%s", data.TotalAmount)
	}

	return &data, total, nil
}

func (h *BillHandler) GeneratePdf(datas []invoice.Data, user entity.User, otp sqlmanager.BillQueryOption, total float64) (string, error) {
	const (
		colCount = 3
		headerWd = 190
	)

	var (
		err error
	)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8FontFromBytes("beeVN", "", bevn.TTF)
	pdf.AddUTF8FontFromBytes("beeVN", "B", bevnbold.TTF)
	pdf.AddUTF8FontFromBytes("beeVN", "I", bevnitalic.TTF)

	pdf.AddPage()
	pdf.SetFont("BeeVN", "", 32)
	pdf.CellFormat(140, 10, "Receipt of Payment", "", 1, "L", false, 0, "")
	pdf.Ln(25)
	pdf.SetFont("BeeVN", "", 14)
	pdf.SetY(25)
	pdf.CellFormat(160, 10, "Account:", "", 0, "L", false, 0, "")

	file, err := os.Open("./logo.png")
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}
	defer file.Close()

	pdf.RegisterImageReader("logo", "png", file)
	if pdf.Ok() {
		pdf.Image("logo", -10, 28,
			30, 0, false, "png", 0, "")
	}

	pdf.Ln(10)
	pdf.SetFont("BeeVN", "B", 14)
	pdf.Cell(50, 8, user.FullName)
	pdf.SetFont("BeeVN", "", 14)
	pdf.CellFormat(140, 8, "36 Ngo 1D Tran Quang Dieu", "", 1, "R", false, 0, "")
	pdf.Cell(50, 8, user.Email)
	pdf.CellFormat(140, 8, "Dong Da, Ha Noi, Viet Nam", "", 1, "R", false, 0, "")

	pdf.SetDrawColor(237, 237, 238)
	pdf.Line(0, 55, 1000, 55)

	pdf.SetY(63)

	header := [colCount]string{" Summary"}
	pdf.SetFont("BeeVN", "", 14)
	// Headers
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(255, 154, 0)
	for colJ := 0; colJ < 1; colJ++ {
		pdf.SetDrawColor(49, 50, 50)
		pdf.SetFont("beeVN", "B", 14)
		pdf.CellFormat(headerWd, 13, header[colJ], "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(24, 24, 24)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("BeeVN", "", 14)
	pdf.CellFormat(3, 16, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(96, 16, "Payment Date:", "", 0, "L", false, 0, "")
	pdf.CellFormat(15, 16, "From", "", 0, "R", false, 0, "")
	pdf.CellFormat(33, 16, cast.ToTime(otp.StartDate).Format("02/01/2006"), "", 0, "R", false, 0, "")
	pdf.CellFormat(7, 16, "to", "", 0, "R", false, 0, "")
	pdf.CellFormat(33, 16, cast.ToTime(otp.EndDate).Format("02/01/2006"), "", 0, "R", false, 0, "")
	pdf.CellFormat(3, 16, "", "R", 1, "L", false, 0, "")

	pdf.CellFormat(3, 1, "", "L", 0, "R", false, 0, "")
	pdf.CellFormat(184, 1, "", "B", 0, "R", false, 0, "")
	pdf.CellFormat(3, 1, "", "R", 1, "R", false, 0, "")

	var textAmount string = fmt.Sprintf("$%s", utils.FormatNumber(utils.ToFixed(math.Abs(total), 2)))
	if total < 0 {
		textAmount = fmt.Sprintf("-$%s", utils.FormatNumber(utils.ToFixed(math.Abs(total), 2)))
	}
	pdf.SetFont("beeVN", "B", 14)
	pdf.CellFormat(3, 16, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(137, 16, "Payment Amount:", "", 0, "L", false, 0, "")
	pdf.CellFormat(47, 16, textAmount, "", 0, "R", false, 0, "")
	pdf.CellFormat(3, 16, "", "R", 1, "L", false, 0, "")
	pdf.CellFormat(190, 3, "", "L,R,B", 1, "R", false, 0, "")

	pdf.Ln(-1)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(255, 154, 0)
	pdf.SetDrawColor(49, 50, 50)
	pdf.CellFormat(headerWd, 13, "  Detail", "1", 1, "L", true, 0, "")

	pdf.SetFont("beeVN", "", 14)
	pdf.SetTextColor(24, 24, 24)
	pdf.SetFillColor(255, 255, 255)
	pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(65, 13, "Payment Date", "", 0, "L", false, 0, "")
	pdf.CellFormat(70, 13, "Payment ID", "", 0, "L", false, 0, "")
	pdf.CellFormat(45, 13, "Amount", "", 0, "R", false, 0, "")
	pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")
	for i, data := range datas {
		pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")
		pdf.CellFormat(65, 13, data.BillDate, "", 0, "L", false, 0, "")
		pdf.CellFormat(70, 13, data.BillCode, "", 0, "L", false, 0, "")
		pdf.CellFormat(45, 13, data.TotalAmount, "", 0, "R", false, 0, "")
		pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")
		pdf.SetFont("beeVN", "", 10)
		//pdf.SetTextColor(224, 224, 224)

		getY := pdf.GetY()
		spaceLeft := 297 - getY - 35
		lenP := len(datas)
		if spaceLeft <= 0 {
			pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
			pdf.AddPage()
			pdf.SetY(30)
			if i+1 < lenP {
				pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")
			}

		} else {
			if lenP == i+1 || spaceLeft < 10 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
			} else {
				pdf.SetFont("beeVN", "", 10)
				pdf.SetTextColor(224, 224, 224)
				pdf.CellFormat(190, 1, "-----------------------------------------------------------------------------------", "L,R", 1, "C", false, 0, "")
				pdf.SetTextColor(24, 24, 24)
			}
		}
		pdf.SetFont("beeVN", "", 14)
	}

	pdf.SetTextColor(24, 24, 24)
	pdf.CellFormat(190, 8, "", "", 1, "L", false, 0, "")
	pdf.SetFont("beeVN", "I", 12)
	pdf.CellFormat(190, 5, "To access billing options, check your statement or for questions regarding this receipt,", "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 10, "log in to your members area at https://ananbay.com", "", 0, "L", false, 0, "")

	fileName := fmt.Sprintf("payment_receipt_%s_%s.pdf", otp.StartDate, otp.EndDate)
	err = pdf.OutputFileAndClose(fileName)
	if err != nil {
		h.Logger.Errorf("Create file pdf error %v", err)
		return "", err
	}

	return fileName, nil
}

func (h *BillHandler) exportBillXlsx(datas []invoice.Data, otp sqlmanager.BillQueryOption) (string, error) {
	header := []string{
		"Mã bill",
		"Mã đơn hàng",
		"Mã lô",
		"Loại phí",
		"Ngày",
		"Số tiền",
		"Nội dung phí",
	}

	f1 := excelize.NewFile()

	for index, item := range header {
		var colStr string
		if index >= 26 {
			colStr = fmt.Sprintf("A%s1", toCharStr(index+1-26))
		} else {
			colStr = fmt.Sprintf("%s1", toCharStr(index+1))

		}
		err := f1.SetCellValue("Sheet1", colStr, item)
		if err != nil {
			h.Logger.Errorf("Write xlsx header error: %v", err)

		}
	}

	fileName := fmt.Sprintf("payment_receipt_%s_%s.xlsx", otp.StartDate, otp.EndDate)
	f, err := os.Create(fileName)
	if err != nil {
		h.Logger.Errorf("Create file tmp: %v", err)
		return "", err
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	defer writer.Flush()
	indexRow := 0
	for _, data := range datas {
		for _, pkg := range data.Packages {
			shipment := ""
			if pkg.ShipmentID > 0 {
				shipment = fmt.Sprintf("#%v", pkg.ShipmentID)
			}
			tmp := []string{}
			tmp = append(tmp, []string{
				data.BillCode,
				pkg.Code,
				shipment,
				"Shipping Fee",
				pkg.CreatedAt.Format("02/01/2006"),
				fmt.Sprintf("%s$", utils.FormatNumber(utils.ToFixed(pkg.ShippingFee, 2))),
				"",
			}...)

			for indexCol, item := range tmp {
				var cellStr string
				if indexCol >= 26 {
					cellStr = fmt.Sprintf("A%s%d", toCharStr(indexCol+1-26), indexRow+2)
				} else {
					cellStr = fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)

				}
				err := f1.SetCellValue("Sheet1", cellStr, item)
				if err != nil {
					h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
				}
			}
			indexRow++
		}

		for _, item := range data.ExtraFee {
			shipment := ""
			if item.ShipmentID > 0 {
				shipment = fmt.Sprintf("#%v", item.ShipmentID)
			}
			tmp := []string{}
			tmp = append(tmp, []string{
				data.BillCode,
				item.PackageCode,
				shipment,
				item.TypeName,
				item.CreatedAt.Format("02/01/2006"),
				fmt.Sprintf("%s$", utils.FormatNumber(utils.ToFixed(item.Amount, 2))),
				item.Description,
			}...)

			for indexCol, item := range tmp {
				var cellStr string
				if indexCol >= 26 {
					cellStr = fmt.Sprintf("A%s%d", toCharStr(indexCol+1-26), indexRow+2)
				} else {
					cellStr = fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)

				}
				err := f1.SetCellValue("Sheet1", cellStr, item)
				if err != nil {
					h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
				}
			}
			indexRow++
		}

		for _, item := range data.RefundFee {
			shipment := ""
			if item.ShipmentID > 0 {
				shipment = fmt.Sprintf("#%v", item.ShipmentID)
			}
			tmp := []string{}
			tmp = append(tmp, []string{
				data.BillCode,
				item.PackageCode,
				shipment,
				"Phí hoàn tiền",
				item.CreatedAt.Format("02/01/2006"),
				fmt.Sprintf("%s$", utils.FormatNumber(utils.ToFixed(item.Amount, 2))),
				item.Description,
			}...)

			for indexCol, item := range tmp {
				var cellStr string
				if indexCol >= 26 {
					cellStr = fmt.Sprintf("A%s%d", toCharStr(indexCol+1-26), indexRow+2)
				} else {
					cellStr = fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)

				}
				err := f1.SetCellValue("Sheet1", cellStr, item)
				if err != nil {
					h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
				}
			}
			indexRow++
		}

		for _, item := range data.ComissionFee {
			shipment := ""
			if item.ShipmentID > 0 {
				shipment = fmt.Sprintf("#%v", item.ShipmentID)
			}
			tmp := []string{}
			tmp = append(tmp, []string{
				data.BillCode,
				item.PackageCode,
				shipment,
				"Phí hoa hồng",
				item.CreatedAt.Format("02/01/2006"),
				fmt.Sprintf("%s$", utils.FormatNumber(utils.ToFixed(item.Amount, 2))),
				item.Description,
			}...)

			for indexCol, item := range tmp {
				var cellStr string
				if indexCol >= 26 {
					cellStr = fmt.Sprintf("A%s%d", toCharStr(indexCol+1-26), indexRow+2)
				} else {
					cellStr = fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)

				}
				err := f1.SetCellValue("Sheet1", cellStr, item)
				if err != nil {
					h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
				}
			}
			indexRow++
		}

	}

	if err := f1.SaveAs(fileName); err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}

	return fileName, nil
}

func toCharStr(i int) string {
	return string(rune('A' - 1 + i))
}
