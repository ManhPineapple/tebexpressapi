package providers

import (
	"errors"
	"fmt"
	"tebexpressapi/pkg/providers/auspost"
	"tebexpressapi/pkg/utils"
)

type AuspostCarrier struct {
	Service *auspost.Auspost
}

func (c *AuspostCarrier) GetCode() string {
	return CarrierTypeAuspost
}

func (c *AuspostCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	in.Weight = utils.Ceil(in.Weight/1000, 2)

	body := auspost.LabelRequest{
		OrderNumber:  in.OrderNumber,
		Code:         in.Code,
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
		DistanceUnit: IBDistanceUnit,
		Weight:       in.Weight,
		Width:        in.Width,
		Length:       in.Length,
		Height:       in.Height,
		PackageID:    in.ID,
		PostmarkDate: in.PostmarkDate,
		ServiceCode:  in.ServiceCode,

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}

	res, message, err := c.Service.Label(body)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	if res.TrackingNumber == "" || res.LabelUrl == "" {
		return nil, &ErrResponse{Messages: []string{"create label error"}}, nil
	}

	result := &ResponseCreateLabel{
		ShipmentID:     res.ShipmentID,
		TrackingNumber: res.TrackingNumber,
		ShippingFee:    res.TotalCost,
		LabelUrl:       res.LabelUrl,
		CarrierService: res.ProductID,
	}

	return result, nil, nil
}

func (c *AuspostCarrier) CreateLabel2(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	return c.CreateLabel(in)
}

func (c *AuspostCarrier) CancelLabel(shipmentID string) (bool, error) {
	errs, err := c.Service.DeleteShipment(shipmentID)
	if err != nil {
		return false, err
	}

	if errs != nil && len(errs.Errors) > 0 {
		messages := []string{}
		for _, v := range errs.Errors {
			messages = append(messages, v.Message)
		}

		return false, &ErrResponse{Messages: messages}
	}

	return true, nil
}

func (c *AuspostCarrier) TrackInfo(trackingNumber string) ([]ResponseTrack, error) {
	res, errs, err := c.Service.Track(trackingNumber)
	if err != nil {
		return nil, err
	}

	if errs != nil && len(errs.Errors) > 0 {
		messages := []string{}
		for _, v := range errs.Errors {
			messages = append(messages, v.Message)
		}

		return nil, &ErrResponse{Messages: messages}
	}

	var result []ResponseTrack
	if len(res.TrackingResults) < 1 {
		return result, nil
	}

	if len(res.TrackingResults[0].TrackableItems) < 1 {
		return result, nil
	}

	if len(res.TrackingResults[0].TrackableItems[0].Events) < 1 {
		return result, nil
	}

	events := res.TrackingResults[0].TrackableItems[0].Events

	count := len(events)
	for i := count - 1; i >= 0; i-- {
		v := events[i]
		result = append(result, ResponseTrack{
			Datetime:    v.Date,
			Description: v.Description,
			Location:    v.Location,
		})
	}

	if len(result) > 0 {
		nowStatus := res.TrackingResults[0].TrackableItems[0].Status
		if res.TrackingResults[0].TrackableItems[0].Status == "" {
			nowStatus = res.TrackingResults[0].Status
		}

		result[len(result)-1].Status = nowStatus
	}

	return result, nil
}

func (c *AuspostCarrier) CreateManifest(arg ManifestRequest) (*ManifestResponse, string, error) {
	shipments := []auspost.OrderShipment{}
	for _, v := range arg.TrackingNumbers {
		shipments = append(shipments, auspost.OrderShipment{ShipmentID: v})
	}

	body := auspost.FormCreateOrder{
		OrderReference: fmt.Sprintf("#%d", arg.ShipmentID),
		PaymentMethod:  "CHARGE_TO_ACCOUNT",
		OrderShipments: shipments,
	}

	manifest, errs, err := c.Service.CreateOrder(body)
	if err != nil {
		return nil, "", err
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs.Errors[0].Message, err
	}

	if manifest.Order.OrderID == "" {
		return nil, "create manifest response is missing", nil
	}

	result := &ManifestResponse{
		RequestID: manifest.Order.OrderID,
		Usps: []Usps{
			{
				ManifestNumber: manifest.Order.OrderID,
				CreatedAt:      manifest.Order.OrderCreationDate,
			},
		},
	}

	return result, "", nil
}

func (c *AuspostCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	return true, nil, nil
}

func (c *AuspostCarrier) EstimateCost(in RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	in.Weight = utils.Ceil(in.Weight/1000, 2)

	body := auspost.LabelRequest{
		OrderNumber:  in.OrderNumber,
		Code:         in.Code,
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
		DistanceUnit: IBDistanceUnit,
		Weight:       in.Weight,
		Width:        in.Width,
		Length:       in.Length,
		Height:       in.Height,
		PackageID:    in.ID,
		PostmarkDate: in.PostmarkDate,

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}

	res, errs, err := c.Service.Estimate(body)
	if err != nil {
		return nil, nil, err
	}

	if errs != nil && len(errs.Errors) > 0 {
		messages := []string{}
		for _, v := range errs.Errors {
			messages = append(messages, v.Message)
		}

		return nil, &ErrResponse{Messages: messages}, err
	}

	if len(res.Shipments) < 1 {
		return nil, nil, errors.New("response is missing")
	}

	shipmentSummary := res.Shipments[0].ShipmentSummary
	result := &ResponseEstimateCost{
		ShippingFee: shipmentSummary.ShippingCost,
		TotalCost:   shipmentSummary.TotalCost,
	}

	return result, nil, nil
}

func (c *AuspostCarrier) UpdateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	in.Weight = utils.Ceil(in.Weight/1000, 2)

	body := auspost.LabelRequest{
		ShipmentID:     in.ShipmentID,
		TrackingNumber: in.TrackingNumber,
		OrderNumber:    in.OrderNumber,
		Code:           in.Code,
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		FullName:       in.FullName,
		Address1:       in.Address1,
		Address2:       in.Address2,
		Company:        in.Company,
		City:           in.City,
		State:          in.State,
		Zipcode:        in.Zipcode,
		Country:        in.Country,
		Phone:          in.Phone,
		Email:          in.Email,
		ItemDetail:     in.ItemName,
		DistanceUnit:   IBDistanceUnit,
		Weight:         in.Weight,
		Width:          in.Width,
		Length:         in.Length,
		Height:         in.Height,
		PackageID:      in.ID,
		PostmarkDate:   in.PostmarkDate,
		ServiceCode:    in.ServiceCode,

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}

	// fmt.Println("===========", body)
	res, msg, err := c.Service.Update(body)
	if err != nil {
		return nil, nil, err
	}

	if msg != "" {
		return nil, &ErrResponse{Messages: []string{msg}}, err
	}

	result := &ResponseCreateLabel{
		ShipmentID:     res.ShipmentID,
		TrackingNumber: res.TrackingNumber,
		ShippingFee:    res.TotalCost,
		LabelUrl:       res.LabelUrl,
		CarrierService: res.ProductID,
	}

	return result, nil, nil
}
