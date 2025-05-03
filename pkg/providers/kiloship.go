package providers

import (
	"errors"
	"fmt"
	"tebexpressapi/pkg/providers/kiloship"
	"time"
)

type KiloshipCarrier struct {
	Service *kiloship.Kiloship
}

func (c *KiloshipCarrier) GetCode() string {
	return CarrierTypeKiloship
}

func (c *KiloshipCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	body := kiloship.KiloshipCreateLabelObject{
		PackageID: in.ID,
		Width:     in.Width,
		Height:    in.Height,
		Length:    in.Length,
		Weight:    in.Weight,
		ToZip:     in.Zipcode,
		ToCity:    in.City,
		ToName:    in.FullName,
		ToState:   in.State,
		ToCountry: in.Country,
		ToStreet1: in.Address1,
		ToStreet2: in.Address2,
		Metadata: []string{
			in.Code,
			fmt.Sprintf("%v", in.OrderNumber),
			fmt.Sprintf("%v", in.DisplayWeight),
		},

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}

	res, err := c.Service.CreateDomesticLabel(body)
	if err != nil {
		return nil, &ErrResponse{Messages: []string{err.Error()}}, err
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.TrackingNumber,
		ShippingFee:    res.ChargeAmount,
		LabelUrl:       res.LabelURL,
		CarrierService: res.Rate.ServicelevelName,
	}

	return result, nil, nil
}

func (c *KiloshipCarrier) TrackInfo(trackingNumber string) ([]ResponseTrack, error) {
	res, err := c.Service.TrackingLabel(trackingNumber)
	if err != nil {
		return nil, err
	}

	var result []ResponseTrack
	for _, event := range res.Data.Events {
		dt, _ := time.Parse("2006-01-02T15:04:05-08:00", event.Timestamp)
		if dt.Year() == 1 {
			dt, _ = time.Parse("2006-01-02T15:04:05-07:00", event.Timestamp)
		}

		country := event.Country
		if country == "" {
			country = "US"
		}
		result = append(result, ResponseTrack{
			Datetime:    dt,
			Status:      event.EventType,
			Description: event.EventType,
			City:        event.City,
			State:       event.State,
			Country:     country,
		})
	}
	return result, nil
}

func (c *KiloshipCarrier) CancelLabel(trackingNumber string) (bool, error) {
	return c.Service.CancelLabel(trackingNumber)
}

func (c *KiloshipCarrier) CreateLabel2(req RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, errors.New("Func wasn't be implemented")
}

func (c *KiloshipCarrier) CreateManifest(req ManifestRequest) (*ManifestResponse, string, error) {
	return nil, "", errors.New("Func wasn't be implemented")
}

func (c *KiloshipCarrier) EstimateCost(req RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	zone, cost, err := c.Service.EstimateCost(req.WarehouseZipcode, req.Zipcode)
	if err != nil {
		return nil, nil, err
	}

	return &ResponseEstimateCost{
		ShippingFee: cost,
		TotalCost:   cost,
		Zone:        zone,
	}, nil, nil
}

func (c *KiloshipCarrier) UpdateLabel(req RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return nil, nil, errors.New("Kiloship doesn't support updating labels.")
}

func (c *KiloshipCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	return false, nil, errors.New("Func wasn't be implemented")
}
