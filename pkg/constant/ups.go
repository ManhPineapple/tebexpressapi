package constant

const (
	ResponseCodeUPSSuccess = "1"

	PackagingCodeCustomerSuppliedPackage = "02"
	PackagingCodeUPSExpressBox           = "21"
	PackagingCodeUPS25KGBox              = "24"
	PackagingCodeUPS10KGBox              = "25"

	UnitOfMeasurementCodeInches      = "IN"
	UnitOfMeasurementCodeCentimeters = "CM"
	UnitOfMeasurementCodeKilograms   = "KGS"
	UnitOfMeasurementCodeLBS         = "Pounds"

	ShipmentChargeTypeTransportation = "01"

	ServiceCodeExpress        = "07"
	ServiceCodeExpedited      = "08"
	ServiceCodeUPSStandard    = "11"
	ServiceCodeWorldWideSaver = "65"

	WebHookEventAccept    = "accepted_tracking_event_received"
	WebHookEventInTransit = "in_transit_tracking_event_received"
	WebHookEventDelivered = "delivered_tracking_event_received"
	WebHookEventReturned  = "returned_to_sender_tracking_event_received"
)
