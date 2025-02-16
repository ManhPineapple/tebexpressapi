package usps

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/axgle/mahonia"
)

// TrackResponse represents the structure of the XML response from USPS API
type TrackResponse struct {
	XMLName   xml.Name   `xml:"TrackResponse"`
	TrackInfo *TrackInfo `xml:"TrackInfo,omitempty"`
}

// TrackInfo represents the TrackInfo element in the USPS API response
type TrackInfo struct {
	TrackDetails []*TrackDetail `xml:"TrackDetail"`
}

// TrackDetail represents each tracking event detail in the USPS API response
type TrackDetail struct {
	Event        string `xml:"Event,omitempty"`
	EventDate    string `xml:"EventDate,omitempty"`
	EventTime    string `xml:"EventTime,omitempty"`
	EventCity    string `xml:"EventCity,omitempty"`
	EventState   string `xml:"EventState,omitempty"`
	EventZIPCode string `xml:"EventZIPCode,omitempty"`
}

func TestTrack(t *testing.T) {
	// Chinese text you want to translate
	chineseText := "你好，世界！"

	// Convert Chinese text to UTF-8 encoded bytes
	enc := mahonia.NewDecoder("gbk")
	if enc == nil {
		fmt.Println("Failed to create decoder")
		return
	}
	utf8Str := enc.ConvertString(chineseText)

	// Print the translated text (assumes conversion works for basic Chinese characters)
	fmt.Printf("Original: %s\nTranslated: %s\n", chineseText, utf8Str)
}
