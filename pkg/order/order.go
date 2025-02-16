package order

import (
	"context"
	"fmt"
	"math/rand"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"
	"time"

	"github.com/redis/go-redis/v9"
)

const GetOrCreateNowBillKey = "get_or_create_now_bill_key_"

func GetOrCreateNowBill(c context.Context, manager *sqlmanager.BillManager, rd *redis.Client, userID int64) (*entity.Bill, error) {
	key := fmt.Sprintf("%s%d", GetOrCreateNowBillKey, userID)

	defer func() {
		if err := rd.Del(c, key).Err(); err != nil {
			fmt.Errorf("remove redis: %v", err)
		}
	}()

	for i := 0; i < 3; i++ {
		ok, err := rd.Get(c, key).Int()
		if err == redis.Nil {
			if err := rd.Set(c, key, 1, time.Duration(30)*time.Second).Err(); err != nil {
				return nil, err
			}

			break
		}

		if err != nil {
			return nil, err
		}

		if ok > 0 {
			ran := rand.Intn(9)
			time.Sleep(time.Duration(ran) * time.Second)
			continue
		}

		if err := rd.Set(c, key, 1, time.Duration(30)*time.Second).Err(); err != nil {
			return nil, err
		}

		break
	}

	return manager.GetOrCreateNowBill(userID)
}
