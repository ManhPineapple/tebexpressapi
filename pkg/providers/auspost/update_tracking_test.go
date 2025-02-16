package auspost

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestUpdate(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	auspost := NewAuspost(nil)

	in := LabelRequest{
		OrderNumber:    "LB00002303",
		PackageID:      16308,
		Code:           "LB00002303",
		ServiceCode:    "AU",
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

	res, msg, err := auspost.Label(in)
	if err != nil {
		t.Error(err)
	}

	if msg != "" {
		t.Errorf("response: %v", msg)
	}

	t.Log(res)

	in.ShipmentID = res.ShipmentID
	in.Weight = 2
	in.Length = 10
	in.Width = 10
	in.Height = 10

	ures, msg, err := auspost.Update(in)
	if err != nil {
		t.Error(err)
	}

	if msg != "" {
		t.Errorf("response: %v", msg)
	}

	t.Log(ures)
}
