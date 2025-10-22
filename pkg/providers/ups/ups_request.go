package ups

type UPSRequest struct {
	ShipmentRequest ShipmentRequest `json:"ShipmentRequest"`
}

type BeeRequest struct {
	Hawbno string `json:"hawbno"`
	Token  string `json:"token"`
}

type ShipmentRequest struct {
	Shipment Shipment `json:"Shipment"`
	Request  struct {
		RequestOption string `json:"RequestOption"`
	}
}

type Shipment struct {
	Description string   `json:"Description"`
	Shipper     ShipInfo `json:"Shipper"`

	ShipTo ShipInfo `json:"ShipTo"`

	ShipFrom ShipInfo `json:"ShipFrom"`

	PaymentInformation PaymentInformation `json:"PaymentInformation"`

	Service Code `json:"Service"`

	Package []Package `json:"Package"`
}

type ShipInfo struct {
	Name                           string  `json:"Name"`
	AttentionName                  string  `json:"AttentionName"`
	ShipperTaxIdentificationNumber string  `json:"TaxIdentificationNumber"`
	Phone                          Phone   `json:"Phone"`
	ShipperNumber                  string  `json:"ShipperNumber"`
	Address                        Address `json:"Address"`
}

type Phone struct {
	Number string `json:"Number"`
}

type Address struct {
	AddressLine       []string `json:"AddressLine"`
	City              string   `json:"City"`
	StateProvinceCode string   `json:"StateProvinceCode"`
	PostalCode        string   `json:"PostalCode"`
	CountryCode       string   `json:"CountryCode"`
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

type InputRateItem struct {
	Length float64 `json:"Length"`
	Width  float64 `json:"Width"`
	Height float64 `json:"Height"`
	Weight float64 `json:"Weight"`
}

type InputAddress struct {
	Name    string `json:"Name"`
	Company string `json:"Company"`
	Phone   string `json:"Phone"`
	Address string `json:"Address"`
	City    string `json:"City"`
	State   string `json:"State"`
	Zipcode string `json:"Zipcode"`
	Country string `json:"Country"`
}

type UPSInputRateRequest struct {
	Items       []InputRateItem
	ShipAddress InputAddress
}

type UPSRateRequest struct {
	RateRequest RateRequest `json:"RateRequest"`
}

type RateRequest struct {
	RateShipment RateShipment `json:"Shipment"`
}

type UserLevelDiscountIndicator struct {
	UserLevelDiscountIndicator string `json:"UserLevelDiscountIndicator"`
}

type RateShipment struct {
	ShipmentRatingOptions *UserLevelDiscountIndicator `json:"ShipmentRatingOptions,omitempty"`
	Shipper               ShipInfo                    `json:"Shipper"`
	ShipTo                ShipInfo                    `json:"ShipTo"`
	ShipFrom              ShipInfo                    `json:"ShipFrom"`
	PaymentInformation    PaymentInformation          `json:"PaymentInformation"`
	Service               Code                        `json:"Service"`
	Package               []RatePackage               `json:"Package"`
}

type RatePackage struct {
	Packaging     Code          `json:"PackagingType"`
	Dimensions    Dimesions     `json:"Dimensions"`
	PackageWeight PackageWeight `json:"PackageWeight"`
}
