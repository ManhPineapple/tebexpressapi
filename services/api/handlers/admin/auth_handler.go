package admin

import (
	"net/http"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthHandler struct {
	Logger *zap.SugaredLogger

	UserManager *sqlmanager.UserManager
}

type SignInUserForm struct {
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
	Password    string `json:"password"`
}

type SignInResponse struct {
	User        interface{} `json:"user"`
	AccessToken string      `json:"access_token"`
}

func NewAuthHandler(l *zap.SugaredLogger, um *sqlmanager.UserManager) *AuthHandler {
	return &AuthHandler{
		Logger: l,

		UserManager: um,
	}
}

func (h *AuthHandler) SignIn() gin.HandlerFunc {
	return func(c *gin.Context) {
		signInInfo := &SignInUserForm{}
		if err := c.ShouldBindJSON(signInInfo); err != nil {
			h.Logger.Errorf("Error while parse request body, details: %v", err)
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		if (signInInfo.Email == "" && signInInfo.PhoneNumber == "") || signInInfo.Password == "" {
			c.JSON(http.StatusBadRequest, "The email/phone_number and password is required")
			return
		}

		if (signInInfo.PhoneNumber != "" && !utils.ValidPhoneNumber(signInInfo.PhoneNumber)) || (signInInfo.Email != "" && !utils.ValidEmail(signInInfo.Email)) {
			c.JSON(http.StatusBadRequest, "Số điện thoại/Email hoặc mật khẩu không chính xác. Vui lòng thử lại!")
			return
		}

		user, err := h.UserManager.GetUserByUniqueKey(&sqlmanager.FetchUserRequest{
			PhoneNumber: signInInfo.PhoneNumber,
			Email:       signInInfo.Email,
		})
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, "Số điện thoại/Email hoặc mật khẩu không chính xác. Vui lòng thử lại!")
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// Compare password
		if !auth.IsCorrectPassword(user.Password, signInInfo.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Số điện thoại/Email hoặc mật khẩu không chính xác. Vui lòng thử lại!",
			})

			return
		}

		if user.Status != constant.UserStatusActive {
			c.JSON(http.StatusForbidden, "Tài khoản không hợp lệ")
			return
		}

		//Create JWT tokenmanager
		tk := &auth.UserToken{
			UserID: user.ID,
			Role:   user.Role,
		}
		tokenString, err := auth.GenerateUserAccessToken(tk)
		if err != nil {
			h.Logger.Errorf("Generate access token error: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// Transform user for response
		dto.UserTransform(user)
		c.JSON(http.StatusOK, SignInResponse{
			user,
			tokenString,
		})
	}
}
