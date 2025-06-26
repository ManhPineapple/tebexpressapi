package customer

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type UploadHandler struct {
	Logger         *zap.SugaredLogger
	LocalS3        storage.S3
	PackageManager *sqlmanager.PackageManager
}

func NewUploadHandler(l *zap.SugaredLogger, s3 storage.S3, pm *sqlmanager.PackageManager) *UploadHandler {
	return &UploadHandler{
		Logger:  l,
		LocalS3: s3,

		PackageManager: pm,
	}
}

func (h *UploadHandler) DownloadLabel() gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Request.URL.Query().Get("url")
		bucketType := c.Request.URL.Query().Get("type")

		if bucketType == "" || url == "" {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		if !utils.ValidSlug(bucketType) {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		bucketName := viper.GetString(fmt.Sprintf("bucket.%s", bucketType))
		if url == "" {
			c.JSON(http.StatusBadRequest, constant.MessageValidateInput)
			return
		}

		// object, err := h.LocalS3.Read(url, bucketName)
		// if err != nil {
		// 	h.Logger.Error(err)
		// 	return
		// }
		// defer object.Body.Close()
		// // If there is no content length, it is a directory

		// fileNames := strings.Split(url, "/")
		// fileName := fileNames[len(fileNames)-1]
		// h.Logger.Info("fileName: ", fileName)
		// c.Header("Content-Type", *object.ContentType)
		// c.Header("Content-Length", fmt.Sprintf("%d", *object.ContentLength))
		// c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%v", fileName))
		// c.Status(http.StatusOK)
		// c.Writer.Write(object.Body.) (c.Writer, object.Body)

		object, err, contentType := h.LocalS3.ReadFileForUpload(url, bucketName)
		if err != nil {
			h.Logger.Error(err)
			return
		}

		pkg, err := h.PackageManager.GetPackage(sqlmanager.PackageQueryOption{
			Label: url,
		})
		if err != nil {
			h.Logger.Error(err)
			return
		}

		now := time.Now()
		pkg.LastPrintLabelAt = &now
		err = h.PackageManager.UpdatePackage(&pkg, pkg.ID)
		if err != nil {
			h.Logger.Error(err)
			return
		}

		// If there is no content length, it is a directory
		body := object.Bytes()
		fileNames := strings.Split(url, "/")
		fileName := fileNames[len(fileNames)-1]
		h.Logger.Info("fileName: ", fileName, *contentType)
		c.Header("Content-Type", *contentType)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Length", strconv.Itoa(len(body)))
		c.Status(http.StatusOK)
		c.Writer.Write(body)

	}
}
