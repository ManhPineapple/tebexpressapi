package kiloship

const gramToOz = 0.0352739619
const USPS_GROUND_ADVENTAGE = "usps_ground_advantage"
const USPS_PRIORITY = "usps_priority"

var ANANBAY_ADDRESS = KiloshipAddress{
	Name:      "ANANBAY",
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
	Name      string `json:"name"`
	Address_1 string `json:"street1"`
	Address_2 string `json:"street2,omitempty"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zipcode   string `json:"zip"`
	Country   string `json:"country"`
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
	Rate           KiloshipRate   `json:"rate"`
	Parcel         KiloshipParcel `json:"parcel"`
	Status         string         `json:"status"`
	LabelURL       string         `json:"labelUrl"`
	Messages       []string       `json:"messages"`
	Metadata       []string       `json:"metadata"`
	ObjectID       string         `json:"objectId"`
	ObjectOwner    string         `json:"objectOwner"`
	ObjectState    string         `json:"objectState"`
	ObjectCreated  string         `json:"objectCreated"`
	ObjectUpdated  string         `json:"objectUpdated"`
	TrackingNumber string         `json:"trackingNumber"`
	TrackingStatus string         `json:"trackingStatus"`
	TrackingURL    string         `json:"trackingUrlProvider"`
	LabelImageURL  string         `json:"labelImageUrl"`
	ChargeAmount   float64        `json:"chargeAmount"`
}

type KiloshipTracking struct {
	Carrier           string               `json:"carrier"`
	Service           string               `json:"service"`
	EstimatedDelivery string               `json:"estimatedDeliveryDate"`
	ActualDelivery    string               `json:"actualDeliveryDate"`
	Events            []KiloshipTrackEvent `json:"events"`
}

type KiloshipTrackEvent struct {
	Timestamp   string `json:"timestamp"`
	Status      string `json:"status"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

type KiloshipTrackingInfoResponse struct {
	Status          string           `json:"status"`
	TrackingStatus  string           `json:"trackingStatus"`
	TrackingNumber  string           `json:"trackingNumber"`
	LabelURL        string           `json:"labelUrl"`
	ChargeAmount    float64          `json:"chargeAmount"`
	CreatedAt       string           `json:"createdAt"`
	UpdatedAt       string           `json:"updatedAt"`
	TrackingDetails KiloshipTracking `json:"trackingDetails"`
}
