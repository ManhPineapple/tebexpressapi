package admin

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/array"
	"tebexpressapi/pkg/utils/file"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PromotionHandler struct {
	Logger    *zap.SugaredLogger
	Redis     *redis.Client
	StorageS3 storage.S3

	PromotionManager *sqlmanager.PromotionManager
	ServiceManager   *sqlmanager.ServiceManager
	UserManager      *sqlmanager.UserManager
}

type createPromotionResponse struct {
	Success bool          `json:"success"`
	Error   string        `json:"error"`
	Errors  []ImportError `json:"errors"`
}

type ImportError struct {
	Line     int64    `json:"line"`
	Value    []string `json:"value"`
	Messages []string `json:"messages"`
}

type CreateSettingPointForm struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Class int     `json:"class"`
	Value float64 `json:"value"`
}

type CreateSettingPoinResponse struct {
	Success bool `json:"success"`
}

type CountResponse struct {
	Count int64 `json:"count"`
}

type FetchPromotionsResponse struct {
	Promotions []*entity.Promotion `json:"promotions"`
}

type FetchPromotionUsersResponse struct {
	Users []entity.User `json:"users"`
}

type GetListSettingPointResponse struct {
	Settings []*entity.SettingPoint `json:"settings"`
}

type AppendForm struct {
	PromotionID int64   `json:"promotion_id"`
	UserID      []int64 `json:"user_id"`
	Tester      bool    `json:"tester"`
}

type AppendUserToPromotionResponse struct {
	Success bool `json:"success"`
}

type UpdatePromotionResponse struct {
	Success bool `json:"success"`
}

type UpdateSettingPointForm struct {
	ID     int64    `json:"id"`
	From   *float64 `json:"from"`
	To     *float64 `json:"to"`
	Value  float64  `json:"value"`
	Class  int      `json:"class"`
	Status int      `json:"status"`
}

type UpdateSettingPointResponse struct {
	Success bool `json:"success"`
}

func NewPromotionManager(l *zap.SugaredLogger, r *redis.Client, s3 storage.S3, prm *sqlmanager.PromotionManager, sm *sqlmanager.ServiceManager, um *sqlmanager.UserManager) *PromotionHandler {
	return &PromotionHandler{
		Logger:    l,
		Redis:     r,
		StorageS3: s3,

		PromotionManager: prm,
		ServiceManager:   sm,
		UserManager:      um,
	}
}

// Create Promotion
func (h *PromotionHandler) CreatePromotion() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		name := string_util.StripTags(strings.TrimSpace(c.Request.PostFormValue("name")))
		if name == "" {
			c.JSON(http.StatusBadRequest, "Tên là bắt buộc")
			return
		}

		if len(name) > 250 {
			c.JSON(http.StatusBadRequest, "Tên không được quá 250 ký tự")
			return
		}

		description := string_util.StripTags(strings.TrimSpace(c.Request.PostFormValue("description")))
		if len(description) > 500 {
			c.JSON(http.StatusBadRequest, "Mô tả không được quá 500 ký tự")
			return
		}

		promotion := &entity.Promotion{
			Name:        name,
			Description: description,
			UserID:      userID,
			Type:        constant.PromotionTypeMarketing,
			Status:      constant.PromotionStatusDraft,
		}

		services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
			Status: constant.StatusActive,
		})
		if err != nil {
			h.Logger.Errorf("get services: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		path, prices, importErrors, message, err := h.parseContentPrice(promotion.Name, h.StorageS3, c.Request, services)
		if message != "" {
			c.JSON(http.StatusBadRequest, message)
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(importErrors) > 0 {
			c.JSON(http.StatusBadRequest, createPromotionResponse{
				Success: false,
				Error:   "File chỉnh sửa giá sai định dạng",
				Errors:  importErrors,
			})
			return
		}

		promotion.S3PathPrice = path
		promotion.Prices = prices

		path, weights, importErrors, message, err := h.parseContentWeight(promotion.Name, h.StorageS3, c.Request)
		if message != "" {
			c.JSON(http.StatusBadRequest, message)
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if len(importErrors) > 0 {
			c.JSON(http.StatusBadRequest, createPromotionResponse{
				Success: false,
				Error:   "File chỉnh sửa cân nặng sai định dạng",
				Errors:  importErrors,
			})
			return
		}

		promotion.S3PathWeight = path
		promotion.Weights = weights

		if err := h.PromotionManager.CreatePromotion(promotion); err != nil {
			h.Logger.Errorf("create product error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, createPromotionResponse{Success: true})
	}
}

func (h *PromotionHandler) parseContentPrice(name string, s3 storage.S3, r *http.Request, services []*entity.Service) (string, []*entity.PromotionPrice, []ImportError, string, error) {
	prices := make([]*entity.PromotionPrice, 0)
	importErrors := make([]ImportError, 0)

	s3path, msg, err := h.readFile(s3, r, name+"_prices", "file_price")
	if msg != "" || err != nil || s3path == "" {
		return "", prices, importErrors, msg, err
	}

	bucketImport := viper.GetString("bucket.static")
	object, err, _ := h.StorageS3.ReadFileForUpload(s3path, bucketImport)
	if err != nil {
		h.Logger.Errorf("Error ReadFile from %s S3 %v", s3path, err)
		return "", prices, importErrors, "", err
	}

	columnService := 0
	columnWeight := 1
	columnPrice := 2

	ef, err := excelize.OpenReader(object)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return "", prices, importErrors, "File chỉnh sửa giá sai định dạng", err
	}

	rows, err := ef.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return "", prices, importErrors, "File chỉnh sửa giá sai định dạng", err
	}

	var keys = []string{}
	var mapPrices = make(map[string]entity.PromotionPrice)

	var mapServices = make(map[string]int64)
	for _, v := range services {
		name := strings.ToUpper(v.Name)
		mapServices[name] = v.ID

		code := strings.ToUpper(v.Code)
		mapServices[code] = v.ID
	}

	for indexRow, row := range rows {
		if len(row) < 3 {
			return "", prices, importErrors, "File chỉnh sửa giá sai định dạng", err
		}

		if indexRow == 0 {
			continue
		}

		errMsg := make([]string, 0)
		var value []string

		serviceCode := strings.ToUpper(strings.TrimSpace(row[columnService]))
		if serviceCode == "" {
			value = append(value, row[columnService])
			errMsg = append(errMsg, "Service code không để trống")
		} else if mapServices[serviceCode] < 1 {
			value = append(value, row[columnService])
			errMsg = append(errMsg, "Service code không hợp lệ")
		}

		weight := cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		if weight <= 0 {
			value = append(value, row[columnWeight])
			errMsg = append(errMsg, "Cân nặng không để trống")
		}
		if weight > constant.PriceMaximumWeight {
			value = append(value, row[columnWeight])
			errMsg = append(errMsg, fmt.Sprintf("Cân nặng không > %v gram", utils.Ceil(constant.PriceMaximumWeight, 0)))
		}

		price := cast.ToFloat64(strings.TrimSpace(row[columnPrice]))
		if price <= 0 {
			value = append(value, row[columnPrice])
			errMsg = append(errMsg, "Giá không để trống")
		}

		key := fmt.Sprintf("%d-%v", mapServices[serviceCode], weight)
		if mapPrices[key].Weight > 0 {
			value = append(value, row[columnWeight])
			errMsg = append(errMsg, "Trùng dữ liệu")
		}

		if len(errMsg) > 0 {
			importErrors = append(importErrors, ImportError{
				Line:     int64(indexRow) + 1,
				Value:    value,
				Messages: errMsg,
			})

			continue
		}

		promotionPrice := entity.PromotionPrice{
			ServiceID: mapServices[serviceCode],
			Weight:    weight,
			Price:     price,
		}

		keys = append(keys, key)
		mapPrices[key] = promotionPrice
	}

	if len(keys) != len(mapPrices) {
		return s3path, prices, importErrors, "File upload sai định dạng", nil
	}

	sort.Strings(keys)
	for _, key := range keys {
		p := mapPrices[key]
		prices = append(prices, &p)
	}

	return s3path, prices, importErrors, "", nil
}

func (h *PromotionHandler) parseContentWeight(name string, s3 storage.S3, r *http.Request) (string, []*entity.PromotionWeight, []ImportError, string, error) {
	items := make([]*entity.PromotionWeight, 0)
	importErrors := make([]ImportError, 0)

	s3path, msg, err := h.readFile(s3, r, name+"_weights", "file_weight")
	if msg != "" || err != nil || s3path == "" {
		return "", items, importErrors, msg, err
	}

	bucketImport := viper.GetString("bucket.static")
	object, err, _ := h.StorageS3.ReadFileForUpload(s3path, bucketImport)
	if err != nil {
		h.Logger.Errorf("Error ReadFile from %s S3 %v", s3path, err)
		return "", items, importErrors, "", err
	}

	columnWeight := 0
	columnErrorWeightAllow := 1

	f, err := excelize.OpenReader(object)
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return "", items, importErrors, "File chỉnh sửa cân nặng sai định dạng!", err
	}

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		h.Logger.Errorf("Error when process excel file: %v", err)
		return "", items, importErrors, "File chỉnh sửa cân nặng sai định dạng!", err
	}

	var keys = []float64{}
	var mapItems = make(map[float64]entity.PromotionWeight)

	for indexRow, row := range rows {
		if indexRow == 0 {
			continue
		}

		errMsg := make([]string, 0)
		var value []string

		weight := cast.ToFloat64(strings.TrimSpace(row[columnWeight]))
		if weight <= 0 {
			value = append(value, row[columnWeight])
			errMsg = append(errMsg, "Mốc cận nặng không để trống")
		}

		price := cast.ToFloat64(strings.TrimSpace(row[columnErrorWeightAllow]))
		if price <= 0 {
			value = append(value, row[columnErrorWeightAllow])
			errMsg = append(errMsg, "Sai số cho phép không để trống")
		}

		if mapItems[weight].Weight > 0 {
			value = append(value, row[columnWeight])
			errMsg = append(errMsg, "Trùng dữ liệu")
		}

		if len(errMsg) > 0 {
			importErrors = append(importErrors, ImportError{
				Line:     int64(indexRow) + 1,
				Value:    value,
				Messages: errMsg,
			})

			continue
		}

		item := entity.PromotionWeight{
			Weight:           weight,
			ErrorWeightAllow: price,
		}

		keys = append(keys, weight)
		mapItems[weight] = item
	}

	if len(keys) != len(mapItems) {
		return "", items, importErrors, "file upload sai format", nil
	}

	sort.Float64s(keys)
	for _, key := range keys {
		p := mapItems[key]
		items = append(items, &p)
	}

	return s3path, items, importErrors, "", nil
}

func (h *PromotionHandler) readFile(s3 storage.S3, r *http.Request, name, fieldName string) (string, string, error) {
	mf, fh, err := r.FormFile(fieldName)
	if err == http.ErrMissingFile {
		return "", "", nil
	}

	if err != nil {
		return "", "", err
	}

	defer mf.Close()

	if !constant.AllowExtensionFileImportPackage[fh.Header.Get("Content-Type")] {
		h.Logger.Errorf("Format file is not allow: ", fh.Header.Get("Content-Type"))
		return "", "File upload sai định dạng.", nil
	}

	bucketImport := viper.GetString("bucket.static")

	userID := cast.ToInt64(r.Header.Get("X-User-Id"))
	filename := strings.ToLower(string_util.ToSlug(name))
	ext := file.GetExtensionFromString(fh.Filename)
	filePath := fmt.Sprintf("promotions/%d_%s_%d%s", userID, filename, time.Now().UnixNano(), ext)

	f, err := fh.Open()
	if err != nil {
		h.Logger.Errorf("Cant Open file: ", err)
		return "", constant.MessageValidateInput, err
	}

	fileBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return "", "", err
	}

	mineType := http.DetectContentType(fileBytes)
	err = s3.UploadFile(bytes.NewReader(fileBytes), filePath, bucketImport, mineType)
	if err != nil {
		h.Logger.Errorf("Error UploadFile to %s S3 %v", filePath, err)
		return "", "", err
	}

	return filePath, "", nil
}

// End Create Promotion

func (h *PromotionHandler) CreateSettingPoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		form := CreateSettingPointForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.From < 0 || form.To <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if form.From >= form.To {
			c.JSON(http.StatusBadRequest, "Giá trị đầu phải nhỏ hơn giá trị cuối !")
			return
		}

		if form.Class != constant.UserClassPriority && form.Class != constant.UserClassPartner && form.Class != constant.UserClassPublic {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		opts := sqlmanager.SettingPointQueryOption{
			Status: constant.SettingEnableStatus,
			Class:  form.Class,
		}

		settings, err := h.PromotionManager.GetSettingPoints(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list setting point error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if form.Value > 0 {
			for _, item := range settings {
				if (form.From > item.From && form.From < item.To) || (form.To > item.From && form.To < item.To) || form.From == item.From || form.To == item.To {
					c.JSON(http.StatusBadRequest, "Khoảng setup đã tồn tại !")
					return
				}
			}
		}

		setting := &entity.SettingPoint{
			From:   form.From,
			To:     form.To,
			Value:  form.Value,
			Class:  form.Class,
			Status: constant.SettingStatusActive,
		}

		err = h.PromotionManager.SaveSettingPoint(setting)
		if err != nil {
			h.Logger.Errorf("Save setting point error: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateSettingPoinResponse{Success: true})
	}
}

func (h *PromotionHandler) CountPromotion() gin.HandlerFunc {
	return func(c *gin.Context) {
		opt := sqlmanager.PromotionQueryOption{
			Search: c.Request.URL.Query().Get("search"),
			Status: cast.ToInt64(c.Request.URL.Query().Get("status")),
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleMarketing {
			opt.UserID = cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		}

		if role != constant.UserRoleAdmin && role != constant.UserRolerBusinessManager {
			opt.Type = constant.PromotionTypeMarketing
		}

		count, err := h.PromotionManager.CountPromotions(opt)
		if err != nil {
			h.Logger.Errorf("count products error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountResponse{Count: count})
	}
}

func (h *PromotionHandler) CountListSettingPoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}
		opts := sqlmanager.SettingPointQueryOption{
			Status: constant.SettingEnableStatus,
		}

		count, err := h.PromotionManager.CountSettingPoints(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list setting point error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CountResponse{count})
	}
}

func (h *PromotionHandler) GetPromotions() gin.HandlerFunc {
	return func(c *gin.Context) {
		offset, limit := httputil.GetRequestPaginate(c.Request)

		opt := sqlmanager.PromotionQueryOption{
			Limit:  limit,
			Offset: offset,
			Search: c.Request.URL.Query().Get("search"),
			Status: cast.ToInt64(c.Request.URL.Query().Get("status")),
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleMarketing {
			opt.UserID = cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		}

		if role != constant.UserRoleAdmin && role != constant.UserRolerBusinessManager {
			opt.Type = constant.PromotionTypeMarketing
		}

		promotions, err := h.PromotionManager.GetPromotions(opt)
		if err != nil && err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		ids := []int64{}
		for _, v := range promotions {
			ids = append(ids, v.ID)
		}

		users, err := h.PromotionManager.GetUsersByPromotionIDs(ids, []int64{})
		if err != nil {
			h.Logger.Errorf("Error while get ids user, details: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapUsers := make(map[int64][]int64)
		for _, v := range users {
			mapUsers[v.PromotionID] = append(mapUsers[v.PromotionID], v.UserID)
		}

		for i, v := range promotions {
			promotions[i].UserIDS = mapUsers[v.ID]
		}

		result := &FetchPromotionsResponse{Promotions: promotions}
		c.JSON(http.StatusOK, result)
	}
}

func (h *PromotionHandler) GetPromotionUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := cast.ToInt64(c.Param("id"))

		isTester := cast.ToBool(c.Request.URL.Query().Get("tester"))
		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if isTester {
			ignoreUsers = []int64{}
		}

		users, err := h.PromotionManager.GetUsersByPromotionID(id, ignoreUsers)
		if err != nil {
			h.Logger.Errorf("fetch users: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, FetchPromotionUsersResponse{Users: users})
	}
}

func (h *PromotionHandler) GetListSettingPoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)
		opts := sqlmanager.SettingPointQueryOption{
			Limit:  limit,
			Offset: offset,
			Order:  "`from` ASC",
			Status: constant.SettingEnableStatus,
		}

		settings, err := h.PromotionManager.GetSettingPoints(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list setting point error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GetListSettingPointResponse{settings})
	}
}

func (h *PromotionHandler) AppendUserToPromotion() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		_, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		form := AppendForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.PromotionID < 1 {
			h.Logger.Errorf("Invalid promotion id %v", userID)
			c.JSON(http.StatusBadRequest, "Promotion_id required")
			return
		}

		currentPromotion, err := h.PromotionManager.GetPromotionByID(form.PromotionID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Promotion Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if currentPromotion.Status != constant.StatusActive {
			c.JSON(http.StatusBadRequest, "Promotion is deactived")
			return
		}

		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if form.Tester {
			ignoreUsers = []int64{}
		}

		var idsDelete, idsAdd []int64

		if len(form.UserID) > 0 {
			users, err := h.PromotionManager.GetUsersByPromotionID(form.PromotionID, ignoreUsers)
			if err != nil {
				h.Logger.Errorf("Error while get ids user, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			var ids = []int64{}
			for _, user := range users {
				if !utils.ContainsNumber(form.UserID, user.ID) {
					idsDelete = append(idsDelete, user.ID)
					continue
				}

				ids = append(ids, user.ID)
			}

			promotions, err := h.PromotionManager.GetMarketingPromotionsByUserIDs(form.UserID, []int64{currentPromotion.ID})
			if err != nil {
				h.Logger.Errorf("Error while get ids user, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			mapUserPromotions := make(map[int64]entity.UserPromotion)
			for _, v := range promotions {
				mapUserPromotions[v.UserID] = v
			}

			for _, id := range form.UserID {
				if !utils.ContainsNumber(ids, id) {
					up := mapUserPromotions[id]
					if currentPromotion.Type == constant.PromotionTypeMarketing && up.PromotionID > 0 && up.PromotionID != currentPromotion.ID {
						name := fmt.Sprintf("#%d", up.UserID)
						if up.User != nil {
							name = up.User.Email

							if up.User.FullName != "" {
								name = up.User.FullName
							}
						}
						c.JSON(http.StatusBadRequest, fmt.Sprintf("Khách hàng %s đã được thêm vào promotion #%d", name, up.PromotionID))
						return
					}

					idsAdd = append(idsAdd, id)
				}
			}
		}

		err = h.PromotionManager.AppendUserToPromotion(form.PromotionID, idsDelete, idsAdd)
		if err != nil {
			h.Logger.Errorf("add user to promotion %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, "Cập nhật thất bại")
			return
		}

		c.JSON(http.StatusOK, AppendUserToPromotionResponse{Success: true})

	}
}

func (h *PromotionHandler) UpdatePromotion() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		promotionID := cast.ToInt64(c.Param("id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		promotion, err := h.PromotionManager.GetPromotionByID(promotionID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get Promotion Detail %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleMarketing && promotion.UserID != userID {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		var promotionPrices = []*entity.PromotionPrice{}
		var promotionWeights = []*entity.PromotionWeight{}

		if promotion.Status != constant.PromotionStatusActive {
			name := string_util.StripTags(strings.TrimSpace(c.Request.PostFormValue("name")))
			if name != "" {
				promotion.Name = name
			}

			if len(name) > 250 {
				c.JSON(http.StatusBadRequest, "Tên không được quá 250 ký tự")
				return
			}

			description := string_util.StripTags(strings.TrimSpace(c.Request.PostFormValue("description")))
			if len(description) > 500 {
				c.JSON(http.StatusBadRequest, "Mô tả không được quá 500 ký tự")
				return
			}

			if description != "" {
				promotion.Description = description
			}

			services, err := h.ServiceManager.GetServices(sqlmanager.ServiceQueryOption{
				Status: constant.StatusActive,
			})
			if err != nil {
				h.Logger.Errorf("get services: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			path, prices, importErrors, message, err := h.parseContentPrice(promotion.Name, h.StorageS3, c.Request, services)
			if message != "" {
				c.JSON(http.StatusBadRequest, message)
				return
			}

			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if len(importErrors) > 0 {
				c.JSON(http.StatusBadRequest, createPromotionResponse{
					Success: false,
					Error:   "File promotion price sai định dạng",
					Errors:  importErrors,
				})
				return
			}

			if path != "" && len(prices) > 0 {
				promotion.S3PathPrice = path
				promotionPrices = prices
			}

			path, weights, importErrors, message, err := h.parseContentWeight(promotion.Name, h.StorageS3, c.Request)
			if message != "" {
				c.JSON(http.StatusBadRequest, message)
				return
			}

			if err != nil {
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if len(importErrors) > 0 {
				c.JSON(http.StatusBadRequest, createPromotionResponse{
					Success: false,
					Error:   "File promotion weight sai định dạng",
					Errors:  importErrors,
				})
				return
			}

			if path != "" && len(weights) > 0 {
				promotion.S3PathWeight = path
				promotionWeights = weights
			}
		}

		if promotion.ID == constant.PromotionPriceByWeight {
			if c.Request.PostFormValue("price") != "" {
				price := utils.Ceil(cast.ToFloat64(strings.TrimSpace(c.Request.PostFormValue("price"))), 2)
				if price <= 0 {
					c.JSON(http.StatusBadRequest, "Giá/KG không hợp lệ hoặc chưa nhập")
					return
				}

				promotion.Price = price
			}
		}

		oldStatus := promotion.Status
		if role == constant.UserRoleAdmin || role == constant.UserRolerBusinessManager {
			status := cast.ToInt(strings.TrimSpace(c.Request.PostFormValue("status")))
			statusValid := map[int]bool{
				constant.PromotionStatusActive:     true,
				constant.PromotionStatusDeactivate: true,
				constant.PromotionStatusDraft:      true,
			}
			if status > 0 {
				if !statusValid[status] {
					c.JSON(http.StatusBadRequest, "Trạng thái không hợp lệ")
					return
				}

				promotion.Status = status
			}
		}

		if oldStatus != promotion.Status && promotion.Status == constant.PromotionStatusActive && promotion.Type == constant.PromotionTypeMarketing {
			users, err := h.PromotionManager.GetUsersByPromotionID(promotion.ID, []int64{})
			if err != nil {
				h.Logger.Errorf("Error while get ids user, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			var ids = []int64{}
			for _, user := range users {
				ids = append(ids, user.ID)
			}

			promotions, err := h.PromotionManager.GetMarketingPromotionsByUserIDs(ids, []int64{promotion.ID})
			if err != nil {
				h.Logger.Errorf("Error while get ids user, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			mapUserPromotions := make(map[int64]entity.UserPromotion)
			for _, v := range promotions {
				mapUserPromotions[v.UserID] = v
			}

			for _, user := range users {
				up := mapUserPromotions[user.ID]
				if up.PromotionID > 0 && up.PromotionID != promotion.ID {
					name := fmt.Sprintf("#%d", up.UserID)
					if up.User != nil {
						name = up.User.Email

						if up.User.FullName != "" {
							name = up.User.FullName
						}
					}

					c.JSON(http.StatusBadRequest, fmt.Sprintf("Khách hàng %s đã được thêm vào promotion #%d", name, up.PromotionID))
					return
				}
			}
		}

		err = h.PromotionManager.Update(promotion, promotionPrices, promotionWeights)
		if err != nil {
			h.Logger.Errorf("update product error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if oldStatus != promotion.Status {
			keyPrice := fmt.Sprintf("%s_%d", calculate.RedisPromotionPriceKey, promotion.ID)
			if err := h.Redis.Del(c, keyPrice).Err(); err != nil {
				h.Logger.Errorf("delete redis promotion price: %v", err)
			}

			keyWeight := fmt.Sprintf("%s_%d", calculate.RedisPromotionWeightKey, promotion.ID)
			if err := h.Redis.Del(c, keyWeight).Err(); err != nil {
				h.Logger.Errorf("delete redis promotion price: %v", err)
			}
		}

		c.JSON(http.StatusOK, UpdatePromotionResponse{Success: true})
	}
}

func (h *PromotionHandler) UpdateSettingPoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID <= 0 {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		form := UpdateSettingPointForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if utils.Float64Value(form.From) < 0 || utils.Float64Value(form.To) <= 0 || form.Value < 0 || form.ID <= 0 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if form.Class != constant.UserClassPriority && form.Class != constant.UserClassPartner && form.Class != constant.UserClassPublic {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if utils.Float64Value(form.From) >= utils.Float64Value(form.To) {
			c.JSON(http.StatusBadRequest, "Giá trị đầu phải nhỏ hơn giá trị cuối !")
			return
		}

		opts := sqlmanager.SettingPointQueryOption{
			Status: constant.SettingEnableStatus,
		}

		settings, err := h.PromotionManager.GetSettingPoints(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Get list setting point error:, %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		mapSettings := make(map[int][]*entity.SettingPoint, 0)

		for _, setting := range settings {
			mapSettings[setting.Class] = append(mapSettings[setting.Class], setting)
		}
		listSetting := mapSettings[form.Class]
		for _, item := range listSetting {
			if item.ID == form.ID {
				continue
			}
			if (utils.Float64Value(form.From) > item.From && utils.Float64Value(form.From) < item.To) ||
				(utils.Float64Value(form.To) > item.From && utils.Float64Value(form.To) < item.To) ||
				utils.Float64Value(form.From) == item.From ||
				utils.Float64Value(form.To) == item.To ||
				(utils.Float64Value(form.From) < item.From && utils.Float64Value(form.To) > item.To) {
				c.JSON(http.StatusBadRequest, "Khoảng setup đã tồn tại !")
				return
			}
		}

		setting, err := h.PromotionManager.GetSettingPointByID(form.ID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		setting.Value = form.Value
		setting.To = utils.Float64Value(form.To)
		setting.From = utils.Float64Value(form.From)
		setting.Class = form.Class
		setting.Status = form.Status
		err = h.PromotionManager.SaveSettingPoint(setting)

		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdatePromotionResponse{Success: true})
	}
}
