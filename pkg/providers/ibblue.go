package providers

import (
	"fmt"
	"log"
	"strings"
	"tebexpressapi/pkg/providers/ibblue"
	"time"
)

const IBDistanceUnit = "in"

type IBBlueCarrier struct {
	Service *ibblue.IBBlue
}

func (c *IBBlueCarrier) GetCode() string {
	return CarrierTypeIBBlue
}

func (c *IBBlueCarrier) CreateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	body := ibblue.LabelRequest{
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
	// fmt.Println("===========", body)
	res, message, err := c.Service.USPSCreateLabel2(body)
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
		CarrierService: res.Usps.MailClass,
		Zone:           res.Usps.Zone,
	}

	return result, nil, nil
}

func (c *IBBlueCarrier) CreateLabel2(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	body := ibblue.LabelRequest{
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

		DisplayWeight:          in.DisplayWeight,
		DomesticCarrierService: in.DomesticCarrierService,
		ServiceCode:            in.ServiceCode,
		HubStateCode:           in.HubStateCode,
		LabelTemplate:          in.LabelTemplate,
		IsExceedPkg:            in.IsExceedPkg,

		WarehouseCompany:  in.WarehouseCompany,
		WarehouseAddress1: in.WarehouseAddress1,
		WarehousePhone:    in.WarehousePhone,
		WarehouseCity:     in.WarehouseCity,
		WarehouseState:    in.WarehouseState,
		WarehouseZipcode:  in.WarehouseZipcode,
		WarehouseCountry:  in.WarehouseCountry,
	}
	// fmt.Println("===========", body)
	res, message, err := c.Service.USPSCreateLabel3(body)
	if err != nil {
		log.Println(err)
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
		CarrierService: res.Usps.MailClass,
		Zone:           res.Usps.Zone,
	}

	return result, nil, nil
}

func (c *IBBlueCarrier) CancelLabel(trackingNumber string) (bool, error) {
	msg, err := c.Service.DeleteLabel(trackingNumber)
	if err != nil {
		return false, err
	}

	if msg != "" {
		return false, &ErrResponse{Messages: []string{msg}}
	}

	return true, nil
}

func (c *IBBlueCarrier) TrackInfo(trackingNumber string) ([]ResponseTrack, error) {
	res, err := c.Service.TrackingInfo(trackingNumber)
	if err != nil {
		return nil, err
	}

	var result []ResponseTrack

	for _, v := range res {
		dt, _ := time.Parse("2006-01-02T15:04:05-08:00", v.Timestamp)
		if dt.Year() == 1 {
			dt, _ = time.Parse("2006-01-02T15:04:05-07:00", v.Timestamp)
		}

		result = append(result, ResponseTrack{
			Datetime: dt,
			City:     v.Location.City,
			State:    v.Location.State,
			Zipcode:  v.Location.Zip,
			Country:  v.Location.Country,
			Status:   v.Status,
		})
	}
	return result, nil
}

func (c *IBBlueCarrier) CreateManifest(arg ManifestRequest) (*ManifestResponse, string, error) {

	body := ibblue.ManifestRequest{
		TrackingNumbers: arg.TrackingNumbers,
		ShipmentID:      arg.ShipmentID,
	}

	manifest, responseError, err := c.Service.CreateManifest(body)
	if err != nil {
		return nil, "", err
	}

	if responseError != "" {
		return nil, responseError, err
	}

	if len(manifest.Usps) <= 0 {
		return nil, "Create manifest error", nil
	}

	result := &ManifestResponse{
		RequestID: manifest.RequestID,
		Usps:      []Usps{},
	}

	for _, item := range manifest.Usps {
		result.Usps = append(result.Usps, Usps{
			ManifestNumber:  item.ManifestNumber,
			CreatedAt:       item.CreatedAt,
			Base64Manifest:  item.Base64Manifest,
			TrackingNumbers: item.TrackingNumbers,
		})

	}

	return result, "", nil
}

func (c *IBBlueCarrier) CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error) {
	body := ibblue.AddressRequest{
		Company:    in.Company,
		Line1:      in.Line1,
		Line2:      in.Line2,
		Line3:      in.Line3,
		City:       in.City,
		State:      in.State,
		PostalCode: in.PostalCode,
		Country:    in.Country,
	}

	res, message, err := c.Service.ValidateAddress(body)
	if err != nil {
		return false, nil, err
	}

	if message != "" {
		return false, &ErrResponse{Messages: []string{message}}, err
	}

	if res.Line1 == "" || res.AddressExists == "N" {
		return false, &ErrResponse{Messages: []string{"Validate address error"}}, nil
	}

	if strings.ToLower(body.Line1) != strings.ToLower(res.Line1) {
		if res.AMSReturnCode != 31 && res.AMSReturnCode != 32 {
			return false, nil, nil
		}
	}

	if strings.ToLower(res.City) != strings.ToLower(in.City) || strings.ToLower(res.State) != strings.ToLower(in.State) {
		return false, nil, nil
	}

	if in.Country != "US" {
		return true, nil, nil
	}

	if !strings.Contains(in.PostalCode, "-") {
		if strings.ToLower(res.Zipcode) != strings.ToLower(in.PostalCode) {
			return false, nil, nil
		}

		return true, nil, nil
	}

	splitZIPCode := strings.Split(in.PostalCode, "-")
	if strings.ToLower(res.Zipcode) != splitZIPCode[0] {
		return false, nil, nil
	}

	if strings.ToLower(res.ZipcodeAddon) != splitZIPCode[1] {
		return false, nil, nil
	}

	return true, nil, nil
}

func (c *IBBlueCarrier) EstimateCost(in RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error) {
	postmarkDate := time.Now().Add(144 * time.Hour)

	if in.PostmarkDate > 0 {
		postmarkDate = time.Now().Add(time.Duration(in.PostmarkDate) * 24 * time.Hour)
	}

	body := ibblue.EstimateCostRequest{
		OrderNumber: in.OrderNumber,
		FromAddress: ibblue.USPSAddress{
			Company:    in.WarehouseCompany,
			Line1:      in.WarehouseAddress1,
			Phone:      in.WarehousePhone,
			City:       in.WarehouseCity,
			State:      in.WarehouseState,
			PostalCode: in.WarehouseZipcode,
			Country:    in.WarehouseCountry,
		},
		ToAddress: ibblue.USPSAddress{
			FirstName:  in.FirstName,
			LastName:   in.LastName,
			Line1:      in.Address1,
			Line2:      in.Address2,
			Company:    in.Company,
			City:       in.City,
			State:      in.State,
			PostalCode: in.Zipcode,
			Country:    in.Country,
			Phone:      in.Phone,
			Email:      in.Email,
		},
		Weight: in.Weight,
		Dimensions: ibblue.Dimensions{
			Width:  in.Width,
			Length: in.Length,
			Height: in.Height,
		},
		IsExceedPackage: in.IsExceedPkg,
		PostmarkDate:    fmt.Sprintf("%sT%s", postmarkDate.Format("2006-01-02"), postmarkDate.Format("15:04:05")),
	}

	res, message, err := c.Service.EstimateCost(body)
	log.Println(message)
	log.Println(err)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	result := &ResponseEstimateCost{
		ShippingFee: res.FeesAmount,
		TotalCost:   res.TotalAmount,
		Zone:        res.Usps.Zone,
	}

	return result, nil, nil
}

func (c *IBBlueCarrier) UpdateLabel(in RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error) {
	body := ibblue.LabelRequest{
		TrackingNumber: in.TrackingNumber,
		DistanceUnit:   IBDistanceUnit,
		Weight:         in.Weight,
		Width:          in.Width,
		Length:         in.Length,
		Height:         in.Height,
	}
	// fmt.Println("===========", body)
	res, message, err := c.Service.USPSUpdateLabel(body)
	if err != nil {
		return nil, nil, err
	}

	if message != "" {
		return nil, &ErrResponse{Messages: []string{message}}, err
	}

	if len(res.Usps.TrackingNumbers) <= 0 || len(res.Base64Labels) <= 0 {
		return nil, &ErrResponse{Messages: []string{"update label error"}}, nil
	}

	result := &ResponseCreateLabel{
		TrackingNumber: res.Usps.TrackingNumbers[0],
		ShippingFee:    res.TotalAmount,
		LabelUrl:       res.Base64Labels[0],
		CarrierService: res.Usps.MailClass,
		Zone:           res.Usps.Zone,
	}

	return result, nil, nil
}
