package providers

import (
	"tebexpressapi/pkg/config"
	"testing"
)

func TestKiloshipCreates(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf_dev.toml"})

	carrier := NewCarrier(CarrierTypeKiloship, 0)

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
