package order

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ups"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils"
	"time"

	"github.com/disintegration/imaging"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func EstimateCost(carrier providers.Carrier, in *entity.Package, warehouse entity.Warehouse) (*providers.ResponseEstimateCost, string, error) {
	if in.Address1 == "" && in.Address2 != "" {
		in.Address1 = in.Address2
	}

	code := ""
	if in.PackageCode != nil {
		code = in.PackageCode.Code
	}

	log.Println("code: ", code)
	// if service code == US48 or INUS or AU or EU -> no need to estimate cost
	if in.Service.Code == constant.ServiceUS48Code || in.Service.Code == constant.ServiceINUSCode || in.Service.Code == constant.ServiceAUCode || in.Service.Code == constant.ServiceEUCode {
		return &providers.ResponseEstimateCost{}, "", nil
	}

	body := providers.RequestCreateLabel{
		ID:                     in.ID,
		Code:                   code,
		OrderNumber:            in.OrderNumber,
		FirstName:              in.Recipient,
		LastName:               in.Recipient,
		FullName:               in.Recipient,
		Company:                in.Company,
		Address1:               in.Address1,
		Address2:               in.Address2,
		City:                   in.City,
		State:                  in.StateCode,
		Zipcode:                in.Zipcode,
		Phone:                  in.PhoneNumber,
		Country:                in.CountryCode,
		ItemName:               in.Detail,
		Weight:                 in.ActualWeight,
		Length:                 in.ActualLength,
		Width:                  in.ActualWidth,
		Height:                 in.ActualHeight,
		WarehouseCompany:       warehouse.Name,
		WarehouseAddress1:      warehouse.Address,
		WarehousePhone:         warehouse.Phone,
		WarehouseCity:          warehouse.City,
		WarehouseState:         warehouse.State,
		WarehouseZipcode:       warehouse.Zipcode,
		WarehouseCountry:       warehouse.Country,
		IsExceedPkg:            in.IsPackageExceed,
		DomesticCarrierService: in.Service.DomesticCarrierService,
		PostmarkDate:           6,
	}

	if body.Weight <= 0 {
		body.Weight = in.Weight
	}

	if body.Length <= 0 {
		body.Length = in.Length
	}

	if body.Height <= 0 {
		body.Height = in.Height
	}

	if body.Width <= 0 {
		body.Width = in.Width
	}

	res, errs, err := carrier.EstimateCost(body)
	if errs != nil {
		return nil, errs.Error(), err
	}
	if err != nil {
		return nil, "", err
	}

	return res, "", nil
}

func StoreLabelS3(s3 storage.S3, link, ext, tracking string) (string, error) {
	bucket := viper.GetString("bucket.labels")
	if ok := labelExtensionIsPDF(link); ok {
		fp := fmt.Sprintf("%s/%s.pdf", time.Now().Format("2006-01-02"), tracking)
		res, err := http.Get(link)
		if err != nil {
			return "", err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Close error: %v", err)
			}
		}(res.Body)

		buf, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", err
		}

		err = s3.UploadFile(bytes.NewBuffer(buf), fp, bucket, constant.ContentTypePDF)
		if err != nil {
			fmt.Errorf("upload base64 label: %v", err)
			return "", err
		}

		return fp, nil
	}

	fp := fmt.Sprintf("%s/%s.%s", time.Now().Format("2006-01-02"), tracking, ext)

	if strings.HasPrefix(strings.ToLower(link), "http") {
		res, err := http.Get(link)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

		// Read a limited number of bytes to detect content type
		head := make([]byte, 512)
		n, err := res.Body.Read(head)
		if err != nil && err != io.EOF {
			return "", fmt.Errorf("read file header: %w", err)
		}

		contentType := http.DetectContentType(head)
		body := io.MultiReader(bytes.NewReader(head[:n]), res.Body)

		switch {
		case contentType == "application/pdf":
			buf, err := io.ReadAll(body)
			if err != nil {
				return "", err
			}
			err = s3.UploadFile(bytes.NewBuffer(buf), fp, bucket, constant.ContentTypePDF)
			if err != nil {
				return "", fmt.Errorf("upload PDF label: %w", err)
			}
			return fp, nil

		case strings.HasPrefix(contentType, "image/"):
			im, err := imaging.Decode(body)
			if err != nil {
				return "", fmt.Errorf("decode image label: %w", err)
			}
			var buf bytes.Buffer
			err = imaging.Encode(&buf, im, imaging.PNG)
			if err != nil {
				return "", fmt.Errorf("encode image label: %w", err)
			}
			err = s3.UploadFile(&buf, fp, bucket, constant.ImageContentTypePNG)
			if err != nil {
				return "", fmt.Errorf("upload image label: %w", err)
			}
			return fp, nil

		default:
			return "", fmt.Errorf("unsupported content type (detected): %s", contentType)
		}
	}

	// Fallback: base64
	decode, err := base64.StdEncoding.DecodeString(link)
	if err != nil {
		return "", fmt.Errorf("decode base64 label: %w", err)
	}

	reader := bytes.NewReader(decode)
	contentType := constant.ImageContentTypePNG
	if ext == "pdf" {
		contentType = constant.ImageContentTypePDF
	}

	err = s3.UploadFile(reader, fp, bucket, contentType)
	if err != nil {
		return "", fmt.Errorf("upload base64 label: %w", err)
	}

	return fp, nil
}

func labelExtensionIsPDF(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	ext := filepath.Ext(u.Path)
	ext = strings.ToLower(ext)

	return ext == ".pdf"
}

func CancelLabel(carrier providers.Carrier, in *entity.Package) (bool, error) {
	if in.Tracking == nil {
		return true, nil
	}

	id := in.Tracking.TrackingNumber
	if in.Tracking.ShipmentID != "" {
		id = in.Tracking.ShipmentID
	}

	return carrier.CancelLabel(id)
}

func CheckIsManifestNow() bool {
	start := viper.GetString("shippo.manifest.start")
	end := viper.GetString("shippo.manifest.end")

	now := time.Now()
	nd := now.Format("2006-01-02")

	ts, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", nd, start))
	if err != nil {
		return false
	}

	te, err := time.Parse("2006-01-02 15:04:05", fmt.Sprintf("%s %s", nd, end))
	if err != nil {
		return false
	}

	if now.Sub(ts) < 0 || now.Sub(te) > 0 {
		return false
	}

	return true
}

func Manifest(packageID int64, trackingNumber string, customerID int64, warehouse *entity.Warehouse, carrier providers.Carrier) (*entity.Manifest, string, error) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return nil, "", err
	}

	t := time.Now().In(loc).Add(144 * time.Hour)
	shipDate := cast.ToTime(time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, t.Nanosecond(), t.Location())).Format(time.RFC3339)
	requestDate := cast.ToTime(shipDate)
	body := providers.ManifestRequest{
		ShipmentID:      packageID,
		TrackingNumbers: []string{trackingNumber},
		Name:            warehouse.Name,
		Line1:           warehouse.Address,
		Line2:           warehouse.Address,
		City:            warehouse.City,
		State:           warehouse.State,
		Zip:             warehouse.Zipcode,
		Country:         warehouse.Country,
		Phone:           warehouse.Phone,
		ShipmentDate:    &requestDate,
	}

	res, msg, err := carrier.CreateManifest(body)
	if err != nil || msg != "" {
		return nil, msg, nil

	}

	if len(res.Usps) < 1 {
		return nil, msg, errors.New("create manifest response not found")
	}

	manifest := &entity.Manifest{
		PackageID:      utils.Int64(packageID),
		ManifestNumber: res.Usps[0].ManifestNumber,
	}

	return manifest, "", nil
}

func StoreLabelManifest(s3 storage.S3, url, manifestNumber string, id int64, carrier string) (string, error) {
	bucket := viper.GetString("bucket.labels")
	filepath := fmt.Sprintf("manifest/%s/%d/%s.png", time.Now().Format("2006-01-02"), id, manifestNumber)

	if carrier == providers.CarrierTypeShippo {
		res, err := http.Get(url)
		if err != nil {
			return "", err
		}

		filepath = fmt.Sprintf("manifest/%s/%d/%s.pdf", time.Now().Format("2006-01-02"), id, manifestNumber)
		err = s3.UploadFile(res.Body, filepath, bucket, constant.ContentTypePDF)
		if err != nil {
			fmt.Errorf("upload base64 label: %v", err)
			return "", err
		}
		return filepath, nil
	}

	if strings.ToLower(url[0:4]) == "http" {
		res, err := http.Get(url)
		if err != nil {
			return "", err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Close error: %v", err)
			}
		}(res.Body)
		im, _, err := image.Decode(res.Body)
		if err != nil {
			fmt.Errorf("decode image label: %v", err)
			return "", err
		}

		src := imaging.Resize(im, 400, 0, imaging.Box)

		var buf bytes.Buffer
		err = imaging.Encode(&buf, src, imaging.PNG)
		if err != nil {
			fmt.Errorf("resize label: %v", err)
			return "", err
		}

		err = s3.UploadFile(&buf, filepath, bucket, constant.ImageContentTypePNG)
		if err != nil {
			fmt.Errorf("upload base64 label: %v", err)
			return "", err
		}

		return filepath, nil
	}

	decode, err := base64.StdEncoding.DecodeString(url)
	if err != nil {
		fmt.Errorf("decode base64 label: %v", err)
		return "", err
	}

	reader := bytes.NewReader(decode)
	err = s3.UploadFile(reader, filepath, bucket, constant.ImageContentTypePNG)
	if err != nil {
		fmt.Errorf("upload labels failed :%v", err)
		return "", err
	}

	return filepath, nil
}

func SendQueueManifest(ids []int64, isFilterShippo bool) error {
	data := entity.DataManifestLabelConsumer{
		Ids:      ids,
		IsShippo: isFilterShippo,
	}

	encode, err := json.Marshal(data)
	if err != nil {
		fmt.Errorf("Error marshaling body: %v", err)
		return err
	}

	fmt.Errorf("push message queue shipment manifest label: %v", string(encode))
	// if err := producer.PublishSimple(constant.QueueManifestLabel, encode); err != nil {
	// 	logger.Log.Error("Error publish message queue shipment-manifest-label: %v", err)
	// 	return err
	// }

	return nil
}

func FBARateExtras(c context.Context, carrier *ups.UPS, packages []*entity.Package, redis *redis.Client) ([]*entity.ExtraFee, error) {
	var upsItems = []ups.InputRateItem{}
	for _, p := range packages {
		upsItems = append(upsItems, ups.InputRateItem{Weight: p.Weight, Length: p.Length, Width: p.Width, Height: p.Height})
	}

	first := packages[0]
	address := first.Address1
	if address == "" {
		address = first.Address2
	}

	res, err := carrier.EstimateCost(ups.UPSInputRateRequest{Items: upsItems, ShipAddress: ups.InputAddress{
		Name:    first.Recipient,
		Company: first.Company,
		Phone:   first.PhoneNumber,
		Address: address,
		City:    first.City,
		State:   first.StateCode,
		Zipcode: first.Zipcode,
		Country: first.CountryCode,
	}})

	if err != nil {
		return nil, err
	}

	rate, err := GetExchangeRate(c, redis)
	if err != nil {
		return nil, err
	}

	extras := []*entity.ExtraFee{}
	if len(res.RateResponse.RatedShipment.ItemizedCharges) > 0 {
		for _, v := range res.RateResponse.RatedShipment.ItemizedCharges {
			amount := cast.ToFloat64(v.MonetaryValue)
			if amount <= 0 {
				continue
			}

			if v.Code != ups.Additional_Handling &&
				v.Code != ups.Large_Package &&
				v.Code != ups.Ship_Additional_Handling &&
				v.Code != ups.Ship_Large_Package {
				continue
			}

			if v.CurrencyCode == "VND" {
				amount = utils.Ceil(amount/rate, 2)
			}

			extras = append(extras, &entity.ExtraFee{
				ExtraFeeTypeID: constant.ExtraFeeTypeOther,
				Description:    ups.AccessorialOrSurchargeCodes[v.Code],
				Status:         constant.ExtraFeeStatusEnable,
				Amount:         amount,
			})
		}
	}

	return extras, nil
}

type RateExchange struct {
	UserId    int64     `json:"user_id"`
	Price     float64   `json:"price"`
	UpdatedAt time.Time `json:"updated_at"`
}

func GetExchangeRate(c context.Context, redis *redis.Client) (float64, error) {
	rate, err := redis.Get(c, constant.RedisKeyRateExChange).Result()
	if err != nil {
		return 0, err
	}

	decode := RateExchange{}
	if err := json.Unmarshal([]byte(rate), &decode); err != nil {
		return 0, err
	}

	return decode.Price, nil
}
