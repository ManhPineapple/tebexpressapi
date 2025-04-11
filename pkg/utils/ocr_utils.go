package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
)

type OCRResponse struct {
	OCRExitCode   int      `json:"OCRExitCode"`
	ErrorMessage  []string `json:"ErrorMessage"`
	ParsedResults []struct {
		ParsedText string `json:"ParsedText"`
	} `json:"ParsedResults"`
}

func GetOcrSpaceOutput(url string) (string, map[string]interface{}, error) {
	// Transform Google Drive view URL to direct download
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
		return "", nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apikey", "K82161124588957")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var ocrResponse OCRResponse
	if err := json.Unmarshal(body, &ocrResponse); err != nil {
		return "", nil, err
	}

	if ocrResponse.OCRExitCode != 1 {
		return "", nil, fmt.Errorf("OCR Error: %v", ocrResponse.ErrorMessage)
	}

	if len(ocrResponse.ParsedResults) == 0 {
		return "", nil, errors.New("No parsed text found")
	}

	parsedText := strings.ReplaceAll(ocrResponse.ParsedResults[0].ParsedText, "\\r\\n", "\n")
	lines := strings.Split(parsedText, "\n")
	mapchange := make(map[string]interface{})
	trackingNumber := ""

	for i := 0; i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), "usps tracking #") && i >= 3 {
			mapchange["recipient"] = strings.TrimSpace(lines[i-3])
			mapchange["address_1"] = strings.TrimSpace(lines[i-2])
			trackingNumber = strings.ReplaceAll(strings.TrimSpace(lines[i+1]), " ", "")

			cityStateZip := strings.TrimSpace(lines[i-1])
			cityStateZipRegex := regexp.MustCompile(`^(.+?)\s([A-Z]{2})\s(\d{5,9}(?:-\d{4})?)$`)
			matches := cityStateZipRegex.FindStringSubmatch(cityStateZip)
			if len(matches) >= 4 {
				mapchange["city"] = matches[1]
				mapchange["state_code"] = matches[2]
				mapchange["zipcode"] = matches[3]
			}
			break
		}
	}

	return trackingNumber, mapchange, nil
}

func GetNslogOcrOutput(pdfURL string) (string, map[string]interface{}, error) {
	// Handle Google Drive URL conversion
	driveRegex := regexp.MustCompile(`drive\.google\.com/file/d/([^/]+)/`)
	if matches := driveRegex.FindStringSubmatch(pdfURL); len(matches) > 1 {
		fileID := matches[1]
		pdfURL = fmt.Sprintf("https://drive.google.com/uc?export=download&id=%s", fileID)
	}

	// Download the PDF
	resp, err := http.Get(pdfURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download PDF: %v", err)
	}
	defer resp.Body.Close()

	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read PDF content: %v", err)
	}

	// call ocr api
	labelBase64 := base64.StdEncoding.EncodeToString(pdfBytes)
	fmt.Print(labelBase64)
	bodyData := map[string]string{"label": labelBase64}
	jsonBody, err := json.Marshal(bodyData)
	if err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest("POST", "http://express.nslogvietnam.com/api/LabelToConsignee", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("CODE", "test")
	req.Header.Set("TOKEN", "jzb2C8YB3NTmQcxGM4JBeXnfnRfj2rH3")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	var apiResp struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", nil, err
	}

	if apiResp.Code != "1" {
		return "", nil, fmt.Errorf("API Error: %s", apiResp.Message)
	}

	data := apiResp.Data
	mapchange := make(map[string]interface{})

	if v, ok := data["name"]; ok {
		mapchange["recipient"] = v
	}
	if v, ok := data["address"]; ok {
		mapchange["address_1"] = v
	}
	if v, ok := data["city"]; ok {
		mapchange["city"] = v
	}
	if v, ok := data["state"]; ok {
		mapchange["state_code"] = v
	}
	if v, ok := data["zipcode"]; ok {
		mapchange["zipcode"] = v
	}

	trackingNumber, _, err := GetOcrSpaceOutput(pdfURL)

	return trackingNumber, mapchange, nil
}
