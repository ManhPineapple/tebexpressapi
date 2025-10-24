package ocrtiktoklabelscheduler

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tebexpressapi/pkg/config"
	"tebexpressapi/pkg/logger"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const DefaultTicker = 120 * time.Minute

type OcrTiktokLabelScheduler struct {
	logger    *zap.SugaredLogger
	redisConn *redis.Client
	mysqlConn *gorm.DB
}

func NewApp(configFile string) *OcrTiktokLabelScheduler {
	app := &OcrTiktokLabelScheduler{
		logger: logger.InitLogger(),
	}

	if len(configFile) > 0 {
		configFiles := strings.Split(configFile, ";")
		err := config.ReadConfigByFiles("toml", configFiles)
		if err != nil {
			app.logger.Panic("initConfig: Could not load conf: ", err)
		}
		app.logger.Info("init config")
	}

	mysqlConn, err := storage.NewMysqlConnection(storage.DefaultMysqlFromConfig(nil))
	if err != nil {
		app.logger.Panic(err)
	}

	redisConn, err := storage.NewRedisConnection(storage.DefaultRedisFromConfig(nil))
	if err != nil {
		app.logger.Panic(err)
	}

	app.mysqlConn = mysqlConn
	app.redisConn = redisConn
	return app
}

func (app *OcrTiktokLabelScheduler) Run() {
	ctx := context.Background()

	packageManager := sqlmanager.NewPackageManager(app.mysqlConn)
	trackingManager := sqlmanager.NewTrackingManager(app.mysqlConn)
	warehouseManager := sqlmanager.NewWareHouseManager(app.mysqlConn)

	handler := NewHandler(ctx, app.logger, app.redisConn, packageManager, trackingManager, warehouseManager)

	cronTime := DefaultTicker
	cron := viper.GetInt64("ocr_tiktok_label.cron")
	if cron > 0 {
		cronTime = time.Duration(cron) * time.Minute
	}

	ticker := time.NewTicker(cronTime)
	defer ticker.Stop()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	keepRunning := true

	handler.Process()

	for keepRunning {
		select {
		case <-ticker.C:
			handler.Process()
		case <-sigterm:
			app.logger.Info("terminating: via signal")
			keepRunning = false
		}
	}
}

func (app *OcrTiktokLabelScheduler) Stop() {
	mysql, _ := app.mysqlConn.DB()
	mysql.Close()
	_ = app.redisConn.Close()

	log.Println("OCR scheduler exiting")
}
