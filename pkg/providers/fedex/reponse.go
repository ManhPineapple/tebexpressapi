package fedex

import "time"

type FedExAuthReponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type CreateFedExShipmentResponse struct {
	TransactionId         string `json:"transactionId"`
	CustomerTransactionId string `json:"customerTransactionId"`
	Output 	OutputShipment `json:"output"`
}

type FedExTrackingInfoResponse struct {
	TransactionId         string      `json:"transactionId"`
	CustomerTransactionId string      `json:"customerTransactionId"`
	Output               	struct {
		CompleteTrackResults []struct {
			TrackingNumber string      `json:"trackingNumber"`
			TrackResults [] struct{
				ScanEvents     []struct {
					Date                 time.Time `json:"date"`
					EventType            string    `json:"eventType"`
					EventDescription     string    `json:"eventDescription"`
					ExceptionCode        string    `json:"exceptionCode"`
					ExceptionDescription string    `json:"exceptionDescription"`
					ScanLocation         struct{
						StreetLines         []string `json:"street_lines"`
						City                string   `json:"city"`
						StateOrProvinceCode string   `json:"stateOrProvinceCode"`
						PostalCode          string   `json:"postalCode"`
						CountryCode         string   `json:"countryCode"`
						CountryName         string   `json:"countryName"`
						Residential         bool     `json:"residential"`
					} `json:"scanLocation"`
					LocationType      string `json:"locationType"`
					DerivedStatusCode string `json:"derivedStatusCode"`
					DerivedStatus     string `json:"derivedStatus"`
				} `json:"scanEvents"`
			} `json:"trackResults"`
		} `json:"completeTrackResults"`
	} `json:"output"`
}

type OutputShipment struct {
	TransactionShipments []Shipment `json:"transactionShipments"`
	JobId                string      `json:"job_id"`
}

type Shipment struct {
	ServiceType       string      `json:"serviceType"`
	ShipDatestamp     string      `json:"shipDatestamp"`
	ServiceCategory   string      `json:"serviceCategory"`
	ShipmentDocuments interface{} `json:"shipment_documents"`
	PieceResponses    []struct{
		MasterTrackingNumber string `json:"masterTrackingNumber"`
		DeliveryDatestamp    string `json:"deliveryDatestamp"`
		TrackingNumber       string `json:"trackingNumber"`
		PackageDocuments     []struct{
			EncodedLabel string `json:"encodedLabel"`
			ContentType  string `json:"contentType"`
			DocType      string `json:"docType"`
		} `json:"packageDocuments"`
	} `json:"pieceResponses"`
	ServiceName string      `json:"serviceName"`
}
