package calculate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/utils"
	"tebexpressapi/pkg/utils/dbgorm"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const GramToPound = 0.00220462262185
const InchToCM = 2.54
const RedisKey = "service-prices"
const RedisPromotionPriceKey = "promotion_price"
const RedisPromotionWeightKey = "promotion_weight"
const MaxWeightPriceAllow = 19950.00
const RateVolumeOversize = 1.2
const KgToGram = 1000

var ErrorNotPrice = errors.New("not price")
var ErrorMaxWeight = errors.New("exceeds the weight")
var ErrorMaxVolume = errors.New("exceeds the volume")
var ErrorNotService = errors.New("not service")
var ErrorMinWeight = errors.New("min the weight")

type CalculatePrice struct {
	mu               sync.Mutex
	DB               *gorm.DB
	Redis            *redis.Client
	prices           []Price
	promotionPrices  []PromotionPrice
	promotionWeights []PromotionWeight
}

type PromotionPrice struct {
	ServiceID int64   `json:"service_id"`
	Weight    float64 `json:"weight"`
	Price     float64 `json:"price"`
}

type PromotionWeight struct {
	Weight           float64 `json:"weight"`
	ErrorWeightAllow float64 `json:"error_weight_allow"`
}

type Price struct {
	ServiceID int64   `json:"service_id"`
	Weight    float64 `json:"weight"`
	Price     float64 `json:"price"`
	UserClass int64   `json:"user_class"`
}

func New(db *gorm.DB, redis *redis.Client) *CalculatePrice {
	c := &CalculatePrice{DB: db, Redis: redis}

	var prices []Price
	err := c.DB.Table("prices").Order("service_id, weight ASC").Find(&prices).Error
	if err != nil {
		panic(err)
	}

	c.prices = prices
	return c
}

func (c *CalculatePrice) LoadPrices(ctx context.Context) error {
	// var prices []Price

	// str, err := c.Redis.Get(ctx, RedisKey).Result()
	// if err != nil && err != redis.Nil {
	// 	return err
	// }

	// if str != "" {
	// 	if err := json.Unmarshal([]byte(str), &prices); err != nil {
	// 		return err
	// 	}

	// 	c.prices = prices
	// 	return nil
	// }

	// err = c.DB.Table("prices").Order("service_id, weight ASC").Find(&prices).Error
	// if err != nil {
	// 	return err
	// }

	// if len(prices) > 0 {
	// 	b, err := json.Marshal(prices)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	log.Println(string(b))

	// 	if err := c.Redis.Set(ctx, RedisKey, string(b), 0).Err(); err != nil {
	// 		return err
	// 	}
	// }

	// c.prices = prices
	return nil
}

func (c *CalculatePrice) LoadPromotionPrices(ctx context.Context, userID int64) error {
	var prices = []PromotionPrice{}
	promotion := &entity.Promotion{}
	err := c.DB.Select("promotions.id").
		Joins(`
			INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id
				AND user_promotions.user_id=? AND user_promotions.status=?`, userID, constant.StatusActive).
		Where("promotions.status=? AND promotions.type=?", constant.PromotionStatusActive, constant.PromotionTypeMarketing).
		First(promotion).Error

	if err == gorm.ErrRecordNotFound {
		c.promotionPrices = prices
		return nil
	}

	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s_%d", RedisPromotionPriceKey, promotion.ID)
	str, err := c.Redis.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if str != "" {
		if err := json.Unmarshal([]byte(str), &prices); err != nil {
			return err
		}

		c.promotionPrices = prices
		return nil
	}

	if err := c.DB.Where("promotion_id=?", promotion.ID).Order("weight ASC").Find(&prices).Error; err != nil {
		return err
	}

	if len(prices) > 0 {
		b, err := json.Marshal(prices)
		if err != nil {
			return err
		}

		if err := c.Redis.Set(ctx, key, string(b), 0).Err(); err != nil {
			return err
		}
	}

	c.promotionPrices = prices
	return nil
}

func (c *CalculatePrice) sort() {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(c.prices)
	if n < 2 {
		return
	}

	mapkeys := make(map[int64][]float64)
	values := make(map[string]Price, n)

	for _, p := range c.prices {
		key := fmt.Sprintf("%v-%v", p.ServiceID, p.Weight)

		if mapkeys[p.ServiceID] == nil {
			mapkeys[p.ServiceID] = []float64{}
		}

		mapkeys[p.ServiceID] = append(mapkeys[p.ServiceID], p.Weight)
		values[key] = p
	}

	i := 0
	for id, keys := range mapkeys {
		sort.Float64s(keys)

		for _, k := range keys {
			key := fmt.Sprintf("%v-%v", id, k)
			c.prices[i] = values[key]
			i++
		}
	}
}

func (c *CalculatePrice) basePrice(weight float64, serviceID int64, userClass int64) (float64, error) {
	if weight <= 0 || (len(c.prices) == 0 && len(c.promotionPrices) == 0) {
		return 0, ErrorNotService
	}

	var max float64 = 0
	var isNotPromotionService bool = true
	if len(c.promotionPrices) > 0 {
		for _, p := range c.promotionPrices {
			if serviceID == p.ServiceID {
				isNotPromotionService = false

				if weight < p.Weight {
					return p.Price, nil
				}

				max = p.Weight
			}
		}
	}

	if isNotPromotionService {
		for _, p := range c.prices {
			if serviceID == p.ServiceID && userClass == p.UserClass {
				log.Println("weight: ", weight, p.Weight)
				if weight < p.Weight {
					return p.Price, nil
				}

				max = p.Weight
			}
		}
	}

	if max == 0 {
		return 0, ErrorNotService
	}

	return max, ErrorNotPrice
}

// Price trả về giá gốc, phí phát sinh, lỗi
func (c *CalculatePrice) Price(ctx context.Context, serviceID int64, userClass int64, weight, length, height, width float64, country string) (float64, float64, error) {
	if err := c.LoadPrices(ctx); err != nil {
		return 0, 0, err
	}

	length, height, width = ParseVolumes(length, height, width)
	wp, bw := CalcPriceWeight(weight, length, height, width, serviceID)
	if wp < MaxWeightPriceAllow && country != "AU" {
		if length > constant.PackageMaxLength {
			return 0, 0, ErrorMaxVolume
		}
	}

	base, err := c.basePrice(wp, serviceID, userClass)
	if err == nil {
		return base, 0, nil
	}

	if err == ErrorNotService {
		return 0, 0, err
	}

	if base < 0.001 {
		base = 0
	}

	if bw {
		return base, 0, ErrorMaxWeight
	}

	numberOver74 := 0
	numberOver54 := 0
	numberOver64 := 0
	if length > 74 {
		numberOver74 += 1
	} else if length > 64 {
		numberOver64 += 1
	} else if length > 54 {
		numberOver54 += 1
	}

	if width > 74 {
		numberOver74 += 1
	} else if width > 64 {
		numberOver64 += 1
	} else if width > 54 {
		numberOver54 += 1
	}

	if height > 74 {
		numberOver74 += 1
	} else if height > 64 {
		numberOver64 += 1
	} else if height > 54 {
		numberOver54 += 1
	}

	var priceOutSize float64
	if numberOver74 >= 2 {
		priceOutSize = 52
	} else if numberOver74 >= 1 {
		priceOutSize = 22
	} else if numberOver64 >= 2 {
		priceOutSize = 12
	} else if numberOver64 >= 1 {
		priceOutSize = 8
	} else if numberOver54 >= 1 {
		priceOutSize = 4
	}

	if base < 0.001 {
		base = 0
	}
	return utils.Ceil(base, 2), priceOutSize, nil
}

// TODO max weight lay tu database
// Price2 trả về giá gốc, phí phát sinh, lỗi
func (c *CalculatePrice) Price2(ctx context.Context, serviceID int64, userClass int64, weight, length, height, width float64, country string) (float64, float64, error) {
	if err := c.LoadPrices(ctx); err != nil {
		return 0, 0, err
	}

	length, height, width = ParseVolumes(length, height, width)
	wp, bw := CalcPriceWeight(weight, length, height, width, serviceID)
	if wp < MaxWeightPriceAllow && country != "AU" {
		if length > constant.PackageMaxLength {
			return 0, 0, ErrorMaxVolume
		}
	}

	base, err := c.basePrice(wp, serviceID, userClass)
	if err != nil {
		if err != ErrorNotPrice {
			return 0, 0, err
		}

		if base == 0 {
			return 0, 0, ErrorNotService
		}

		if base < 0.001 {
			base = 0
		}

		if bw {
			return base, 0, ErrorMaxWeight
		}
		return base, 0, ErrorMaxVolume
	}

	if base < 0.001 {
		base = 0
	}

	return base, 0, nil
}

// Bổ sung tính năng check quá cỡ
// Có 1 cạnh >54cm => thu thêm 4$
// Có 1 cạnh > 64cm => thu thêm 8$
// Có 2 cạnh > 64cm => thu thêm 12$
// Có 1 cạnh > 74cm => thu thêm 10$
// Có 2 cạnh > 74cm => thu thêm 30$
func (c *CalculatePrice) Price3(ctx context.Context, userID, serviceID, userClass int64, weight, length, height, width float64, country string) (float64, float64, error) {
	if err := c.LoadPromotionPrices(ctx, userID); err != nil {
		return 0, 0, err
	}

	if err := c.LoadPrices(ctx); err != nil {
		return 0, 0, err
	}

	length, height, width = ParseVolumes(length, height, width)
	wp, bw := CalcPriceWeight(weight, length, height, width, serviceID)

	base, err := c.basePrice(wp, serviceID, userClass)
	if err != nil {
		if err != ErrorNotPrice {
			return 0, 0, err
		}

		if base == 0 {
			return 0, 0, ErrorNotService
		}

		if base < 0.001 {
			base = 0
		}

		if bw {
			return utils.Ceil(base, 2), 0, ErrorMaxWeight
		}
		return utils.Ceil(base, 2), 0, ErrorMaxVolume
	}

	if base < 0.001 {
		base = 0
	}
	return utils.Ceil(base, 2), 0, nil
}

func (c *CalculatePrice) Discount(userID int64, weight float64) (float64, error) {
	price, err := c.PromotionPriceByWeight(userID)
	if err != nil {
		return 0, err
	}

	var discount float64 = 0
	if price > 0 && weight > constant.MaxWeightShippingPackage {
		discount = price * weight / 1000
	}

	return utils.Ceil(discount, 2), nil
}

func (c *CalculatePrice) CalculateExceedPackagePrice(weight float64, cost float64) (float64, error) {
	rate := viper.GetFloat64("fee.exceed_package")
	if rate == 0 {
		return 0, errors.New("config rate exceed fee not found")
	}
	fee := rate * weight / 1000
	return utils.Ceil(1.1*(cost+fee), 2), nil
}

func (c *CalculatePrice) GetFbaPackagePrice(price, weight, length, height, width float64) float64 {
	log.Println("before parse: ", length, height, width)
	length, height, width = ParseVolumes(length, height, width)
	log.Println("parse: ", length, height, width)
	wp, _ := CalcFBAPriceWeight(weight, length, height, width)
	log.Println("CalcFBAPriceWeight: ", wp)
	if float64(int(wp)) > wp {
		wp = float64(int(wp) + 1)
	}
	totalPrice := wp * price / constant.KgToGram
	return utils.Ceil(totalPrice, 2)
}

func (c *CalculatePrice) GetExtraFeeBaterry() float64 {
	totalPrice := viper.GetFloat64("fee.battery")
	return utils.Ceil(totalPrice, 2)
}

func (c *CalculatePrice) GetRatePriceByTotalWeight(ctx context.Context, weight float64, serviceID, userClass int64) (float64, float64, error) {
	log.Println("GetRatePriceByTotalWeight: ", weight, " - ", serviceID, " - ", userClass)
	if err := c.LoadPrices(ctx); err != nil {
		return 0, 0, err
	}

	if len(c.prices) < 1 {
		return 0, 0, ErrorNotPrice
	}

	var checkFirstLoop = false
	var max float64
	for _, p := range c.prices {
		if serviceID == p.ServiceID && userClass == p.UserClass {
			if !checkFirstLoop {
				checkFirstLoop = true
				if weight < p.Weight {
					return 0, 0, nil
				}
			}

			if weight < p.Weight {
				return p.Price, 0, nil
			}

			max = p.Weight
		}
	}

	if max > 0 {
		return 0, max, nil
	}

	return 0, 0, ErrorNotService
}

func SizeToWeight(length, height, width float64) float64 {
	return (length * height * width) / 5
}

func AUBSizeToWeight(length, height, width float64) float64 {
	return (length * height * width) / 4
}

// CalcPriceWeight khối lượng tính cước
// "GIÁ CƯỚC TẠM TÍNH DỰA TRÊN ""KHỐI LƯỢNG TÍNH CƯỚC"".
// Khối lượng tính cước thuộc khoảng nào thì tính giá cước dựa trên khoảng đó.
// Làm thế nào để xác định ""KHỐI LƯỢNG TÍNH CƯỚC"".
// DỰA TRÊN phép so sánh: Khối lượng thực (KG) và Khối lượng thể tích: DÀI*RỘNG*CAO/5000 (CM)"
// DỰA TRÊN phép so sánh: Khối lượng thực (KG) và Khối lượng thể tích: DÀI*RỘNG*CAO/4000 (CM) đối với AUB"
// 1. Khối lượng thực tế > Khối lượng thể tích => khối lượng tính cước là KHỐI LƯỢNG THỰC TẾ
// 2. Khối lượng thực tế < Khối lượng thể tích => Khối lượng tính cước là KHỐI LƯỢNG THỂ TÍCH
// params weight unit grams, length & height & width unit centimeter
// return float (unit grams)
func CalcPriceWeight(weight, length, height, width float64, serviceID int64) (float64, bool) {
	wv := SizeToWeight(length, height, width)
	// if serviceID == 18 { // AUB service
	// 	wv = AUBSizeToWeight(length, height, width)
	// }
	if wv > weight {
		return wv, false
	}

	return weight, true
}

func CalcFBAPriceWeight(weight, length, height, width float64) (float64, bool) {
	wv := SizeToWeight(length, height, width)
	wv = cast.ToFloat64(int(math.Ceil(wv / 1000)))
	wv = wv * 1000
	if wv > weight {
		return wv, false
	}

	return weight, true
}

// CalcPriceOutSize phụ phí quá cỡ
// L + (2xH) + (2xW) < 50 inch: Không có phụ phí
// L + (2xH) + (2xW) < 50 inch: Phụ phí theo công thức
// L*H*W/166 Rồi làm tròn lên con số gần nhất, sau đó *$1.5
// params length & height & width unit centimeter
// return float (USD)
func CalcPriceOutSize(l, h, w float64) float64 {
	l = l / InchToCM
	h = h / InchToCM
	w = w / InchToCM

	p := (l + 2*(h+w))
	if p < 50 {
		return 0
	}

	r := (l * h * w) / 166
	price := 1.5 * Ceil(r, 0)
	return price
}

// ParseVolumes phan tich volume nhap vao
// để tránh trường hợp tính giá quá cỡ
// voi gia tri nho nhat la height
// gia tri lon nhat la length
// con lai la width
func ParseVolumes(l, h, w float64) (length, height, width float64) {
	length = l
	height = h
	width = w

	if length < height {
		height, length = length, height
	}

	if width < height {
		height, width = width, height
	}

	if length < width {
		width, length = length, width
	}

	return
}

func Ceil(v float64, d int) float64 {
	e := math.Pow10(d)
	return math.Ceil(v*e) / e
}

func Round(v float64) int64 {
	return int64(math.Round(v))
}

func Floor(v float64, d int) float64 {
	e := math.Pow10(d)
	return math.Floor(v*e) / e
}

// PeakFee tính phí cao điểm
func PeakFee(weight float64) float64 {
	rate := viper.GetFloat64("extra_fees.peak_fee")
	if rate < 0 {
		return 0
	}

	return rate
}

func (c *CalculatePrice) LoadPromotionWeights(ctx context.Context, userID int64) error {
	var prices = []PromotionWeight{}

	promotion := &entity.Promotion{}
	err := c.DB.Select("promotions.id").
		Joins("INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id AND user_promotions.user_id=? AND user_promotions.status=?", userID, constant.StatusActive).
		Where("promotions.status=? AND promotions.type=?", constant.PromotionStatusActive, constant.PromotionTypeMarketing).
		First(promotion).Error

	if err == gorm.ErrRecordNotFound {
		c.promotionWeights = prices
		return nil
	}

	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s_%d", RedisPromotionWeightKey, promotion.ID)
	str, err := c.Redis.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if str != "" {
		if err := json.Unmarshal([]byte(str), &prices); err != nil {
			return err
		}

		c.promotionWeights = prices
		return nil
	}

	if err := c.DB.Where("promotion_id=?", promotion.ID).Order("weight ASC").Find(&prices).Error; err != nil {
		return err
	}

	if len(prices) > 0 {
		b, err := json.Marshal(prices)
		if err != nil {
			return err
		}

		if err := c.Redis.Set(ctx, key, string(b), 0).Err(); err != nil {
			return err
		}
	}

	c.promotionWeights = prices
	return nil
}

func (c *CalculatePrice) PromotionFixWeight(ctx context.Context, userID int64, weight float64) (float64, error) {
	if err := c.LoadPromotionWeights(ctx, userID); err != nil {
		return 0, err
	}

	if len(c.promotionWeights) > 0 {
		var max float64 = 0
		for _, p := range c.promotionWeights {
			if weight < p.Weight {
				return weight - p.ErrorWeightAllow, nil
			}

			max = p.ErrorWeightAllow
		}

		return weight - max, nil
	}

	var count int64
	db := c.DB.Model(&entity.UserPromotion{}).Where("promotions.status=?", constant.StatusActive).Limit(1)
	db = db.Joins(`
			INNER JOIN promotions ON promotions.id=user_promotions.promotion_id
				AND user_promotions.user_id=?
				AND user_promotions.promotion_id=?
				AND user_promotions.status=? `, userID, constant.PromotionFixWeightID, constant.StatusActive)
	db = db.Count(&count)

	if err := db.Error; err != nil {
		return 0, err
	}

	if count > 0 {
		return BasePromotionFixWeight(weight), nil
	}

	return weight, nil
}

func BasePromotionFixWeight(weight float64) float64 {
	if weight >= 300 {
		return weight - 7
	}

	return weight - 5
}

func (c *CalculatePrice) PromotionPriceByWeight(userID int64) (float64, error) {
	promotion := &entity.Promotion{}

	err := c.DB.Select("promotions.id, promotions.price").
		Joins(`INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id AND user_promotions.user_id=? AND user_promotions.status=?`, userID, constant.StatusActive).
		Where("promotions.id=? AND promotions.status=?", constant.PromotionPriceByWeight, constant.PromotionStatusActive).
		First(promotion).Error

	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}

	if err != nil {
		return 0, err
	}

	return promotion.Price, nil
}

func (c *CalculatePrice) PromotionExtras(pkg *entity.Package, extraFees []entity.ExtraFee, price float64) ([]entity.ExtraFee, error) {
	fees := []entity.ExtraFee{}

	currentFees := extraFees
	if len(currentFees) < 1 && pkg.ID > 0 {
		items := []entity.ExtraFee{}
		db := c.DB.Where("package_id=? AND extra_fees.status=? ", pkg.ID, constant.ExtraFeeStatusEnable)
		db = db.Joins("LEFT JOIN extra_fee_types ON extra_fee_types.id = extra_fees.extra_fee_type_id")
		db = db.Where("extra_fee_types.status=?", constant.ExtraFeeStatusEnable)
		db = db.Find(&items)

		if err := db.Error; err != nil {
			return fees, err
		}

		currentFees = items
	}

	var oversizeExtraFeeAmount float64 = 0
	// var discountExtraFeeAmount float64 = 0
	for _, fee := range currentFees {
		if fee.ExtraFeeTypeID == constant.ExtraFeeTypeOversize {
			oversizeExtraFeeAmount += fee.Amount
		}
	}

	// extra fee discount
	if pkg.Status == constant.PackageStatusCreated && pkg.Service.Code != constant.ServiceFBACode {
		discount, err := c.Discount(pkg.UserID, pkg.Weight)
		if err != nil {
			return fees, err
		}

		if discount > 0 {
			now := time.Now()
			fees = append(fees, entity.ExtraFee{
				Model: dbgorm.Model{
					CreatedAt: now,
					UpdatedAt: now,
				},
				PackageID:      utils.Int64(pkg.ID),
				ExtraFeeTypeID: constant.ExtraFeeTypeDiscount,
				Amount:         -discount,
				Status:         constant.ExtraFeeStatusEnable,
			})
		}
	}

	// end extra fee discount

	// extra fee promotion oversize
	promotion := &entity.Promotion{}
	err := c.DB.Select("promotions.id, promotions.price").
		Joins(`INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id AND user_promotions.user_id=? AND user_promotions.status=?`, pkg.UserID, constant.StatusActive).
		Where("promotions.id=? AND promotions.status=?", constant.PromotionOversizeFee, constant.PromotionStatusActive).
		First(promotion).Error

	if err == gorm.ErrRecordNotFound || pkg.Service.Code == constant.ServiceFBACode {
		return fees, nil
	}

	if err != nil {
		return fees, err
	}

	var weight float64 = pkg.ActualWeight
	var length float64 = pkg.ActualLength
	var height float64 = pkg.ActualHeight
	var width float64 = pkg.ActualWidth

	if weight <= 0 {
		weight = pkg.Weight
	}

	if length <= 0 {
		length = pkg.Length
	}

	if height <= 0 {
		height = pkg.Height
	}

	if width <= 0 {
		width = pkg.Width
	}

	vw := SizeToWeight(length, height, width)
	if vw <= weight*RateVolumeOversize {
		return fees, nil
	}

	extraFee := &entity.ExtraFeeType{}
	if err := c.DB.Where("id=? AND status=?", constant.ExtraFeeTypeOversize, constant.ExtraFeeStatusEnable).First(extraFee).Error; err != nil {
		return fees, err
	}

	var amount float64 = 0.15 * price
	if pkg.Status != constant.PackageStatusCreated {
		amount = amount - oversizeExtraFeeAmount
	}

	if amount <= 0 {
		return fees, nil
	}

	now := time.Now()
	fees = append(fees, entity.ExtraFee{
		Model: dbgorm.Model{
			CreatedAt: now,
			UpdatedAt: now,
		},
		PackageID:      utils.Int64(pkg.ID),
		ExtraFeeTypeID: constant.ExtraFeeTypeOversize,
		Amount:         utils.Ceil(amount, 2),
		Description:    extraFee.Name,
		Status:         constant.ExtraFeeStatusEnable,
	})

	return fees, nil
}

func LengthAndGirth(length, height, width float64) float64 {
	l, h, w := ParseVolumes(length, height, width)

	return 2*(h+w) + l
}

var maxWeight = constant.PackageMaxWeight / GramToPound
var maxLengthAndGirth = InchToCM * constant.PackageMaxLengthAndGirth

func ValidateWeight(weight float64, isEN bool) (msgWeight string) {
	if weight > maxWeight {
		max := utils.Floor(maxWeight, 2)
		msgWeight = fmt.Sprintf("Trọng lượng không vượt quá %v gram", max)
		if isEN {
			msgWeight = fmt.Sprintf("The weight must be less than or equal to %v gram", max)
		}
	}

	return
}

func ValidateLengthAndGirth(length, height, width float64, isEN bool) (msgLengthAndGirth string) {
	l, h, w := ParseVolumes(length, height, width)
	lag := LengthAndGirth(l, h, w)
	if lag > maxLengthAndGirth {
		max := utils.Floor(maxLengthAndGirth, 2)
		msgLengthAndGirth = fmt.Sprintf("“Dài + 2*(cao + rộng)” không vượt quá %v cm", max)
		if isEN {
			msgLengthAndGirth = fmt.Sprintf("“2*(weight + width) + length” must be less than or equal to %v cm", max)
		}
	}

	return
}

func ValidateDimension(weight, length, height, width float64, isEN bool) (msgLength string, msgHeight string, msgWidth string) {
	pw, _ := CalcPriceWeight(weight, length, height, width, 0)

	var max float64 = constant.PackageMaxLength
	if pw >= MaxWeightPriceAllow {
		max = constant.PackageMaxLengthOversize
	} else {
		l, _, _ := ParseVolumes(length, height, width)
		if l > constant.PackageMaxLength {
			max = constant.PackageMaxLengthOversize
		}
	}

	if length > max {
		msgLength = fmt.Sprintf("Chiều dài không được vượt quá %v cm", max)
		if isEN {
			msgLength = fmt.Sprintf("The length must be less than or equal to %v cm", max)
		}
	}

	if height > max {
		msgHeight = fmt.Sprintf("Chiều cao không được vượt quá %v cm", max)
		if isEN {
			msgHeight = fmt.Sprintf("The height must be less than or equal to %v cm", max)
		}
	}

	if width > max {
		msgWidth = fmt.Sprintf("Chiều rộng không được vượt quá %v cm", max)
		if isEN {
			msgWidth = fmt.Sprintf("The width must be less than or equal to %v cm", max)
		}
	}

	return
}

func ValidateVolumes(weight, length, height, width float64, isEN bool) []string {
	var messages []string

	msgWeight := ValidateWeight(weight, isEN)
	if msgWeight != "" {
		messages = append(messages, msgWeight)
	}

	msgLengthAndGirth := ValidateLengthAndGirth(length, height, width, isEN)
	if msgLengthAndGirth != "" {
		messages = append(messages, msgLengthAndGirth)
	}

	msgLength, msgHeight, msgWidth := ValidateDimension(weight, length, height, width, isEN)
	if msgLength != "" {
		messages = append(messages, msgLength)
	}
	if msgHeight != "" {
		messages = append(messages, msgHeight)
	}

	if msgWidth != "" {
		messages = append(messages, msgWidth)
	}

	return messages
}

func (c *CalculatePrice) CheckIsExceed(ctx context.Context, userID, serviceID, userClass int64, weight, length, height, width float64) (bool, error) {
	if err := c.LoadPromotionPrices(ctx, userID); err != nil {
		return false, err
	}

	if err := c.LoadPrices(ctx); err != nil {
		return false, err
	}

	length, height, width = ParseVolumes(length, height, width)
	if length > constant.PackageMaxLength {
		return true, nil
	}

	wp, _ := CalcPriceWeight(weight, length, height, width, serviceID)
	base, err := c.basePrice(wp, serviceID, userClass)
	if err == nil {
		return false, nil
	}

	if err != ErrorNotPrice {
		return false, err
	}

	if base == 0 {
		return false, ErrorNotService
	}

	return true, nil
}

func (c *CalculatePrice) PromotionInsured(userID int64) (bool, float64, error) {
	promotion := &entity.Promotion{}
	err := c.DB.Select("promotions.id, promotions.price").
		Joins(`INNER JOIN user_promotions ON user_promotions.promotion_id=promotions.id AND user_promotions.user_id=? AND user_promotions.status=?`, userID, constant.StatusActive).
		Where("promotions.id=? AND promotions.status=?", constant.PromotionInsured, constant.PromotionStatusActive).
		First(promotion).Error

	if err == gorm.ErrRecordNotFound {
		return false, 0, nil
	}

	if err != nil {
		return false, 0, err
	}

	return true, promotion.Price, nil
}

func (c *CalculatePrice) GetServiceExtraFeee(w, h, l float64, service entity.Service) float64 {
	if w > constant.LimitExtraFee2 || h > constant.LimitExtraFee2 || l > constant.LimitExtraFee2 {
		return service.ExtraFee_2
	}

	if w > constant.LimitExtraFee1 || h > constant.LimitExtraFee1 || l > constant.LimitExtraFee1 {
		return service.ExtraFee_1
	}
	return 0
}
