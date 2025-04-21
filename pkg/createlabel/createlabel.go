package createlabel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/providers"
	"tebexpressapi/pkg/providers/ibblue"
	"tebexpressapi/pkg/utils"

	"github.com/redis/go-redis/v9"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const (
	RedisKeyVolumes    = "fake-volumes"
	RedisKeyDimensions = "fake-dimensions"
	LabelTypeOld       = 1
	LabelTypeNew       = 2
	LabelTypeUpdate    = 3

	// GramToOz const convert 1 gram to ounces
	GramToOz = 0.03527392

	// CentimeterToInches const convert 1 centimeter to inch
	CentimeterToInches = 0.3937007874

	// LbToGram const convert 1 pound to grams
	LbToGram = 453.59237

	// FirstClassLimit Limit of first class (oz)
	FirstClassLimit = 16

	MassUnitG  = "g"
	MassUnitKG = "kg"
	MassUnitLB = "lb"
	MassUnitOX = "oz"

	DistanceUnitCM = "cm"
	DistanceUnitIN = "in"
	DistanceUnitFT = "ft"
	DistanceUnitM  = "m"
	DistanceUnitMM = "mm"
	DistanceUnitYD = "yd"

	StatusSuccess       = "SUCCESS"
	StatusQueued        = "QUEUED"
	StatusError         = "ERROR"
	RedisPrefixKeyPoint = "SIZE_POINT"
)

var FirstStep = []float64{5, 9, 13, 16}

type CreateLabel struct {
	db         *gorm.DB
	redis      *redis.Client
	volumes    []entity.FakeVolume
	dimensions []entity.FakeDimension
}

type ResponseCreateLabel struct {
	providers.ResponseCreateLabel
	Weight      float64 `json:"weight"`
	Length      float64 `json:"length"`
	Height      float64 `json:"height"`
	Width       float64 `json:"width"`
	CarrierCode string  `json:"carrier_code"`
}

type DataPrice struct {
	Zone      string  `json:"zone"`
	Price     float64 `json:"price"`
	CarrierID int64   `json:"carrier_id"`
}

func Init(db *gorm.DB, redis *redis.Client) *CreateLabel {
	return &CreateLabel{db: db, redis: redis}
}

func (c *CreateLabel) LoadVolumes(ctx context.Context) error {
	var volumes []entity.FakeVolume

	str, err := c.redis.Get(ctx, RedisKeyVolumes).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if str != "" {
		if err := json.Unmarshal([]byte(str), &volumes); err != nil {
			return err
		}

		c.volumes = volumes
		return nil
	}

	err = c.db.Table("fake_volumes").
		Select("milestone,weight,dimension").
		Order("milestone DESC").Find(&volumes).Error

	if err != nil {
		return err
	}

	b, err := json.Marshal(volumes)
	if err != nil {
		return err
	}

	if err := c.redis.Set(ctx, RedisKeyVolumes, string(b), 0).Err(); err != nil {
		return err
	}

	c.volumes = volumes
	return nil
}

func (c *CreateLabel) LoadDimensions(ctx context.Context) error {
	var dimensions []entity.FakeDimension

	str, err := c.redis.Get(ctx, RedisKeyDimensions).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if str != "" {
		if err := json.Unmarshal([]byte(str), &dimensions); err != nil {
			return err
		}

		c.dimensions = dimensions
		return nil
	}

	err = c.db.Table("fake_dimensions").Select("`to`,`from`,rate").Find(&dimensions).Error
	if err != nil {
		return err
	}

	b, err := json.Marshal(dimensions)
	if err != nil {
		return err
	}

	if err := c.redis.Set(ctx, RedisKeyDimensions, string(b), 0).Err(); err != nil {
		return err
	}

	c.dimensions = dimensions
	return nil
}

// Fake trả về giảm khối lượng, giảm kích thước
func (c *CreateLabel) FakeVolume(weight, volume float64) (float64, float64) {
	var devSize float64 = 1
	var minusWeight float64 = 0

	for _, v := range c.volumes {
		if weight <= v.Milestone {
			minusWeight = v.Weight
		}

		if volume <= v.Milestone {
			devSize = v.Dimension
		}
	}

	if devSize < 1 {
		devSize = 1
	}

	if weight >= volume {
		return minusWeight, 1
	}

	return 0, devSize
}

func (c *CreateLabel) FakeCubic(length, height, width float64) float64 {
	cubic := length * height * width
	for _, v := range c.dimensions {
		if cubic >= v.From && cubic <= v.To {
			return v.Rate
		}
	}

	return 1
}

func (c *CreateLabel) Fake(ctx context.Context, weight, length, height, width float64) (float64, float64, float64, float64, error) {
	if err := c.LoadVolumes(ctx); err != nil {
		return 0, 0, 0, 0, err
	}

	if err := c.LoadDimensions(ctx); err != nil {
		return 0, 0, 0, 0, err
	}

	volume := calculate.SizeToWeight(length, height, width)
	minusWeight, devSize := c.FakeVolume(weight, volume)

	weight = weight - minusWeight
	length = length / devSize
	height = height / devSize
	width = width / devSize

	rate := c.FakeCubic(length, height, width)
	length = length * rate
	height = height * rate
	width = width * rate

	return weight, length, height, width, nil
}

// Fake trả về giảm khối lượng, giảm kích thước
func (c *CreateLabel) GetBreakPoinWeightService(ctx context.Context) (float64, error) {
	if err := c.LoadVolumes(ctx); err != nil {
		return 0, errors.New("missing breakpoint")
	}

	maxWeight := float64(constant.MaxWeightShippingPackage)
	var result float64
	var index int
	for i := len(c.volumes) - 1; i > 0; i-- {
		if (c.volumes[i].Milestone - c.volumes[i].Weight) == maxWeight { // match now
			result = c.volumes[i].Milestone
			break
		}

		if (c.volumes[i].Milestone - c.volumes[i].Weight) > maxWeight {
			index = i
			break
		}
	}

	if index > 0 {
		start := c.volumes[index+1].Milestone
		end := c.volumes[index].Milestone
		// loop from start+1 to end - 1
		for j := start + 1; j < end-1; j++ {
			if j-c.volumes[index].Weight >= maxWeight {
				result = j
				break
			}
		}
	}

	if result > 0 {
		return result, nil
	}

	return 0, errors.New("missing breakpoint")
}

func GetCarrierCode(in providers.RequestCreateLabel, redis *redis.Client) string {
	weight, length, height, width := in.Weight, in.Length, in.Height, in.Width

	if in.DistanceUnit == DistanceUnitIN {
		width = width * CentimeterToInches
		length = length * CentimeterToInches
		height = height * CentimeterToInches
	}

	if in.MassUnit == MassUnitG || in.MassUnit == "" {
		weight = weight * GramToOz
	}

	if width > 15 && length > 15 && height > 15 {
		if in.IsExceedPkg {
			return viper.GetString("provider.oversize_service")
		}

		return viper.GetString("provider.largesize_service")
	}

	if (width > 18 && length > 18) || (height > 18 && length > 18) || (height > 18 && width > 18) {
		if in.IsExceedPkg {
			return viper.GetString("provider.oversize_service")
		}

		return viper.GetString("provider.largesize_service")
	}

	if weight > FirstClassLimit {
		if in.IsExceedPkg {
			return viper.GetString("provider.oversize_service")
		}

		return viper.GetString("provider.largesize_service")
	}

	return viper.GetString("provider.smallsize_service")
}

func (c *CreateLabel) Request(ctx context.Context, body providers.RequestCreateLabel, carrier providers.Carrier, customerID int64, labelType int) (*ResponseCreateLabel, *providers.ErrResponse, error) {
	log.Println("payload: ", body.Weight, body.Length, body.Height, body.Width)

	carrierCode := ""

	if body.Country != "AU" {
		if labelType == LabelTypeNew {
			zone := ""
			if body.Zone > 0 {
				zone = fmt.Sprintf("Zone%d", body.Zone)
			}

			code, err := c.GetCarrierCode(ctx, entity.Package{
				Weight:          body.Weight,
				Length:          body.Length,
				Height:          body.Height,
				Width:           body.Width,
				IsPackageExceed: body.IsExceedPkg,
				Service:         &entity.Service{Code: body.FullServiceCode},
			}, customerID, zone)

			log.Println("GetCarrierCode: ", err, customerID, zone)
			if err != nil {
				return nil, nil, err
			}

			log.Printf("weight: %v, length: %v, height: %v, width: %v, zone: %v, carrier: %v", body.Weight, body.Length, body.Height, body.Width, zone, carrierCode)

			if code != "" {
				carrier = providers.NewCarrier(code, customerID)
				if carrier == nil {
					return nil, nil, errors.New("carrier is invalid")
				}
			}

			carrierCode = code

			if code == providers.CarrierTypeShippo && body.PostmarkDate > 6 {
				body.PostmarkDate = 6
			}
		}

		weight, length, height, width, err := c.Fake(ctx, body.Weight, body.Length, body.Height, body.Width)
		if err != nil {
			return nil, nil, err
		}

		body.Weight = weight
		body.Length = length
		body.Height = height
		body.Width = width
	}

	result := &ResponseCreateLabel{
		Weight:      body.Weight,
		Length:      body.Length,
		Height:      body.Height,
		Width:       body.Width,
		CarrierCode: carrierCode,
	}

	log.Println("payload: ", result)

	if body.DisplayWeight <= 0 {
		body.DisplayWeight = body.Weight
	}

	if body.LabelTemplate != ibblue.TemplateTebexpress && body.LabelTemplate != ibblue.TemplateWLTebexpress {
		body.LabelTemplate = ibblue.TemplateTebexpress
	}

	log.Println("label template: ", body.LabelTemplate, labelType)
	body.LabelTemplate = ""
	labelType = LabelTypeOld

	var auditError *providers.ErrResponse
	var res *providers.ResponseCreateLabel
	var err error

	switch labelType {
	case LabelTypeOld:
		res, auditError, err = carrier.CreateLabel(body)
	case LabelTypeNew:
		res, auditError, err = carrier.CreateLabel2(body)
	case LabelTypeUpdate:
		res, auditError, err = carrier.UpdateLabel(body)
	default:
		res, auditError, err = carrier.CreateLabel(body)
	}

	if res != nil {
		result.ShipmentID = res.ShipmentID
		result.Zone = res.Zone
		result.LabelUrl = res.LabelUrl
		result.ShippingFee = res.ShippingFee
		result.TrackingNumber = res.TrackingNumber
		result.CarrierService = res.CarrierService
	}

	if result.Zone < 1 {
		result.Zone = body.Zone
	}

	if labelType == LabelTypeNew {
		result, auditError, err = c.CheckSwitchCarrier(auditError, body, carrierCode, customerID, result)
		return result, auditError, err
	}

	return result, auditError, err
}

func (c *CreateLabel) CheckSwitchCarrier(auditError *providers.ErrResponse, body providers.RequestCreateLabel, carrierCode string, customerID int64, result *ResponseCreateLabel) (*ResponseCreateLabel, *providers.ErrResponse, error) {
	var res *providers.ResponseCreateLabel
	var err error

	if auditError == nil {
		return result, auditError, err
	}

	carrierCheck := []string{providers.CarrierTypeShippo, providers.CarrierTypeIBBlue}
	mapErrMess := make(map[string][]string)
	for _, code := range carrierCheck {
		if code == providers.CarrierTypeShippo {
			mapErrMess[code] = append(mapErrMess[code], "Recipient address invalid")
			mapErrMess[code] = append(mapErrMess[code], "You are required to have a valid payment method on file to purchase labels")
		} else {
			mapErrMess[code] = append(mapErrMess[code], "Insufficient funds")
		}
	}

	if utils.ContainsString(carrierCheck, carrierCode) {
		isResendRequest := false
		for _, msg := range mapErrMess[carrierCode] {
			if strings.Contains(auditError.Error(), msg) {
				isResendRequest = true
			}
		}
		if isResendRequest {
			for _, code := range carrierCheck {
				if carrierCode != code {
					carrierCode = code
					break
				}
			}
			carrier := providers.NewCarrier(carrierCode, customerID)
			if carrier == nil {
				return nil, nil, errors.New("carrier is invalid")
			}

			res, auditError, err = carrier.CreateLabel2(body)
			if res != nil {
				result.CarrierCode = carrierCode
				result.ShipmentID = res.ShipmentID
				result.Zone = res.Zone
				result.LabelUrl = res.LabelUrl
				result.ShippingFee = res.ShippingFee
				result.TrackingNumber = res.TrackingNumber
				result.CarrierService = res.CarrierService
			}

			if result.Zone < 1 {
				result.Zone = body.Zone
			}
		}
	}

	return result, auditError, err
}

type CarrierPrice struct {
	CarrierID int64   `json:"carrier_id"`
	Price     float64 `json:"price"`
	Zone      string  `json:"zone"`
}

type Data struct {
	Point float64        `json:"point"`
	Data  []CarrierPrice `json:"data"`
}

func (c *CreateLabel) ClearCacheRedisCarrierPrice(ctx context.Context) error {
	err := c.redis.Del(ctx, fmt.Sprintf("%s_service", constant.ServiceSizeOverCode), fmt.Sprintf("%s_service", constant.ServiceSizeLargeCode), fmt.Sprintf("%s_service", constant.ServiceSizeSmallCode)).Err()
	return err
}

func (c *CreateLabel) GetRedisCarrierCode(ctx context.Context, point, zone string) (string, error) {
	zones := make(map[string]string)
	result, err := c.redis.Get(ctx, point).Result()
	log.Println("redis get: ", result, point, zone, err)
	if err == redis.Nil || result == "" {
		log := &entity.CheckPriceLog{}
		if err := c.db.Order("created_at DESC").First(&log).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return providers.CarrierTypeIBBlue, nil
			}

			return "", err
		}

		var data = []Data{}
		if err := json.Unmarshal([]byte(log.Data), &data); err != nil {
			return "", err
		}

		carriers := []entity.Carrier{}
		if err := c.db.Find(&carriers).Error; err != nil {
			return "", err
		}

		mapCarriers := make(map[int64]string)
		for _, c := range carriers {
			mapCarriers[c.ID] = strings.TrimSpace(c.Code)
		}

		for _, v := range data {
			key := GetKeyStepWeight(v.Point)
			zoneCarrier := make(map[string]string)
			zonePriceMin := make(map[string]float64)

			for _, cp := range v.Data {
				if zoneCarrier[cp.Zone] == "" || zonePriceMin[cp.Zone] == 0 || zonePriceMin[cp.Zone] > cp.Price {
					zoneCarrier[cp.Zone] = mapCarriers[cp.CarrierID]
					zonePriceMin[cp.Zone] = cp.Price
				}
			}

			b, err := json.Marshal(zoneCarrier)
			if err != nil {
				return "", err
			}

			if err := c.redis.Set(ctx, key, string(b), -1).Err(); err != nil {
				return "", err
			}

			if key == point {
				zones = zoneCarrier
			}
		}
	} else if result != "" {
		if err := json.Unmarshal([]byte(result), &zones); err != nil {
			return "", err
		}
	}

	if zones[zone] == "" {
		return "", errors.New("carrier not found")
	}

	return zones[zone], nil
}

func GetWeightStep() []float64 {
	steps := FirstStep
	max := float64(constant.MaxWeightOz)
	for i := FirstStep[len(FirstStep)-1] + 16; i <= max; i += 16 {
		steps = append(steps, i)
	}
	return steps
}

func GetKeyStepWeight(w float64) string {
	steps := GetWeightStep()

	for i := 0; i <= len(steps)-1; i++ {
		if w < steps[i] {
			return fmt.Sprintf("%s_%v", RedisPrefixKeyPoint, i+1)
		}
	}
	return fmt.Sprintf("%s_%v", RedisPrefixKeyPoint, len(steps)+1)
}

func (c *CreateLabel) GetCarrierSize(in providers.RequestCreateLabel) string {
	if in.IsExceedPkg {
		return fmt.Sprintf("%s_%v", RedisPrefixKeyPoint, len(GetWeightStep())+1)
	}

	weight, length, height, width := in.Weight, in.Length, in.Height, in.Width
	volumeWeight := calculate.SizeToWeight(length, height, width)

	priceWeight := weight
	if volumeWeight > weight {
		priceWeight = volumeWeight
	}

	priceWeight = priceWeight * GramToOz
	return GetKeyStepWeight(priceWeight)
}

func (c *CreateLabel) GetCarrierCode(ctx context.Context, sp entity.Package, customerID int64, zone string) (string, error) {
	return providers.CarrierTypeKiloship, nil
	if zone == "" {
		log.Println("serviceCode: ", sp.Service.Code)
		switch sp.Service.Code {
		case constant.ServiceUS48Code:
			return providers.CarrierTypeDarius, nil
		case constant.ServiceINUSCode:
			return providers.CarrierTypeDarius, nil
		case constant.ServiceAUCode:
			return providers.CarrierTypeDarius, nil
		case constant.ServiceEUCode:
			return providers.CarrierTypeDarius, nil
		default:
			return providers.CarrierTypeIBBlue, nil
		}
	}

	body := providers.RequestCreateLabel{
		Weight:       sp.ActualWeight,
		Height:       sp.ActualHeight,
		Length:       sp.ActualLength,
		Width:        sp.ActualWidth,
		DistanceUnit: DistanceUnitIN,
		IsExceedPkg:  sp.IsPackageExceed,
	}

	if body.Weight <= 0 {
		body.Weight = sp.Weight
	}

	if body.Length <= 0 {
		body.Length = sp.Length
	}

	if body.Width <= 0 {
		body.Width = sp.Width
	}

	if body.Height <= 0 {
		body.Height = sp.Height
	}

	weight, length, height, width, err := c.Fake(ctx, body.Weight, body.Length, body.Height, body.Width)
	log.Println("fake: ", err)
	if err != nil {
		return "", err
	}

	body.Weight = weight
	body.Length = length
	body.Height = height
	body.Width = width

	sizePoint := ""
	if sp.IsPackageExceed {
		sizePoint = fmt.Sprintf("%s_%v", RedisPrefixKeyPoint, len(GetWeightStep())+1)
	} else {
		sizePoint = c.GetCarrierSize(body)
	}

	return c.GetRedisCarrierCode(ctx, sizePoint, zone)
}

func WeightToPointNumber(w float64) int {
	steps := GetWeightStep()

	for i := 0; i <= len(steps)-1; i++ {
		if w < steps[i] {
			return i + 1
		}
	}
	return len(steps) + 1
}
