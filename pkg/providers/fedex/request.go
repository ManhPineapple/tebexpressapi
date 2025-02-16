package fedex

type CreateFexExShipmentRequest struct {
	RequestedShipment    RequestShipment `json:"requestedShipment"`
	LabelResponseOptions string          `json:"labelResponseOptions"`
	AccountNumber        AccountNumber   `json:"accountNumber"`
}

type TrackingFedExShipmentRequest struct {
	TrackingInfo         []Tracking `json:"trackingInfo"`
	IncludeDetailedScans bool                 `json:"includeDetailedScans"`
}

type Tracking struct {
	TrackingNumberInfo TrackingData `json:"trackingNumberInfo"`
}

type TrackingData struct {
	TrackingNumber string `json:"trackingNumber"`
}

type RequestShipment struct {
	CustomsClearanceDetail    CustomsClearance           `json:"customsClearanceDetail"`
	Shipper                   AddressInfo                `json:"shipper"`
	Recipients                []AddressInfo              `json:"recipients"`
	PickupType                string                     `json:"pickupType"`
	ServiceType               string                     `json:"serviceType"`
	PackagingType             string                     `json:"packagingType"`
	ShippingChargesPayment    Payment                    `json:"shippingChargesPayment"`
	LabelSpecification        LabelSpecification         `json:"labelSpecification"`
	RequestedPackageLineItems []RequestedPackageLineItem `json:"requestedPackageLineItems"`
}

type CustomsClearance struct {
	DutiesPayment     Payment       `json:"dutiesPayment"`
	TotalCustomsValue DeclaredValue `json:"totalCustomsValue"`
	Commodities       []Commodity   `json:"commodities"`
}

type Commodity struct {
	Description          string              `json:"description"`
	CountryOfManufacture string              `json:"countryOfManufacture"`
	Weight               Weight              `json:"weight"`
	UnitPrice            DeclaredValue       `json:"unitPrice"`
	AdditionalMeasures   []AdditionalMeasure `json:"additionalMeasures"`
	QuantityUnits        string              `json:"quantityUnits"`
	Quantity             int                 `json:"quantity"`
	NumberOfPieces       int                 `json:"numberOfPieces"`
}

type AdditionalMeasure struct {
	Quantity int    `json:"quantity"`
	Units    string `json:"units"`
}

type DeclaredValue struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Payment struct {
	PaymentType string `json:"paymentType"`
}

type LabelSpecification struct {
	LabelStockType string `json:"labelStockType"`
	ImageType      string `json:"imageType"`
}

type AddressInfo struct {
	Address Address `json:"address"`
	Contact Contact `json:"contact"`
}

type Address struct {
	StreetLines         []string `json:"streetLines"`
	StateOrProvinceCode string   `json:"stateOrProvinceCode"`
	PostalCode          string   `json:"postalCode"`
	City                string   `json:"city"`
	CountryCode         string   `json:"countryCode"`
}

type Contact struct {
	PhoneNumber string `json:"phoneNumber"`
	PersonName  string `json:"personName"`
	CompanyName string `json:"companyName"`
}

type RequestedPackageLineItem struct {
	Weight     Weight    `json:"weight"`
	Dimensions Dimension `json:"dimensions"`
}

type Weight struct {
	Units string  `json:"units"`
	Value float64 `json:"value"`
}

type Dimension struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Units  string  `json:"units"`
}

type AccountNumber struct {
	Value string `json:"value"`
}
