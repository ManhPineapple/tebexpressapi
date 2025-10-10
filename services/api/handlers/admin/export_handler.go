package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"tebexpressapi/pkg/utils/file"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type ExportHandler struct {
	Logger *zap.SugaredLogger

	S3 storage.S3

	PackageManager  *sqlmanager.PackageManager
	ShipmentManager *sqlmanager.ShipmentManager
}

type ExportPackageForm struct {
	IDS []int64 `json:"ids"`
}
type ExportPackageAdminReponse struct {
	Download string `json:"download"`
}

type ExportShipmentForm struct {
	ID int64 `json:"id"`
}
type ExportShipmentReponse struct {
	Download string `json:"download"`
}
type ShipmentExport struct {
	dbgorm.Model
	ID             float64 `json:"id"`
	Code           string  `json:"code"`
	TrackingNumber string  `json:"tracking_number"`
	Detail         string  `json:"detail"`
	ShippingFee    float64 `json:"shipping_fee"`
	Recipient      string  `json:"recipient"`
	Address1       string  `json:"address_1"`
	FullName       string  `json:"full_name"`
	OrderNumber    string  `json:"order_number"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	Length         float64 `json:"length"`
	Weight         float64 `json:"weight"`
	ContainerCode  string  `json:"container_code"`
	Status         int     `json:"status"`
}

func NewExportHandler(l *zap.SugaredLogger, s3 storage.S3, pm *sqlmanager.PackageManager, sm *sqlmanager.ShipmentManager) *ExportHandler {
	return &ExportHandler{
		Logger: l,
		S3:     s3,

		PackageManager:  pm,
		ShipmentManager: sm,
	}
}

func (h *ExportHandler) DownloadLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Request.URL.Query().Get("url")
		bucketType := c.Request.URL.Query().Get("type")

		if bucketType == "" || url == "" {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if !utils.ValidSlug(bucketType) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		bucketName := viper.GetString("bucket.labels")
		if url == "" {
			c.JSON(http.StatusBadGateway, constant.MessageValidateInput)
			return
		}

		h.Logger.Info(url, bucketName)
		object, err := h.S3.Read(url, bucketName)
		if err != nil {
			h.Logger.Error(err)
			return
		}
		defer object.Body.Close()
		// If there is no content length, it is a directory
		pkg, err := h.PackageManager.GetPackage(sqlmanager.PackageQueryOption{
			Label: url,
		})
		if err != nil {
			h.Logger.Error(err)
			return
		}

		now := time.Now()
		pkg.LastPrintLabelAt = &now
		err = h.PackageManager.UpdatePackage(&pkg, pkg.ID)
		if err != nil {
			h.Logger.Error(err)
			return
		}

		c.Header("Content-Type", *object.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", *object.ContentLength))
		c.Status(http.StatusOK)
		io.Copy(c.Writer, object.Body)
	}
}

func (h *ExportHandler) ExportPackage() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		exportForm := &ExportPackageForm{}
		if err := c.ShouldBindJSON(exportForm); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		options := sqlmanager.PackageQueryOption{
			IDs: exportForm.IDS,
			Preload: []string{
				"Service",
				"ExtraFee",
			},
			SortBy: "created_at",
			Sort:   "ASC",
		}

		if len(options.IDs) == 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		packages, err := h.PackageManager.GetPackages(options)

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
		c.JSON(http.StatusOK, ExportPackageAdminReponse{Download: filePath})
	}
}

func (h *ExportHandler) ExportShipment() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		exportForm := &ExportShipmentForm{}
		if err := c.ShouldBindJSON(exportForm); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if exportForm.ID < 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		var items []ShipmentExport
		err := h.ShipmentManager.GetShipmentExport(&items, exportForm.ID)

		if err != nil {
			h.Logger.Errorf("Get shipment error %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		filePath, err := h.exportShipmentXlsx(items, userID)
		if err != nil {
			h.Logger.Errorf("Error while export csv file %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		c.JSON(http.StatusOK, ExportShipmentReponse{Download: filePath})
	}
}
func (h *ExportHandler) exportCsvPackageXlsx(packages []entity.Package, userID int64) (string, error) {
	header := []string{
		"Mã AB",
		"Mã tracking",
		"Mã đơn hàng",
		"Chi tiết hàng hóa",
		"Mã lô",
		"Ngày tạo",
		"Ngày chấp nhận",
		"Tên khách hàng",
		"Tên người nhận",
		"SĐT người nhận",
		"Địa chỉ nhận",
		"Địa chỉ nhận (phụ)",
		"Thành phố",
		"Mã vùng",
		"Mã bưu điện",
		"Mã quốc gia",
		"Trọng lượng",
		"Dài",
		"Rộng",
		"Cao",
		"Dịch vụ",
		"Tổng cước",
		"Phí giao",
		"Phí phát sinh",
		"Trạng thái",
	}

	f1 := excelize.NewFile()
	sheetName := "Sheet1"

	// Write header row
	for index, item := range header {
		colStr := fmt.Sprintf("%s1", AdmintoCharStr(index+1))
		if err := f1.SetCellValue(sheetName, colStr, item); err != nil {
			h.Logger.Errorf("Write xlsx header error: %v", err)
		}
	}

	fileName := fmt.Sprintf("danh_sach_van_don_%d_%02d_%02d.xlsx",
		time.Now().Day(), int(time.Now().Month()), time.Now().Year(),
	)

	for indexRow, Package := range packages {
		var trackingNumber string
		if Package.Tracking != nil {
			trackingNumber = Package.Tracking.TrackingNumber
		}

		pkgCode := ""
		if Package.PackageCode != nil {
			pkgCode = Package.PackageCode.Code
		}

		if Package.Status == constant.PackageStatusCreated {
			pkgCode = ""
		}

		tmp := []string{
			pkgCode,
			trackingNumber,
			Package.OrderNumber,
			Package.Detail,
		}

		// lấy mã lô
		options := sqlmanager.ShipmentQueryOptions{
			LoadContainer: true,
			PackageID:     Package.ID,
		}

		shipmentNumber := ""
		shipment, _ := h.ShipmentManager.GetShipment(options)
		if shipment != nil {
			shipmentNumber = cast.ToString(shipment.ID)
		}

		// tính phí phát sinh
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

		var checkinTimeStr string
		if Package.CheckinWarehouseAt != nil {
			checkinTimeStr = Package.CheckinWarehouseAt.Add(7 * time.Hour).Format("2006-01-02T15:04:05")
		}

		tmp = append(tmp, []string{
			shipmentNumber,
			cast.ToString(Package.CreatedAt.Add(7 * time.Hour).Format("2006-01-02T15:04:05")),
			cast.ToString(checkinTimeStr),
			Package.User.FullName,
			Package.Recipient,
			Package.PhoneNumber,
			Package.Address1,
			Package.Address2,
			Package.City,
			Package.StateCode,
			Package.Zipcode,
			Package.CountryCode,
			cast.ToString(Package.Weight),
			cast.ToString(Package.Length),
			cast.ToString(Package.Width),
			cast.ToString(Package.Height),
			Package.Service.Name,
			fmt.Sprintf("%.2f", Package.ShippingFee+extraFee),
			fmt.Sprintf("%.2f", Package.ShippingFee),
			fmt.Sprintf("%.2f", extraFee),
			cast.ToString(constant.MapTextStatusAdminPackage[Package.Status]),
		}...)

		for indexCol, item := range tmp {
			cellStr := fmt.Sprintf("%s%d", toCharStr(indexCol+1), indexRow+2)
			if err := f1.SetCellValue(sheetName, cellStr, item); err != nil {
				h.Logger.Errorf("Write xlsx row %v column %v error: %v", indexRow+2, indexCol+1, err)
			}
		}
	}

	// Save the file locally
	if err := f1.SaveAs(fileName); err != nil {
		h.Logger.Errorf("save file error: %v", err)
		return "", err
	}

	filePath := fmt.Sprintf("export/package/%s/%s%s",
		viper.GetString("env"),
		file.MakeUploadUserIDPath(userID, ""),
		fileName,
	)

	buffer, err := ioutil.ReadFile(fileName)
	if err != nil {
		h.Logger.Errorf("read file error: %v", err)
		return "", err
	}
	defer os.Remove(fileName)

	contentType := constant.ContentTypeXlsxExcel2007Above
	bucketExport := viper.GetString("bucket.export_packages")
	if err := h.S3.UploadFile(bytes.NewBuffer(buffer), filePath, bucketExport, contentType); err != nil {
		h.Logger.Errorf("upload file error: %v", err)
		return "", err
	}

	return filePath, nil
}

func (h *ExportHandler) exportShipmentXlsx(shipment []ShipmentExport, userID int64) (string, error) {
	header := []string{
		"ID Đơn hàng",
		"Ananbay tracking",
		"Last mile tracking",
		"Mã đơn hàng",
		"Chi tiết hàng hóa",
		"Tên khách hàng",
		"Tên người nhận",
		"Địa chỉ",
		"Phí",
		"Weight",
		"Length",
		"Width",
		"Height",
		"Mã kiện",
		"Trạng thái đơn hàng ",
	}

	f1 := excelize.NewFile()

	for index, item := range header {
		colStr := fmt.Sprintf("%s1", toCharStr(index+1))
		err := f1.SetCellValue("Sheet1", colStr, item)
		if err != nil {
			h.Logger.Errorf("Write xlsx header error: %v", err)
		}
	}

	fileName := fmt.Sprintf("danh_sach_don_hang_%d_%02d_%02d.xlsx", time.Now().Day(), int(time.Now().Month()), time.Now().Year())
	f, err := os.Create(fileName)
	if err != nil {
		h.Logger.Errorf("Create file tmp: %v", err)
		return "", err
	}
	defer f.Close()
	defer os.Remove(fileName)
	writer := csv.NewWriter(f)
	defer writer.Flush()

	for indexRow, item := range shipment {
		tmp := []string{}
		tmp = append(tmp, []string{
			cast.ToString(item.ID),
			item.Code,
			item.TrackingNumber,
			item.OrderNumber,
			item.Detail,
			item.FullName,
			item.Recipient,
			item.Address1,
			cast.ToString(item.ShippingFee),
			cast.ToString(item.Weight),
			cast.ToString(item.Length),
			cast.ToString(item.Width),
			cast.ToString(item.Height),
			item.ContainerCode,
			cast.ToString(constant.MapTextStatusAdminPackage[item.Status]),
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

	filePath := fmt.Sprintf("export/shipment/%s/%s%s", viper.GetString("env"), file.MakeUploadUserIDPath(userID, ""), fileName)

	buffer, err := ioutil.ReadFile(fileName)
	if err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}
	f.Read(buffer)

	contentType := constant.ContentTypeXlsxExcel2007Above
	bucketExport := viper.GetString("bucket.export_packages")
	err = h.S3.UploadFile(bytes.NewBuffer(buffer), filePath, bucketExport, contentType)

	if err != nil {
		h.Logger.Errorf("read file error: %v/n", err)
		return "", err
	}

	return filePath, nil
}

func AdmintoCharStr(i int) string {
	var col string
	for i > 0 {
		rem := (i - 1) % 26
		col = string(rune('A'+rem)) + col
		i = (i - 1) / 26
	}
	return col
}

func toCharStr(i int) string {
	return string(rune('A' - 1 + i))
}
