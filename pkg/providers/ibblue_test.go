package providers

import (
	"bytes"
	"encoding/base64"
	"image/png"
	"log"
	"os"
	"tebexpressapi/pkg/config"
	"testing"
)

func TestIBBlue(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeIBBlue, 0)

	res, errAudit, err := carrier.CreateLabel(RequestCreateLabel{
		ID:        16308,
		FirstName: "Joe",
		LastName:  "Biden",
		FullName:  "Joe Biden",
		Code:      "LB00002303",
		Address1:  "86 DUST COMMANDER DR",
		City:      "HARRISON",
		State:     "OH",
		Zipcode:   "45030-8951",
		Country:   "US",
		ItemName:  "Chi tiet hang hoa",
		Weight:    116,
		Width:     5,
		Length:    10,
		Height:    1,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	t.Log(res.TrackingNumber)
	//t.Log(res.LabelUrl)

	rc, err := carrier.CancelLabel(res.TrackingNumber)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(rc)
}

func TestIBBlue2(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeIBBlue, 0)

	res, errAudit, err := carrier.CreateLabel2(RequestCreateLabel{
		ID:           16308,
		FirstName:    "Joe",
		LastName:     "Biden",
		FullName:     "Joe Biden",
		Code:         "LB00002303",
		Address1:     "86 DUST COMMANDER DR",
		City:         "HARRISON",
		State:        "OH",
		Zipcode:      "45030-8951",
		Country:      "US",
		ItemName:     "Chi tiet hang hoa",
		Weight:       116,
		Width:        5,
		Length:       10,
		Height:       1,
		ServiceCode:  "P",
		HubStateCode: "TX",
	})
	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	buf, err := base64.StdEncoding.DecodeString(res.LabelUrl)
	if err != nil {
		t.Fatal(err)
		return
	}

	r := bytes.NewReader(buf)

	im, err := png.Decode(r)
	if err != nil {
		t.Fatal(err)
		return
	}

	fb, err := os.Create("label.jpg")
	if err != nil {
		log.Fatal(err)
	}

	defer fb.Close()
	err = png.Encode(fb, im)
	if err != nil {
		log.Fatal(err)
	}
}

func TestIBBlueEstimateCost(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeIBBlue, 0)

	res, errAudit, err := carrier.EstimateCost(RequestCreateLabel{
		ID:        16308,
		FirstName: "Joe",
		LastName:  "Biden",
		FullName:  "Joe Biden",
		Code:      "LB00002303",
		Address1:  "86 DUST COMMANDER DR",
		City:      "HARRISON",
		State:     "OH",
		Zipcode:   "45030-8951",
		Country:   "US",
		ItemName:  "Chi tiet hang hoa",
		Weight:    116,
		Width:     5,
		Length:    10,
		Height:    1,
	})

	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	t.Log(res.ShippingFee, res.TotalCost)
}

func TestIBBlueUpdate(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})

	carrier := NewCarrier(CarrierTypeIBBlue, 0)

	res, errAudit, err := carrier.CreateLabel2(RequestCreateLabel{
		ID:           16308,
		FirstName:    "Joe",
		LastName:     "Biden",
		FullName:     "Joe Biden",
		Code:         "LB00002303",
		Address1:     "86 DUST COMMANDER DR",
		City:         "HARRISON",
		State:        "OH",
		Zipcode:      "45030-8951",
		Country:      "US",
		ItemName:     "Chi tiet hang hoa",
		Weight:       116,
		Width:        5,
		Length:       10,
		Height:       1,
		ServiceCode:  "P",
		HubStateCode: "TX",
	})
	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	t.Logf("tracking number: %v", res.TrackingNumber)

	ures, errAudit, err := carrier.UpdateLabel(RequestCreateLabel{
		TrackingNumber: res.TrackingNumber,
		Weight:         120,
		Width:          6,
		Length:         11,
		Height:         2,
	})
	if err != nil {
		t.Error(err)
		return
	}

	if errAudit != nil {
		t.Error(errAudit)
		return
	}

	t.Log(ures)
}
