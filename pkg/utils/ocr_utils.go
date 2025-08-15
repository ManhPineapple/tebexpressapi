package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

	re := regexp.MustCompile(`9\d{3}(?:\s?\d{4}){4}\s?\d{2}`)
	match := re.FindString(parsedText)
	trackingNumber = strings.ReplaceAll(match, " ", "")

	for i := 0; i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), "usps tracking #") && i >= 3 {
			mapchange["recipient"] = strings.TrimSpace(lines[i-3])
			mapchange["address_1"] = strings.TrimSpace(lines[i-2])
			// trackingNumber = strings.ReplaceAll(strings.TrimSpace(lines[i+1]), " ", "")

			cityStateZip := strings.TrimSpace(lines[i-1])
			cityStateZipRegex := regexp.MustCompile(`^(.+?)\s([A-Z]{2})\s(\d{5,9}(?:-\d{4})?)$`)

			matches := cityStateZipRegex.FindStringSubmatch(cityStateZip)
			if len(matches) >= 4 {
				mapchange["city"] = matches[1]
				mapchange["state_code"] = matches[2]
				mapchange["zipcode"] = matches[3]
				mapchange["country_code"] = "US"
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
		log.Printf("Failed to send request: %v. Falling back to OCRSpace", err)
		return GetOcrSpaceOutput(pdfURL)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v. Falling back to OCRSpace", err)
		return GetOcrSpaceOutput(pdfURL)
	}

	var apiResp struct {
		Code    string                 `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		log.Printf("Failed to unmarshal response: %v. Falling back to OCRSpace", err)
		return GetOcrSpaceOutput(pdfURL)
	}

	if apiResp.Code != "1" {
		log.Printf("API returned error code: %s, message: %s. Falling back to OCRSpace", apiResp.Code, apiResp.Message)
		return GetOcrSpaceOutput(pdfURL)
	}

	if !validateApiData(apiResp.Data) {
		log.Printf("API returned invalid data: %+v. Falling back to OCRSpace", apiResp.Data)
		return GetOcrSpaceOutput(pdfURL)
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

	mapchange["country_code"] = "US"
	trackingNumber, _, err := GetOcrSpaceOutput(pdfURL)

	return trackingNumber, mapchange, nil
}

func validateApiData(data map[string]interface{}) bool {
	requiredFields := []string{"name", "address", "zipcode", "state", "city"}

	for _, field := range requiredFields {
		val, ok := data[field].(string)
		if !ok || val == "" {
			return false
		}
	}

	// zipcode: must be only digits and length between 4-10
	if zipcode, _ := data["zipcode"].(string); !regexp.MustCompile(`^\d{5}(-\d{4})?$`).MatchString(zipcode) {
		return false
	}

	// state: must be alphabetic (2–20 chars) or valid US state code
	if state, _ := data["state"].(string); !regexp.MustCompile(`^[A-Za-z]{2,20}$`).MatchString(state) {
		return false
	}

	// city: reject if too short or too garbled (you could refine this)
	if city, _ := data["city"].(string); len(city) < 2 {
		return false
	}

	return true
}
