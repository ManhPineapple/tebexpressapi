package auth

import (
	"fmt"
	"log"
	"net/http"
	"tebexpressapi/pkg/constant"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

func (m *Auth) AuthenticationCustomerVerify(authInfo *AuthInfo) gin.HandlerFunc {
	return func(c *gin.Context) {
		userToken, err := ParseUserAccessToken(c.Request)
		if err != nil {
			log.Println(err)
			c.String(http.StatusUnauthorized, ErrTokenParse)
			c.Abort()
			return
		}

		if userToken.UserID < 1 {
			c.String(http.StatusUnauthorized, constant.HttpErrorTokenExpired.Error())
			c.Abort()
			return
		}

		// If user role change, need fix this
		if authInfo.UserRoles != nil && len(authInfo.UserRoles) > 0 {
			if isEnabledRole, ok := authInfo.UserRoles[userToken.Role]; !ok || !isEnabledRole {
				fmt.Printf("user role %v", constant.HttpErrorForbidden.Error())
				fmt.Printf("Auth slice %v", authInfo.UserRoles)
				fmt.Printf("Role user %v", userToken.Role)
				c.String(http.StatusUnauthorized, fmt.Sprintf("user role %v", constant.HttpErrorForbidden.Error()))
				c.Abort()
				return
			}
		}

		// Assign data to request header
		c.Request.Header.Set("X-User-Id", cast.ToString(userToken.UserID))
		c.Request.Header.Set("X-User-Role", cast.ToString(userToken.Role))

		if userToken.Role == constant.UserRoleCustomer {
			shopID := cast.ToString(userToken.ActiveShopID)

			// Too many queries are using shopId for filter but if the shopId is less than 1 the where statement
			// maybe ignored by the sqlmanager. For sellers, we should always filter by the shopId to avoid the data that
			// does not belong to them.
			// So this is the fucking trick for now.
			if userToken.ActiveShopID < 1 {
				shopID = cast.ToString(constant.FakeShopId)
			}

			c.Request.Header.Set("X-Active-Shop-Id", shopID)

		}

		// Assign data to request header
		c.Request.Header.Set("X-User-Id", cast.ToString(userToken.UserID))
		c.Request.Header.Set("X-User-Role", cast.ToString(userToken.Role))
		c.Next()
	}
}
