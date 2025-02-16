package auspost

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestCreateLabel(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	auspost := NewAuspost(nil)

	payload := LabelRequest{
		OrderNumber:    "LB00002303",
		PackageID:      16308,
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		MiddleName:     "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "123 Centre Road",
		Address2:       "",
		City:           "Sydney",
		State:          "NSW",
		Zipcode:        "2000",
		Phone:          "0412345678",
		Email:          "",
		Country:        "AU",
		ItemDetail:     "",
		MassUnit:       "",
		Weight:         1,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "",

		WarehouseCompany:  "John Citizen",
		WarehouseAddress1: "1 Main Street",
		WarehousePhone:    "0401234567",
		WarehouseCity:     "MELBOURNE",
		WarehouseState:    "VIC",
		WarehouseZipcode:  "3000",
		WarehouseCountry:  "AU",
	}

	res, errs, err := auspost.CreateShipment(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Errorf("response: %v", errs)
	}

	t.Log(res)
	if len(res.Shipments) < 1 {
		t.Errorf("response: %v", res)
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
						Group:      "Express Post",
						Layout:     "A4-3pp",
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

	resLabel, errs, err := auspost.CreateLabel(payloadLabel)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Errorf("response: %v", errs)
	}

	t.Log(resLabel)
}
