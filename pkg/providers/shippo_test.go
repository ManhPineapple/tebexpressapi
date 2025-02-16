package providers

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestCreateLabel(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "1922 Lake Roberts Landing Drive",
		Address2:       "",
		City:           "WINTER GARDEN",
		State:          "FL",
		Zipcode:        "34787",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         100,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462-4014",
		WarehouseCountry:  "US",
	}

	res, errs, err := carrier.CreateLabel(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	t.Log(res)
}

func TestCancelLabel(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "1922 Lake Roberts Landing Drive",
		Address2:       "",
		City:           "WINTER GARDEN",
		State:          "FL",
		Zipcode:        "34787",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         100,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462-4014",
		WarehouseCountry:  "US",
	}

	res, errs, err := carrier.CreateLabel(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	ok, err := carrier.CancelLabel(res.ShipmentID)
	if err != nil {
		t.Error(err)
	}

	t.Log(ok)
}

func TestTrackInfo(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	res, err := carrier.TrackInfo("SHIPPO_DELIVERED")
	if err != nil {
		t.Error(err)
	}

	t.Log(res)
}

func TestCreateManifest(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:                16308,
		OrderNumber:       "LB00002303",
		Code:              "",
		TrackingNumber:    "LB00002303",
		FirstName:         "",
		LastName:          "",
		FullName:          "Jane Smith",
		Company:           "",
		Address1:          "1922 Lake Roberts Landing Drive",
		Address2:          "",
		City:              "WINTER GARDEN",
		State:             "FL",
		Zipcode:           "34787",
		Phone:             "0412345678",
		Email:             "",
		Country:           "US",
		ItemName:          "T-Shirt",
		MassUnit:          "g",
		Weight:            100,
		Length:            10,
		Width:             10,
		Height:            10,
		DistanceUnit:      "cm",
		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462-4014",
		WarehouseCountry:  "US",
	}

	res, errs, err := carrier.CreateLabel(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	body := ManifestRequest{
		TrackingNumbers: []string{res.ShipmentID},
		Name:            payload.WarehouseCompany,
		Line1:           payload.WarehouseAddress1,
		Phone:           payload.WarehousePhone,
		City:            payload.WarehouseCity,
		State:           payload.WarehouseState,
		Zip:             payload.WarehouseZipcode,
		Country:         payload.WarehouseCountry,
	}

	manifest, message, err := carrier.CreateManifest(body)
	if err != nil {
		t.Error(err)
	}

	if message != "" {
		t.Error(message)
	}

	t.Log(manifest)
}

func TestCheckAddress(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "215 Clayton St.",
		Address2:       "",
		City:           "San Francisco",
		State:          "CA",
		Zipcode:        "94117",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         100,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462",
		WarehouseCountry:  "US",
	}

	body := RequestCheckPackageAddress{
		Company:    payload.Company,
		Line1:      payload.Address1,
		Line2:      payload.Address2,
		City:       payload.City,
		State:      payload.State,
		PostalCode: payload.Zipcode,
		Country:    payload.Country,
	}

	res, errs, err := carrier.CheckPackageAddress(body)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	t.Log(res)
}

func TestEstimateCost(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "514 N Broadway",
		Address2:       "",
		City:           "Tecumseh",
		State:          "OK",
		Zipcode:        "74873",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         500,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "Warehouse CA",
		WarehouseAddress1: "15436 Brookhust St",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Westminter",
		WarehouseState:    "CA",
		WarehouseZipcode:  "92683",
		WarehouseCountry:  "US",
	}

	res, errs, err := carrier.EstimateCost(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(err)
	}

	t.Log(res)
}

func TestUpdateLabel(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeShippo, 0)

	payload := RequestCreateLabel{
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Jane Smith",
		Company:        "",
		Address1:       "1922 Lake Roberts Landing Drive",
		Address2:       "",
		City:           "WINTER GARDEN",
		State:          "FL",
		Zipcode:        "34787",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         100,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462-4014",
		WarehouseCountry:  "US",
	}

	res, errs, err := carrier.CreateLabel(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	payload = RequestCreateLabel{
		ShipmentID:     res.ShipmentID,
		ID:             16308,
		OrderNumber:    "LB00002303",
		Code:           "",
		TrackingNumber: "LB00002303",
		FirstName:      "",
		LastName:       "",
		FullName:       "Van Hanh",
		Company:        "",
		Address1:       "1922 Lake Roberts Landing Drive",
		Address2:       "",
		City:           "WINTER GARDEN",
		State:          "FL",
		Zipcode:        "34787",
		Phone:          "0412345678",
		Email:          "",
		Country:        "US",
		ItemName:       "T-Shirt",
		MassUnit:       "g",
		Weight:         100,
		Length:         10,
		Width:          10,
		Height:         10,
		DistanceUnit:   "cm",

		WarehouseCompany:  "ANANBAY LLC",
		WarehouseAddress1: "1625 Radcliff ave",
		WarehousePhone:    "1234567890",
		WarehouseCity:     "Bronx",
		WarehouseState:    "NY",
		WarehouseZipcode:  "10462-4014",
		WarehouseCountry:  "US",
	}

	res, errs, err = carrier.UpdateLabel(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Error(errs)
	}

	t.Log(res)
}
