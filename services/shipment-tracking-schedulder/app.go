package shipmenttrackingschedulder

import (
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

const DefaultTicker = 1 * time.Minute

type ShipmentTrackingScheduler struct {
	logger *zap.SugaredLogger

	redisConn *redis.Client
	mysqlConn *gorm.DB
}

func NewApp(configFile string) *ShipmentTrackingScheduler {
	app := &ShipmentTrackingScheduler{
		logger: logger.InitLogger(),
	}

	if len(configFile) > 0 {
		configFiles := strings.Split(configFile, ";")

		// read by input file
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

func (app *ShipmentTrackingScheduler) Run() {
	userManager := sqlmanager.NewUserManager(app.mysqlConn)
	packageManager := sqlmanager.NewPackageManager(app.mysqlConn)
	billManager := sqlmanager.NewBillManager(app.mysqlConn)
	trackingManager := sqlmanager.NewTrackingManager(app.mysqlConn)
	containerManager := sqlmanager.NewContainerManager(app.mysqlConn)
	shipmentManager := sqlmanager.NewShipmentManager(app.mysqlConn)

	handler := NewHandler(app.logger, containerManager, shipmentManager, trackingManager, packageManager, userManager, billManager)

	cronTime := DefaultTicker
	cron := viper.GetInt64("shipment_tracking.cron")
	if cron > 0 {
		cronTime = time.Duration(cron) * time.Minute
	}

	ticker := time.NewTicker(cronTime)
	defer func() {

	}()

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	keepRunning := true
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

func (h *ShipmentTrackingScheduler) Stop() {
	mysql, _ := h.mysqlConn.DB()
	mysql.Close()
	_ = h.redisConn.Close()

	log.Println("Server exiting")
}
