package auspost

import (
	"encoding/json"
	"time"
)

type (
	OrderShipment struct {
		ShipmentID string `json:"shipment_id"`
	}

	FormCreateOrder struct {
		OrderReference string          `json:"order_reference"`
		PaymentMethod  string          `json:"payment_method"`
		OrderShipments []OrderShipment `json:"shipments"`
	}

	CreateOrderResponse struct {
		Order struct {
			OrderID           string `json:"order_id"`
			OrderReference    string `json:"order_reference"`
			OrderCreationDate string `json:"order_creation_date"`
			PaymentMethod     string `json:"payment_method"`

			OrderSummary struct {
				TotalCost              float64                `json:"total_cost"`
				TotalCostExGst         float64                `json:"total_cost_ex_gst"`
				TotalGst               float64                `json:"total_gst"`
				TotalWeight            float64                `json:"total_weight"`
				Status                 string                 `json:"status"`
				NumberOfShipments      int                    `json:"number_of_shipments"`
				NumberOfItems          int                    `json:"number_of_items"`
				DangerousGoodsIncluded bool                   `json:"dangerous_goods_included"`
				ShippingMethods        map[string]int         `json:"shipping_methods"`
				TrackingSummary        map[string]interface{} `json:"tracking_summary"`
			} `json:"order_summary"`

			Shipments []struct {
				ShipmentID           string    `json:"shipment_id"`
				ShipmentReference    string    `json:"shipment_reference"`
				EmailTrackingEnabled bool      `json:"email_tracking_enabled"`
				MovementType         string    `json:"movement_type"`
				ChargeToAccount      string    `json:"charge_to_account"`
				ShipmentCreationDate time.Time `json:"shipment_creation_date"`
				ShipmentModifiedDate time.Time `json:"shipment_modified_date"`

				Items []struct {
					ProductID            string  `json:"product_id"`
					ItemID               string  `json:"item_id"`
					ItemReference        string  `json:"item_reference"`
					Weight               float64 `json:"weight"`
					AuthorityToLeave     bool    `json:"authority_to_leave"`
					SafeDropEnabled      bool    `json:"safe_drop_enabled"`
					AllowPartialDelivery bool    `json:"allow_partial_delivery"`

					TrackingDetails struct {
						ArticleID     string `json:"article_id"`
						ConsignmentID string `json:"consignment_id"`
						BarcodeID     string `json:"barcode_id"`
					} `json:"tracking_details"`

					ItemSummary struct {
						TotalCost      float64 `json:"total_cost"`
						TotalCostExGst float64 `json:"total_cost_ex_gst"`
						TotalGst       float64 `json:"total_gst"`
						Status         string  `json:"status"`
					} `json:"item_summary"`
				} `json:"items"`

				ShipmentSummary struct {
					TotalCost      float64 `json:"total_cost"`
					TotalCostExGst float64 `json:"total_cost_ex_gst"`
					TotalGst       float64 `json:"total_gst"`
					FuelSurcharge  float64 `json:"fuel_surcharge"`
					Status         string  `json:"status"`

					TrackingSummary map[string]int `json:"tracking_summary"`
					NumberOfItems   int            `json:"number_of_items"`
				} `json:"shipment_summary"`
			} `json:"shipments"`
		} `json:"order"`
	}
)

func (au *Auspost) CreateOrder(in FormCreateOrder) (*CreateOrderResponse, *ResponseErrors, error) {
	var res interface{}
	if err := au.Client.Put("/shipping/v1/orders", in, &res); err != nil {
		return nil, nil, err
	}

	result := &CreateOrderResponse{}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(b, result)
	if err != nil {
		return nil, nil, err
	}

	if result.Order.OrderID != "" {
		return result, nil, nil
	}

	errors := &ResponseErrors{}
	err = json.Unmarshal(b, errors)
	if err != nil {
		return nil, nil, err
	}

	return nil, errors, nil
}
