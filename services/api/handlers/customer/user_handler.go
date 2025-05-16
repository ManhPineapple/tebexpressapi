package customer

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	Logger *zap.SugaredLogger

	UserManager    *sqlmanager.UserManager
	SettingManager *sqlmanager.SettingManager
	ServiceManager *sqlmanager.ServiceManager
}

type GetUserResponse struct {
	User         *entity.User           `json:"user"`
	Promotion    *dto.PromotionCustomer `json:"promotion"`
	PushBookmark bool                   `json:"push_bookmark"`
}

type GetTokenResponse struct {
	Token string `json:"token"`
}

type GenerateTokenForm struct {
	Name        string `json:"name"`
	IsDefault   bool   `json:"is_default"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
	City        string `json:"city"`
	District    string `json:"district"`
	Wards       string `json:"wards"`
}

type GenerateTokenResponse struct {
	Success bool `json:"success"`
}

type UpdateUserForm struct {
	FullName        string `json:"full_name"`
	Birthday        string `json:"birthday"`
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	PushBookmark    bool   `json:"push_bookmark"`
}

type UpdateUserResponse struct {
	User interface{} `json:"user"`
}

func NewUserHandler(l *zap.SugaredLogger, r *redis.Client, um *sqlmanager.UserManager, sm *sqlmanager.SettingManager, srm *sqlmanager.ServiceManager) *UserHandler {
	return &UserHandler{
		Logger: l,

		UserManager:    um,
		SettingManager: sm,
		ServiceManager: srm,
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

		user.HoldingMoney, err = h.UserManager.GetHoldingMoney(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		if err != nil {
			h.Logger.Errorf("Get user holding money %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		queryOptions := sqlmanager.SettingQueryOption{
			Key:    constant.BookmarkPushSettingKey,
			UserID: userID,
		}

		setting, err := h.SettingManager.GetSetting(queryOptions)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Error fetch setting webhook url: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		var pushBookmark bool
		if setting != nil && setting.ID > 0 {
			pushBookmark = cast.ToBool(setting.Value)
		}

		// user.Promotions = promotions
		result := &GetUserResponse{User: user, PushBookmark: pushBookmark}
		dto.UserTransform(user)

		c.JSON(http.StatusOK, result)
	}
}

func (h *UserHandler) Token() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, "User id required")
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

		if user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		userToken, err := h.UserManager.GetUserTokenByUserID(userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageNotFound)
			return
		}

		c.JSON(http.StatusOK, GetTokenResponse{Token: userToken.Token})
	}
}

func (h *UserHandler) Reset() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			c.JSON(http.StatusBadRequest, "User id required")
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

		if user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		token := fmt.Sprintf("%s:%s", user.Email, uuid.New().String())
		if user.Email == "" {
			token = fmt.Sprintf("%s:%s", user.PhoneNumber, uuid.New().String())
		}
		tokenBase64 := base64.StdEncoding.EncodeToString([]byte(token))

		err = h.UserManager.ResetTokenByUserID(userID, tokenBase64)
		if err != nil {
			h.Logger.Errorf("Get user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		c.JSON(http.StatusOK, GenerateTokenResponse{Success: true})
	}
}

func (h *UserHandler) Update() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		if userID < 1 {
			h.Logger.Errorf("Invalid user id %v", userID)
			c.JSON(http.StatusBadRequest, "User id required")
			return
		}

		role := cast.ToString(c.Request.Header.Get("X-User-Role"))
		if role != constant.UserRoleCustomer {
			c.JSON(http.StatusForbidden, constant.MessagePermissionDenied)
			return
		}

		user, err := h.UserManager.GetUserByID(userID)
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, "Cập nhật thất bại")
			return
		}

		if err != nil {
			h.Logger.Errorf("Get GetUserByID %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}
		newForm := UpdateUserForm{}
		decoder := json.NewDecoder(c.Request.Body)

		if err := decoder.Decode(&newForm); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if !utils.ValidFullName(newForm.FullName) {
			c.JSON(http.StatusBadRequest, "Tên không hợp lệ")
			return
		}

		if newForm.Birthday != "" && !utils.ValidDate(newForm.Birthday) {
			c.JSON(http.StatusBadRequest, "Ngày sinh không hợp lệ")
			return
		}

		if len(newForm.FullName) < 1 {
			c.JSON(http.StatusBadRequest, "Tên không được để trống")
			return
		}

		if newForm.CurrentPassword != "" || newForm.NewPassword != "" {
			if errValidPassword := utils.ValidatePassword(newForm.CurrentPassword); errValidPassword != "" {
				c.JSON(http.StatusBadRequest, errValidPassword)
				return
			}

			if errValidPassword := utils.ValidatePassword(newForm.NewPassword); errValidPassword != "" {
				c.JSON(http.StatusBadRequest, errValidPassword)
				return
			}

			if !auth.IsCorrectPassword(user.Password, newForm.CurrentPassword) {
				c.JSON(http.StatusBadRequest, "Mật khẩu hiện tại không đúng")
				return
			}

			if auth.IsCorrectPassword(user.Password, newForm.NewPassword) {
				c.JSON(http.StatusBadRequest, "Mật khẩu mới phải khác mật khẩu hiện tại ")
				return
			}
		}

		//hash password
		if newForm.NewPassword != "" {
			user.Password = h.hashPasswordV2(newForm.NewPassword)
		}

		user.FullName = newForm.FullName
		user.Birthday = newForm.Birthday

		userUpdated, err := h.UserManager.UpdateUser(user)
		if err != nil {
			h.Logger.Errorf("Update your account user %v error, %v", userID, err)
			c.JSON(http.StatusInternalServerError, "Cập nhật thất bại")
			return
		}

		queryOptions := sqlmanager.SettingQueryOption{
			Key:    constant.BookmarkPushSettingKey,
			UserID: userID,
		}

		setting, err := h.SettingManager.GetSetting(queryOptions)
		if err != nil && err != gorm.ErrRecordNotFound {
			h.Logger.Errorf("Error fetch setting webhook url: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if setting.ID == 0 {
			setting.Key = constant.BookmarkPushSettingKey
			setting.Status = constant.SettingEnableStatus
			setting.UserID = userID
			setting.Value = cast.ToString(newForm.PushBookmark)
		} else {
			setting.Value = cast.ToString(newForm.PushBookmark)
		}

		err = h.SettingManager.SaveSetting(setting)
		if err != nil {
			h.Logger.Errorf("Error save setting push bookmark: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		dto.UserTransform(userUpdated)
		c.JSON(http.StatusOK, UpdateUserResponse{
			userUpdated,
		})
	}
}

func (h *UserHandler) hashPasswordV2(pwd string) string {
	pwdBytes := []byte(pwd)
	hash, err := bcrypt.GenerateFromPassword(pwdBytes, bcrypt.MinCost)
	if err != nil {
		h.Logger.Errorf("Error while hash password %v, details: %v", pwd, err)
	}
	return string(hash)
}
