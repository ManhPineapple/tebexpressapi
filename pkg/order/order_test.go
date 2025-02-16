package order

import (
	"context"
	"fmt"
	"tebexpressapi/pkg/config"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"

	"sync"
	"testing"
	"time"
)

func TestGetOrCreateNowBill(t *testing.T) {
	config.ReadConfigByFiles("toml", []string{"conf/conf.toml"})
	db, _ := storage.NewMysqlConnection(storage.DefaultMysqlFromConfig(nil))

	rd, _ := storage.NewRedisConnection(nil)

	manager := sqlmanager.NewBillManager(db)

	var wg sync.WaitGroup

	userID := int64(2527)

	date := time.Now().Add(7 * time.Hour).Format("2006-01-02")
	sdb := db.Where("user_id = ?", userID)
	sdb = sdb.Where("DATE_FORMAT(convert_tz(created_at, @@session.time_zone,'+07:00') ,'%Y-%m-%d %H:%i:%s') BETWEEN ?  AND ?", fmt.Sprintf("%s 00:00:00", date), fmt.Sprintf("%s 23:59:59", date))

	if err := sdb.Delete(&entity.Bill{}).Error; err != nil {
		t.Error(err)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			bill, err := GetOrCreateNowBill(context.Background(), manager, rd, userID)
			if err != nil {
				t.Error(err)
			}

			t.Logf("now bill id: %v", bill.ID)
		}()
	}

	wg.Wait()

	mysql, _ := db.DB()
	mysql.Close()
	rd.Close()
}
