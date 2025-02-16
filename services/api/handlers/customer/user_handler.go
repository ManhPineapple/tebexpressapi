package customer

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"go.uber.org/zap"
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
