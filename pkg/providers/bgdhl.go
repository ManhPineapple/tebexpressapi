package providers

import (
	"fmt"
	"tebexpressapi/pkg/providers/bgfulfillment"

	"github.com/spf13/cast"
)

type BGDHLCarrier struct {
	Service *bgfulfillment.BGFulfillment
}

func (c *BGDHLCarrier) GetCode() string {
	return CarrierTypeBGDHL
}

func (c *BGDHLCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
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

	res, err := c.Service.DHLCreateLabel(body)
	if err != nil {
		return nil, nil, err
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.TrackingNumber,
		ShippingFee:    cast.ToFloat64(res.Rate.Amount),
		LabelUrl:       res.Base64Labels,
	}
	return result, nil, nil
}

func (c *BGDHLCarrier) CreateLabel2(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, nil
}

func (c *BGDHLCarrier) CancelLabel(track string) (bool, error) {
	msg, err := c.Service.DeleteLabel(track)
	if err != nil {
		return false, err
	}

	if msg != "" {
		return false, &ErrResponse{Messages: []string{msg}}
	}

	return true, nil
}

func (c *BGDHLCarrier) TrackInfo(track string) ([]ResponseTrack, error) {
	return nil, fmt.Errorf("Carrier not support get track info")
}

func (c *BGDHLCarrier) CreateManifest(arg ManifestRequest) (*ManifestResponse, string, error) {
	return nil, "", fmt.Errorf("Carrier not support create manifest")
}

func (c *BGDHLCarrier) EstimateCost(in RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	return nil, nil, fmt.Errorf("Carrier not support estimate cost")
}

func (c *BGDHLCarrier) UpdateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, nil
}

func (c *BGDHLCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	return true, nil, nil
}
