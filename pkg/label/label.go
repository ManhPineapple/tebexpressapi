package label

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/label/font/bevietnamproblack"
	"tebexpressapi/pkg/label/font/bevietnamprobold"
	"tebexpressapi/pkg/label/font/bevietnamproextrabold"
	"tebexpressapi/pkg/label/font/bevietnampromedium"
	"tebexpressapi/pkg/label/tmp"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"time"

	"github.com/spf13/viper"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/qr"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	BaseWidth     = 551
	BaseHeight    = 787
	BarcodeWidth  = 353
	BarcodeHeight = 86
	LineHeight    = 45
	QRCodeSize    = 74
)

type Input struct {
	OrderNumber string
	Code        string
	Logo        string
	LogoImage   *image.Image
	Service     string
	Name        string
	From        string
	Address1    string
	Address2    string
	City        string
	Zipcode     string
	State       string
	Country     string
	Weight      float64
	Index       int
	Total       int
}

type SettingLabelForm struct {
	LogoUrl    string `json:"logo_url"`
	PreviewUrl string `json:"preview_url"`
	ShipFrom   string `json:"ship_from"`
}

var (
	UniformColor    = image.NewUniform(color.RGBA{0, 0, 0, 255})
	ErrorInvalidRow = errors.New("invalid rows ship from")
)

func GenerateBase64(in Input, env string) (string, error) {
	iLogo, err := makeLogo("ND")
	if err != nil {
		log.Fatal(err)
	}

	in.LogoImage = &iLogo
	im, err := GenerateImage(in, env)
	if err != nil {
		return "", err
	}

	b := &bytes.Buffer{}
	png.Encode(b, im)
	base := base64.StdEncoding.EncodeToString(b.Bytes())
	return base, nil
}

func GenerateReader(in Input, env string) (io.Reader, error) {
	im, err := GenerateImage(in, env)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(nil)
	png.Encode(b, im)
	return b, nil
}

func GenerateReaderPreview(in Input) (io.Reader, error) {
	im, err := GenerateImagePreview(in)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBuffer(nil)
	png.Encode(b, im)
	return b, nil
}

func GeneratePreviewBase64(in Input) (string, error) {
	im, err := GenerateImagePreview(in)
	if err != nil {
		return "", err
	}

	b := &bytes.Buffer{}
	png.Encode(b, im)
	base := base64.StdEncoding.EncodeToString(b.Bytes())
	return base, nil
}

func GenerateImagePreview(in Input) (image.Image, error) {
	var im image.Image
	var err error
	if in.LogoImage != nil {
		im, _, err = image.Decode(bytes.NewReader(tmp.TIM_CUS_PRE))
		if err != nil {
			return nil, err
		}
	} else {
		im, _, err = image.Decode(bytes.NewReader(tmp.TIM_PRE))
		if err != nil {
			return nil, err
		}
	}

	beExtra, err := sfnt.Parse(bevietnamproextrabold.TTF)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, BaseWidth, BaseHeight))

	// Draw layout
	draw.Draw(dst, image.Rect(0, 0, BaseWidth, BaseHeight), im, image.Point{}, draw.Src)

	// Draw ShipFrom
	face, err := opentype.NewFace(beExtra, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	lines := strings.Split(in.From, "\n")
	if len(lines) > 3 {
		return nil, ErrorInvalidRow
	}

	// Draw ship from
	y := 48
	maxTextWidthFrom := 210

	if len(lines) > 3 {
		return nil, ErrorInvalidRow
	}

	for _, line := range lines {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}

		n := len(text)
		widthText := 0
		textDisplay := text

		for i := 0; i < n; i++ {
			widthText += font.MeasureString(face, text[i:i+1]).Ceil()
			if widthText > maxTextWidthFrom {
				textDisplay = text[0:i]
				break
			}
		}

		d := font.Drawer{
			Dst:  dst,
			Src:  UniformColor,
			Face: face,
			Dot:  fixed.P(230, y),
		}

		d.DrawString(textDisplay)
		y += 20
	}

	//draw logo
	if in.LogoImage != nil {
		imgLogo := *in.LogoImage
		ox := (123-imgLogo.Bounds().Dx())/2 + 15
		oy := (123-imgLogo.Bounds().Dy())/2 + 15
		oxe := ox + imgLogo.Bounds().Dx()
		oye := oy + imgLogo.Bounds().Dy()
		draw.Draw(dst, image.Rect(ox, oy, oxe, oye), imgLogo, imgLogo.Bounds().Min, draw.Over)
	}

	return dst, nil
}

func GenerateImage(in Input, env string) (image.Image, error) {
	var im image.Image
	var err error
	if in.LogoImage != nil {
		im, err = jpeg.Decode(bytes.NewReader(tmp.TIM_CUS))
		if err != nil {
			return nil, err
		}
	} else {
		im, err = jpeg.Decode(bytes.NewReader(tmp.TIM))
		if err != nil {
			return nil, err
		}
	}

	beBlack, err := sfnt.Parse(bevietnamproblack.TTF)
	if err != nil {
		return nil, err
	}

	beExtra, err := sfnt.Parse(bevietnamproextrabold.TTF)
	if err != nil {
		return nil, err
	}

	beMedium, err := sfnt.Parse(bevietnampromedium.TTF)
	if err != nil {
		return nil, err
	}

	beBold, err := sfnt.Parse(bevietnamprobold.TTF)
	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, BaseWidth, BaseHeight))

	// Draw layout
	draw.Draw(dst, image.Rect(0, 0, BaseWidth, BaseHeight), im, image.Point{}, draw.Src)

	// Draw ShipFrom
	face, err := opentype.NewFace(beExtra, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	lines := strings.Split(in.From, "\n")
	if len(lines) > 3 {
		return nil, ErrorInvalidRow
	}

	// Draw ship from
	y := 48
	maxTextWidthFrom := 210

	if len(lines) > 3 {
		return nil, ErrorInvalidRow
	}

	for _, line := range lines {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}

		n := len(text)
		widthText := 0
		textDisplay := text

		for i := 0; i < n; i++ {
			widthText += font.MeasureString(face, text[i:i+1]).Ceil()
			if widthText > maxTextWidthFrom {
				textDisplay = text[0:i]
				break
			}
		}

		d := font.Drawer{
			Dst:  dst,
			Src:  UniformColor,
			Face: face,
			Dot:  fixed.P(230, y),
		}

		d.DrawString(textDisplay)
		y += 20
	}

	//draw logo

	if in.LogoImage != nil {
		imgLogo := *in.LogoImage
		ox := (123-imgLogo.Bounds().Dx())/2 + 15
		oy := (123-imgLogo.Bounds().Dy())/2 + 15
		oxe := ox + imgLogo.Bounds().Dx()
		oye := oy + imgLogo.Bounds().Dy()
		draw.Draw(dst, image.Rect(ox, oy, oxe, oye), imgLogo, imgLogo.Bounds().Min, draw.Over)
	}

	// Draw to
	face, err = opentype.NewFace(beExtra, &opentype.FaceOptions{
		Size:    22,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}
	y = 175 + textHeight(in.Name, face)
	x := 90
	d := font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(in.Name)

	face, err = opentype.NewFace(beMedium, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}
	y += 28
	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(in.Address1)
	y += 28
	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(in.City)
	y += 28
	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(in.Zipcode)
	y += 28
	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(in.State)

	// draw service
	face, err = opentype.NewFace(beBlack, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P((210-textWidth(strings.ToUpper(in.Service), face))/2+15, 463),
	}
	d.DrawString(strings.ToUpper(in.Service))

	// draw weight:
	face, err = opentype.NewFace(beMedium, &opentype.FaceOptions{
		Size:    18,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(330, 463),
	}
	d.DrawString(fmt.Sprintf("%v g", in.Weight))

	// draw order number:
	//in.OrderNumber = fmt.Sprintf("#%s",in.OrderNumber)
	lineWidth := 0
	splitStrings := strings.Split(in.OrderNumber, "")
	for _, splitstr := range splitStrings {
		strWidth := font.MeasureString(face, splitstr).Ceil()
		if lineWidth+int(strWidth) < 202 {
			d := font.Drawer{
				Dst:  dst,
				Src:  UniformColor,
				Face: face,
				Dot:  fixed.P(lineWidth+320, 398),
			}
			lineWidth += strWidth
			d.DrawString(splitstr)
		} else {
			break
		}
	}
	// draw country
	face, err = opentype.NewFace(beBlack, &opentype.FaceOptions{
		Size:    54,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P((210-textWidth(in.Country, face))/2+15, 412),
	}
	d.DrawString(in.Country)

	//set font face barcode
	face, err = opentype.NewFace(beBold, &opentype.FaceOptions{
		Size:    14,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P((BaseWidth-textWidth("ANANBAY TRACKING", face))/2, 532),
	}

	d.DrawString("ANANBAY TRACKING")
	// draw barcode
	bc, err := generateBarcode(in.Code, env)
	if err != nil {
		return nil, err
	}

	ox := (BaseWidth - bc.Bounds().Dx()) / 2
	oy := 548
	oxe := ox + bc.Bounds().Dx()
	oye := oy + bc.Bounds().Dy()

	if oye >= 634 {
		oye = 634
	}

	draw.Draw(dst, image.Rect(ox, oy, oxe, oye), bc, bc.Bounds().Min, draw.Over)

	// qc, err := generateQRCode(in.Code)
	// if err != nil {
	// 	return nil, err
	// }

	// draw.Draw(dst, image.Rect(450, 27, 450+QRCodeSize, 27+QRCodeSize), qc, qc.Bounds().Min, draw.Over)

	if oye < 633 {
		for {
			oy = oye
			oye = oy + bc.Bounds().Dy()
			if oye > 633 {
				oye = 633
			}

			draw.Draw(dst, image.Rect(ox, oy, oxe, oye), bc, bc.Bounds().Min, draw.Over)
		}
	}

	face, err = opentype.NewFace(beBold, &opentype.FaceOptions{
		Size:    11,
		DPI:     72,
		Hinting: font.HintingNone,
	})

	if err != nil {
		return nil, err
	}

	wt := textWidth(in.Code, face)
	d = font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P((BaseWidth-wt)/2, 651),
	}

	d.DrawString(in.Code)

	if in.Index > 0 && in.Total > 0 {
		text := fmt.Sprintf("%d/%d", in.Index, in.Total)
		face, err = opentype.NewFace(beMedium, &opentype.FaceOptions{
			Size:    14,
			DPI:     102,
			Hinting: font.HintingNone,
		})
		if err != nil {
			return nil, err
		}

		d = font.Drawer{
			Dst:  dst,
			Src:  UniformColor,
			Face: face,
			Dot:  fixed.P((BaseWidth-textWidth(text, face))/2, 722),
		}

		d.DrawString(text)
	}

	// draw cross out
	if env == "development" {
		drawLineRightToLeft(dst, 0, 0, 551, 787, 10, UniformColor)
		drawLineLeftToRight(dst, 0, 787, 551, 0, 10, UniformColor)
	}

	return dst, nil
}

// draw bottom right to top left
func drawLineRightToLeft(img draw.Image, x1, y1, x2, y2 int, size int, col color.Color) {
	dx, dy := x2-x1, y2-y1
	a := float64(dy) / float64(dx)
	b := int(float64(y1) - a*float64(x1))

	img.Set(x1, y1, col)
	for x := x1 + 1; x <= x2; x++ {
		y := int(a*float64(x)) + b
		for s := 0; s < size/2; s++ {
			img.Set(x-s, y-s, col)
			img.Set(x+s, y+s, col)
		}
	}
}

// draw bottom left to top right
func drawLineLeftToRight(img draw.Image, x1, y1, x2, y2 int, size int, col color.Color) {
	dx, dy := 2*(x2-x1), y1-y2
	e, slope := dy, 2*dy
	for ; dy != 0; dy-- {
		for s := 0; s < size/2; s++ {
			img.Set(x1-s, y1+s, col)
			img.Set(x1+s, y1-s, col)
		}
		y1--
		e -= dx
		if e < 0 {
			x1++
			e += slope
		}
	}
}

func generateBarcode(code string, env string) (barcode.Barcode, error) {
	encode, err := code128.Encode(code)
	if err != nil {
		return nil, err
	}

	bcWidth := BarcodeWidth
	if env == "development" {
		bcWidth = 500
	}

	return barcode.Scale(encode, bcWidth, BarcodeHeight)
}

func generateQRCode(code string) (image.Image, error) {
	encode, err := qr.Encode(code, qr.H, qr.Auto)
	if err != nil {
		return nil, err
	}

	return barcode.Scale(encode, QRCodeSize, QRCodeSize)
}

func drawTextCenter(im *image.RGBA, text string, face font.Face, y int) {
	wt := textWidth(text, face)
	d := &font.Drawer{
		Dst:  im,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P((BaseWidth-wt)/2, y),
	}

	d.DrawString(text)
}

func textSize(text string, face font.Face) (width, height int) {
	bounds, _ := font.BoundString(face, text)
	width = int((bounds.Max.X - bounds.Min.X) / 64)
	height = int((bounds.Max.Y - bounds.Min.Y) / 64)
	return
}

func textWidth(text string, face font.Face) int {
	bounds, _ := font.BoundString(face, text)
	return int((bounds.Max.X - bounds.Min.X) / 64)
}

func textHeight(text string, face font.Face) int {
	bounds, _ := font.BoundString(face, text)
	return int((bounds.Max.Y - bounds.Min.Y) / 64)
}

func ImageToByte(name string) {
	f, err := os.Open(name)
	if err != nil {
		println(err)
		return
	}

	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		println(err)
		return
	}

	fd, err := os.Create("data.txt")
	if err != nil {
		println(err)
		return
	}

	defer f.Close()

	str := ""
	for i, v := range b {
		str += fmt.Sprintf("0x%02x,", v)
		if (i+1)%16 == 0 {
			str += "\n"
		}
	}

	fd.WriteString(str)
}

func ImageToByteGo(in, out, namevar string) {
	f, err := os.Open(in)
	if err != nil {
		println(err)
		return
	}

	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		println(err)
		return
	}

	fd, err := os.Create(out)
	if err != nil {
		println(err)
		return
	}

	defer f.Close()

	str := "package tmp\n\n"
	str += "var " + namevar + " = []byte{\n"
	for i, v := range b {
		str += fmt.Sprintf("0x%02x,", v)
		if (i+1)%16 == 0 {
			str += "\n"
		}
	}

	str += "\n}"

	fd.WriteString(str)
}

func CreateLabel(sp *entity.Package, SettingManager *sqlmanager.SettingManager, StorageS3 storage.S3) (string, string, error) {
	if sp.PackageCode == nil {
		return "", "", errors.New("cant find package code")
	}

	if sp.Service == nil {
		return "", "", errors.New("cant find service")
	}

	env := viper.GetString("env")

	start := time.Now()
	input := Input{
		OrderNumber: sp.OrderNumber,
		Code:        sp.PackageCode.Code,
		Logo:        sp.Service.Name[0:1],
		Service:     sp.Service.Name,
		Name:        sp.Recipient,
		From:        constant.DefaultShipFromLabel,
		Address1:    sp.Address1,
		Address2:    sp.Address2,
		City:        sp.City,
		Zipcode:     sp.Zipcode,
		State:       sp.StateCode,
		Country:     sp.CountryCode,
		Weight:      sp.Weight,
	}
	// setting, err := SettingManager.GetSetting(sqlmanager.SettingQueryOption{
	// 	UserID: sp.UserID,
	// 	Key:    constant.CustomizeLabelSettingKey,
	// })
	// if err != nil && err != gorm.ErrRecordNotFound {
	// 	return "", "", errors.New("Error fetch setting label")
	// }

	// if setting != nil && setting.ID > 0 {
	// 	settingLabel := SettingLabelForm{}
	// 	err = json.Unmarshal([]byte(setting.Value), &settingLabel)
	// 	if err != nil {
	// 		fmt.Errorf("Unmashal setting error: %v", err)
	// 		return "", "", err
	// 	}
	// 	if settingLabel.LogoUrl != "" {
	// 		bucket := viper.GetString("bucket.labels")
	// 		buf, err := StorageS3.ReadFile(settingLabel.LogoUrl, bucket)
	// 		if err != nil {
	// 			fmt.Errorf("Read s3 file error: %v", err)
	// 			return "", "", err
	// 		}

	// 		logoImg, _, err := image.Decode(buf)
	// 		if err != nil {
	// 			fmt.Errorf("Decode image error: %v", err)
	// 			return "", "", err
	// 		}
	// 		input.LogoImage = &logoImg
	// 	}
	// 	if settingLabel.ShipFrom != "" {
	// 		input.From = settingLabel.ShipFrom
	// 	}
	// }

	a, _ := json.Marshal(input)
	log.Println("hihi: ", string(a))
	log.Println("hihi: ", env)
	base, err := GenerateBase64(input, env)

	log.Printf("execution time generate label: %v", time.Since(start))

	if err != nil {
		fmt.Errorf("generate label: %v", err)
		return "", "", err
	}

	start = time.Now()

	decode, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		fmt.Errorf("generate: %v", err)
		return "", "", err
	}

	reader := bytes.NewReader(decode)
	bucket := viper.GetString("bucket.labels")
	filepath := fmt.Sprintf("%s/%s.png", time.Now().Format("2006-01-02"), sp.PackageCode.Code)

	err = StorageS3.UploadFile(reader, filepath, bucket, constant.ImageContentTypePNG)
	if err != nil {
		fmt.Errorf("upload labels failed :%v", err)
		return "", "", err
	}

	log.Printf("execution time upload label to s3: %v", time.Since(start))

	return base, filepath, err
}

func CreateFBALabel(sp *entity.Package, SettingManager *sqlmanager.SettingManager, StorageS3 storage.S3, index, total int) (string, string, error) {
	if sp.PackageCode == nil {
		return "", "", errors.New("cant find package code")
	}

	if sp.Service == nil {
		return "", "", errors.New("cant find service")
	}

	env := viper.GetString("env")

	start := time.Now()
	input := Input{
		OrderNumber: sp.OrderNumber,
		Code:        sp.PackageCode.Code,
		Logo:        "",
		Service:     sp.Service.Name,
		Name:        sp.Recipient,
		From:        constant.DefaultShipFromLabel,
		Address1:    sp.Address1,
		Address2:    sp.Address2,
		City:        sp.City,
		Zipcode:     sp.Zipcode,
		State:       sp.StateCode,
		Country:     sp.CountryCode,
		Weight:      sp.Weight,
		Index:       index,
		Total:       total,
	}

	a, _ := json.Marshal(input)
	log.Println("in img: ", string(a))
	base, err := GenerateBase64(input, env)

	log.Printf("execution time generate label: %v", time.Since(start))

	if err != nil {
		log.Printf("generate label: %v", err)
		return "", "", err
	}

	start = time.Now()

	decode, err := base64.StdEncoding.DecodeString(base)
	if err != nil {
		log.Printf("generate: %v", err)
		return "", "", err
	}

	reader := bytes.NewReader(decode)
	bucket := viper.GetString("bucket.labels")
	filepath := fmt.Sprintf("%s/%s.png", time.Now().Format("2006-01-02"), sp.PackageCode.Code)

	err = StorageS3.UploadFile(reader, filepath, bucket, constant.ImageContentTypePNG)
	if err != nil {
		fmt.Errorf("upload labels failed :%v", err)
		return "", "", err
	}

	log.Printf("execution time upload label to s3: %v", time.Since(start))

	return base, filepath, err
}

func makeLogo(t string) (image.Image, error) {
	dst := image.NewRGBA(image.Rect(0, 0, 100, 100))
	beBold, err := sfnt.Parse(bevietnamprobold.TTF)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(beBold, &opentype.FaceOptions{
		Size:    64,
		DPI:     72,
		Hinting: font.HintingNone,
	})
	if err != nil {
		return nil, err
	}

	d := font.Drawer{
		Dst:  dst,
		Src:  UniformColor,
		Face: face,
		Dot:  fixed.P(5, 70),
	}
	d.DrawString(t)

	return dst, nil
}
