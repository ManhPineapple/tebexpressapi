package dto

import (
	"time"
)

type GetOrdersInContainerDTO struct {
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	ID               int64  `json:"id"`
	FullName         string `json:"full_name"`
	RefId            string `json:"ref_id"`
	Status           string `json:"status"`
	TrackingNumber   string `json:"tracking_number"`
	TrackingLabelURL string `json:"tracking_label_url"`
}

type GetListShippingOrdersDTO struct {
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	ID             int64  `json:"id"`
	FullName       string `json:"full_name"`
	RefId          string `json:"ref_id"`
	Status         string `json:"status"`
	StatusTracking string `json:"status_tracking"`
	TrackingNumber string `json:"tracking_number"`
}

type ShippingContainerHistoryDTO struct {
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ID        int64  `json:"id"`

	OrderId         int64     `json:"order_id"`
	ShippingOrderId int64     `json:"shipping_order_id"`
	TrackingNumber  string    `json:"tracking_number"`
	Location        string    `json:"location"`
	ShipTime        time.Time `json:"ship_time"`
	Status          string    `json:"status"`
}

type ShippingPackageHistory struct {
	ShippingPackageID int64     `json:"shipping_package_id"`
	Description       string    `json:"description"`
	Location          string    `json:"location"`
	ShipTime          time.Time `json:"ship_time"`
	Status            string    `json:"status"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

type UPSRequest struct {
	ShipmentRequest ShipmentRequest `json:"ShipmentRequest"`
}

type ShipmentRequest struct {
	Shipment Shipment `json:"Shipment"`
}

type Shipment struct {
	Description string   `json:"Description"`
	Shipper     ShipInfo `json:"Shipper"`

	ShipTo ShipInfo `json:"ShipTo"`

	ShipFrom ShipInfo `json:"ShipFrom"`

	PaymentInformation PaymentInformation `json:"PaymentInformation"`

	Service Code `json:"Service"`

	Package []Package `json:"Package"`

	LabelSpecification LabelSpecification `json:"LabelSpecification"`
}

type LabelSpecification struct {
	LabelImageFormat Code `json:"LabelImageFormat"`
}

type ShipInfo struct {
	Name          string  `json:"Name"`
	AttentionName string  `json:"AttentionName"`
	Phone         Phone   `json:"Phone"`
	ShipperNumber string  `json:"ShipperNumber"`
	Address       Address `json:"Address"`
}

type Phone struct {
	Number string `json:"Number"`
}

type Address struct {
	AddressLine       string `json:"AddressLine"`
	City              string `json:"City"`
	StateProvinceCode string `json:"StateProvinceCode"`
	PostalCode        string `json:"PostalCode"`
	CountryCode       string `json:"CountryCode"`
}

type PaymentInformation struct {
	ShipmentCharge `json:"ShipmentCharge"`
}

type ShipmentCharge struct {
	Type        string      `json:"Type"`
	BillShipper BillShipper `json:"BillShipper"`
}

type BillShipper struct {
	AccountNumber string `json:"AccountNumber"`
}

type Code struct {
	Code string `json:"Code"`
}

type Package struct {
	Packaging Code `json:"Packaging"`

	Dimensions Dimesions `json:"Dimensions"`

	PackageWeight PackageWeight `json:"PackageWeight"`
}

type Dimesions struct {
	UnitOfMeasurement Code   `json:"UnitOfMeasurement"`
	Length            string `json:"Length"`
	Width             string `json:"Width"`
	Height            string `json:"Height"`
}

type PackageWeight struct {
	UnitOfMeasurement Code   `json:"UnitOfMeasurement"`
	Weight            string `json:"Weight"`
}

type UPSResponse struct {
	ShipmentResponse ShipmentResponse `json:"ShipmentResponse"`
}
type ResponseStatus struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}
type Response struct {
	ResponseStatus       ResponseStatus `json:"ResponseStatus"`
	TransactionReference string         `json:"TransactionReference"`
}
type BaseServiceCharge struct {
	CurrencyCode  string `json:"CurrencyCode"`
	MonetaryValue string `json:"MonetaryValue"`
}
type TransportationCharges struct {
	CurrencyCode  string `json:"CurrencyCode"`
	MonetaryValue string `json:"MonetaryValue"`
}
type ItemizedCharges struct {
	Code          string `json:"Code"`
	CurrencyCode  string `json:"CurrencyCode"`
	MonetaryValue string `json:"MonetaryValue"`
	SubType       string `json:"SubType,omitempty"`
}
type ServiceOptionsCharges struct {
	CurrencyCode  string `json:"CurrencyCode"`
	MonetaryValue string `json:"MonetaryValue"`
}
type TotalCharges struct {
	CurrencyCode  string `json:"CurrencyCode"`
	MonetaryValue string `json:"MonetaryValue"`
}
type ShipmentCharges struct {
	BaseServiceCharge     BaseServiceCharge     `json:"BaseServiceCharge"`
	TransportationCharges TransportationCharges `json:"TransportationCharges"`
	ItemizedCharges       []ItemizedCharges     `json:"ItemizedCharges"`
	ServiceOptionsCharges ServiceOptionsCharges `json:"ServiceOptionsCharges"`
	TotalCharges          TotalCharges          `json:"TotalCharges"`
}
type UnitOfMeasurement struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}
type BillingWeight struct {
	UnitOfMeasurement UnitOfMeasurement `json:"UnitOfMeasurement"`
	Weight            string            `json:"Weight"`
}
type ImageFormat struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}
type ShippingLabel struct {
	ImageFormat  ImageFormat `json:"ImageFormat"`
	GraphicImage string      `json:"GraphicImage"`
	HTMLImage    string      `json:"HTMLImage"`
}
type PackageResults struct {
	TrackingNumber        string                `json:"TrackingNumber"`
	ServiceOptionsCharges ServiceOptionsCharges `json:"ServiceOptionsCharges"`
	ShippingLabel         ShippingLabel         `json:"ShippingLabel"`
}
type ShipmentResults struct {
	ShipmentCharges              ShipmentCharges `json:"ShipmentCharges"`
	BillingWeight                BillingWeight   `json:"BillingWeight"`
	ShipmentIdentificationNumber string          `json:"ShipmentIdentificationNumber"`
	PackageResults               PackageResults  `json:"PackageResults"`
}
type ShipmentResponse struct {
	Response        Response        `json:"Response"`
	ShipmentResults ShipmentResults `json:"ShipmentResults"`
}
