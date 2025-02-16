package providers

import (
	"fmt"
	"strings"
	"tebexpressapi/pkg/providers/auspost"
	"tebexpressapi/pkg/providers/bgfulfillment"
	"tebexpressapi/pkg/providers/darius"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/spf13/viper"
)

const (
	// CarrierTypeBG BG USPS
	CarrierTypeBG = "BG"

	// CarrierTypeBGDHL BG DHL
	CarrierTypeBGDHL = "BGDHL"

	// CarrierTypeStamps is Stamps
	CarrierTypeStamps = "STAMPS"

	// CarrierTypeIBBlue IB BLUE
	CarrierTypeIBBlue = "IBBLUE"

	CarrierTypeIBBlueTest = "IBBLUE_TEST"

	CarrierTypeAuspost = "AUSPOST"

	CarrierTypeShippo = "SHIPPO"

	CarrierTypeDarius = "DARIUS"
)

type (
	// RequestCreateLabel a body of request create a label
	RequestCreateLabel struct {
		ShipmentID     string  `json:"shipment_id,omitempty"`
		ID             int64   `json:"id"`
		Code           string  `json:"code"`
		OrderNumber    string  `json:"order_number"`
		TrackingNumber string  `json:"tracking_number"`
		FirstName      string  `json:"first_name"`
		LastName       string  `json:"last_name"`
		FullName       string  `json:"full_name"`
		Company        string  `json:"company"`
		Address1       string  `json:"address1"`
		Address2       string  `json:"address2"`
		City           string  `json:"city"`
		State          string  `json:"state"`
		Zipcode        string  `json:"zipcode"`
		Phone          string  `json:"phone"`
		Email          string  `json:"email"`
		Country        string  `json:"country"`
		ItemName       string  `json:"item_name"`
		Weight         float64 `json:"weight"`
		Length         float64 `json:"length"`
		Width          float64 `json:"width"`
		Height         float64 `json:"height"`
		MassUnit       string  `json:"mass_unit"`
		DistanceUnit   string  `json:"distance_unit"`
		PostmarkDate   int64   `json:"postmark_date"`

		DisplayWeight          float64 `json:"-"`
		ServiceCode            string  `json:"-"`
		FullServiceCode        string  `json:"-"`
		DomesticCarrierService string  `json:"-"`
		HubStateCode           string  `json:"-"`
		LabelTemplate          string  `json:"-"`
		IsExceedPkg            bool    `json:"-"`
		Zone                   int     `json:"-"`

		WarehouseCompany  string `json:"warehouse_company"`
		WarehouseAddress1 string `json:"warehouse_address1"`
		WarehousePhone    string `json:"warehouse_phone"`
		WarehouseCity     string `json:"warehouse_city"`
		WarehouseState    string `json:"warehouse_state"`
		WarehouseZipcode  string `json:"warehouse_zipcode"`
		WarehouseCountry  string `json:"warehouse_country"`
	}

	// A struct for update label.
	// RequestUpdateLabel struct {
	// 	TrackingNumber string  `json:"tracking_number"`
	// 	PostmarkDate   int64   `json:"postmark_date"`
	// 	Weight         float64 `json:"weight"`
	// 	WeightUnit     string  `json:"weight_unit"`
	// 	Length         float64 `json:"length"`
	// 	Width          float64 `json:"width"`
	// 	Height         float64 `json:"height"`
	// 	DimensionsUnit string  `json:"dimensions_unit"`
	// }

	// ResponseCreateLabel result of request create a label
	ResponseCreateLabel struct {
		ShipmentID     string  `json:"shipment_id"`
		TrackingNumber string  `json:"tracking_number"`
		ShippingFee    float64 `json:"fee_shipping"`
		LabelUrl       string  `json:"label_url"`
		CarrierService string  `json:"carrier_service"`
		Zone           int     `json:"zone"`
	}

	// ResponseUpdateLabel struct {
	// 	ShippingFee float64 `json:"fee_shipping"`
	// 	LabelUrl    string  `json:"label_url"`
	// }

	// ResponseTrack result of request traking information
	ResponseTrack struct {
		Datetime    time.Time `json:"datetime"`
		City        string    `json:"city"`
		State       string    `json:"state"`
		Zipcode     string    `json:"zipcode"`
		Country     string    `json:"country"`
		Status      string    `json:"status"`
		Type        string    `json:"type"`
		Location    string    `json:"location"`
		Description string    `json:"description"`
	}

	ErrResponse struct {
		Messages []string
	}

	// ManifestRequest struct
	ManifestRequest struct {
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

	// ManifestResponse result of request create manifest
	ManifestResponse struct {
		RequestID string `json:"request_id"`
		Usps      []Usps `json:"usps"`
	}
	Usps struct {
		ShipmentID      string   `json:"shipment_id"`
		ManifestNumber  string   `json:"manifest_number"`
		CreatedAt       string   `json:"created_at"`
		TrackingNumbers []string `json:"tracking_numbers"`
		Base64Manifest  string   `json:"base64_manifest"`
	}

	RequestCheckPackageAddress struct {
		Company    string `json:"company_name"`
		Line1      string `json:"line1"`
		Line2      string `json:"line2"`
		Line3      string `json:"line3"`
		City       string `json:"city"`
		State      string `json:"state_province"`
		PostalCode string `json:"postal_code"`
		Country    string `json:"country_code"`
	}

	CheckPackageAddressResponse struct {
		Line1        string `json:"line1"`
		Line2        string `json:"line2"`
		LastLine     string `json:"last_line"`
		City         string `json:"city"`
		State        string `json:"state_province"`
		Zipcode      string `json:"zipcode"`
		ZipcodeAddon string `json:"zipcode_addon"`
		Country      string `json:"country_code"`
		Verify       int    `json:"verify"`
	}

	// ResponseCreateLabel result of request create a label
	ResponseEstimateCost struct {
		ShippingFee float64 `json:"shipping_fee"`
		TotalCost   float64 `json:"total_cost"`
		Zone        int     `json:"zone"`
	}

	RequestUpdateLabel struct {
		TrackingNumber string  `json:"tracking_number"`
		Weight         float64 `json:"weight"`
		Length         float64 `json:"length"`
		Width          float64 `json:"width"`
		Height         float64 `json:"height"`
		DistanceUnit   string  `json:"distance_unit"`
		PostmarkDate   int64   `json:"postmark_date"`
	}

	ResponseUpdateLabel struct {
		TrackingNumber string  `json:"tracking_number"`
		ShippingFee    float64 `json:"fee_shipping"`
		LabelUrl       string  `json:"label_url"`
	}
)

func (e *ErrResponse) Error() string {
	return strings.Join(e.Messages, ", ")
}

type Carrier interface {
	GetCode() string
	// CreateLabel create a label
	CreateLabel(RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error)
	CreateLabel2(RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error)

	// CancelLabel cancel a label
	CancelLabel(string) (bool, error)

	// TrackInfo get list track information timeline
	TrackInfo(string) ([]ResponseTrack, error)

	// CreateManifest create manifest
	CreateManifest(ManifestRequest) (*ManifestResponse, string, error)

	EstimateCost(RequestCreateLabel) (*ResponseEstimateCost, *ErrResponse, error)

	UpdateLabel(RequestCreateLabel) (*ResponseCreateLabel, *ErrResponse, error)

	CheckPackageAddress(in RequestCheckPackageAddress) (bool, *ErrResponse, error)
}

type CarrierIBBlue interface {
	// CheckPackageAddress validate package address by IBBlue
	CheckPackageAddress(RequestCheckPackageAddress) (bool, *ErrResponse, error)
}

func NewCarrier(name string, customerID int64) Carrier {
	switch name {
	case CarrierTypeBG:
		return &BGCarrier{Service: bgfulfillment.NewBG(nil)}
	case CarrierTypeBGDHL:
		return &BGDHLCarrier{Service: bgfulfillment.NewBG(nil)}
	case CarrierTypeAuspost:
		if !utils.IsCustomerBlacklist(customerID) {
			return &AuspostCarrier{Service: auspost.NewAuspost(nil)}
		}

		return &AuspostCarrier{Service: auspost.NewAuspost(&auspost.Options{
			Username:      viper.GetString("provider.auspost_test.username"),
			Password:      viper.GetString("provider.auspost_test.password"),
			AccountNumber: viper.GetString("provider.auspost_test.account_number"),
			BaseURL:       viper.GetString("provider.auspost_test.base_url"),
			Address: auspost.Address{
				Name:     viper.GetString("provider.auspost_test.name"),
				Suburb:   viper.GetString("provider.auspost_test.suburb"),
				Lines:    []string{viper.GetString("provider.auspost_test.line1")},
				State:    viper.GetString("provider.auspost_test.state_province"),
				Postcode: viper.GetString("provider.auspost_test.postcode"),
				Country:  viper.GetString("provider.auspost_test.country_code"),
				Phone:    viper.GetString("provider.auspost_test.phone"),
				Email:    viper.GetString("provider.auspost_test.email"),
			},
			ProductExpressID:     viper.GetString("provider.auspost_test.product_express_id"),
			ProductParcelPostID:  viper.GetString("provider.auspost_test.product_parcel_post_id"),
			ProductExpressName:   viper.GetString("provider.auspost_test.product_express_name"),
			ProductExpressLayout: viper.GetString("provider.auspost_test.product_express_layout"),
		})}

	case CarrierTypeIBBlue:
		if !utils.IsCustomerBlacklist(customerID) {
			return &IBBlueCarrier{Service: ibblue.NewIBBlue(nil)}
		}

		return &IBBlueCarrier{Service: ibblue.NewIBBlue(&ibblue.Options{
			BaseURL:         viper.GetString("provider.ibblue_test.base_url"),
			APIKey:          viper.GetString("provider.ibblue_test.api_key"),
			WeightUnit:      viper.GetString("provider.ibblue_test.weight_unit"),
			ImageFormat:     viper.GetString("provider.ibblue_test.image_format"),
			ImageResolution: viper.GetInt("provider.ibblue_test.image_resolution"),
			DimensionsUnit:  viper.GetString("provider.ibblue_test.dimensions_unit"),

			Address: &ibblue.USPSAddress{
				Company:    viper.GetString("provider.ibblue_test.company_name"),
				City:       viper.GetString("provider.ibblue_test.city_locality"),
				Line1:      viper.GetString("provider.ibblue_test.address_line1"),
				State:      viper.GetString("provider.ibblue_test.state_province"),
				PostalCode: viper.GetString("provider.ibblue_test.postal_code"),
				Country:    viper.GetString("provider.ibblue_test.country_code"),
				Phone:      viper.GetString("provider.ibblue_test.phone"),
			},
			Usps: &ibblue.USPS{
				Shape:                viper.GetString("provider.ibblue.shape"),
				MailClass:            viper.GetString("provider.ibblue.mail_class"),
				MailClassPriority:    viper.GetString("provider.ibblue.mail_class_priority"),
				MailClassParcel:      viper.GetString("provider.ibblue.mail_class_parcel"),
				ImageSize:            viper.GetString("provider.ibblue.image_size"),
				GdeOriginCountryCode: viper.GetString("provider.ibblue.gde_origin_country_code"),
			},
		})}
	case CarrierTypeDarius:
		return &DariusCarrier{
			Service: darius.NewDarius(nil),
		}
	default:
		fmt.Errorf(fmt.Sprintf("Don't support carrier %s", name))
	}

	return nil
}

func NewCarrierValidateAddress() CarrierIBBlue {
	return &IBBlueCarrier{Service: ibblue.NewIBBlue(nil)}
}
