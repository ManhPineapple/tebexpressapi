package admin

import (
	"fmt"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/array"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	Logger *zap.SugaredLogger

	UserManager *sqlmanager.UserManager
}

type GetUserResponse struct {
	User *entity.User `json:"user"`
}

type GetListUsersResponse struct {
	Users []dto.UserResponseDto `json:"users"`
}

type CountListUsersResponse struct {
	Count int64 `json:"count"`
}

var MapIntPrice = map[string][]int64{
	"public":   []int64{constant.UserClassPublic},
	"partner":  []int64{constant.UserClassPartner},
	"priority": []int64{constant.UserClassPriority},
}

type GetUsersByRoleResponse struct {
	Users []dto.UserResponseDto `json:"users"`
}

type CreateUserResponse struct {
	Success bool `json:"success"`
}

type CreateUserForm struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
	Password    string `json:"password"`
	SlackID     string `json:"slack_id"`

	CustomerIDs []int64 `json:"customer_id"`
}

type UpdateUserResponse struct {
	Success bool `json:"success"`
}

type UpdateStatusUserForm struct {
	UserID    int64   `json:"id"`
	Status    int64   `json:"status"`
	SupportID []int64 `json:"support_id"`
}

type UpdateStatusUserResponse struct {
	Success bool `json:"success"`
}

type UpdateUserForm struct {
	Email        string  `json:"email"`
	PhoneNumber  string  `json:"phone_number"`
	FullName     string  `json:"full_name"`
	Role         string  `json:"role"`
	Password     string  `json:"password"`
	SlackID      string  `json:"slack_id"`
	CustomerID   []int64 `json:"customer_id"`
	CustomerSwap []int64 `json:"customer_swap"`
	SupportID    int64   `json:"support_id"`
}

type updateUserInfoForm struct {
	DebtMaxAmount   float64 `json:"debt_max_amount"`
	DebtMaxDay      int     `json:"debt_max_day"`
	Class           int64   `json:"class"`
	CancelMaxAmount float64 `json:"cancel_max_amount"`
	RefundDay       int     `json:"refund_day"`
}

type updateUserInfoResponse struct {
	UserInfo interface{} `json:"user_info"`
}

func NewUserHandler(l *zap.SugaredLogger, um *sqlmanager.UserManager) *UserHandler {
	return &UserHandler{
		Logger: l,

		UserManager: um,
	}
}

func (h *UserHandler) Get() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		if user.ReferralCode == "" {
			s, _ := utils.GenerateRandomString(15)
			user.ReferralCode = s
			change := map[string]interface{}{
				"referral_code": s,
				"updated_at":    time.Now(),
			}
			err = h.UserManager.UpdateInfoUser(change, userID)
			if err != nil {
				h.Logger.Errorf("Update user error, %v", userID, err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}
		}

		result := &GetUserResponse{User: user}
		dto.UserTransform(user)

		c.JSON(http.StatusOK, result)
	}
}

func (h *UserHandler) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		if userID < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
		}

		offset, limit := httputil.GetRequestPaginate(c.Request)

		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if cast.ToBool(c.Request.URL.Query().Get("tester")) {
			ignoreUsers = []int64{}
		}

		arrStatus := []int{}
		status := cast.ToString(c.Request.URL.Query().Get("status"))
		if len(cast.ToString(c.Request.URL.Query().Get("arrStatus"))) > 0 {
			for _, status := range strings.Split(cast.ToString(c.Request.URL.Query().Get("arrStatus")), ",") {
				if cast.ToInt(status) >= 0 {
					arrStatus = append(arrStatus, cast.ToInt(status))
				} else {
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}
			}
		}

		if len(arrStatus) > 0 {
			status = ""
		}

		inRoles := []string{}
		rolesFilter := cast.ToString(c.Request.URL.Query().Get("roles"))
		if rolesFilter != "" {
			for _, item := range strings.Split(rolesFilter, ",") {
				val := strings.TrimSpace(item)
				if !utils.ValidSlug(val) {
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}

				inRoles = append(inRoles, val)
			}
		}

		// Get users
		opts := sqlmanager.UserQueryOption{
			ID:        cast.ToInt64(c.Request.URL.Query().Get("id")),
			Search:    cast.ToString(c.Request.URL.Query().Get("search")),
			Status:    status,
			ArrStatus: arrStatus,
			Offset:    offset,
			Limit:     limit,
			Role:      cast.ToString(c.Request.URL.Query().Get("role")),
			IgnoreIDs: ignoreUsers,
			InRole:    inRoles,
			PartnerID: user.PartnerID,
		}

		priceArr := strings.Split(cast.ToString(c.Request.URL.Query().Get("price_arr")), ",")
		for _, price := range priceArr {
			opts.PriceType = append(opts.PriceType, MapIntPrice[price]...)
		}

		if cast.ToBool(c.Request.URL.Query().Get("prepay")) {
			opts.IsPrePay = true
		}

		if cast.ToBool(c.Request.URL.Query().Get("postpaid")) {
			opts.IsPostpaid = true
		}

		if cast.ToInt64(c.Request.URL.Query().Get("appraiser_id")) == userID {
			opts.AppraiserID = cast.ToInt64(c.Request.URL.Query().Get("appraiser_id"))
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		if opts.Role != "" && utils.InvalidTag(opts.Role) {
			c.JSON(http.StatusBadRequest, "Quyền không hợp lệ")
			return
		}

		if opts.Status != "" && utils.InvalidTag(opts.Status) {
			c.JSON(http.StatusBadRequest, "Trạng thái không hợp lệ")
			return
		}

		if opts.Role == "" {
			opts.InRole = []string{
				constant.UserRoleSupport,
				constant.UserRoleSale,
				constant.UserRoleSupportLeader,
				constant.UserRoleAdmin,
				constant.UserRoleAccountant,
				constant.UserRoleWarehouse,
				constant.UserRoleHub,
				constant.UserRoleMarketing,
				constant.UserRolerShipPartner,
				constant.UserRolerBusinessManager,
				constant.UserRoleSaleOperation,
			}
			if role == constant.UserRolerBusinessManager {
				opts.InRole = []string{
					// constant.UserRoleSale,
					constant.UserRoleSupport,
				}
			}
		}

		if opts.Role == constant.UserRoleAppraiser {
			opts.InRole = []string{
				constant.UserRoleSupport,
				constant.UserRoleSale,
				constant.UserRoleSupportLeader,
				constant.UserRoleAdmin,
			}
		}

		opts.IsHasInfo = true
		opts.IsHasReferring = true

		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportQuery = userID
		}

		users, err := h.UserManager.GetUsersByOptions(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorw("Get users  %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var userDTO []dto.UserResponseDto
		err = httputil.Transform(users, &userDTO)
		if err != nil {
			h.Logger.Errorf("Get users  %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		for i, _ := range userDTO {
			userDTO[i].CustomerID, err = h.UserManager.GetCustomerIDBySupportID(userDTO[i].ID)
			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("Error while get ids customer, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			userDTO[i].SupportID, err = h.UserManager.GetSupportIDByCustomerID(userDTO[i].ID)
			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("Error while get ids customer, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			phoneNumber := ""
			if role == constant.UserRoleAdmin {
				phoneNumber = userDTO[i].PhoneNumber
			}
			userDTO[i].PhoneNumber = phoneNumber

			if users[i].UserInfo != nil {
				appraiser, err := h.UserManager.GetUserByID(users[i].UserInfo.AppraiserID)
				if err != nil && err != gorm.ErrRecordNotFound {
					h.Logger.Errorf("Error while get ids customer, details: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}
				userDTO[i].AppraiserName = appraiser.FullName
				userDTO[i].AppraiserID = users[i].UserInfo.AppraiserID
				userDTO[i].TaxCode = users[i].UserInfo.TaxCode
				userDTO[i].Volume = users[i].UserInfo.Volume
				userDTO[i].ItemType = users[i].UserInfo.ItemType
				userDTO[i].WarehouseAddress = users[i].UserInfo.WarehouseAddress
			}

			if users[i].ReferringUser != nil {
				userDTO[i].RefID = utils.Int64(users[i].ReferringUser.ID)
				userDTO[i].ReferringName = utils.String(users[i].ReferringUser.FullName)
				if users[i].ReferringUser.Role == constant.UserRoleCustomer {
					userDTO[i].IsCustomerRefer = true
				}
			}
		}

		result := GetListUsersResponse{Users: userDTO}
		c.JSON(http.StatusOK, result)
	}
}

func (h *UserHandler) Count() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		role := cast.ToString(c.Request.Header.Get("X-User-Role"))

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
		}

		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if cast.ToBool(c.Request.URL.Query().Get("tester")) {
			ignoreUsers = []int64{}
		}

		str := strings.TrimSpace(cast.ToString(c.Request.URL.Query().Get("not_in")))
		if str != "" {
			notInIDs := array.SliceStringToSliceInt(strings.Split(str, ","))
			ignoreUsers = append(ignoreUsers, notInIDs...)
		}

		inRoles := []string{}
		rolesFilter := cast.ToString(c.Request.URL.Query().Get("roles"))
		if rolesFilter != "" {
			for _, item := range strings.Split(rolesFilter, ",") {
				role := strings.TrimSpace(item)
				if !utils.ValidSlug(role) {
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}

				inRoles = append(inRoles, role)
			}
		}

		// Get users
		opts := sqlmanager.UserQueryOption{
			Search:    cast.ToString(c.Request.URL.Query().Get("search")),
			Status:    cast.ToString(c.Request.URL.Query().Get("status")),
			Role:      cast.ToString(c.Request.URL.Query().Get("role")),
			InRole:    inRoles,
			IgnoreIDs: ignoreUsers,
			IsHasInfo: true,
			PartnerID: user.PartnerID,
		}

		priceArr := strings.Split(cast.ToString(c.Request.URL.Query().Get("price_arr")), ",")
		for _, price := range priceArr {
			opts.PriceType = append(opts.PriceType, MapIntPrice[price]...)
		}

		if cast.ToBool(c.Request.URL.Query().Get("prepay")) {
			opts.IsPrePay = true
		}

		if cast.ToBool(c.Request.URL.Query().Get("postpaid")) {
			opts.IsPostpaid = true
		}

		if opts.Search != "" && utils.InvalidTag(opts.Search) {
			c.JSON(http.StatusBadRequest, "Từ khóa không hợp lệ")
			return
		}

		if opts.Role != "" && utils.InvalidTag(opts.Role) {
			c.JSON(http.StatusBadRequest, "Quyền không hợp lệ")
			return
		}

		if opts.Status != "" && utils.InvalidTag(opts.Status) {
			c.JSON(http.StatusBadRequest, "Trạng thái khóa không hợp lệ")
			return
		}

		if opts.Role == "" {
			opts.InRole = []string{
				constant.UserRoleSupport,
				constant.UserRoleAdmin,
				constant.UserRoleAccountant,
				constant.UserRoleWarehouse,
				constant.UserRoleHub,
				constant.UserRoleMarketing,
				constant.UserRolerShipPartner,
				constant.UserRolerBusinessManager,
				constant.UserRoleSale,
			}

			if role == constant.UserRolerBusinessManager {
				opts.InRole = []string{
					constant.UserRoleSale,
				}
			}

		}
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportQuery = userID
		}

		if opts.Role == constant.UserRoleAppraiser {
			opts.InRole = []string{
				constant.UserRoleSupport,
				constant.UserRoleSupportLeader,
				constant.UserRoleSale,
			}
		}

		count, err := h.UserManager.CountUserByOptions(opts)
		if err != nil {
			h.Logger.Errorw(fmt.Sprintf("Error while count user by user id %v error, %v", userID, err))
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		result := &CountListUsersResponse{Count: count}
		c.JSON(http.StatusOK, result)
	}
}

func (h *UserHandler) Role() gin.HandlerFunc {
	return func(c *gin.Context) {
		searchKey := cast.ToString(c.Request.URL.Query().Get("search"))
		roleFilter := cast.ToString(c.Request.URL.Query().Get("role"))
		code := cast.ToString(c.Request.URL.Query().Get("code"))
		offset, limit := httputil.GetRequestPaginate(c.Request)
		notLimit := cast.ToBool(c.Request.URL.Query().Get("not_limit"))
		customerID := cast.ToInt64(c.Request.URL.Query().Get("user_id"))
		status := cast.ToString(c.Request.URL.Query().Get("status"))
		rolesFilter := cast.ToString(c.Request.URL.Query().Get("roles"))

		if utils.InvalidTag(searchKey) || !utils.ValidSlug(roleFilter) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		ignoreUsers := array.SliceStringToSliceInt(strings.Split(viper.GetString("blacklist.users"), ","))
		if cast.ToBool(c.Request.URL.Query().Get("tester")) {
			ignoreUsers = []int64{}
		}

		str := strings.TrimSpace(cast.ToString(c.Request.URL.Query().Get("not_in")))
		if str != "" {
			notInIDs := array.SliceStringToSliceInt(strings.Split(str, ","))
			ignoreUsers = append(ignoreUsers, notInIDs...)
		}

		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		user, err := h.UserManager.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
		}

		arrStatus := []int{}
		if len(cast.ToString(c.Request.URL.Query().Get("arrStatus"))) > 0 {
			for _, status := range strings.Split(cast.ToString(c.Request.URL.Query().Get("arrStatus")), ",") {
				if cast.ToInt(status) >= 0 {
					arrStatus = append(arrStatus, cast.ToInt(status))
				} else {
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}
			}
		}

		if len(arrStatus) > 0 {
			status = ""
		}

		inRoles := []string{}
		if rolesFilter != "" {
			for _, item := range strings.Split(rolesFilter, ",") {
				role := strings.TrimSpace(item)
				if !utils.ValidSlug(role) {
					c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
					return
				}

				inRoles = append(inRoles, role)
			}
		}

		if roleFilter == "" && len(inRoles) < 1 {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		// Get users
		opts := sqlmanager.UserQueryOption{
			Search:    searchKey,
			Status:    status,
			ArrStatus: arrStatus,
			Offset:    offset,
			Role:      roleFilter,
			InRole:    inRoles,
			Code:      code,
			ID:        customerID,
			IsHasInfo: true,
			IgnoreIDs: ignoreUsers,
			PartnerID: user.PartnerID,
		}

		if opts.Role == constant.UserRoleAppraiser {
			opts.InRole = []string{
				constant.UserRoleSupport,
				constant.UserRoleSale,
				constant.UserRoleSupportLeader,
			}
		}

		if !notLimit {
			opts.Limit = limit
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role == constant.UserRoleSupport || role == constant.UserRoleSale {
			opts.SupportQuery = userID
		}

		users, err := h.UserManager.GetUsersByOptions(opts)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorw(fmt.Sprintf("Get users  error, %v", err))
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var userDTO []dto.UserResponseDto
		err = httputil.Transform(users, &userDTO)
		if err != nil {
			h.Logger.Errorw(fmt.Sprintf("Get users error, %v", err))
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var ids []int64
		for i, _ := range userDTO {
			if users[i].UserInfo != nil && users[i].UserInfo.AppraiserID > 0 {
				ids = append(ids, users[i].UserInfo.AppraiserID)
			}

		}

		listAppraiser, err := h.UserManager.GetUsers(sqlmanager.UserQueryOption{
			IDS: ids,
		})

		if err != nil {
			h.Logger.Errorw(fmt.Sprintf("Get users error, %v", err))
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var mListAppr = make(map[int64]*entity.User, 0)

		for _, user := range listAppraiser {
			mListAppr[user.ID] = user
		}

		for i, _ := range userDTO {

			phoneNumber := ""
			if role == constant.UserRoleAdmin {
				phoneNumber = userDTO[i].PhoneNumber
			}
			userDTO[i].PhoneNumber = phoneNumber

			if users[i].UserInfo != nil {
				if mListAppr[users[i].UserInfo.AppraiserID] != nil {
					userDTO[i].AppraiserName = mListAppr[users[i].UserInfo.AppraiserID].FullName
				}
				userDTO[i].AppraiserID = users[i].UserInfo.AppraiserID
				userDTO[i].TaxCode = users[i].UserInfo.TaxCode
				userDTO[i].Volume = users[i].UserInfo.Volume
				userDTO[i].ItemType = users[i].UserInfo.ItemType
				userDTO[i].WarehouseAddress = users[i].UserInfo.WarehouseAddress
			}

		}

		result := GetUsersByRoleResponse{Users: userDTO}

		c.JSON(http.StatusOK, result)
	}
}

func (h *UserHandler) Create() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		form := &CreateUserForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		// role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		// if role == constant.UserRolerBusinessManager && form.Role != constant.UserRoleSupport && form.Role != constant.UserRoleSale && form.Role != constant.UserRoleSaleOperation {
		// 	c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
		// 	return
		// }

		if validate := h.validateSignUpInfo(form); len(validate) > 0 {
			c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"message": constant.MessageValidateInput,
				"errors":  validate,
			})
			return
		}

		userInfo, err := h.parseRequestUserInfo(form)
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		userInfo.PartnerID = entity.PartnerMap[c.Request.Host]
		if err = h.UserManager.AdminCreateUser(userInfo, form.CustomerIDs); err != nil {
			h.Logger.Errorf("Error while create user, details: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, CreateUserResponse{Success: true})
	}
}

func (h *UserHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Param("id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		form := &UpdateUserForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		// role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		// if role == constant.UserRolerBusinessManager && form.Role != constant.UserRoleSupport && form.Role != constant.UserRoleSale && form.Role != constant.UserRoleSaleOperation {
		// 	c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
		// 	return
		// }

		current, err := h.UserManager.FetchUserWithoutInfo(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if messages := h.validateExistSignUpInfo(form, current); len(messages) > 0 {
			c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"message": constant.MessageValidateInput,
				"errors":  messages,
			})
			return
		}

		if err != nil {
			h.Logger.Errorf("get user: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		user, err := h.parseRequestUpdatedUserInfo(form)
		if err != nil {
			h.Logger.Errorf("parse user: %s", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var idsDelete, idsAdd []int64
		if form.Role == constant.UserRoleSupport || (form.Role == "" && current.Role == constant.UserRoleSupport) || form.Role == constant.UserRoleSale || (form.Role == "" && current.Role == constant.UserRoleSale) {
			ids, err := h.UserManager.GetCustomerIDBySupportID(userID)
			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("Error while get ids customer, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			if form.SupportID > 0 {
				for _, id := range form.CustomerSwap {
					if !utils.ContainsNumber(ids, id) {
						c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
						return
					}
				}

				idsDelete = append(idsDelete, form.CustomerSwap...)

				// Kiem tra support duoc chuyen da duoc support chua
				aids, err := h.UserManager.GetCustomerIDBySupportID(form.SupportID)
				if err != nil && err != gorm.ErrRecordNotFound {
					h.Logger.Errorf("Error while get ids customer, details: %v", err)
					c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
					return
				}

				for _, id := range form.CustomerSwap {
					if !utils.ContainsNumber(aids, id) {
						idsAdd = append(idsAdd, id)
					}
				}

			} else {
				for _, id := range ids {
					if !utils.ContainsNumber(form.CustomerID, id) {
						idsDelete = append(idsDelete, id)
					}
				}

				for _, id := range form.CustomerID {
					if !utils.ContainsNumber(ids, id) {
						idsAdd = append(idsAdd, id)
					}
				}
			}
		}

		user.ID = current.ID
		if err := h.UserManager.AdminUpdateUser(user, idsDelete, idsAdd, form.SupportID); err != nil {
			h.Logger.Errorf("Error while update user, details: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, UpdateUserResponse{Success: true})
	}
}

func (h *UserHandler) UpdateStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		// role := transhttp.GetUserRoleFromRequest(r)
		// if role != constant.UserRoleAdmin {
		// 	c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
		// 	return
		// }

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
		form := UpdateStatusUserForm{}
		if err := c.ShouldBindJSON(&form); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.UserID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		user, err := h.UserManager.GetUserByID(form.UserID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		var idsDelete, idsAdd []int64

		if len(form.SupportID) > 0 {
			form.Status = user.Status
			ids, err := h.UserManager.GetSupportIDByCustomerID(form.UserID)
			if err != nil && err != gorm.ErrRecordNotFound {
				h.Logger.Errorf("Error while get ids customer, details: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			for _, id := range ids {
				if !utils.ContainsNumber(form.SupportID, id) {
					idsDelete = append(idsDelete, id)
				}
			}

			for _, id := range form.SupportID {
				if !utils.ContainsNumber(ids, id) {
					idsAdd = append(idsAdd, id)
				}
			}
		}

		if form.Status != constant.UserStatusActive &&
			form.Status != constant.UserStatusDeactive {

			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return

		}

		err = h.UserManager.UpdateStatusUser(form.UserID, form.Status, form.SupportID, idsDelete, idsAdd, user.Status)
		if err != nil {
			h.Logger.Errorf("Update user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, "Cập nhật thất bại")
			return
		}

		c.JSON(http.StatusOK, UpdateStatusUserResponse{Success: true})
	}
}

func (h *UserHandler) Info() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Param("user_id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, "User ID is missing")
			return
		}

		form := &updateUserInfoForm{}
		if err := c.ShouldBindJSON(form); err != nil {
			h.Logger.Errorf("parse body: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if form.Class <= 0 {
			c.JSON(http.StatusBadRequest, "User class is required !")
			return
		}

		ui, err := h.UserManager.GetUserInfoByUserID(userID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get user info: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if err == gorm.ErrRecordNotFound {
			ui = &entity.UserInfo{
				UserID: userID,
			}
		}

		ui.DebtMaxAmount = form.DebtMaxAmount
		ui.DebtMaxDay = form.DebtMaxDay
		ui.CancelMaxAmount = form.CancelMaxAmount
		ui.RefundDay = form.RefundDay

		user, err := h.UserManager.GetUserByID(userID)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("get user error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if err := h.UserManager.SaveUser(ui, user, form.Class); err != nil {
			h.Logger.Errorf("save user info: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, updateUserInfoResponse{ui})
	}
}

func (h *UserHandler) validateSignUpInfo(info *CreateUserForm) []string {
	messages := make([]string, 0)

	if info.Email == "" && info.PhoneNumber == "" {
		messages = append(messages, "Tài khoản không được để trống")
	}

	if info.Email != "" {
		if !utils.ValidEmail(info.Email) {
			messages = append(messages, "Tài khoản không hợp lệ")
		} else {
			isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
				Email: info.Email,
			})

			if isExist {
				messages = append(messages, "Tài khoản đã được sử dụng, vui lòng chọn tài khoản khác")
			}
		}
	}

	if info.PhoneNumber != "" {
		phonenumberLength := len(info.PhoneNumber)
		if 1 > phonenumberLength || 20 < phonenumberLength {
			messages = append(messages, "Tài khoản không được để trống, phải từ 1 đến 20 ký tự")
		} else if !utils.ValidPhoneNumber(info.PhoneNumber) {
			messages = append(messages, "Tài khoản không hợp lệ")
		} else {
			isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
				PhoneNumber: info.PhoneNumber,
			})

			if isExist {
				messages = append(messages, "Tài khoản đã được sử dụng, vui lòng chọn tài khoản khác")
			}
		}
	}

	if info.SlackID != "" {
		isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
			SlackID: info.SlackID,
		})

		if isExist {
			messages = append(messages, "Slack ID đã được sử dụng, vui lòng chọn Slack ID khác")
		}
	}

	if !utils.ValidFullName(info.FullName) {
		messages = append(messages, "Tên không hợp lệ")
	}

	usernameLength := len(info.FullName)
	if usernameLength < 1 {
		messages = append(messages, "Tên không được để trống")
	}

	if info.Role == "" {
		messages = append(messages, "Quyền không được để trống")
	}

	if info.Role != constant.UserRoleAdmin &&
		info.Role != constant.UserRoleAccountant &&
		info.Role != constant.UserRoleWarehouse &&
		info.Role != constant.UserRoleSupport &&
		info.Role != constant.UserRoleSale &&
		info.Role != constant.UserRoleHub &&
		info.Role != constant.UserRoleSupportLeader &&
		info.Role != constant.UserRoleMarketing &&
		info.Role != constant.UserRolerBusinessManager &&
		info.Role != constant.UserRolerShipPartner &&
		info.Role != constant.UserRoleSaleOperation &&
		info.Role != constant.UserRoleCustomer {
		messages = append(messages, "Quyền không tồn tại")
	}

	if (info.Role == constant.UserRoleSupport || info.Role == constant.UserRoleSale) && len(info.CustomerIDs) < 1 {
		messages = append(messages, "Khách hàng không được để trống")
	}

	if errValidPassword := utils.ValidatePassword(info.Password); errValidPassword != "" {
		messages = append(messages, errValidPassword)
	}

	return messages
}

func (h *UserHandler) validateExistSignUpInfo(form *UpdateUserForm, current *entity.User) []string {
	messages := make([]string, 0)

	if form.Email != "" {
		if !utils.ValidEmail(form.Email) {
			messages = append(messages, "Tài khoản không hợp lệ")
		} else if form.Email != current.Email {
			isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{Email: form.Email})

			if isExist {
				messages = append(messages, "Tài khoản đã được sử dụng, vui lòng chọn tài khoản khác")
			}
		}
	}

	if form.PhoneNumber != "" {
		if length := len(form.PhoneNumber); 1 > length || 20 < length {
			messages = append(messages, "Tài khoản không được để trống, phải từ 1 đến 20 ký tự")
		} else if !utils.ValidPhoneNumber(form.PhoneNumber) {
			messages = append(messages, "Tài khoản không hợp lệ")
		} else if form.PhoneNumber != current.PhoneNumber {
			isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{PhoneNumber: form.PhoneNumber})

			if isExist {
				messages = append(messages, "Tài khoản đã được sử dụng, vui lòng chọn tài khoản khác")
			}
		}
	}

	if form.SlackID != "" && form.SlackID != current.SlackID {
		isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
			SlackID: form.SlackID,
		})

		if isExist {
			messages = append(messages, "Slack ID đã được sử dụng, vui lòng chọn Slack ID khác")
		}
	}

	if !utils.ValidFullName(form.FullName) {
		messages = append(messages, "Tên không hợp lệ")
	}

	usernameLength := len(form.FullName)
	if usernameLength < 1 {
		messages = append(messages, "Tên không được để trống")
	}

	if form.Role == "" {
		messages = append(messages, "Quyền không được để trống")
	}

	if form.Role != constant.UserRoleAdmin &&
		form.Role != constant.UserRoleAccountant &&
		form.Role != constant.UserRoleWarehouse &&
		form.Role != constant.UserRoleSupport &&
		form.Role != constant.UserRoleSale &&
		form.Role != constant.UserRoleHub &&
		form.Role != constant.UserRoleSupportLeader &&
		form.Role != constant.UserRoleMarketing &&
		form.Role != constant.UserRolerBusinessManager &&
		form.Role != constant.UserRolerShipPartner &&
		form.Role != constant.UserRoleSaleOperation &&
		form.Role != constant.UserRoleCustomer {
		messages = append(messages, "Quyền không tồn tại")
	}

	if (form.Role == constant.UserRoleSupport || form.Role == constant.UserRoleSale) && len(form.CustomerID) < 1 {
		messages = append(messages, "Khách hàng không được để trống")
	}

	if form.Password == "" {
		return messages
	}

	if msg := utils.ValidatePassword(form.Password); msg != "" {
		messages = append(messages, msg)
	}

	return messages
}

func (h *UserHandler) parseRequestUserInfo(Info *CreateUserForm) (*entity.User, error) {
	user := &entity.User{
		FullName:    strings.TrimSpace(Info.FullName),
		Email:       strings.TrimSpace(Info.Email),
		PhoneNumber: strings.TrimSpace(Info.PhoneNumber),
		Role:        Info.Role,
		SlackID:     Info.SlackID,
	}

	// Hash password
	if Info.Password != "" {
		user.Password = h.hashPassword(Info.Password)
	}

	return user, nil
}

func (h *UserHandler) parseRequestUpdatedUserInfo(form *UpdateUserForm) (*entity.User, error) {
	user := &entity.User{
		FullName:    strings.TrimSpace(form.FullName),
		Email:       strings.TrimSpace(form.Email),
		PhoneNumber: strings.TrimSpace(form.PhoneNumber),
		Role:        form.Role,
		SlackID:     form.SlackID,
	}

	// Hash password
	if form.Password != "" {
		user.Password = h.hashPassword(form.Password)
	}

	return user, nil
}

func (h *UserHandler) hashPassword(pwd string) string {
	pwdBytes := []byte(pwd)
	hash, err := bcrypt.GenerateFromPassword(pwdBytes, bcrypt.MinCost)
	if err != nil {
		h.Logger.Errorf("Error while hash password %v, details: %v", pwd, err)
	}
	return string(hash)
}
