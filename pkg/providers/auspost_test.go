package providers

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestAuspostCreateLabel(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeAuspost, 0)
	res, errAudit, err := carrier.CreateLabel(RequestCreateLabel{
		OrderNumber:    "LB00002303",
		ID:             16308,
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
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
		ItemName:       "",
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
	})
	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	t.Log(res)
}
