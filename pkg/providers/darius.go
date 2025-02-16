package providers

import (
	"log"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/providers/darius"

	"github.com/spf13/cast"
)

type DariusCarrier struct {
	Service *darius.Darius
}

func (c *DariusCarrier) GetCode() string {
	return CarrierTypeDarius
}

func (c *DariusCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	log.Println("Darius CreateLabel")
	body := darius.LabelRequest{
		Code:               in.Code,
		OrderNumber:        in.OrderNumber,
		ServiceCode:        in.FullServiceCode,
		ConsigneeMobile:    in.Phone,
		ConsigneeName:      in.FullName,
		ConsigneeAddress:   in.Address1,
		ConsigneeTelephone: in.Phone,
		Country:            in.Country,
		ConsigneeState:     in.State,
		ConsigneeCity:      in.City,
		ConsigneePostcode:  in.Zipcode,
		Weight:             cast.ToString(in.Weight / cast.ToFloat64(calculate.KgToGram)),
		VolumeHeight:       cast.ToString(in.Height),
		VolumeLength:       cast.ToString(in.Length),
		VolumeWidth:        cast.ToString(in.Width),
		VolumeWeight:       cast.ToString(in.Weight),
	}

	res, message, err := c.Service.USPSCreateLabel(body)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	if res.TrackingNumber == "" || res.OrderId == "" {
		return nil, &ErrResponse{Messages: []string{"create label error"}}, nil
	}

	label, err := c.Service.GetLabel(res.OrderId)
	if err != nil {
		return nil, nil, err
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.TrackingNumber,
		LabelUrl:       label,
	}

	return result, nil, nil
}

func (c *DariusCarrier) CreateLabel2(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	log.Println("Darius CreateLabel2")
	body := darius.LabelRequest{
		Code:               in.Code,
		OrderNumber:        in.OrderNumber,
		ServiceCode:        in.FullServiceCode,
		ConsigneeMobile:    in.Phone,
		ConsigneeName:      in.FullName,
		ConsigneeAddress:   in.Address1,
		ConsigneeTelephone: in.Phone,
		Country:            in.Country,
		ConsigneeState:     in.State,
		ConsigneeCity:      in.City,
		ConsigneePostcode:  in.Zipcode,
		Weight:             cast.ToString(in.Weight / cast.ToFloat64(calculate.KgToGram)),
		VolumeHeight:       cast.ToString(in.Height),
		VolumeLength:       cast.ToString(in.Length),
		VolumeWidth:        cast.ToString(in.Width),
		VolumeWeight:       cast.ToString(in.Weight),
	}

	res, message, err := c.Service.USPSCreateLabel(body)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	if res.TrackingNumber == "" || res.OrderId == "" {
		return nil, &ErrResponse{Messages: []string{"create label error"}}, nil
	}

	label, err := c.Service.GetLabel(res.OrderId)
	if err != nil {
		return nil, nil, err
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.TrackingNumber,
		LabelUrl:       label,
	}

	return result, nil, nil
}

func (c *DariusCarrier) CancelLabel(trackingNumber string) (bool, error) {
	log.Println("Darius CancelLabel")
	return true, nil
}

func (c *DariusCarrier) TrackInfo(trackingNumber string) ([]ResponseTrack, error) {
	log.Println("Darius TrackInfo")
	return nil, nil
}

func (c *DariusCarrier) CreateManifest(arg ManifestRequest) (*ManifestResponse, string, error) {
	log.Println("Darius CreateManifest")
	return nil, "", nil
}

func (c *DariusCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	log.Println("Darius CheckPackageAddress")
	return true, nil, nil
}

func (c *DariusCarrier) EstimateCost(in RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	log.Println("Darius EstimateCost")
	return nil, nil, nil
}

func (c *DariusCarrier) UpdateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	log.Println("Darius UpdateLabel")
	return nil, nil, nil
}
