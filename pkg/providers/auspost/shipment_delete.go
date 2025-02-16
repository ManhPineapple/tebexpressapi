package auspost

import (
	"io"
)

func (au *Auspost) DeleteShipment(shipmentID string) (*ResponseErrors, error) {
	var errors *ResponseErrors = &ResponseErrors{}
	err := au.Client.Delete("/shipping/v1/shipments/"+shipmentID, nil, errors)
	if err == io.EOF {
		return nil, nil
	}

	if len(errors.Errors) > 0 && errors.Errors[0].Name == "SHIPMENT_NOT_FOUND_ERROR" {
		return nil, nil
	}

	return errors, err
}
