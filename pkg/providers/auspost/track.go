package auspost

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type (
	TrackEvent struct {
		Location    string    `json:"location"`
		Description string    `json:"description"`
		Date        time.Time `json:"date"`
	}

	TrackItem struct {
		ArticleID   string       `json:"article_id"`
		ProductType string       `json:"product_type"`
		Status      string       `json:"status"`
		Events      []TrackEvent `json:"events"`
	}

	Tracking struct {
		TrackingID string `json:"tracking_id"`
		Status     string `json:"status"`
		Errors     []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"errors"`

		TrackableItems []struct {
			TrackItem
			ConsignmentID string      `json:"consignment_id"`
			NumberOfItems int         `json:"number_of_items"`
			Items         []TrackItem `json:"items"`
		} `json:"trackable_items"`

		Consignment struct {
			Status string       `json:"status"`
			Events []TrackEvent `json:"events"`
		} `json:"consignment"`
	}

	TrackResponse struct {
		TrackingResults []Tracking `json:"tracking_results"`
	}
)

func (au *Auspost) Track(ids ...string) (*TrackResponse, *ResponseErrors, error) {
	if len(ids) < 1 {
		return nil, nil, errors.New("ids is missing")
	}

	s := strings.Join(ids, ",")

	var buf []byte
	var response interface{}

	if au.ISEnvDevelopment {
		buf = []byte(fmt.Sprintf(DemoDeliveryEvents, s, s))
	} else {
		if err := au.Client.Get("/shipping/v1/track/?tracking_ids="+s, &response, nil); err != nil {
			return nil, nil, err
		}

		b, err := json.Marshal(response)
		if err != nil {
			return nil, nil, err
		}

		buf = b
	}

	result := &TrackResponse{}

	if err := json.Unmarshal(buf, result); err != nil {
		return nil, nil, err
	}

	if len(result.TrackingResults) > 0 {
		return result, nil, nil
	}

	errors := &ResponseErrors{}
	if err := json.Unmarshal(buf, errors); err != nil {
		return nil, nil, err
	}

	return nil, errors, nil
}
