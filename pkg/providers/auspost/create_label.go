package auspost

import "encoding/json"

type (
	LabelGroup struct {
		Group      string `json:"group"`
		Layout     string `json:"layout"`
		Branded    bool   `json:"branded"`
		LeftOffset int    `json:"left_offset"`
		TopOffset  int    `json:"top_offset"`
	}

	LabelPreference struct {
		Type   string       `json:"type"`
		Format string       `json:"format"`
		Groups []LabelGroup `json:"groups"`
	}

	LabelShipmentItem struct {
		ItemID string `json:"item_id"`
	}

	LabelShipment struct {
		ShipmentID string               `json:"shipment_id"`
		Items      *[]LabelShipmentItem `json:"items,omitempty"`
	}

	FormCreateLabel struct {
		WaitForLabelUrl        bool              `json:"wait_for_label_url,omitempty"`
		UnlabelledArticlesOnly bool              `json:"unlabelled_articles_only,omitempty"`
		Preferences            []LabelPreference `json:"preferences"`
		Shipments              []LabelShipment   `json:"shipments"`
	}

	Label struct {
		RequestID       string          `json:"request_id"`
		URL             string          `json:"url"`
		Status          string          `json:"status"`
		RequestDate     string          `json:"request_date"`
		URLCreationDate string          `json:"url_creation_date"`
		Shipments       []LabelShipment `json:"shipments"`
		ShipmentIDs     []string        `json:"shipment_ids"`
		LabelProperties struct {
			Format   string `json:"format"`
			PageSize string `json:"page_size"`
		} `json:"label_properties"`
	}

	CreateLabelResponse struct {
		Labels []Label `json:"labels"`
	}
)

func (au *Auspost) CreateLabel(payload FormCreateLabel) (*CreateLabelResponse, *ResponseErrors, error) {
	var response interface{}
	if err := au.Client.Post("/shipping/v1/labels", payload, &response); err != nil {
		return nil, nil, err
	}

	b, err := json.Marshal(response)
	if err != nil {
		return nil, nil, err
	}

	result := &CreateLabelResponse{}
	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, nil, err
	}

	if len(result.Labels) > 0 {
		return result, nil, nil
	}

	errors := &ResponseErrors{}
	err = json.Unmarshal(b, errors)
	if err != nil {
		return nil, nil, err
	}

	return nil, errors, nil
}
