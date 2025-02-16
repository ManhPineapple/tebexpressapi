package packages

import (
	"context"
	"encoding/json"
	"tebexpressapi/pkg/calculate"
	"tebexpressapi/pkg/createlabel"

	"github.com/redis/go-redis/v9"
)

const RedisKeyUSEstimateDelivery = "us_estimate_delivery_logs"
const RedisKeyAUEstimateDelivery = "au_estimate_delivery"

func EstimateDelivery(c context.Context, r *redis.Client, cl *createlabel.CreateLabel, country string, zone int, weight, length, height, width float64, serviceID int64) (float64, error) {
	if country == "AU" {
		estimate, err := r.Get(c, RedisKeyAUEstimateDelivery).Float64()
		if err != nil {
			return 0, err
		}

		return estimate, nil
	}

	res, err := r.Get(c, RedisKeyUSEstimateDelivery).Result()
	if err != nil {
		return 0, err
	}

	result := make(map[int]map[int]float64)
	if err := json.Unmarshal([]byte(res), &result); err != nil {
		return 0, err
	}

	if len(result) == 0 {
		return 0, nil
	}

	we, l, h, w, err := cl.Fake(c, weight, length, height, width)
	if err != nil {
		return 0, err
	}

	wp, _ := calculate.CalcPriceWeight(we, l, h, w, serviceID)
	point := createlabel.WeightToPointNumber(wp * createlabel.GramToOz)

	if len(result[point]) == 0 {
		return 0, nil
	}

	return result[point][zone], nil
}
