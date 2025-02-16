package bgfulfillment

import (
	"fmt"
	"time"
)

const (
	BaseURL = "https://seller.bgfulfillment.com"

	// GramToOz const convert 1 gram to ounces
	GramToOz = 0.03527392

	// CentimeterToInches const convert 1 centimeter to inch
	CentimeterToInches = 0.3937007874

	// LbToGram const convert 1 pound to grams
	LbToGram = 453.59237

	// FirstClassLimit Limit of first class (oz)
	FirstClassLimit = 16
)

type (
	Options struct {
		APIKey          string
		Weight          float64 `json:"weight"`
		WeightUnit      string  `json:"weight_unit"`
		ImageFormat     string  `json:"image_format"`
		ImageResolution int     `json:"image_resolution"`
		DimensionsUnit  string  `json:"dimensions_unit"`
		Address         *USPSAddress
		Usps            *USPS
	}

	Dimensions struct {
		Width  float64 `json:"width"`
		Length float64 `json:"length"`
		Height float64 `json:"height"`
	}

	ErrorResponse struct {
		Code    string      `json:"code"`
		Message interface{} `json:"message"`
	}

	LabelRequest struct {
		OrderName    string  `json:"order_name"`
		FirstName    string  `json:"first_name"`
		MiddleName   string  `json:"middle_name"`
		LastName     string  `json:"last_name"`
		FullName     string  `json:"full_name"`
		Company      string  `json:"company"`
		Address1     string  `json:"line1"`
		Address2     string  `json:"line2"`
		City         string  `json:"city"`
		State        string  `json:"state"`
		Zipcode      string  `json:"zipcode"`
		Phone        string  `json:"phone"`
		Email        string  `json:"email"`
		Country      string  `json:"country_code"`
		ItemDetail   string  `json:"item_detail"`
		MassUnit     string  `json:"mass_unit"`
		Weight       float64 `json:"weight"`
		Length       float64 `json:"length"`
		Width        float64 `json:"width"`
		Height       float64 `json:"height"`
		DistanceUnit string  `json:"distance_unit"`

		WarehouseCompany  string `json:"warehouse_company"`
		WarehouseAddress1 string `json:"warehouse_address1"`
		WarehousePhone    string `json:"warehouse_phone"`
		WarehouseCity     string `json:"warehouse_city"`
		WarehouseState    string `json:"warehouse_state"`
		WarehouseZipcode  string `json:"warehouse_zipcode"`
		WarehouseCountry  string `json:"warehouse_country"`
	}
)

// USPS struct
type (
	USPSRequest struct {
		APIKey          string      `json:"api_key"`
		RequestID       string      `json:"request_id"`
		FromAddress     USPSAddress `json:"from_address"`
		ToAddress       USPSAddress `json:"to_address"`
		Weight          float64     `json:"weight"`
		WeightUnit      string      `json:"weight_unit"`
		ImageFormat     string      `json:"image_format"`
		ImageResolution int         `json:"image_resolution"`
		Dimensions      Dimensions  `json:"dimensions"`
		DimensionsUnit  string      `json:"dimensions_unit"`
		Usps            USPS        `json:"usps"`
	}

	USPSAddress struct {
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

	USPS struct {
		Shape             string `json:"shape"`
		MailClass         string `json:"mail_class"`
		MailClassPriority string `json:"mail_class_priority"`
		ImageSize         string `json:"image_size"`
	}

	USPSResponse struct {
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
)

// DHL struct
type (
	DhlRequest struct {
		OrderNumber string      `json:"order_number"`
		APIKey      string      `json:"api_key"`
		Shipment    DHLShipment `json:"shipment"`
	}

	DHLShipment struct {
		AddressFrom DHLAddress   `json:"address_from"`
		AddressTo   DHLAddress   `json:"address_to"`
		Parcels     []DHLParcels `json:"parcels"`
	}

	DHLParcels struct {
		Length       string `json:"length"`
		Width        string `json:"width"`
		Height       string `json:"height"`
		DistanceUnit string `json:"distance_unit"`
		Weight       string `json:"weight"`
		MassUnit     string `json:"mass_unit"`
	}

	DHLResponse struct {
		ObjectState   string    `json:"object_state"`
		Status        string    `json:"status"`
		ObjectCreated time.Time `json:"object_created"`
		ObjectUpdated time.Time `json:"object_updated"`
		ObjectID      string    `json:"object_id"`
		ObjectOwner   string    `json:"object_owner"`
		Test          bool      `json:"test"`
		Rate          struct {
			ObjectId          string `json:"object_id"`
			Amount            string `json:"amount"`
			Currency          string `json:"currency"`
			AmountLocal       string `json:"amount_local"`
			CurrencyLocal     string `json:"currency_local"`
			Provider          string `json:"provider"`
			ServicelevelName  string `json:"servicelevel_name"`
			ServicelevelToken string `json:"servicelevel_token"`
			CarrierAccount    string `json:"carrier_account"`
		} `json:"rate"`
		TrackingNumber string `json:"tracking_number"`
		TrackingStatus struct {
			ObjectCreated time.Time `json:"object_created"`
			ObjectUpdated time.Time `json:"object_updated"`
			ObjectID      string    `json:"object_id"`
			Status        string    `json:"status"`
			StatusDetails string    `json:"status_details"`
			StatusDate    *string   `json:"status_date"`
			Location      *string   `json:"location"`
		} `json:"tracking_status"`
		TrackingHistory      []interface{} `json:"tracking_history"`
		Eta                  *string       `json:"eta"`
		TrackingUrlProvider  string        `json:"tracking_url_provider"`
		LabelUrl             string        `json:"label_url"`
		CommercialInvoiceUrl string        `json:"commercial_invoice_url"`
		Messages             []struct {
			Source string `json:"source"`
			Code   string `json:"code"`
			Text   string `json:"text"`
		}
		Order    *string `json:"order"`
		Metadata string  `json:"metadata"`
		Parcel   string  `json:"parcel"`
		Billing  struct {
			Payments []interface{} `json:"payments"`
		} `json:"billing"`
		Base64Labels string `json:"base64_labels"`
	}

	DHLAddress struct {
		Name    string `json:"name,omitempty"`
		Street1 string `json:"street1,omitempty"`
		Street2 string `json:"street2,omitempty"`
		City    string `json:"city,omitempty"`
		State   string `json:"state,omitempty"`
		Zip     string `json:"zip,omitempty"`
		Country string `json:"country,omitempty"`
	}
)

func (err *ErrorResponse) Error() string {
	return fmt.Sprintf("%s: %v", err.Code, err.Message)
}
