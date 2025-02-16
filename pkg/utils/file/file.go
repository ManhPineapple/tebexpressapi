package file

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"tebexpressapi/pkg/utils/string_util"
	"time"

	"github.com/google/uuid"
)

func MakeOrderPath(sellerId, shopId int64, fileP *multipart.FileHeader) string {
	t := time.Now()
	day := t.Format("2006-01-02")
	unixID := uuid.New().String()

	fileExt, _ := ExtensionContentType(fileP.Header.Get("Content-Type"))
	filename := strings.Split(fileP.Filename, ".csv")
	fileName := fmt.Sprintf("%d/%s/%s___%s%s", shopId, day, filename[0], unixID, fileExt)
	filePath := strconv.FormatInt(sellerId, 10) + "/" + fileName

	return filePath
}

func MakeFilePath(userID int64, fh *multipart.FileHeader) string {
	t := time.Now()
	day := t.Format("2006-01-02")
	unixID := uuid.New().String()

	fileExt, _ := ExtensionContentType(fh.Header.Get("Content-Type"))
	filename := strings.ReplaceAll(fh.Filename, ".csv", "")
	filePath := fmt.Sprintf("%d/%s/%s___%s%s", userID, day, filename, unixID, fileExt)

	return filePath
}

func MakeChunkOrderPath(sellerId, shopId int64) string {
	t := time.Now()
	fileName := fmt.Sprintf("%d/%s/", shopId, t.Format("2006-01-02-15-04-05"))
	filePath := strconv.FormatInt(sellerId, 10) + "/" + fileName

	return filePath
}

func MakePathExport(userID int64) string {
	t := time.Now()
	filePath := strconv.FormatInt(userID, 10) + "/" + t.Format("2006-01-02-15-04-05")
	return filePath
}

func MakeTrackingNumberPath(supplierId int64) string {
	filePath := RandomFilenamePath(supplierId, ".csv")

	return filePath
}

func MakeExportPath(supplierId, ConsignmentId int64) string {
	filePath := strconv.FormatInt(supplierId, 10) + "/" + RandomFilenamePath(ConsignmentId, ".csv")

	return filePath
}

func MakeDesignPath(sellerId int64, sku string) string {
	filePath := strconv.FormatInt(sellerId, 10) + "/" + sku + "/"

	return filePath
}

func MakeUploadPath(userId int64, folder string) string {
	if folder == "" {
		return fmt.Sprintf("%d/", userId)
	}
	filePath := fmt.Sprintf("%s/%d/", folder, userId)
	return filePath
}
func MakeUploadUserIDPath(userId int64, folder string) string {
	if folder == "" {
		return fmt.Sprintf("%d/", userId)
	}
	filePath := fmt.Sprintf("%d/", userId)
	return filePath
}

func RandomFilenamePath(id int64, ext string) string {
	t := time.Now()
	fileName := fmt.Sprintf("%d/%s%s/%s%s", id, t.Format("2006-01"), t.Format("02-03-04-05"), uuid.New().String(), ext)

	return fileName
}

func RandomFilename(ext string) string {
	return fmt.Sprintf("%s%s", uuid.New().String(), ext)
}

func MakeFilename(name string) string {
	ext := GetExtensionFromString(name)
	fileName := strings.ReplaceAll(name, ext, "")
	fileName = string_util.ToSlug(fileName)

	return fmt.Sprintf("%s%s", fileName, ext)
}

func MakeChunkFilename(name string, chunkIndex int) string {
	ext := GetExtensionFromString(name)
	fileName := strings.ReplaceAll(name, ext, "")
	fileName = string_util.ToSlug(fileName)

	return fmt.Sprintf("%s_%d%s", fileName, chunkIndex, ext)
}

func ExtensionContentType(contentType string) (string, error) {
	switch contentType {
	case "image/png":
		return ".png", nil
	case "image/jpeg":
		return ".jpg", nil
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ".xlsx", nil
	case "application/vnd.ms-excel",
		"text/csv",
		"application/csv",
		"text/x-csv",
		"application/x-csv",
		"text/x-comma-separated-values",
		"text/comma-separated-values":
		return ".csv", nil
	default:
		return "", errors.New("File invalid")
	}
}

// GetFileNameAndExt extracts name and extension from path
func GetFileNameAndExt(path string) (string, string) {
	split := strings.Split(path, "/")
	name := split[len(split)-1]
	split = strings.Split(name, ".")
	ext := split[len(split)-1]
	nSplit := split[:len(split)-1]
	name = strings.Join(nSplit, "")

	return name, ext
}
