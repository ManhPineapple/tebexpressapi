package auspost

import (
	"errors"
)

func (au *Auspost) Update(in LabelRequest) (*LabelResponse, string, error) {
	if in.ShipmentID == "" {
		return nil, "", errors.New("missing shipment id")
	}

	res, errs, err := au.UpdateShipment(in.ShipmentID, in)
	if err != nil {
		return nil, "", err
	}

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs.Errors[0].Message, err
	}

	if len(res.Items) < 1 {
		return nil, "", errors.New("response not found")
	}

	item := res.Items[0]
	result := &LabelResponse{
		ShipmentID:     res.ShipmentID,
		ProductID:      item.ProductID,
		TrackingNumber: item.TrackingDetails.ArticleID,
		TotalCost:      res.ShipmentSummary.TotalCost,
		ShippingCost:   res.ShipmentSummary.ShippingCost,
		TotalCostExGst: res.ShipmentSummary.TotalCostExGst,
		TotalGst:       res.ShipmentSummary.TotalGst,
	}

	payloadLabel := FormCreateLabel{
		WaitForLabelUrl:        true,
		UnlabelledArticlesOnly: false,
		Preferences: []LabelPreference{
			{
				Type:   "PRINT",
				Format: "PDF",
				Groups: []LabelGroup{
					{
						Group:      au.ProductExpressName,
						Layout:     au.ProductExpressLayout,
						Branded:    false,
						LeftOffset: 0,
						TopOffset:  0,
					},
				},
			},
		},
		Shipments: []LabelShipment{
			{
				ShipmentID: res.ShipmentID,
			},
		},
	}

	resLabel, errs, err := au.CreateLabel(payloadLabel)
	if err != nil {
		return result, "", err
	}

	if errs != nil && len(errs.Errors) > 0 {
		return result, errs.Errors[0].Message, err
	}

	if len(resLabel.Labels) < 1 {
		return result, "", errors.New("response miss label")
	}

	result.LabelUrl = resLabel.Labels[0].URL
	return result, "", nil
}
