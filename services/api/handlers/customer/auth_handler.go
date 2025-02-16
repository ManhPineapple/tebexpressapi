package customer

import (
	"fmt"
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil/auth"
	"tebexpressapi/pkg/models/dto"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	Logger *zap.SugaredLogger
	Redis  *redis.Client

	PartnerMap map[string]int64

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

type SignUpUser struct {
	entity.User
	TkExpire     string `json:"tk_expire"`
	ReferralCode string `json:"referral_code"`
}

type SignUpUserForm struct {
	User SignUpUser `json:"user"`
}

type SignUpResponse struct {
	User interface{} `json:"user"`
}

func NewAuthHandler(l *zap.SugaredLogger, r *redis.Client, um *sqlmanager.UserManager) *AuthHandler {
	return &AuthHandler{
		Logger: l,
		Redis:  r,

		PartnerMap: entity.PartnerMap,

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
			h.Logger.Errorf("Error get user: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		// Compare password
		if !auth.IsCorrectPassword(user.Password, signInInfo.Password) {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"error": "Số điện thoại/Email hoặc mật khẩu không chính xác. Vui lòng thử lại!",
			})
			return
		}

		if user.Status == constant.UserStatusDeactive {
			c.JSON(http.StatusForbidden, "Tài khoản của bạn đã ngừng hoạt động")
			return
		}

		// If Account is not yet activated, send a new email confirm
		if user.Status == constant.UserStatusInactive {
			c.JSON(http.StatusForbidden, map[string]interface{}{
				"user":  user,
				"error": "Tài khoản của bạn chưa được kích hoạt. Liên hệ support để được hỗ trợ",
			})
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

func (h *AuthHandler) SignUp() gin.HandlerFunc {
	return func(c *gin.Context) {
		signUpInfo := &SignUpUserForm{}
		if err := c.ShouldBindJSON(signUpInfo); err != nil {
			c.JSON(http.StatusBadRequest, constant.MessageParseRequestBody)
			return
		}

		validate := h.validateSignUpInfo(signUpInfo)
		if len(validate) > 0 {
			c.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
				"message": constant.MessageValidateInput,
				"errors":  validate,
			})
			return
		}

		var referralUserID int64
		if signUpInfo.User.TkExpire != "" {
			code := strings.TrimSpace(signUpInfo.User.TkExpire)
			key := fmt.Sprintf("%s_%s", "invite_token", code)

			str, err := h.Redis.Get(c, key).Result()
			if err == redis.Nil {
				c.JSON(http.StatusNotFound, constant.MessageNotFound)
				return
			}

			if err != nil {
				h.Logger.Errorf("get info customer invite in redis: %v", err)
				c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
				return
			}

			arr := strings.Split(str, ":")
			if len(arr) < 2 {
				c.JSON(http.StatusNotFound, constant.MessageNotFound)
				return
			}

			email := arr[0]

			if !strings.EqualFold(signUpInfo.User.Email, email) {
				c.JSON(http.StatusBadRequest, "Mã giới thiệu không hợp lệ hoặc đã hết hạn")
				return
			}

		}

		userInfo, err := h.parseRequestUserInfo(signUpInfo)
		if err != nil {
			h.Logger.Errorf("Error parse request, details: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		if referralUserID > 0 {
			userInfo.RefID = utils.Int64(referralUserID)
		}

		if signUpInfo.User.TkExpire != "" {
			userInfo.Status = constant.UserStatusActive
		}

		userInfo.PartnerID = entity.PartnerMap[c.Request.Host]
		user, err := h.UserManager.CreateUser(userInfo, referralUserID)
		if err != nil {
			h.Logger.Errorf("Error while create user, details: %v", err)
			c.JSON(http.StatusInternalServerError, constant.MessageServerInternalError)
			return
		}

		dto.UserTransform(user)
		c.JSON(http.StatusOK, SignUpResponse{
			user,
		})
	}
}

func (h *AuthHandler) parseRequestUserInfo(SignUpV2Info *SignUpUserForm) (*entity.User, error) {
	userInfo := SignUpV2Info.User
	userInfo.User.FullName = strings.TrimSpace(userInfo.User.FullName)
	userInfo.User.Email = strings.TrimSpace(userInfo.User.Email)

	user := userInfo.User

	user.Role = constant.UserRoleCustomer
	user.Status = constant.UserStatusInactive
	user.Class = constant.UserClassPublic

	// Hash password
	user.Password = h.hashPasswordV2(user.Password)

	return &user, nil
}

func (h *AuthHandler) validateSignUpInfo(info *SignUpUserForm) []string {
	messages := make([]string, 0)

	usernameLength := len(info.User.FullName)
	if usernameLength < 1 {
		messages = append(messages, "Tên tài khoản không được để trống")
	}
	// } else if !utils.ValidFullName(info.User.FullName) {
	// 	messages = append(messages, "Tên tài khoản không hợp lệ")
	// }

	if info.User.Package == 0 {
		messages = append(messages, "Vui lòng chọn quy mô vận chuyển")
	} else if _, exist := constant.DefinedPackageValue[info.User.Package]; !exist {
		messages = append(messages, "Quy mô vận chuyển không hợp lệ")
	}

	phonenumberLength := len(info.User.PhoneNumber)
	if 1 > phonenumberLength || 20 < phonenumberLength {
		messages = append(messages, "Số điện thoại không được để trống, phải từ 1 đến 20 ký tự")
	} else if !utils.ValidPhoneNumber(info.User.PhoneNumber) {
		messages = append(messages, "Số điện thoại không hợp lệ")
	} else {
		isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
			PhoneNumber: info.User.PhoneNumber,
		})

		if isExist {
			messages = append(messages, "Số điện thoại đã được sử dụng, vui lòng chọn số điện thoại khác")
		}
	}

	if info.User.Email == "" {
		messages = append(messages, "Email không được để trống")
	} else if !utils.ValidEmail(info.User.Email) {
		messages = append(messages, "Email không hợp lệ")
	} else {
		isExist := h.UserManager.IsExistUniqueKey(&sqlmanager.FetchUserRequest{
			Email: info.User.Email,
		})

		if isExist {
			messages = append(messages, "Email đã được sử dụng, vui lòng chọn email khác")
		}
	}

	if errValidPassword := utils.ValidatePassword(info.User.Password); errValidPassword != "" {
		messages = append(messages, errValidPassword)
	}

	return messages
}

func (h *AuthHandler) hashPasswordV2(pwd string) string {
	pwdBytes := []byte(pwd)
	hash, err := bcrypt.GenerateFromPassword(pwdBytes, bcrypt.MinCost)
	if err != nil {
		h.Logger.Errorf("Error while hash password %v, details: %v", pwd, err)
	}
	return string(hash)
}
