package auth

import (
	"fmt"
	"log"
	"net/http"
	"tebexpressapi/pkg/constant"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

const (
	ErrTokenParse            string = "cant parse user access token"
	ErrParseUserClaimsFailed string = "Parse user claims failed"
)

type AuthInfo struct {
	Enable         bool
	IsCustomer     bool
	UserRoles      map[string]bool
	RequiredFields map[string]bool
}

func (m *Auth) AuthenticationVerify(authInfo *AuthInfo) gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken, err := ParseUserAccessToken(c.Request)
		if err != nil {
			log.Println(err)
			c.String(http.StatusUnauthorized, ErrTokenParse)
			c.Abort()
		}

		if userToken.UserID < 1 {
			c.String(http.StatusUnauthorized, constant.HttpErrorTokenExpired.Error())
			c.Abort()
		}

		// If user role change, need fix this
		if authInfo.UserRoles != nil && len(authInfo.UserRoles) > 0 {
			if isEnabledRole, ok := authInfo.UserRoles[userToken.Role]; !ok || !isEnabledRole {
				fmt.Errorf("user role %v", constant.HttpErrorForbidden.Error())
				fmt.Errorf("Auth slice %v", authInfo.UserRoles)
				fmt.Errorf("Role user %v", userToken.Role)

				c.String(http.StatusUnauthorized, fmt.Sprintf("user role %v", constant.HttpErrorForbidden.Error()))
				c.Abort()
			}
		}

		// Assign data to request header
		c.Request.Header.Set("X-User-Id", cast.ToString(userToken.UserID))
		c.Request.Header.Set("X-User-Role", cast.ToString(userToken.Role))
		c.Next()
	}
}
