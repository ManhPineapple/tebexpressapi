package yun

import "fmt"

const (
	// GramToOz const convert 1 gram to ounces
	GramToOz = 0.03527392

	// CentimeterToInches const convert 1 centimeter to inch
	CentimeterToInches = 0.3937007874

	// LbToGram const convert 1 pound to grams
	LbToGram = 453.59237

	// FirstClassLimit Limit of first class (oz)
	FirstClassLimit = 16

	TemplateTebexpress   = "Ananbay"
	TemplateWLTebexpress = "Ananbay2"
)

type Options struct {
	BaseURL        string
	UserName       string
	Password       string
	ShippingMethod string
	Sender         *Sender
}

type Sender struct {
	CountryCode string
	Company     string
	Street      string
	City        string
	State       string
	Zip         string
}

type LabelRequest struct {
	//OrderName    string  `json:"order_name"`
	OrderNumber    string  `json:"order_number"`
	PackageID      int64   `json:"package_id"`
	Code           string  `json:"code"`
	TrackingNumber string  `json:"tracking_number"`
	FirstName      string  `json:"first_name"`
	MiddleName     string  `json:"middle_name"`
	LastName       string  `json:"last_name"`
	FullName       string  `json:"full_name"`
	Company        string  `json:"company"`
	Address1       string  `json:"line1"`
	Address2       string  `json:"line2"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Zipcode        string  `json:"zipcode"`
	Phone          string  `json:"phone"`
	Email          string  `json:"email"`
	Country        string  `json:"country_code"`
	ItemDetail     string  `json:"item_detail"`
	MassUnit       string  `json:"mass_unit"`
	Weight         float64 `json:"weight"`
	Length         float64 `json:"length"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	DistanceUnit   string  `json:"distance_unit"`
	PostmarkDate   int64   `json:"postmark_date"`

	DisplayWeight          float64 `json:"-"`
	ServiceCode            string  `json:"-"`
	DomesticCarrierService string  `json:"-"`
	HubStateCode           string  `json:"-"`
	LabelTemplate          string  `json:"-"`
	IsExceedPkg            bool    `json:"-"`

	WarehouseCompany  string `json:"warehouse_company"`
	WarehouseAddress1 string `json:"warehouse_address1"`
	WarehousePhone    string `json:"warehouse_phone"`
	WarehouseCity     string `json:"warehouse_city"`
	WarehouseState    string `json:"warehouse_state"`
	WarehouseZipcode  string `json:"warehouse_zipcode"`
	WarehouseCountry  string `json:"warehouse_country"`
}

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"`
}

func (err *ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %v", err.Code, err.Message)
}

type USPSResponse struct {
	RequestID             string        `json:"request_id"`
	Weight                float64       `json:"weight"`
	PricingWeight         float64       `json:"pricing_weight"`
	PricingCubicFt        interface{}   `json:"pricing_cubic_ft"`
	WeightUnit            string        `json:"weight_unit"`
	Dimensions            interface{}   `json:"dimensions"`
	DimensionsUnit        interface{}   `json:"dimensions_unit"`
	Value                 float64       `json:"value"`
	PostmarkDate          string        `json:"postmark_date"`
	Status                string        `json:"status"`
	PostageAmount         float64       `json:"postage_amount"`
	FeesAmount            float64       `json:"fees_amount"`
	TotalAmount           float64       `json:"total_amount"`
	EstimatedDeliveryDays float64       `json:"estimated_delivery_days"`
	FromAddress           USPSAddress   `json:"from_address"`
	ToAddress             USPSAddress   `json:"to_address"`
	Metadata              []interface{} `json:"metadata"`
	Usps                  struct {
		MailClass string `json:"mail_class"`
		Fees      []struct {
			Name string  `json:"name"`
			Fee  float64 `json:"fee"`
		} `json:"fees"`
		Pricing         string      `json:"pricing"`
		Zone            int         `json:"zone"`
		CountryGroup    interface{} `json:"country_group"`
		CanadaZone      interface{} `json:"canada_zone"`
		TrackingNumbers []string    `json:"tracking_numbers"`
	} `json:"usps"`
	RefundDetail   interface{} `json:"refund_detail"`
	ManifestDetail interface{} `json:"manifest_detail"`
	User           struct {
		BillingType    string  `json:"billing_type"`
		AccountType    string  `json:"account_type"`
		AccountBalance float64 `json:"account_balance"`
	} `json:"user"`
	Base64Labels   []string      `json:"base64_labels"`
	Carrier        string        `json:"carrier"`
	Adjustments    []interface{} `json:"adjustments"`
	IsReturn       bool          `json:"is_return"`
	ShippingStatus interface{}   `json:"shipping_status"`
	CustomsForm    interface{}   `json:"customs_form"`
}

type USPSAddress struct {
	FirstName      string `json:"first_name,omitempty"`
	MiddleName     string `json:"middle_name,omitempty"`
	LastName       string `json:"last_name,omitempty"`
	Company        string `json:"company_name,omitempty"`
	Line1          string `json:"line1,omitempty"`
	Line2          string `json:"line2,omitempty"`
	Line3          string `json:"line3,omitempty"`
	City           string `json:"city,omitempty"`
	State          string `json:"state_province,omitempty"`
	PostalCode     string `json:"postal_code,omitempty"`
	Phone          string `json:"phone_number,omitempty"`
	Sms            string `json:"sms,omitempty"`
	Email          string `json:"email,omitempty"`
	Country        string `json:"country_code,omitempty"`
	IsoCountryCode string `json:"iso_country_code,omitempty"`
}

type ManifestRequest struct {
	TrackingNumbers []string `json:"tracking_numbers"`
	ShipmentID      int64    `json:"shipment_id"`
}

type CreateManifestRequest struct {
	RequestID       string      `json:"request_id"`
	ImageFormat     string      `json:"image_format"`
	ImageResolution int         `json:"image_resolution"`
	Usps            UspsRequest `json:"usps"`
}

type UspsRequest struct {
	TrackingNumbers []string `json:"tracking_numbers"`
}

type AddressRequest struct {
	Company    string `json:"company_name"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2"`
	Line3      string `json:"line3"`
	City       string `json:"city"`
	State      string `json:"state_province"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country_code"`
}

type CleanseAddressResponse struct {
	Line1            string `json:"line1"`
	Line2            string `json:"line2"`
	Line3            string `json:"line3,omitempty"`
	LastLine         string `json:"last_line"`
	City             string `json:"city"`
	State            string `json:"state_province"`
	Zipcode          string `json:"zip5"`
	ZipcodeAddon     string `json:"zip4"`
	DPBC             string `json:"dpbc"`
	RecordType       string `json:"record_type"`
	RDI              string `json:"rdi"`
	DPV              string `json:"dpv"`
	DPVCommnent      string `json:"dpv_comment"`
	CarrierRoute     string `json:"carrier_route"`
	AddressExists    string `json:"address_exists"`
	AMSReturnCode    int    `json:"ams_return_code"`
	RawAMSReturnCode int    `json:"raw_ams_return_code"`
	AMSVersion       string `json:"ams_version"`
	AMSDate          string `json:"ams_date"`
}

type EstimateCostRequest struct {
	RequestID       string      `json:"request_id"`
	OrderNumber     string      `json:"order_number"`
	FromAddress     USPSAddress `json:"from_address"`
	ToAddress       USPSAddress `json:"to_address"`
	PostmarkDate    string      `json:"postmark_date"`
	Weight          float64     `json:"weight"`
	WeightUnit      string      `json:"weight_unit"`
	ImageFormat     string      `json:"image_format"`
	Dimensions      Dimensions  `json:"dimensions"`
	DimensionsUnit  string      `json:"dimensions_unit"`
	ImageResolution int         `json:"image_resolution"`
	Usps            USPS        `json:"usps"`
	IsExceedPackage bool        `json:"-"`
}

type Dimensions struct {
	Width  float64 `json:"width"`
	Length float64 `json:"length"`
	Height float64 `json:"height"`
}

type EstimateCostResponse struct {
	RequestID             string      `json:"request_id"`
	FromAddress           USPSRequest `json:"from_address"`
	ToAddress             USPSRequest `json:"to_address"`
	Weight                float64     `json:"weight"`
	WeightUnit            string      `json:"weight_unit"`
	PricingWeight         float64     `json:"pricing_weight"`
	PricingCubicFt        interface{} `json:"pricing_cubic_ft"`
	Dimensions            interface{} `json:"dimensions"`
	DimensionsUnit        string      `json:"dimensions_unit"`
	Value                 float64     `json:"value"`
	PostageAmount         float64     `json:"postage_amount"`
	FeesAmount            float64     `json:"fees_amount"`
	TotalAmount           float64     `json:"total_amount"`
	EstimatedDeliveryDays int         `json:"estimated_delivery_days"`
	Usps                  struct {
		MailClass string `json:"mail_class"`
		Pricing   string `json:"pricing"`
		Fees      []struct {
			Name string  `json:"name"`
			Fee  float64 `json:"fee"`
		} `json:"fees"`
		Zone int `json:"zone"`
	} `json:"usps"`
	User struct {
		AccountBalance float64 `json:"account_balance"`
		BillingType    string  `json:"billing_type"`
		AccountType    string  `json:"account_type"`
	} `json:"user"`
}

type USPSRequest struct {
	CustomerOrderNumber string    `json:"CustomerOrderNumber"`
	ShippingMethodCode  string    `json:"ShippingMethodCode"`
	PackageCount        int       `json:"PackageCount"`
	Weight              float64   `json:"Weight"`
	Parcels             []*Parcel `json:"Parcels"`
	Receiver            *Receiver `json:"Receiver"`
}

type Parcel struct {
	EName        string  `json:"EName"`
	CName        string  `json:"CName"`
	Quantity     int     `json:"Quantity"`
	UnitPrice    float64 `json:"UnitPrice"`
	UnitWeight   float64 `json:"UnitWeight"`
	SKU          string  `json:"SKU"`
	CurrencyCode string  `json:"CurrencyCode"`
}

type Receiver struct {
	CountryCode string `json:"CountryCode"`
	FirstName   string `json:"FirstName"`
	LastName    string `json:"LastName"`
	Street      string `json:"Street"`
	City        string `json:"City"`
	State       string `json:"State"`
	Zip         string `json:"Zip"`
	Phone       string `json:"Phone"`
}
type USPS struct {
	Shape                string `json:"shape"`
	MailClass            string `json:"mail_class"`
	MailClassPriority    string `json:"mail_class_priority"`
	MailClassParcel      string `json:"mail_class_parcel"`
	ImageSize            string `json:"image_size"`
	GdeOriginCountryCode string `json:"gde_origin_country_code"`
}

type Metadata struct {
	Rubberstamp1 string `json:"rubberstamp1"`
	Rubberstamp2 string `json:"rubberstamp2"`
	Rubberstamp3 string `json:"rubberstamp3"`
	Rubberstamp4 string `json:"rubberstamp4"`
	Reference1   string `json:"reference1"`
	Reference2   string `json:"reference2"`
	Reference3   string `json:"reference3"`
}

type CreateManifestResponse struct {
	RequestID string      `json:"request_id"`
	Usps      []Usps      `json:"usps"`
	Dhl       interface{} `json:"dhl"`
	Ib        interface{} `json:"ib"`
}

type Usps struct {
	Auto            bool     `json:"auto"`
	ManifestNumber  string   `json:"manifest_number"`
	CreatedAt       string   `json:"created_at"`
	FacilityType    string   `json:"facility_type"`
	FacilityZipCode string   `json:"facility_zip_code"`
	PriorityCount   int      `json:"priority_count"`
	ExpressCount    int      `json:"express_count"`
	PmiCount        int      `json:"pmi_count"`
	EmiCount        int      `json:"emi_count"`
	GxgCount        int      `json:"gxg_count"`
	OtherCount      int      `json:"other_count"`
	TrackingNumbers []string `json:"tracking_numbers"`
	Base64Manifest  string   `json:"base64_manifest"`
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
