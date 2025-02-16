package darius

import "fmt"

type Options struct {
	BaseURL        string
	LabelURL       string
	Username       string
	Password       string
	CustomerId     string
	CustomerUserId string
}

type AuthResponse struct {
	CustomerId     string `json:"customer_id"`
	CustomerUserId string `json:"customer_userid"`
	Ack            string `json:"ack"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"`
}

func (err *ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %v", err.Code, err.Message)
}

type Product struct {
	ExpressType      string `json:"express_type"`
	ProductId        string `json:"product_id"`
	ProductShortName string `json:"product_shortname"`
}

type LabelRequest struct {
	Code               string `json:"code"`
	OrderNumber        string `json:"order_number"`
	ServiceCode        string `json:"service_code"`
	BuyerId            string `json:"buyerid"`
	ConsigneeMobile    string `json:"consignee_mobile"`
	ConsigneeName      string `json:"consignee_name"`
	ConsigneeAddress   string `json:"consignee_address"`
	ConsigneeTelephone string `json:"consignee_telephone"`
	Country            string `json:"country"`
	ConsigneeState     string `json:"consignee_state"`
	ConsigneeCity      string `json:"consignee_city"`
	ConsigneePostcode  string `json:"consignee_postcode"`
	Weight             string `json:"weight"`
	VolumeHeight       string `json:"volume_height"`
	VolumeLength       string `json:"volume_length"`
	VolumeWidth        string `json:"volume_width"`
	VolumeWeight       string `json:"volume_weight"`
}

type OrderVolumeParam struct {
	VolumeHeight string `json:"volume_height"`
	VolumeLength string `json:"volume_length"`
	VolumeWidth  string `json:"volume_width"`
	VolumeWeight string `json:"volume_weight"`
}

type OrderInvoiceParam struct {
	InvoiceAmount string `json:"invoice_amount"`
	InvoicePcs    string `json:"invoice_pcs"`
	InvoiceTitle  string `json:"invoice_title"`
	InvoiceWeight string `json:"invoice_weight"`
	SKU           string `json:"sku"`
}
type USPSRequest struct {
	BuyerId           string               `json:"buyerid"`
	OrderPiece        string               `json:"order_piece"`
	ConsigneeMobile   string               `json:"consignee_mobile"`
	OrderReturnSign   string               `json:"order_returnsign"`
	TradeType         string               `json:"trade_type"`
	DutyType          string               `json:"duty_type"`
	ConsigneeName     string               `json:"consignee_name"`
	ConsigneeAddress  string               `json:"consignee_address"`
	Country           string               `json:"country"`
	ConsigneeState    string               `json:"consignee_state"`
	ConsigneeCity     string               `json:"consignee_city"`
	ConsigneePostcode string               `json:"consignee_postcode"`
	CustomerId        string               `json:"customer_id"`
	CustomerUserId    string               `json:"customer_userid"`
	ProductId         string               `json:"product_id"`
	Weight            string               `json:"weight"`
	CargoType         string               `json:"cargo_type"`
	OrderVolumeParam  []*OrderVolumeParam  `json:"orderVolumeParam"`
	OrderInvoiceParam []*OrderInvoiceParam `json:"orderInvoiceParam"`
	ShipperName       string               `json:"shipper_name,omitempty"`
	ShipperAddress1   string               `json:"shipper_address1,omitempty"`
	ShipperPostcode   string               `json:"shipper_postcode,omitempty"`
	ShipperCity       string               `json:"shipper_city,omitempty"`
	ShipperCountry    string               `json:"shipper_country,omitempty"`
	ShipperTelephone  string               `json:"shipper_telephone,omitempty"`
}

type USPSResponse struct {
	Ack               string `json:"ack"`
	Attr1             string `json:"attr1"`
	Attr2             string `json:"attr2"`
	ChannelCode       string `json:"channel_code"`
	IsDelay           string `json:"is_delay"`
	IsRemote          string `json:"is_remote"`
	IsResidential     string `json:"is_residential"`
	LabelUrl          string `json:"label_url"`
	Message           string `json:"message"`
	OrderId           string `json:"order_id"`
	OrderPrivateCode  string `json:"order_privatecode"`
	OrderTransferCode string `json:"order_transfercode"`
	ReferenceNumber   string `json:"reference_number"`
	ReturnAddress     string `json:"return_address"`
	TrackingNumber    string `json:"tracking_number"`
}

type CreateLabelResponse struct {
	Ack               string `json:"ack"`
	Attr1             string `json:"attr1"`
	Attr2             string `json:"attr2"`
	ChannelCode       string `json:"channel_code"`
	IsDelay           string `json:"is_delay"`
	IsRemote          string `json:"is_remote"`
	IsResidential     string `json:"is_residential"`
	LabelUrl          string `json:"label_url"`
	Message           string `json:"message"`
	OrderId           string `json:"order_id"`
	OrderPrivateCode  string `json:"order_privatecode"`
	OrderTransferCode string `json:"order_transfercode"`
	ReferenceNumber   string `json:"reference_number"`
	ReturnAddress     string `json:"return_address"`
	TrackingNumber    string `json:"tracking_number"`
}

type TrackingResponse struct {
	Status   string `json:"status"`
	Location struct {
		City    string `json:"city"`
		State   string `json:"state"`
		Zip     string `json:"zip"`
		Country string `json:"country"`
	} `json:"location"`
	Timestamp string `json:"timestamp"`
}
