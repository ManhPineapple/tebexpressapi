package ups

type UPSResponse struct {
	ShipmentResponse ShipmentResponse `json:"ShipmentResponse"`
}
type UPSResponseOnePackage struct {
	ShipmentResponse ShipmentResponseOnePackage `json:"ShipmentResponse"`
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
	ShipmentCharges              ShipmentCharges  `json:"ShipmentCharges"`
	BillingWeight                BillingWeight    `json:"BillingWeight"`
	ShipmentIdentificationNumber string           `json:"ShipmentIdentificationNumber"`
	PackageResults               []PackageResults `json:"PackageResults"`
}

type ShipmentResultsOnePackage struct {
	ShipmentCharges              ShipmentCharges `json:"ShipmentCharges"`
	BillingWeight                BillingWeight   `json:"BillingWeight"`
	ShipmentIdentificationNumber string          `json:"ShipmentIdentificationNumber"`
	PackageResults               PackageResults  `json:"PackageResults"`
}

type ShipmentResponse struct {
	Response        Response        `json:"Response"`
	ShipmentResults ShipmentResults `json:"ShipmentResults"`
}

type ShipmentResponseOnePackage struct {
	Response        Response                  `json:"Response"`
	ShipmentResults ShipmentResultsOnePackage `json:"ShipmentResults"`
}

type TrackDetailResponse struct {
	TrackResponse TrackResponse      `json:"trackResponse"`
	ErrorResponse TrackResponseError `json:"response"`
}

type BeeTrackDetailResponse struct {
	TrackInfo []struct {
		HouseBillNo    string `json:"HouseBillNo"`
		MasterBillNo   string `json:"MasterBillNo"`
		PartnerName    string `json:"PartnerName"`
		PartnerAddress string `json:"PartnerAddress"`
		TaxCode        string `json:"TaxCode"`
	} `json:"trackinfo"`

	TrackingDetails []struct {
		Status        string `json:"Status"`
		Location      string `json:"Location"`
		FlightNo      string `json:"FlightNo"`
		FlightDate    string `json:"FlightDate"`
		DepartureTime string `json:"DepartureTime"`
		ArrivalTime   string `json:"ArrivalTime"`
		Description   string `json:"Description"`
		TrackingDate  string `json:"TrackingDate"`
	} `json:"TrackingDetails"`
}
type BeeTrackDetailResponseErr struct {
	TrackInfo []struct {
		Errors    string `json:"errors"`
		MsgResult string `json:"msg_result"`
	}
	TrackingDetails []struct {
		Status        string `json:"Status"`
		Location      string `json:"Location"`
		FlightNo      string `json:"FlightNo"`
		FlightDate    string `json:"FlightDate"`
		DepartureTime string `json:"DepartureTime"`
		ArrivalTime   string `json:"ArrivalTime"`
		Description   string `json:"Description"`
		TrackingDate  string `json:"TrackingDate"`
	} `json:"TrackingDetails"`
}
type TrackResponseError struct {
	TrackError []TrackError `json:"errors"`
}

type TrackError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TrackResponse struct {
	TrackShipment []TrackShipment `json:"shipment"`
}

type TrackShipment struct {
	TrackPackage []TrackPackage `json:"package"`
}

type TrackPackage struct {
	TrackingNumber string `json:"trackingNumber"`

	Activity []Activity `json:"activity"`
}

type Activity struct {
	ActivityLocation ActivityLocation `json:"location"`
	ActivityStatus   ActivityStatus   `json:"status"`
	Date             string           `json:"date"`
	Time             string           `json:"time"`
}

type ActivityLocation struct {
	ActivityAddress ActivityAddress `json:"address"`
}

type ActivityAddress struct {
	City          string `json:"city"`
	StateProvince string `json:"stateProvince"`
	PostalCode    string `json:"postalCode"`
	Country       string `json:"country"`
}

type ActivityStatus struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Code        string `json:"code"`
}

type UPSRateResponse struct {
	RateResponse RateResponse `json:"RateResponse"`
}

type CodeItem struct {
	Code        string `json:"Code"`
	Description string `json:"Description"`
}

type RateResponse struct {
	Response struct {
		ResponseStatus       ResponseStatus `json:"ResponseStatus"`
		Alert                []CodeItem     `json:"Alert"`
		TransactionReference string         `json:"TransactionReference"`
	} `json:"Response"`

	RatedShipment struct {
		Service               CodeItem              `json:"Service"`
		RatedShipmentAlert    []CodeItem            `json:"Alert"`
		BillingWeight         BillingWeight         `json:"BillingWeight"`
		TransportationCharges TransportationCharges `json:"TransportationCharges"`
		BaseServiceCharge     BaseServiceCharge     `json:"BaseServiceCharge"`
		ServiceOptionsCharges ServiceOptionsCharges `json:"ServiceOptionsCharges"`
		TotalCharges          TotalCharges          `json:"TotalCharges"`
		ItemizedCharges       []ItemizedCharges     `json:"ItemizedCharges"`
		RatedPackage          interface{}           `json:"RatedPackage"`
	} `json:"RatedShipment"`
}
