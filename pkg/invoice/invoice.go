package invoice

import (
	"fmt"
	"os"

	bevn "tebexpressapi/pkg/invoice/font/BeVn"
	bevnbold "tebexpressapi/pkg/invoice/font/BeVnBold"
	bevnitalic "tebexpressapi/pkg/invoice/font/BeeVnItalic"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/spf13/cast"
)

type Package struct {
	ID             int64     `json:"id"`
	Code           string    `json:"code"`
	TrackingNumber string    `json:"tracking_number"`
	ShippingFee    float64   `json:"shipping_fee"`
	Status         int       `json:"status"`
	ShipmentID     int64     `json:"shipment_id"`
	UpdatedAt      time.Time `json:"update_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type ExtraFee struct {
	ID          int64     `json:"id"`
	TypeName    string    `json:"type_name"`
	PackageID   int64     `json:"package_id"`
	ShipmentID  int64     `json:"shipment_id"`
	PackageCode string    `json:"package_code"`
	CouponCode  string    `json:"coupon_code"`
	BillID      *int64    `json:"bill_id"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Status      int       `json:"status"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
}

type Data struct {
	Email          string
	FullName       string
	BillCode       string
	BillDate       string
	TotalAmount    string
	ShippingFee    string
	TotalExtraFee  string
	TotalRefundFee string
	TotalComFee    string
	Packages       []Package
	ExtraFee       []ExtraFee
	RefundFee      []ExtraFee
	ComissionFee   []ExtraFee
}

func GeneratePdf(data Data, bill *entity.Bill, user *entity.User) (string, error) {
	const (
		colCount = 3
		colWd    = 60.0
		headerWd = 190
		marginH  = 15.0
		lineHt   = 5.5
		cellGap  = 2.0
		wd       = 210
	)

	var (
		err error
	)

	// _, filename, _, ok := runtime.Caller(0)
	// if !ok {
	// 	logger.Log.Errorf("Can not get path file ")
	// 	return "", nil
	// }
	// pathF := path.Dir(filename)

	pdf := gofpdf.New("P", "mm", "A4", "")

	pdf.AddUTF8FontFromBytes("beeVN", "", bevn.TTF)
	pdf.AddUTF8FontFromBytes("beeVN", "B", bevnbold.TTF)
	pdf.AddUTF8FontFromBytes("beeVN", "I", bevnitalic.TTF)

	pdf.SetHeaderFuncMode(func() {
		file, err := os.Open("./logo.png")
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer file.Close()

		pdf.RegisterImageReader("logo", "png", file)
		if pdf.Ok() {
			pdf.Image("logo", 10, 6,
				30, 0, false, "png", 0, "")
		}

		pdf.SetFont("BeeVN", "B", 28)

		// pdf.Image(pathF+"/logo.png", 10, 6, 30, 0, false, "", 0, "")
		pdf.SetY(4)
		pdf.Cell(50, 0, "")
		pdf.CellFormat(140, 10, "Receipt of Payment", "", 1, "R", false, 0, "")
		pdf.Ln(20)
	}, true)
	pdf.SetFooterFunc(func() {
		// Position at 1.5 cm from bottom
		pdf.SetY(-15)
		pdf.SetFont("BeeVN", "I", 8)
		// Text color in gray
		pdf.SetTextColor(128, 128, 128)
		// Page number
		pdf.CellFormat(0, 10, fmt.Sprintf(" %d", pdf.PageNo()),
			"", 0, "R", false, 0, "")
	})
	pdf.AddPage()
	pdf.SetFont("BeeVN", "", 14)
	pdf.SetY(20)
	pdf.Cell(50, 10, "36 Ngo 1D Tran Quang Dieu")
	pdf.CellFormat(140, 10, "Account: "+data.Email, "", 1, "R", false, 0, "")
	pdf.Cell(140, 10, "Dong Da, Ha Noi, Viet Nam")
	pdf.SetFont("BeeVN", "B", 14)
	pdf.CellFormat(50, 10, data.FullName, "", 1, "R", false, 0, "")
	pdf.SetDrawColor(237, 237, 238)
	pdf.Line(0, 50, 1000, 50)

	pdf.SetY(70)
	header := [colCount]string{"  Summary"}

	pdf.SetFont("BeeVN", "", 14)
	// Headers
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(255, 154, 0)
	for colJ := 0; colJ < 1; colJ++ {
		pdf.SetDrawColor(49, 50, 50)
		pdf.SetFont("beeVN", "B", 14)
		pdf.CellFormat(headerWd, 13, header[colJ], "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(24, 24, 24)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("BeeVN", "", 14)
	pdf.CellFormat(3, 20, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(137, 20, "Payment ID:", "", 0, "L", false, 0, "")
	pdf.CellFormat(47, 20, bill.Code, "", 0, "R", false, 0, "")
	pdf.CellFormat(3, 20, "", "R", 1, "L", false, 0, "")
	pdf.CellFormat(3, 6, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(137, 6, "Payment Date:", "", 0, "L", false, 0, "")
	pdf.CellFormat(47, 6, data.BillDate, "", 0, "R", false, 0, "")
	pdf.CellFormat(3, 6, " ", "R", 1, "L", false, 0, "")
	pdf.CellFormat(190, 8, "", "L,R", 1, "R", false, 0, "")
	pdf.Line(14, 115, 195, 115)
	pdf.SetFont("beeVN", "B", 14)
	pdf.CellFormat(3, 10, "", "L,B", 0, "L", false, 0, "")
	pdf.CellFormat(137, 10, "Payment Amount:", "B", 0, "L", false, 0, "")
	pdf.CellFormat(47, 10, data.TotalAmount, "B", 0, "R", false, 0, "")
	pdf.CellFormat(3, 10, "", "R,B", 1, "L", false, 0, "")

	pdf.Ln(-1)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFillColor(255, 154, 0)
	for colJ := 0; colJ < 1; colJ++ {
		pdf.SetDrawColor(49, 50, 50)
		pdf.CellFormat(headerWd, 13, "  Detail", "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(24, 24, 24)
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("beeVN", "B", 14)
	pdf.CellFormat(3, 15, "", "L", 0, "L", false, 0, "")
	pdf.CellFormat(137, 15, "Phí vận đơn:", "B", 0, "L", false, 0, "")
	pdf.CellFormat(47, 15, "$"+cast.ToString(data.ShippingFee), "B", 0, "R", false, 0, "")
	pdf.CellFormat(3, 15, "", "R", 1, "L", false, 0, "")
	pdf.SetFont("beeVN", "", 14)

	pdf.SetDrawColor(49, 50, 50)
	// pdf.Line(14, 165, 195, 165)
	for i, packageItem := range data.Packages {
		pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")
		pdf.CellFormat(135, 13, packageItem.Code, "", 0, "L", false, 0, "")
		pdf.CellFormat(45, 13, "$"+utils.FormatNumber(packageItem.ShippingFee), "", 0, "R", false, 0, "")
		pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")

		getY := pdf.GetY()
		spaceLeft := 297 - getY - 35
		lenP := len(data.Packages)
		if spaceLeft <= 0 {
			pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
			pdf.AddPage()
			pdf.SetY(30)
			if i+1 < lenP || len(data.ExtraFee) > 0 || len(data.RefundFee) > 0 {
				pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")
			}

		} else {
			if lenP == i+1 && len(data.ExtraFee) < 1 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
			} else if lenP == i+1 && spaceLeft < 10 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
			} else {
				pdf.SetFont("beeVN", "", 10)
				pdf.CellFormat(190, 1, "-----------------------------------------------------------------------------------", "L,R", 1, "C", false, 0, "")
			}
		}
		pdf.SetFont("beeVN", "", 14)
	}

	if len(data.ExtraFee) > 0 {
		pdf.SetFont("beeVN", "", 14)

		getY := pdf.GetY()
		spaceLeft := 297 - getY - 35
		if spaceLeft < 10 {
			pdf.AddPage()
			pdf.SetY(30)
			pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")

		} else {
			pdf.CellFormat(190, 3, "", "L,R", 1, "L", false, 0, "")

		}
		pdf.SetFont("beeVN", "B", 14)
		pdf.CellFormat(3, 15, "", "L", 0, "L", false, 0, "")
		pdf.CellFormat(137, 15, "Phí phát sinh:", "B", 0, "L", false, 0, "")
		pdf.CellFormat(47, 15, "$"+data.TotalExtraFee, "B", 0, "R", false, 0, "")
		pdf.CellFormat(3, 15, "", "R", 1, "L", false, 0, "")
		pdf.SetFont("beeVN", "", 14)

		pdf.SetDrawColor(49, 50, 50)
		// pdf.Line(14, 165, 195, 165)
		for i, item := range data.ExtraFee {
			code := item.PackageCode
			if code == "" && item.ShipmentID > 0 {
				code = fmt.Sprintf("FBA #%d", item.ShipmentID)
			}

			pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")
			pdf.CellFormat(135, 13, code, "", 0, "L", false, 0, "")
			pdf.CellFormat(45, 13, "$"+utils.FormatNumber(item.Amount), "", 0, "R", false, 0, "")
			pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")

			getY := pdf.GetY()
			spaceLeft := 297 - getY - 35
			lenP := len(data.ExtraFee)
			if spaceLeft <= 0 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				pdf.AddPage()
				pdf.SetY(30)
				if i+1 < lenP || len(data.RefundFee) > 0 {
					pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")
				}

			} else {
				if lenP == i+1 && len(data.RefundFee) < 1 {
					pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				} else if lenP == i+1 && spaceLeft < 10 {
					pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				} else {
					pdf.SetFont("beeVN", "", 10)
					pdf.CellFormat(190, 1, "-----------------------------------------------------------------------------------", "L,R", 1, "C", false, 0, "")
				}
			}
			pdf.SetFont("beeVN", "", 14)
		}
	}

	if len(data.RefundFee) > 0 {
		pdf.SetFont("beeVN", "", 14)

		getY := pdf.GetY()
		spaceLeft := 297 - getY - 35
		if spaceLeft < 10 {
			pdf.AddPage()
			pdf.SetY(30)
			pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")

		} else {
			pdf.CellFormat(190, 3, "", "L,R", 1, "L", false, 0, "")

		}
		pdf.SetFont("beeVN", "B", 14)
		pdf.CellFormat(3, 15, "", "L", 0, "L", false, 0, "")
		pdf.CellFormat(137, 15, "Hoàn tiền:", "B", 0, "L", false, 0, "")
		pdf.CellFormat(47, 15, "-$"+data.TotalRefundFee, "B", 0, "R", false, 0, "")
		pdf.CellFormat(3, 15, "", "R", 1, "L", false, 0, "")
		pdf.SetFont("beeVN", "", 14)

		pdf.SetDrawColor(49, 50, 50)
		// pdf.Line(14, 165, 195, 165)
		for i, item := range data.RefundFee {
			pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")

			if item.CouponCode != "" {
				pdf.CellFormat(135, 13, fmt.Sprintf("Coupon %s", item.CouponCode), "", 0, "L", false, 0, "")
			} else {
				pdf.CellFormat(135, 13, item.PackageCode, "", 0, "L", false, 0, "")
			}

			pdf.CellFormat(45, 13, "-$"+utils.FormatNumber(item.Amount), "", 0, "R", false, 0, "")
			pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")

			getY := pdf.GetY()
			spaceLeft := 297 - getY - 35
			lenP := len(data.RefundFee)
			if spaceLeft <= 0 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				pdf.AddPage()
				pdf.SetY(30)
				if i+1 < lenP {
					pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")
				}

			} else {
				if lenP == i+1 {
					pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				} else {
					pdf.SetFont("beeVN", "", 10)
					pdf.CellFormat(190, 1, "-----------------------------------------------------------------------------------", "L,R", 1, "C", false, 0, "")

				}
			}
			pdf.SetFont("beeVN", "", 14)

		}
	}

	if len(data.ComissionFee) > 0 {
		pdf.SetFont("beeVN", "", 14)

		getY := pdf.GetY()
		spaceLeft := 297 - getY - 35
		if spaceLeft < 10 {
			pdf.AddPage()
			pdf.SetY(30)
			pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")

		} else {
			pdf.CellFormat(190, 3, "", "L,R", 1, "L", false, 0, "")

		}
		pdf.SetFont("beeVN", "B", 14)
		pdf.CellFormat(3, 15, "", "L", 0, "L", false, 0, "")
		pdf.CellFormat(137, 15, "Hoa hồng:", "B", 0, "L", false, 0, "")
		pdf.CellFormat(47, 15, "-$"+data.TotalComFee, "B", 0, "R", false, 0, "")
		pdf.CellFormat(3, 15, "", "R", 1, "L", false, 0, "")
		pdf.SetFont("beeVN", "", 14)

		pdf.SetDrawColor(49, 50, 50)
		// pdf.Line(14, 165, 195, 165)
		for i, item := range data.ComissionFee {
			pdf.CellFormat(5, 13, "", "L", 0, "L", false, 0, "")

			pdf.CellFormat(135, 13, item.Description, "", 0, "L", false, 0, "")
			pdf.CellFormat(45, 13, "-$"+utils.FormatNumber(item.Amount), "", 0, "R", false, 0, "")
			pdf.CellFormat(5, 13, "", "R", 1, "L", false, 0, "")

			getY := pdf.GetY()
			spaceLeft := 297 - getY - 35
			lenP := len(data.ComissionFee)
			if spaceLeft <= 0 {
				pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				pdf.AddPage()
				pdf.SetY(30)
				if i+1 < lenP {
					pdf.CellFormat(190, 1, "", "T,L,R", 1, "", false, 0, "")
				}

			} else {
				if lenP == i+1 {
					pdf.CellFormat(190, 1, "", "B,L,R", 1, "", false, 0, "")
				} else {
					pdf.SetFont("beeVN", "", 10)
					pdf.CellFormat(190, 1, "-----------------------------------------------------------------------------------", "L,R", 1, "C", false, 0, "")

				}
			}
			pdf.SetFont("beeVN", "", 14)

		}
	}

	pdf.CellFormat(190, 8, "", "", 1, "L", false, 0, "")

	pdf.SetFont("beeVN", "I", 12)
	pdf.CellFormat(190, 5, "To access billing options, check your statement or for questions regarding this receipt,", "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 10, "log in to your members area at https://ananbay.com/", "", 0, "L", false, 0, "")

	fileName := fmt.Sprintf("payment_receipt_%s.pdf", bill.Code)

	err = pdf.OutputFileAndClose(fileName)
	if err != nil {
		fmt.Errorf("Create file pdf error %v", err)
		return "", err
	}

	return fileName, nil
}
