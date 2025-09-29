package consumer

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"tebexpressapi/pkg/config"
	"tebexpressapi/pkg/rabbitmq"
	"tebexpressapi/pkg/rabbitmq/consumers"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"
)

func StartConsumers(configFile string) {
	if len(configFile) > 0 {
		configFiles := strings.Split(configFile, ";")

		err := config.ReadConfigByFiles("toml", configFiles)
		if err != nil {
			log.Panic("initConfig: Could not load conf: ", err)
		}
	}

	conn, err := rabbitmq.NewConnection(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := rabbitmq.NewChannel(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	mysqlConn, err := storage.NewMysqlConnection(storage.DefaultMysqlFromConfig(nil))
	if err != nil {
		log.Panic(err)
	}

	pm := sqlmanager.NewPackageManager(mysqlConn)
	tm := sqlmanager.NewTrackingManager(mysqlConn)
	wm := sqlmanager.NewWareHouseManager(mysqlConn)
	s3 := storage.NewAmazonS3(nil)

	// Start OCR consumer
	ocrConsumer, err := consumers.NewOcrConsumer(ch, pm, tm, wm)
	if err != nil {
		log.Fatalf("failed to init OCR consumer: %v", err)
	}
	go func() {
		if err := ocrConsumer.Start(context.Background()); err != nil {
			log.Fatalf("OCR consumer stopped: %v", err)
		}
	}()

	// Start Upload consumer
	uploadConsumer, err := consumers.NewUploadConsumer(ch, pm, s3)
	if err != nil {
		log.Fatalf("failed to init Upload consumer: %v", err)
	}
	go func() {
		if err := uploadConsumer.Start(context.Background()); err != nil {
			log.Fatalf("OCR consumer stopped: %v", err)
		}
	}()

	log.Println("Consumers started. Waiting for messages...")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	<-sig
	log.Println("Shutting down consumers...")
}
