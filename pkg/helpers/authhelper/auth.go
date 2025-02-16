package authhelper

import (
	"context"
	"fmt"
	"tebexpressapi/pkg/constant"

	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/sqlmanager"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func CheckCancelLimit(c context.Context, ui *entity.UserInfo, redisDriver *redis.Client) (string, error) {
	var cancelMaxAmount float64 = constant.DefaultCancelMaxAMount
	if ui != nil {
		cancelMaxAmount = ui.CancelMaxAmount
	}

	rKeyCancel := fmt.Sprintf("%s_%d", "cancel_amount", ui.UserID)
	cancelAmount, err := redisDriver.Get(c, rKeyCancel).Float64()
	if err != nil && err != redis.Nil {
		fmt.Errorf("get redis : %s", err)
		return "", err
	}

	if cancelAmount > cancelMaxAmount {
		return "Account exceeds order creation limit", nil
	}

	return "", nil
}

func CheckCancelLimit2(c context.Context, userID int64, manager *sqlmanager.UserManager, redisDriver *redis.Client) (string, error) {
	ui, err := manager.GetUserInfoByUserID(userID)
	if err != nil && err != gorm.ErrRecordNotFound {
		fmt.Errorf("get redis : %s", err)
		return "", err
	}

	var cancelMaxAmount float64 = constant.DefaultCancelMaxAMount
	if ui != nil {
		cancelMaxAmount = ui.CancelMaxAmount
	}

	rKeyCancel := fmt.Sprintf("%s_%d", "cancel_amount", ui.UserID)
	cancelAmount, err := redisDriver.Get(c, rKeyCancel).Float64()
	if err != nil && err != redis.Nil {
		fmt.Errorf("get redis : %s", err)
		return "", err
	}

	if cancelAmount > cancelMaxAmount {
		return "Account exceeds order creation limit", nil
	}

	return "", nil
}
