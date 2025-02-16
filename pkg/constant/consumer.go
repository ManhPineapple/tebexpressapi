package constant

const (
	// ProcessOKCode marks message as done
	ProcessOKCode int64 = 200
	// ProcessFailRetryCode marks message as fail and retry
	ProcessFailRetryCode = 500
	// ProcessFailDropCode marks message as fail and drop
	ProcessFailDropCode = 400
	// ProcessFailReproduceCode --
	ProcessFailReproduceCode = 302
)

const QueueShipmentCreateLabel = "shipment-create-label"
const QueueShipmentEstimateCost = "shipment-estimate-cost"
const QueueShipmentCreateLabelPDF = "shipment-create-labels-pdf"
const QueueShipmentRefundConsumer = "shipment-refund"
const QueueShipmentCancelCarrierConsumer = "shipment-cancel-carrier"
const QueueShipmentEstimateExceedPackage =  "shipment-estimate-exceed-package-consumer"
const QueueShipmentExportPackage = "shipment-export-package"
const QueueManifestLabel = "shipment-manifest-label"
