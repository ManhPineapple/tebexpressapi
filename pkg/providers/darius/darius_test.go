package darius

import (
	"log"
	"tebexpressapi/pkg/constant"
	"testing"
)

func TestClient(t *testing.T) {
	c := NewDarius(&Options{
		BaseURL:  "http://119.91.146.95:8082",
		Username: "TEST",
		Password: "123456",
	})

	log.Println(c.CustomerId)
	log.Println(c.CustomerUserId)
}

func TestUSPSCreateLabel(t *testing.T) {
	c := NewDarius(&Options{
		BaseURL:  "http://119.91.146.95:8082",
		Username: "TEST",
		Password: "123456",
	})

	resp, _, err := c.USPSCreateLabel(LabelRequest{
		ServiceCode:        constant.ServiceUS48Code,
		ConsigneeMobile:    "123456",
		ConsigneeName:      "John Doe",
		ConsigneeAddress:   "7405 W STIRRUP AVE",
		ConsigneeTelephone: "1 314-282-9402",
		Country:            "US",
		ConsigneeState:     "ID",
		ConsigneeCity:      "BOISE",
		ConsigneePostcode:  "83709-6464",
		Weight:             "46",
		VolumeHeight:       "5",
		VolumeLength:       "11",
		VolumeWidth:        "7",
		VolumeWeight:       "46",
	})

	log.Println(err)
	log.Println(resp.Ack)
}

func TestGetLabel(t *testing.T) {
	c := NewDarius(&Options{
		BaseURL:  "http://119.91.146.95:8082",
		LabelURL: "http://119.91.146.95:8089",
		Username: "TEST",
		Password: "123456",
	})

	c.GetLabel("969792")
}
