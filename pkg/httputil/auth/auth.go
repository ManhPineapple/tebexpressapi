package auth

import (
	"fmt"
	"net/http"
	"tebexpressapi/pkg/constant"

	"tebexpressapi/pkg/models/entity"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

type Auth struct {
	db *gorm.DB
}

func Init(db *gorm.DB) *Auth {
	return &Auth{db: db}
}

func (m *Auth) VerifyCustomer() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, token := AuthBasic(c.Request)

		if id == "" || token == "" {
			c.String(http.StatusUnauthorized, constant.MessageUnauthorized)
			c.Abort()
		}

		db := m.db.Select("users.*")
		db = db.Where("(users.email=? OR users.phone_number=?) AND user_tokens.token=?", id, id, token)
		db = db.Where("user_tokens.status=?", constant.UserStatusActive)
		db = db.Where("users.status=?", constant.UserStatusActive)
		db.Joins("INNER JOIN user_tokens ON user_tokens.user_id=users.id")

		user := &entity.User{}
		db = db.First(user)

		if db.Error != nil || user.ID <= 0 || user.Status != constant.UserStatusActive {
			fmt.Errorf("authentic: %v", db.Error)
			c.String(http.StatusUnauthorized, constant.APIResponseMessageUnauthorized)
			c.Abort()
		}

		c.Request.Header.Set("X-User-Id", cast.ToString(user.ID))
		c.Request.Header.Set("X-User-Role", cast.ToString(user.Role))
		c.Next()
	}
}
