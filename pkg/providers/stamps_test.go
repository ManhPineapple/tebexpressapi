package providers

import (
	"tebexpressapi/pkg/config"
	"testing"
	"time"
)

func TestStamps(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	carrier := NewCarrier(CarrierTypeStamps, 0)

	res, errAudit, err := carrier.CreateLabel(RequestCreateLabel{
		ID:        16308,
		FirstName: "Joe",
		LastName:  "Biden",
		FullName:  "Joe Biden",
		Code:      "LB00002303",
		Address1:  "1 S. 680",
		City:      "Mapleton",
		State:     "UT",
		Zipcode:   "84664",
		Country:   "US",
		ItemName:  "Chi tiet hang hoa",
		Weight:    116,
		Width:     5,
		Length:    10,
		Height:    2,
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
	time.Sleep(10 * time.Second)

	timelines, err := carrier.TrackInfo(res.TrackingNumber)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(timelines)
	time.Sleep(10 * time.Second)

	rc, err := carrier.CancelLabel(res.TrackingNumber)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(rc)
}
