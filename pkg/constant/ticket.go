package constant

const TicketStatusPending = 2
const TicketStatusProcessed = 3
const TicketStatusApplying = 1

const TicketTypeDefault = 1
const TicketTypeReship = 2
const TicketTypeRefund = 3

const TicketMessageLive = 1
const TicketMessageDelete = 2

const TicketStatusRepCustomer = 0
const TicketStatusRepAdmin = 1

const TicketMaxAttachFile = int64(5 * (1 << 20))

const TicketReasonChangeOrderInfo = 1
const TicketReasonCancelOrder = 2
const TicketReasonProductionTimeAndTracking = 3
const TicketReasonLabel = 4
const TicketReasonAfterFulfill = 5
const TicketReasonOther = 6

const TicketMaxRefundAmount = 50

var AllowTicketFileType = map[string]bool{
	"image/png":                     true,
	"image/jpeg":                    true,
	"image/jpg":                     true,
	"text/csv":                      true,
	"text/plain":                    true,
	"text/x-csv":                    true,
	"application/csv":               true,
	"application/x-csv":             true,
	"text/x-comma-separated-values": true,
	"text/comma-separated-values":   true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}
