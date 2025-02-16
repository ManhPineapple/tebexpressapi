package constant

const UploadMaxAttachFile = int64(5 * (1 << 20))

var UploadAllowFileTypes = map[string]bool{
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
	"application/vnd.ms-excel":      true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
}

var UploadAllowFileExtensions = map[string]bool{
	".jpg":  true,
	".png":  true,
	".jpeg": true,
	".csv":  true,
	".xlsx": true,
}
