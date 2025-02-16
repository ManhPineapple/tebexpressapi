package providers

import (
	"fmt"
	"tebexpressapi/pkg/providers/bgfulfillment"

	"github.com/spf13/cast"
)

const DefaultDistanceUnit = "in"

type BGCarrier struct {
	Service *bgfulfillment.BGFulfillment
}

func (c *BGCarrier) GetCode() string {
	return CarrierTypeBG
}

func (c *BGCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	body := bgfulfillment.LabelRequest{
		OrderName:    cast.ToString(in.ID),
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		FullName:     in.FullName,
		Address1:     in.Address1,
		Address2:     in.Address2,
		Company:      in.Company,
		City:         in.City,
		State:        in.State,
		Zipcode:      in.Zipcode,
		Country:      in.Country,
		Phone:        in.Phone,
		Email:        in.Email,
		ItemDetail:   in.ItemName,
		DistanceUnit: DefaultDistanceUnit,
		Weight:       in.Weight,
		Width:        in.Width,
		Length:       in.Length,
		Height:       in.Height,

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}

	res, message, err := c.Service.USPSCreateLabel(body)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	if len(res.Usps.TrackingNumbers) <= 0 || len(res.Base64Labels) <= 0 {
		return nil, &ErrResponse{Messages: []string{"create label error"}}, nil
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.Usps.TrackingNumbers[0],
		ShippingFee:    res.TotalAmount,
		LabelUrl:       res.Base64Labels[0],
	}
	return result, nil, nil
}

func (c *BGCarrier) CreateLabel2(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, nil
}

func (c *BGCarrier) CancelLabel(track string) (bool, error) {
	msg, err := c.Service.DeleteLabel(track)
	if err != nil {
		return false, err
	}

	if msg != "" {
		return false, &ErrResponse{Messages: []string{msg}}
	}

	return true, nil
}

func (c *BGCarrier) TrackInfo(track string) ([]ResponseTrack, error) {
	return nil, fmt.Errorf("Carrier not support get track info")
}

func (c *BGCarrier) CreateManifest(arg ManifestRequest) (*ManifestResponse, string, error) {
	return nil, "", fmt.Errorf("Carrier not support create manifest")
}

func (c *BGCarrier) EstimateCost(in RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	return nil, nil, fmt.Errorf("Carrier not support estimate cost")
}

func (c *BGCarrier) UpdateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, nil
}

func (c *BGCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	return true, nil, nil
}
