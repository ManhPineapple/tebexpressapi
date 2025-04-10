package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
)

type OCRResponse struct {
	OCRExitCode   int      `json:"OCRExitCode"`
	ErrorMessage  []string `json:"ErrorMessage"`
	ParsedResults []struct {
		ParsedText string `json:"ParsedText"`
	} `json:"ParsedResults"`
}

func GetOcrOutput(url string) (string, error) {
	// Transform ggdrive view url to download url
	driveRegex := regexp.MustCompile(`drive\.google\.com/file/d/([^/]+)/`)
	matches := driveRegex.FindStringSubmatch(url)
	if len(matches) > 1 {
		fileID := matches[1]
		url = fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
	}

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	_ = writer.WriteField("language", "eng")
	_ = writer.WriteField("isOverlayRequired", "false")
	_ = writer.WriteField("url", url)
	_ = writer.WriteField("iscreatesearchablepdf", "false")
	_ = writer.WriteField("issearchablepdfhidetextlayer", "false")
	_ = writer.WriteField("filetype", "pdf")

	writer.Close()

	req, err := http.NewRequest("POST", "https://api.ocr.space/parse/image", &requestBody)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apikey", "K82161124588957")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ocrResponse OCRResponse
	if err := json.Unmarshal(body, &ocrResponse); err != nil {
		return "", err
	}

	if ocrResponse.OCRExitCode != 1 {
		return "", errors.New(fmt.Sprintf("OCR Error: %v", ocrResponse.ErrorMessage))
	}

	// Return parsed text if available
	if len(ocrResponse.ParsedResults) > 0 {
		return ocrResponse.ParsedResults[0].ParsedText, nil
	}

	return "", errors.New("No parsed text found")
}
