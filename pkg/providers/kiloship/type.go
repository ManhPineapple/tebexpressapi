package kiloship

import "time"

const gramToOz = 0.0352739619
const cmToInch = 0.393701
const USPS_GROUND_ADVANTAGE = "usps_ground_advantage"
const USPS_PRIORITY = "usps_priority"

var ANANBAY_ADDRESS = KiloshipAddress{
	Name:      "ANANBAY Inc",
	Address_1: "142 BARRINGTON LN",
	City:      "LEWISVILLE",
	State:     "TX",
	Zipcode:   "75067-9002",
	Country:   "US",
}

type KiloshipErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Title  string `json:"title"`
			Detail string `json:"detail"`
			Source struct {
				Parameter string `json:"parameter"`
			} `json:"source"`
		} `json:"errors"`
	} `json:"error"`
}

type KiloshipOptions struct {
	BaseURL string
	ApiKey  string
}

type KiloshipAddress struct {
	Name          string `json:"name,omitempty"`
	FirstName     string `json:"firstName,omitempty"` // used in create SCAN form
	LastName      string `json:"lastName,omitempty"`  // used in create SCAN form
	Address_1     string `json:"street1,omitempty"`
	Address_2     string `json:"street2,omitempty"`
	StreetAddress string `json:"streetAddress,omitempty"` // used in create SCAN form
	City          string `json:"city,omitempty"`
	State         string `json:"state,omitempty"`
	Zipcode       string `json:"zip,omitempty"`
	ScanZipcode   string `json:"ZIPCode,omitempty"`
	Country       string `json:"country,omitempty"`
}

type KiloshipParcel struct {
	Width        string `json:"width"`
	Height       string `json:"height"`
	Length       string `json:"length"`
	Weight       string `json:"weight"`
	MassUnit     string `json:"massUnit"`
	DistanceUnit string `json:"distanceUnit"`
	WeightUOM    string `json:"weightUOM"`
}

type KiloshipShipment struct {
	Async       bool             `json:"async"`
	AddressTo   KiloshipAddress  `json:"addressTo"`
	AddressFrom KiloshipAddress  `json:"addressFrom"`
	Parcels     []KiloshipParcel `json:"parcels"`
}

type KiloshipCreateLabelRequest struct {
	Shipment          KiloshipShipment `json:"shipment"`
	ServiceLevelToken string           `json:"servicelevelToken"`
	Metadata          []string         `json:"metadata"`
}

type KiloshipCreateLabelObject struct {
	PackageID int64    `json:"package_id"`
	Width     float64  `json:"width"`
	Height    float64  `json:"height"`
	Length    float64  `json:"length"`
	Weight    float64  `json:"weight"`
	ToZip     string   `json:"to_zip"`
	ToCity    string   `json:"to_city"`
	ToName    string   `json:"to_name"`
	ToState   string   `json:"to_state"`
	ToCountry string   `json:"to_country"`
	ToStreet1 string   `json:"to_street1"`
	ToStreet2 string   `json:"to_street2"`
	Metadata  []string `json:"metadata"`

	FullServiceCode   string `json:"full_service_code"`
	WarehouseCompany  string `json:"warehouse_company"`
	WarehouseAddress1 string `json:"warehouse_address1"`
	WarehousePhone    string `json:"warehouse_phone"`
	WarehouseCity     string `json:"warehouse_city"`
	WarehouseState    string `json:"warehouse_state"`
	WarehouseZipcode  string `json:"warehouse_zipcode"`
	WarehouseCountry  string `json:"warehouse_country"`
}

type KiloshipRate struct {
	Amount            string  `json:"amount"`
	Currency          string  `json:"currency"`
	ObjectID          string  `json:"objectId"`
	Provider          string  `json:"provider"`
	AmountLocal       float64 `json:"amountLocal"`
	CurrencyLocal     string  `json:"currencyLocal"`
	CarrierAccount    string  `json:"carrierAccount"`
	ServicelevelName  string  `json:"servicelevelName"`
	ServicelevelToken string  `json:"servicelevelToken"`
}

type KiloshipCreateLabelResponse struct {
	Rate   KiloshipRate `json:"rate"`
	Parcel struct {
		Weight    float64 `json:"weight"`
		WeightUOM string  `json:"weightUOM"`
	} `json:"parcel"`
	Status         string   `json:"status"`
	LabelURL       string   `json:"labelUrl"`
	Messages       []string `json:"messages"`
	Metadata       []string `json:"metadata"`
	ObjectID       string   `json:"objectId"`
	ObjectOwner    string   `json:"objectOwner"`
	ObjectState    string   `json:"objectState"`
	ObjectCreated  string   `json:"objectCreated"`
	ObjectUpdated  string   `json:"objectUpdated"`
	TrackingNumber string   `json:"trackingNumber"`
	TrackingStatus string   `json:"trackingStatus"`
	TrackingURL    string   `json:"trackingUrlProvider"`
	LabelImageURL  string   `json:"labelImageUrl"`
	ChargeAmount   float64  `json:"chargeAmount"`
}

type KiloshipTracking struct {
	Events []KiloshipTrackEvent `json:"trackingEvents"`
}

type KiloshipTrackEvent struct {
	Timestamp string `json:"GMTTimestamp"`
	EventType string `json:"eventType"`

	City    string `json:"eventCity"`
	State   string `json:"eventState"`
	Country string `json:"eventCountry"`
}

type KiloshipTrackingInfoResponse struct {
	ChargeAmount float64          `json:"chargeAmount"`
	Data         KiloshipTracking `json:"data"`
}

type KiloshipCancelLabelResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type ManifestRequest struct {
	TrackingNumbers []string   `json:"tracking_numbers"`
	ShipmentID      int64      `json:"shipment_id"`
	ShipmentDate    *time.Time `json:"shipment_date"`
	Name            string     `json:"Name"`
	Line1           string     `json:"line1"`
	Line2           string     `json:"line2"`
	City            string     `json:"city"`
	State           string     `json:"state"`
	Zip             string     `json:"zip"`
	Country         string     `json:"country"`
	Phone           string     `json:"phone"`
}

type KiloshipManifestRequest struct {
	MailingDate                  string          `json:"mailingDate" validate:"required"`
	EntryFacilityZIPCode         string          `json:"entryFacilityZIPCode" validate:"required"`
	DestinationEntryFacilityType string          `json:"destinationEntryFacilityType" validate:"required,oneof=NONE DESTINATION_NETWORK_DISTRIBUTION_CENTER DESTINATION_SECTIONAL_CENTER_FACILITY DESTINATION_DELIVERY_UNIT DESTINATION_SERVICE_HUB"`
	FromAddress                  KiloshipAddress `json:"fromAddress" validate:"required,dive"`
	Shipment                     struct {
		TrackingNumbers []string `json:"trackingNumbers"`
	} `json:"shipment"`
	OverwriteMailingDate bool `json:"overwriteMailingDate"`
}

type KiloshipManifestResponse struct {
	Form           string          `json:"form"`
	ImageType      string          `json:"imageType"`
	LabelType      string          `json:"labelType"`
	MailingDate    string          `json:"mailingDate"`
	ManifestNumber string          `json:"manifestNumber"`
	FromAddress    KiloshipAddress `json:"fromAddress"`
	Shipment       struct {
		TrackingNumbers []string `json:"trackingNumbers"`
	} `json:"shipment"`
	ScanFormImage string `json:"scanFormImage"`
}
