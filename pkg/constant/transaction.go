package constant

const TransactionLogTypeTopup = 1
const TransactionLogTypePay = 2
const TransactionLogTypeRefund = 4
const TransactionLogTypePayoneer = 5
const TransactionLogTypePingPong = 6
const TransactionLogTypeAffliate = 7

const TransactionStatusProcess = 1
const TransactionStatusSuccess = 2
const TransactionStatusFailure = 3
const TransactionStatusRefund = 4
const TransactionStatusDraft = 5

const NotifyTransactionText = "Nạp tiền"

var MapTextActionTransaction = map[int64]string{
	TransactionStatusSuccess: "được duyệt",
	TransactionStatusFailure: "bị từ chối",
}

var MapTypeTransaction = map[int64]string{
	TransactionLogTypePayoneer: "Payoneer",
	TransactionLogTypePingPong: "PingPong",
	TransactionLogTypeTopup:    "Chuyển khoản",
}
