package calculate

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"log"
	"os"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/storage"
	"tebexpressapi/pkg/utils/dbgorm"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/cast"
	"github.com/spf13/viper"

	"gorm.io/gorm"
)

func db() *gorm.DB {
	db, err := storage.NewMysqlConnection(&storage.MysqlConfiguration{
		Host:      "localhost",
		Port:      3303,
		Username:  "dev",
		Password:  "$7=SgXa]5(j(dLAs",
		Database:  "shipment_dev",
		Charset:   "utf8",
		ParseTime: true,
		Debug:     true,
	})

	if err != nil {
		log.Panic(err)
	}

	return db
}

func rd() *redis.Client {
	b, err := os.ReadFile(viper.GetString("redis.secret"))
	if err != nil {
		return nil
	}

	var data map[string]interface{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return nil
	}

	ro := &redis.Options{
		Addr:     viper.GetString("redis.addr"),
		Password: cast.ToString(data["redis_password"]),
		DB:       0,
	}

	if viper.GetBool("redis.is_tls") {
		ro.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	rdb := redis.NewClient(ro)
	return rdb
}

func TestPriceMax(t *testing.T) {
	manager := New(db(), rd())
	manager.LoadPrices(context.Background())

	price, extraFee, err := manager.Price(context.Background(), 11, 5000, 20, 60, 15, 10, "")
	t.Logf("price: %v, extraFee: %v", price, extraFee)
	if err == ErrorMaxVolume {
		t.Errorf("Cân nặng vượt quá %v grams", price-1)
	}

	if err == ErrorMaxWeight {
		t.Errorf("Cân nặng vượt quá %v grams", price-1)
	}

	if err != nil {
		t.Error(err)
	}
}

func TestPrice(t *testing.T) {
	manager := New(db(), rd())
	manager.LoadPrices(context.Background())

	price, extraFee, err := manager.Price(context.Background(), 11, 50, 1, 1, 1, 1, "")
	if err != nil {
		t.Error(err)
	}

	if price <= 0 {
		t.Error("price is zero")
	}

	t.Logf("price: %v, extraFee: %v", price, extraFee)

	price, extraFee, err = manager.Price(context.Background(), 11, 150, 1, 1, 1, 1, "")
	if err != nil {
		t.Error(err)
	}

	t.Logf("price: %v, extraFee: %v", price, extraFee)

	if price <= 0 {
		t.Error("price is zero")
	}

	t.Logf("price: %v", price)

	price, extraFee, err = manager.Price(context.Background(), 10, 50, 1, 1, 1, 1, "")
	if err != nil {
		t.Error(err)
	}

	if price <= 0 {
		t.Error("price is zero")
	}

	t.Logf("price: %v, extraFee: %v", price, extraFee)
}

func TestPrice2(t *testing.T) {
	manager := New(db(), rd())

	price, extraFee, err := manager.Price2(context.Background(), 2, 100, 25, 25, 30, 10, "")
	if err != nil {
		t.Error(err)
	}

	if price <= 0 {
		t.Error("price is zero")
	}

	t.Logf("price: %v, extra: %v", price, extraFee)

	t.Logf("price: %v, extraFee: %v", price, extraFee)
}

func TestPriceWeight(t *testing.T) {
	var (
		weight float64 = 100 // unit grams
		lenght float64 = 20  // unit centimeter
		height float64 = 2   // unit centimeter
		width  float64 = 10  // unit centimeter
	)

	wp, _ := CalcPriceWeight(weight, lenght, height, width, 0)
	if wp != 100 {
		t.Error("Incorrect calculate price weight (weight = 100, lenght = 20cm, height = 2cm, width = 10cm)")
	}

	t.Logf("calculate price weight (100, 20, 2, 10): %v", wp)

	weight = 70
	wp, _ = CalcPriceWeight(weight, lenght, height, width, 0) // 20*2*10/5 = 80 > 70
	if wp != 80 {
		t.Error("Incorrect calculate price weight (weight = 70g, lenght = 20cm, height = 2cm, width = 10cm)")
	}

	t.Logf("calculate price weight(70, 20, 2, 10): %v", wp)
}

func TestPriceOutSize(t *testing.T) {
	var (
		lenght float64 = 20 // unit centimeter
		height float64 = 2  // unit centimeter
		width  float64 = 10 // unit centimeter
	)

	p := CalcPriceOutSize(lenght, height, width)
	t.Logf("calculate price out size (20, 2, 10): %v", p)
	if p != 0 {
		t.Error("Incorrect calculate price out size (lenght = 20cm, height = 2cm, width = 10cm)")
	}

	lenght = 60
	height = 15
	width = 30

	r := (lenght / InchToCM * height / InchToCM * width / InchToCM) / 166
	outprice := Ceil(1.5*Ceil(r, 2), 2)

	p = CalcPriceOutSize(lenght, height, width)
	t.Logf("calculate price out size (60, 15, 30): %v", p)
	if p != outprice {
		t.Error("Incorrect calculate price out size (lenght = 60cm, height = 15cm, width = 30cm)")
	}
}

func TestParseVolumes(t *testing.T) {
	l, h, w := ParseVolumes(3, 4, 5)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}

	l, h, w = ParseVolumes(3, 5, 4)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}

	l, h, w = ParseVolumes(5, 4, 3)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}

	l, h, w = ParseVolumes(5, 3, 4)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}

	l, h, w = ParseVolumes(4, 3, 5)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}

	l, h, w = ParseVolumes(4, 5, 3)
	if l != 5 || h != 3 || w != 4 {
		t.Error("Incorrect parse")
	}
}

func TestCeil(t *testing.T) {
	if Ceil(7.22, 0) != 8 {
		t.Error("Ceil fail")
	}

	if Ceil(7.72, 0) != 8 {
		t.Error("Ceil fail")
	}

	if Ceil(-7.72, 0) != -7 {
		t.Error("Ceil fail")
	}

	if Ceil(-7.12, 0) != -7 {
		t.Error("Ceil fail")
	}
}

func TestRound(t *testing.T) {
	if Round(7.12) != 7 {
		t.Errorf("Ceil fail %v: %v", 7.12, Round(7.12))
	}

	if Round(7.72) != 8 {
		t.Errorf("Ceil fail %v: %v", 7.72, Round(7.72))
	}

	if Round(-7.12) != -7 {
		t.Errorf("Ceil fail %v: %v", -7.72, Round(-7.72))
	}

	if Round(-7.72) != -8 {
		t.Errorf("Ceil fail %v: %v", -7.72, Round(-7.72))
	}
}

func TestFloor(t *testing.T) {
	if Floor(7.12, 0) != 7 {
		t.Error("Ceil fail")
	}

	if Floor(7.72, 0) != 7 {
		t.Error("Ceil fail")
	}

	if Floor(-7.72, 0) != -8 {
		t.Error("Ceil fail")
	}

}

func TestLoadPromotionWeights(t *testing.T) {
	manager := New(db(), rd())
	w, err := manager.PromotionFixWeight(context.Background(), 2514, 201)
	if err != nil {
		t.Log(err)
	}

	t.Logf("prices: %v", w)
	t.Logf("prices: %v", manager.promotionWeights)
}

func TestPromotionExtraFees(t *testing.T) {
	manager := New(db(), rd())

	p := &entity.Package{
		Model: dbgorm.Model{
			ID: 125947,
		},
		Weight:       1800,
		Height:       9,
		Length:       40,
		Width:        30,
		ActualWeight: 1800,
		ActualHeight: 9,
		ActualLength: 40,
		ActualWidth:  30,
		UserID:       2527,
	}

	price, _, err := manager.Price3(context.Background(), p.UserID, 12, 1, p.Weight, p.Length, p.Height, p.Width, "")
	if err != nil {
		t.Log(err)
	}

	fees, err := manager.PromotionExtras(p, nil, price)
	if err != nil {
		t.Log(err)
	}

	b, err := json.Marshal(fees)
	if err != nil {
		t.Log(err)
	}

	t.Logf("extra fees: %v", string(b))
}
