package auspost

import (
	"errors"
)

type LabelRequest struct {
	ShipmentID     string  `json:"shipment_id"`
	OrderNumber    string  `json:"order_number"`
	PackageID      int64   `json:"package_id"`
	Code           string  `json:"code"`
	TrackingNumber string  `json:"tracking_number"`
	FirstName      string  `json:"first_name"`
	MiddleName     string  `json:"middle_name"`
	LastName       string  `json:"last_name"`
	FullName       string  `json:"full_name"`
	Company        string  `json:"company"`
	Address1       string  `json:"line1"`
	Address2       string  `json:"line2"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Zipcode        string  `json:"zipcode"`
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	Country        string  `json:"country_code"`
	ItemDetail     string  `json:"item_detail"`
	MassUnit       string  `json:"mass_unit"`
	Weight         float64 `json:"weight"`
	Length         float64 `json:"length"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	DistanceUnit   string  `json:"distance_unit"`
	PostmarkDate   int64   `json:"postmark_date"`

	DisplayWeight float64 `json:"-"`
	ServiceCode   string  `json:"-"`
	HubStateCode  string  `json:"-"`
	LabelTemplate string  `json:"-"`
	IsExceedPkg   bool    `json:"-"`

	WarehouseCompany  string `json:"warehouse_company"`
	WarehouseAddress1 string `json:"warehouse_address1"`
	WarehousePhone    string `json:"warehouse_phone"`
	WarehouseCity     string `json:"warehouse_city"`
	WarehouseState    string `json:"warehouse_state"`
	WarehouseZipcode  string `json:"warehouse_zipcode"`
	WarehouseCountry  string `json:"warehouse_country"`
}

type LabelResponse struct {
	ShipmentID     string  `json:"shipment_id"`
	ProductID      string  `json:"product_id"`
	TrackingNumber string  `json:"tracking_number"`
	LabelUrl       string  `json:"label_url"`
	ShippingCost   float64 `json:"shipping_cost"`
	TotalCost      float64 `json:"total_cost"`
	TotalCostExGst float64 `json:"total_cost_ex_gst"`
	TotalGst       float64 `json:"total_gst"`
}

func (au *Auspost) Label(in LabelRequest) (*LabelResponse, string, error) {
	to := Address{
		Name:         in.FullName,
		Lines:        []string{in.Address1},
		Suburb:       in.City,
		State:        in.State,
		Postcode:     in.Zipcode,
		Phone:        in.Phone,
		Email:        in.Email,
		Country:      in.Country,
		BusinessName: in.Company,
	}

	if in.Address2 != "" {
		to.Lines = append(to.Lines, in.Address2)
	}

	res, errs, err := au.CreateShipment(in)
	if err != nil {
		return nil, "", err
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs.Errors[0].Message, err
	}

	if len(res.Shipments) < 1 {
		return nil, "", errors.New("response not found")
	}

	shipment := res.Shipments[0]
	if len(shipment.Items) < 1 {
		return nil, "", errors.New("response miss items")
	}

	item := shipment.Items[0]

	result := &LabelResponse{
		ShipmentID:     shipment.ShipmentID,
		TrackingNumber: item.TrackingDetails.ArticleID,
		TotalCost:      shipment.ShipmentSummary.TotalCost,
		ShippingCost:   shipment.ShipmentSummary.ShippingCost,
		TotalCostExGst: shipment.ShipmentSummary.TotalCostExGst,
		TotalGst:       shipment.ShipmentSummary.TotalGst,
		ProductID:      item.ProductID,
	}

	payloadLabel := FormCreateLabel{
		WaitForLabelUrl:        true,
		UnlabelledArticlesOnly: false,
		Preferences: []LabelPreference{
			{
				Type:   "PRINT",
				Format: "PDF",
				Groups: []LabelGroup{
					{
						Group:      au.ProductExpressName,
						Layout:     au.ProductExpressLayout,
						Branded:    false,
						LeftOffset: 0,
						TopOffset:  0,
					},
				},
			},
		},
		Shipments: []LabelShipment{
			{
				ShipmentID: res.Shipments[0].ShipmentID,
			},
		},
	}

	resLabel, errs, err := au.CreateLabel(payloadLabel)
	if err != nil {
		return result, "", err
	}

	if errs != nil && len(errs.Errors) > 0 {
		return result, errs.Errors[0].Message, err
	}

	if len(resLabel.Labels) < 1 {
		return result, "", errors.New("response miss label")
	}

	result.LabelUrl = resLabel.Labels[0].URL
	return result, "", nil
}
