package auspost

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestEstimate(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	auspost := NewAuspost(nil)

	payload := LabelRequest{
		City:             "Sydney",
		State:            "NSW",
		Zipcode:          "2000",
		Country:          "AU",
		WarehouseCity:    "MELBOURNE",
		WarehouseState:   "VIC",
		WarehouseZipcode: "3000",
		WarehouseCountry: "AU",
		Length:           5,
		Height:           5,
		Width:            5,
		Weight:           5,
		OrderNumber:      "ANANBAY",
	}

	res, errs, err := auspost.Estimate(payload)
	if err != nil {
		t.Error(err)
	}

	if errs != nil {
		t.Errorf("response: %v", errs)
	}

	t.Log(res)
}
