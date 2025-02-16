package usps

import "time"

type Address struct {
	Name                        string `json:"name"`
	Phone                       string `json:"phone"`
	CompanyName                 string `json:"company_name"`
	AddressLine1                string `json:"address_line1"`
	AddressLine2                string `json:"address_line2"`
	AddressLine3                string `json:"address_line3"`
	CityLocality                string `json:"city_locality"`
	StateProvince               string `json:"state_province"`
	PostalCode                  string `json:"postal_code"`
	CountryCode                 string `json:"country_code"`
	AddressResidentialIndicator string `json:"address_residential_indicator"`
}

type RequestBody struct {
	Shipment Shipment `json:"shipment"`
}

type Dimensions struct {
	Unit   string  `json:"unit"`
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Shipment struct {
	ServiceCode     string            `json:"service_code,omitempty"`
	ValidateAddress string            `json:"validate_address,omitempty"`
	ShipTo          Address           `json:"ship_to"`
	ShipFrom        Address           `json:"ship_from"`
	PackagesRequest []PackagesRequest `json:"packages"`
}

type ResponseBody struct {
	LabelID         string         `json:"label_id"`
	Status          string         `json:"status"`
	ShipmentID      string         `json:"shipment_id"`
	ShipDate        time.Time      `json:"ship_date"`
	CreatedAt       time.Time      `json:"created_at"`
	ShipmentCost    CurrencyAmount `json:"shipment_cost"`
	InsuranceCost   CurrencyAmount `json:"insurance_cost"`
	TrackingNumber  string         `json:"tracking_number"`
	IsReturnLabel   bool           `json:"is_return_label"`
	RmaNumber       interface{}    `json:"rma_number"`
	IsInternational bool           `json:"is_international"`
	BatchID         string         `json:"batch_id"`
	CarrierID       string         `json:"carrier_id"`
	ServiceCode     string         `json:"service_code"`
	PackageCode     string         `json:"package_code"`
	Voided          bool           `json:"voided"`
	VoidedAt        interface{}    `json:"voided_at"`
	LabelFormat     string         `json:"label_format"`
	LabelLayout     string         `json:"label_layout"`
	Trackable       bool           `json:"trackable"`
	LabelImageID    interface{}    `json:"label_image_id"`
	CarrierCode     string         `json:"carrier_code"`
	TrackingStatus  string         `json:"tracking_status"`
	LabelDownload   LabelDownload  `json:"label_download"`
	FormDownload    interface{}    `json:"form_download"`
	InsuranceClaim  interface{}    `json:"insurance_claim"`
	Packages        []Packages     `json:"packages"`
	ChargeEvent     string         `json:"charge_event"`
}

type CurrencyAmount struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type LabelDownload struct {
	Pdf  string `json:"pdf"`
	Png  string `json:"png"`
	Zpl  string `json:"zpl"`
	Href string `json:"href"`
}

type Weight struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type LabelMessages struct {
	Reference1 interface{} `json:"reference1"`
	Reference2 interface{} `json:"reference2"`
	Reference3 interface{} `json:"reference3"`
}

type PackagesRequest struct {
	Weight     Weight     `json:"weight"`
	Dimensions Dimensions `json:"dimensions"`
}

type Packages struct {
	Weight            Weight         `json:"weight"`
	Dimensions        Dimensions     `json:"dimensions"`
	PackageCode       string         `json:"package_code"`
	InsuredValue      CurrencyAmount `json:"insured_value"`
	TrackingNumber    string         `json:"tracking_number,omitempty"`
	LabelMessages     LabelMessages  `json:"label_messages,omitempty"`
	ExternalPackageID interface{}    `json:"external_package_id,omitempty"`
}

type ErrorResponse struct {
	RequestID string   `json:"request_id"`
	Errors    []Errors `json:"errors"`
}

type Errors struct {
	ErrorSource string `json:"error_source"`
	ErrorType   string `json:"error_type"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
}

type PackageResource struct {
	CompanyName                 string
	Name                        string
	Phone                       string
	AddressLine1                string
	AddressLine2                string
	CityLocality                string
	StateProvince               string
	PostalCode                  string
	CountryCode                 string
	AddressResidentialIndicator string

	Weight float64
	Height float64
	Length float64
	Width  float64

	ServiceCode string
}

type ResultResource struct {
	TrackingNumber string
	TrackingStatus string
	LabelURL       string
	ShipmentCost   float64
	InsuranceCost  float64

	ErrorMessage string
}

type TrackingResult struct {
	TrackingNumber           string          `json:"tracking_number"`
	StatusCode               string          `json:"status_code"`
	StatusDescription        string          `json:"status_description"`
	CarrierStatusCode        string          `json:"carrier_status_code"`
	CarrierStatusDescription string          `json:"carrier_status_description"`
	ShipDate                 string          `json:"ship_date"`
	EstimatedDeliveryDate    string          `json:"estimated_delivery_date"`
	ActualDeliveryDate       string          `json:"actual_delivery_date"`
	ExceptionDescription     string          `json:"exception_description"`
	Events                   []EventShipping `json:"events"`
}
type EventShipping struct {
	OccurredAt        time.Time `json:"occurred_at"`
	CarrierOccurredAt time.Time `json:"carrier_occurred_at"`
	Description       string    `json:"description"`
	CityLocality      string    `json:"city_locality"`
	StateProvince     string    `json:"state_province"`
	PostalCode        string    `json:"postal_code"`
	CountryCode       string    `json:"country_code"`
	CompanyName       string    `json:"company_name"`
	Signer            string    `json:"signer"`
	EventCode         string    `json:"event_code"`
}

type CarrierService struct {
	CarrierID               string `json:"carrier_id"`
	CarrierCode             string `json:"carrier_code"`
	ServiceCode             string `json:"service_code"`
	Name                    string `json:"name"`
	Domestic                bool   `json:"domestic"`
	International           bool   `json:"international"`
	IsMultiPackageSupported bool   `json:"is_multi_package_supported"`
}

type CarrierPackage struct {
	PackageID   string `json:"package_id"`
	PackageCode string `json:"package_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CarrierOption struct {
	Name         string `json:"name"`
	DefaultValue string `json:"default_value"`
	Description  string `json:"description"`
}

type Carrier struct {
	CarrierID                         string           `json:"carrier_id"`
	CarrierCode                       string           `json:"carrier_code"`
	AccountNumber                     string           `json:"account_number"`
	RequiresFundedAmount              bool             `json:"requires_funded_amount"`
	Balance                           float64          `json:"balance"`
	Nickname                          string           `json:"nickname"`
	FriendlyName                      string           `json:"friendly_name"`
	Primary                           bool             `json:"primary"`
	HasMultiPackageSupportingServices bool             `json:"has_multi_package_supporting_services"`
	SupportsLabelMessages             bool             `json:"supports_label_messages"`
	Services                          []CarrierService `json:"services"`
	Packages                          []CarrierPackage `json:"packages"`
	Options                           []CarrierOption  `json:"options"`
}

type ServicesResponse struct {
	Services []CarrierService `json:"services"`
}

type Rate struct {
	RateID                string         `json:"rate_id"`
	RateType              string         `json:"rate_type"`
	CarrierID             string         `json:"carrier_id"`
	ShippingAmount        CurrencyAmount `json:"shipping_amount"`
	InsuranceAmount       CurrencyAmount `json:"insurance_amount"`
	ConfirmationAmount    CurrencyAmount `json:"confirmation_amount"`
	OtherAmount           CurrencyAmount `json:"other_amount"`
	Zone                  int            `json:"zone"`
	PackageType           string         `json:"package_type"`
	DeliveryDays          int            `json:"delivery_days"`
	GuaranteedService     bool           `json:"guaranteed_service"`
	EstimatedDeliveryDate time.Time      `json:"estimated_delivery_date"`
	CarrierDeliveryDays   string         `json:"carrier_delivery_days"`
	ShipDate              time.Time      `json:"ship_date"`
	NegotiatedRate        bool           `json:"negotiated_rate"`
	ServiceType           string         `json:"service_type"`
	ServiceCode           string         `json:"service_code"`
	Trackable             bool           `json:"trackable"`
	CarrierCode           string         `json:"carrier_code"`
	CarrierNickname       string         `json:"carrier_nickname"`
	CarrierFriendlyName   string         `json:"carrier_friendly_name"`
	ValidationStatus      string         `json:"validation_status"`
	WarningMessages       []interface{}  `json:"warning_messages"`
	ErrorMessages         []interface{}  `json:"error_messages"`
}

type RateError struct {
	ErrorSource string `json:"error_source"`
	ErrorType   string `json:"error_type"`
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
	carrierID   string `json:"carrier_id"`
	CarrierCode string `json:"carrier_code"`
	CarrierName string `json:"carrier_name"`
}

type RateResponse struct {
	Rates         []Rate        `json:"rates"`
	InvalidRates  []interface{} `json:"invalid_rates"`
	RateRequestID string        `json:"rate_request_id"`
	ShipmentID    string        `json:"shipment_id"`
	CreatedAt     time.Time     `json:"created_at"`
	Status        string        `json:"status"`
	Errors        []RateError   `json:"errors"`
}

type AdvancedOptions struct {
	BillToAccount     interface{} `json:"bill_to_account"`
	BillToCountryCode interface{} `json:"bill_to_country_code"`
	BillToParty       interface{} `json:"bill_to_party"`
	BillToPostalCode  interface{} `json:"bill_to_postal_code"`
	ContainsAlcohol   bool        `json:"contains_alcohol"`
	DeliveredDutyPaid bool        `json:"delivered_duty_paid"`
	NonMachinable     bool        `json:"non_machinable"`
	SaturdayDelivery  bool        `json:"saturday_delivery"`
	DryIce            bool        `json:"dry_ice"`
	DryIceWeight      interface{} `json:"dry_ice_weight"`
	FreightClass      interface{} `json:"freight_class"`
	CustomField1      interface{} `json:"custom_field1"`
	CustomField2      interface{} `json:"custom_field2"`
	CustomField3      interface{} `json:"custom_field3"`
	CollectOnDelivery interface{} `json:"collect_on_delivery"`
}

type RateResult struct {
	RateResponse       RateResponse    `json:"rate_response"`
	ShipmentID         string          `json:"shipment_id"`
	CarrierID          string          `json:"carrier_id"`
	ServiceCode        string          `json:"service_code"`
	ExternalShipmentID string          `json:"external_shipment_id"`
	ShipDate           time.Time       `json:"ship_date"`
	CreatedAt          time.Time       `json:"created_at"`
	ModifiedAt         time.Time       `json:"modified_at"`
	ShipmentStatus     string          `json:"shipment_status"`
	ShipTo             Address         `json:"ship_to"`
	ShipFrom           Address         `json:"ship_from"`
	WarehouseID        string          `json:"warehouse_id"`
	ReturnTo           Address         `json:"return_to"`
	Confirmation       string          `json:"confirmation"`
	Customs            string          `json:"customs"`
	ExternalOrderID    string          `json:"external_order_id"`
	OrderSourceCode    string          `json:"order_source_code"`
	AdvancedOptions    AdvancedOptions `json:"advanced_options"`
	InsuranceProvider  string          `json:"insurance_provider"`
	Tags               []interface{}   `json:"tags"`
	Packages           []Packages      `json:"packages"`
	TotalWeight        Weight          `json:"total_weight"`
	Items              []interface{}   `json:"items"`
}

type RateOptions struct {
	CarrierIds   []string `json:"carrier_ids,omitempty"`
	ServiceCodes []string `json:"service_codes,omitempty"`
	PackageTypes []string `json:"package_types,omitempty"`
}

type RateRequest struct {
	RateOptions RateOptions `json:"rate_options"`
	Shipment    Shipment    `json:"shipment"`
}

type RateForm struct {
	ValidateAddress             string `json:"validate_address"`
	Name                        string `json:"name"`
	Phone                       string `json:"phone"`
	CompanyName                 string `json:"company_name"`
	AddressLine1                string `json:"address_line1"`
	AddressLine2                string `json:"address_line2"`
	CityLocality                string `json:"city_locality"`
	StateProvince               string `json:"state_province"`
	PostalCode                  string `json:"postal_code"`
	CountryCode                 string `json:"country_code"`
	AddressResidentialIndicator string `json:"address_residential_indicator"`

	ServiceCodes []string `json:"service_codes,omitempty"`
	PackageTypes []string `json:"package_types,omitempty"`

	Weight float64 `json:"weight"`
	Height float64 `json:"height"`
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
}
