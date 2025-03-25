package customer

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils/string_util"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProductHandler struct {
	Logger *zap.SugaredLogger

	ProductManager *sqlmanager.ProductManager
}

type CreateProductRequest struct {
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Stock    int64   `json:"stock"`
	Price    float64 `json:"price"`
	Weight   float64 `json:"weight"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
	Height   float64 `json:"height"`
	Detail   string  `json:"detail"`
	Material string  `json:"material"`
	Country  string  `json:"country"`
}

type CreateProductResponse struct {
	Product *entity.Product `json:"product"`
}

type GetListProductsResponse struct {
	Products []*entity.Product `json:"products"`
}

type CountListProductsResponse struct {
	Count int64 `json:"count"`
}

type UpdateForm struct {
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Stock    int64   `json:"stock"`
	Price    float64 `json:"price"`
	Weight   float64 `json:"weight"`
	Width    float64 `json:"width"`
	Length   float64 `json:"length"`
	Height   float64 `json:"height"`
	Country  string  `json:"country"`
	Detail   string  `json:"detail"`
	Material string  `json:"material"`
}

type UpdateProductResponse struct {
	Success bool `json:"success"`
}

type DeleteProductResponse struct {
	Success bool `json:"success"`
}

func NewProductHandler(l *zap.SugaredLogger, prm *sqlmanager.ProductManager) *ProductHandler {
	return &ProductHandler{
		Logger: l,

		ProductManager: prm,
	}
}

func (h *ProductHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": constant.MessagePermissionDenied})
			return
		}

		CreateProductInfo := &CreateProductRequest{}
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(CreateProductInfo); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if CreateProductInfo.Name = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(CreateProductInfo.Name); CreateProductInfo.Name == "" {
			c.JSON(http.StatusBadRequest, "Tên hàng hóa không để trống")
			return
		} else {
			if len(CreateProductInfo.Name) > 50 {
				c.JSON(http.StatusBadRequest, "Tên hàng hóa tối đa là 50 ký tự")
				return
			}
		}

		CreateProductInfo.SKU = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(CreateProductInfo.SKU)
		if CreateProductInfo.SKU == "" {
			c.JSON(http.StatusBadRequest, "SKU không để trống")
			return
		}
		if len(CreateProductInfo.SKU) > 50 {
			c.JSON(http.StatusBadRequest, "SKU tối đa là 50 ký tự")
			return
		}

		if CreateProductInfo.Price <= 0 {
			c.JSON(http.StatusBadRequest, "Giá sản phẩm không hợp lệ")
			return
		}

		if strings.ToUpper(CreateProductInfo.Country) != constant.UsCountryCode && strings.ToUpper(CreateProductInfo.Country) != constant.AuCountryCode {
			c.JSON(http.StatusBadRequest, "Quốc gia không hợp lệ")
			return
		}

		if strings.ToUpper(CreateProductInfo.Country) == constant.UsCountryCode {
			if CreateProductInfo.Weight > 31751.46 {
				c.JSON(http.StatusBadRequest, "Trọng lượng tối đa là 31751.46 Gram")
				return
			}

			if CreateProductInfo.Width > 90 {
				c.JSON(http.StatusBadRequest, "Chiều rộng tối đa là 90cm")
				return
			}

			if CreateProductInfo.Length > 90 {
				c.JSON(http.StatusBadRequest, "Chiều dài tối đa là 90cm")
				return
			}

			if CreateProductInfo.Height > 90 {
				c.JSON(http.StatusBadRequest, "Chiều cao tối đa là 90cm")
				return
			}

			if CreateProductInfo.Length+2*(CreateProductInfo.Width+CreateProductInfo.Height) > 330.2 {
				c.JSON(http.StatusBadRequest, "Chiều dài + 2*(chiều cao + chiều rộng)” không vượt quá 330.2 cm")
				return
			}
		}

		if strings.ToUpper(CreateProductInfo.Country) == constant.AuCountryCode {
			if CreateProductInfo.Weight > 22000 {
				c.JSON(http.StatusBadRequest, "Trọng lượng tối đa là 22000 Gram")
				return
			}

			if CreateProductInfo.Width > 105 {
				c.JSON(http.StatusBadRequest, "Chiều rộng tối đa là 105cm")
				return
			}

			if CreateProductInfo.Length > 105 {
				c.JSON(http.StatusBadRequest, "Chiều dài tối đa là 105cm")
				return
			}

			if CreateProductInfo.Height > 105 {
				c.JSON(http.StatusBadRequest, "Chiều cao tối đa là 105cm")
				return
			}

			if CreateProductInfo.Length*CreateProductInfo.Width*CreateProductInfo.Height > 250000 {
				c.JSON(http.StatusBadRequest, "Chiều dài*chiều rộng*chiều cao không vượt quá 0.25m3")
				return
			}

		}

		Product := &entity.Product{
			Name:     CreateProductInfo.Name,
			UserID:   userID,
			SKU:      CreateProductInfo.SKU,
			Stock:    CreateProductInfo.Stock,
			Price:    CreateProductInfo.Price,
			Weight:   CreateProductInfo.Weight,
			Width:    CreateProductInfo.Width,
			Length:   CreateProductInfo.Length,
			Height:   CreateProductInfo.Height,
			Detail:   CreateProductInfo.Detail,
			Material: CreateProductInfo.Material,
			Country:  strings.ToUpper(CreateProductInfo.Country),
			Status:   constant.StatusActive,
		}

		product, err := h.ProductManager.CreateProductOrAddStock(Product)
		if err != nil {
			h.Logger.Errorf("create product error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateProductResponse{Product: product})
	}
}

func (h *ProductHandler) Import() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": constant.MessagePermissionDenied})
			return
		}

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed"})
			return
		}
		defer file.Close()

		xlsxFile, err := excelize.OpenReader(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file format"})
			return
		}

		rows, err := xlsxFile.GetRows("Sheet1")
		if err != nil || len(rows) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Empty or invalid sheet"})
			return
		}

		var total int
		var importSuccess int
		var importErrors []map[string]interface{}

		for i, row := range rows[1:] {
			if len(row) < 9 {
				importErrors = append(importErrors, map[string]interface{}{
					"line":     i + 2,
					"value":    row,
					"messages": "Thiếu trường bắt buộc",
				})
				continue
			}

			product := &entity.Product{
				Name:     row[0],
				SKU:      row[1],
				Stock:    cast.ToInt64(row[2]),
				Detail:   row[3],
				Material: row[4],
				Country:  strings.ToUpper(row[5]),
				Weight:   cast.ToFloat64(row[6]),
				Length:   cast.ToFloat64(row[7]),
				Width:    cast.ToFloat64(row[8]),
				Height:   cast.ToFloat64(row[9]),
				UserID:   userID,
				Status:   constant.StatusActive,
			}

			if err := validateProduct(product); err != nil {
				importErrors = append(importErrors, map[string]interface{}{
					"line":     i + 2,
					"value":    row,
					"messages": err.Error(),
				})
				continue
			}

			_, err := h.ProductManager.CreateProductOrAddStock(product)
			if err != nil {
				importErrors = append(importErrors, map[string]interface{}{
					"line":     i + 2,
					"value":    row,
					"messages": "Failed to insert into database",
				})
				continue
			}

			importSuccess++
			total++
		}

		if len(importErrors) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"total":          total,
				"import_success": importSuccess,
				"errors":         importErrors,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":          total,
			"import_success": importSuccess,
			"errors":         nil,
		})
	}
}

func validateProduct(p *entity.Product) error {
	if p.Name == "" {
		return errors.New("Tên sản phẩm là bắt buộc")
	}
	if len(p.SKU) > 50 {
		return errors.New("Độ dài SKU không được vượt quá 50 ký tự")
	}
	if p.Weight <= 0 {
		return errors.New("Khối lượng phải lớn hơn 0")
	}
	if p.Length <= 0 || p.Width <= 0 || p.Height <= 0 {
		return errors.New("Kích thước phải lớn hơn 0")
	}
	return nil
}

func (h *ProductHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)
		search := c.Request.URL.Query().Get("search")
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		opt := sqlmanager.ProductQueryOption{
			UserID: userID,
			Limit:  limit,
			Offset: offset,
			Search: search,
			Status: constant.StatusActive,
		}
		products, err := h.ProductManager.GetProducts(opt)
		if err != nil {
			h.Logger.Errorf("Get products error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		result := &GetListProductsResponse{Products: products}

		c.JSON(http.StatusOK, result)
	}
}

func (h *ProductHandler) ProductLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		productID := cast.ToInt64(c.Param("product_id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		productLog, err := h.ProductManager.GetProductLogs(productID)
		if err != nil {
			h.Logger.Errorf("Get product logs error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, productLog)
	}
}

func (h *ProductHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Request.URL.Query().Get("search")
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		opt := sqlmanager.ProductQueryOption{
			UserID: userID,
			Search: search,
			Status: constant.StatusActive,
		}
		count, err := h.ProductManager.CountProducts(opt)
		if err != nil {
			h.Logger.Errorf("count products error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountListProductsResponse{Count: count})
	}
}

func (h *ProductHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		productID := cast.ToInt64(c.Param("product_id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		currentProduct, err := h.ProductManager.GetProductByID(productID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Product Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if currentProduct.UserID != userID {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		UpdateForm := &UpdateForm{}
		decoder := json.NewDecoder(c.Request.Body)
		if err := decoder.Decode(UpdateForm); err != nil {
			h.Logger.Errorf("Error parsing json: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if UpdateForm.Name = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.Name); UpdateForm.Name == "" {
			c.JSON(http.StatusBadRequest, "Tên hàng hóa không để trống")
			return
		} else {
			if len(UpdateForm.Name) > 50 {
				c.JSON(http.StatusBadRequest, "Tên hàng hóa tối đa là 50 ký tự")
				return
			}
		}

		UpdateForm.SKU = string_util.RemoveInvalidUTF8CharactersAndTrimSpace(UpdateForm.SKU)
		if UpdateForm.SKU == "" {
			c.JSON(http.StatusBadRequest, "SKU không để trống")
			return
		} else {
			if len(UpdateForm.SKU) > 50 {
				c.JSON(http.StatusBadRequest, "SKU tối đa là 50 ký tự")
				return
			}
		}
		if UpdateForm.Stock < 0 {
			c.JSON(http.StatusBadRequest, "Số lượng không hợp lệ")
			return
		}
		if strings.ToUpper(UpdateForm.Country) != constant.UsCountryCode && strings.ToUpper(UpdateForm.Country) != constant.AuCountryCode {
			c.JSON(http.StatusBadRequest, "Quốc gia không hợp lệ")
			return
		}

		if strings.ToUpper(UpdateForm.Country) == constant.UsCountryCode {
			if UpdateForm.Weight > 31751.46 {
				c.JSON(http.StatusBadRequest, "Trọng lượng tối đa là 31751.46 Gram")
				return
			}

			if UpdateForm.Width > 90 {
				c.JSON(http.StatusBadRequest, "Chiều rộng tối đa là 90cm")
				return
			}

			if UpdateForm.Length > 90 {
				c.JSON(http.StatusBadRequest, "Chiều dài tối đa là 90cm")
				return
			}

			if UpdateForm.Height > 90 {
				c.JSON(http.StatusBadRequest, "Chiều cao tối đa là 90cm")
				return
			}

			if UpdateForm.Length+2*(UpdateForm.Width+UpdateForm.Height) > 330.2 {
				c.JSON(http.StatusBadRequest, "Chiều dài + 2*(chiều cao + chiều rộng)” không vượt quá 330.2 cm")
				return
			}
		}

		if strings.ToUpper(UpdateForm.Country) == constant.AuCountryCode {
			if UpdateForm.Weight > 22000 {
				c.JSON(http.StatusBadRequest, "Trọng lượng tối đa là 22000 Gram")
				return
			}

			if UpdateForm.Width > 105 {
				c.JSON(http.StatusBadRequest, "Chiều rộng tối đa là 105cm")
				return
			}

			if UpdateForm.Length > 105 {
				c.JSON(http.StatusBadRequest, "Chiều dài tối đa là 105cm")
				return
			}

			if UpdateForm.Height > 105 {
				c.JSON(http.StatusBadRequest, "Chiều cao tối đa là 105cm")
				return
			}

			if UpdateForm.Length*UpdateForm.Width*UpdateForm.Height > 250000 {
				c.JSON(http.StatusBadRequest, "Chiều dài*chiều rộng*chiều cao không vượt quá 0.25m3")
				return
			}
		}

		mapchange := make(map[string]interface{})

		if UpdateForm.Name != currentProduct.Name {
			mapchange["name"] = UpdateForm.Name
		}

		if UpdateForm.SKU != currentProduct.SKU {
			isExist, err := h.ProductManager.CheckSKUExist(UpdateForm.SKU, userID)
			if err != nil {
				h.Logger.Errorf("check SKU exist error: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
			if isExist {
				c.JSON(http.StatusBadRequest, "SKU đã tồn tại!")
				return
			}

			mapchange["sku"] = UpdateForm.SKU
		}

		if UpdateForm.Weight != currentProduct.Weight {
			mapchange["weight"] = UpdateForm.Weight
		}

		if UpdateForm.Stock != currentProduct.Stock {
			mapchange["stock"] = UpdateForm.Stock
		}

		if UpdateForm.Price != currentProduct.Price {
			mapchange["price"] = UpdateForm.Price
		}

		if UpdateForm.Width != currentProduct.Width {
			mapchange["width"] = UpdateForm.Width
		}

		if UpdateForm.Length != currentProduct.Length {
			mapchange["length"] = UpdateForm.Length
		}

		if UpdateForm.Height != currentProduct.Height {
			mapchange["height"] = UpdateForm.Height
		}

		if UpdateForm.Detail != currentProduct.Detail {
			mapchange["detail"] = UpdateForm.Detail
		}

		if UpdateForm.Material != currentProduct.Material {
			mapchange["material"] = UpdateForm.Material
		}

		if strings.ToUpper(UpdateForm.Country) != currentProduct.Country {
			mapchange["country"] = strings.ToUpper(UpdateForm.Country)
		}

		if len(mapchange) == 0 {
			c.JSON(http.StatusOK, UpdateProductResponse{Success: true})
			return
		}

		err = h.ProductManager.UpdateProduct(productID, mapchange)
		if err != nil {
			h.Logger.Errorf("update product error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdateProductResponse{Success: true})

	}
}

func (h *ProductHandler) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		productID := cast.ToInt64(c.Param("product_id"))

		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		currentProduct, err := h.ProductManager.GetProductByID(productID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Product Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if currentProduct.UserID != userID {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		mapchange := map[string]interface{}{
			"Status": constant.StatusDeactive,
		}

		err = h.ProductManager.UpdateProduct(productID, mapchange)
		if err != nil {
			h.Logger.Errorf("delete product error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, DeleteProductResponse{Success: true})
	}
}
