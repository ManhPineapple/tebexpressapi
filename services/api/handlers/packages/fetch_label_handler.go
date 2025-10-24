package packages

import (
	"net/http"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/httputil"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func (h *PackageHandler) FetchLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := cast.ToInt64(c.Request.Header.Get("X-User-Id"))
		code := strings.TrimSpace(c.Param("code"))
		if code == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"Ananbay tracking is invalid"},
			})

			return
		}

		var result struct {
			ID    int64  `json:"id"`
			Label string `json:"label"`
		}

		err := h.PackageManager.FetchPackage(sqlmanager.PackageQueryOption{
			Code:   code,
			UserID: userId,
		}, "packages.id,packages.label", &result)

		if err == gorm.ErrRecordNotFound || result.ID == 0 {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"invalid token"},
			})

			return
		}

		if err != nil {
			h.Logger.Errorf("Get package error: %v", err)
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		if result.Label == "" {
			c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
				Error:    constant.MessageValidateInput,
				Messages: []string{"invalid tracking number"},
			})

			return
		}
		bucketName := viper.GetString("bucket.labels")
		url := result.Label

		// var tracking struct {
		// 	ID       int64  `json:"id"`
		// 	LabelURL string `json:"label_url"`
		// }
		// err = h.TrackingManager.GetTrackingByField(sqlmanager.TrackingOption{
		// 	PackageID: result.ID,
		// 	Status:    constant.TrackingStatusSuccess,
		// }, "id,label_url", &tracking)

		// if err != nil && err != gorm.ErrRecordNotFound {
		// 	h.Logger.Errorf("Get tracking error: %v", err)
		// 	c.JSON(http.StatusBadRequest, httputil.ErrorResponse{
		// 		Error: constant.MessageServerInternalError,
		// 	})

		// 	return
		// }

		// if tracking.ID > 0 {
		// 	url = tracking.LabelURL
		// }

		data, err := h.StorageS3.PreAssign(url, bucketName)
		if err != nil {
			h.Logger.Errorf("Read file error: %v", err)
			c.JSON(http.StatusInternalServerError, httputil.ErrorResponse{
				Error: constant.MessageServerInternalError,
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"url": data,
		})
	}
}
