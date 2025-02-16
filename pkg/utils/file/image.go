package file

import (
	"archive/zip"
	"fmt"
	"image/jpeg"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"tebexpressapi/pkg/constant"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
)

func ResizeImageFromByteArray(byteArray []byte, path string, width int) error {
	u, err := url.Parse(path)
	if err != nil {
		return err
	}

	filePath := strings.TrimLeft(u.Path, "/")
	dirPath := constant.DirStorage + filepath.Dir(filePath)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		os.MkdirAll(dirPath, os.ModePerm)
	}

	filePath = constant.DirStorage + filePath

	extension := GetExtensionFromString(path)
	filename := RandomFilename("." + extension)

	err = ioutil.WriteFile(filename, byteArray, 0644)
	if err != nil {
		return err
	}

	defer os.Remove(filename)
	if extension == ".webp" {
		return ResizeImageFromWeb(filename, filePath, width)
	}
	return ResizeImageFromFile(filename, filePath, width)
}

func ResizeImageFromWeb(filename, filePath string, width int) error {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Failed to open image 1: %v\n", err)
		return err
	}

	src, err := webp.Decode(file)
	if err != nil {
		fmt.Printf("Failed to open image 2: %v\n", err)
		return err
	}

	w := uint(width)
	src = resize.Resize(w, 0, src, resize.Lanczos3)

	out, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to open image 3: %v\n", err)
	}
	defer out.Close()
	err = jpeg.Encode(out, src, nil)
	return err
}

func ResizeImageFromFile(filename, filePath string, width int) error {
	src, err := imaging.Open(filename)
	if err != nil {
		fmt.Printf("Failed to open image: %v\n", err)
		return err
	}

	src = imaging.Resize(src, width, 0, imaging.Lanczos)
	err = imaging.Save(src, filePath)
	return err
}

func ResizeImageFromInput(file multipart.File, filePath string, width int) error {
	src, err := imaging.Decode(file)
	if err != nil {
		fmt.Printf("Failed to open image: %v\n", err)
		return err
	}

	src = imaging.Resize(src, width, 0, imaging.Lanczos)
	err = imaging.Save(src, filePath)
	return err
}

func GetExtensionFromString(str string) string {
	var extension = filepath.Ext(str)

	return extension
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func ZipFiles(filename string, files map[string]string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// Add files to zip
	for name, filePath := range files {
		if err = addFileToZip(zipWriter, filePath, name, true); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zipWriter *zip.Writer, localFileName, filename string, deleteLocalFile bool) error {
	fileToZip, err := os.Open(localFileName)
	if err != nil {
		return err
	}
	defer fileToZip.Close()
	if deleteLocalFile {
		defer os.Remove(localFileName)
	}

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Using FileInfoHeader() above only uses the basename of the file. If we want
	// to preserve the folder structure we can overwrite this with the full path.
	header.Name = filename

	// Change to deflate to gain better compression
	// see http://golang.org/pkg/archive/zip/#pkg-constants
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}
