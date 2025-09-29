package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"tebexpressapi/pkg/order"
	"tebexpressapi/pkg/rabbitmq"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/storage"

	"github.com/rabbitmq/amqp091-go"
)

type UploadConsumer struct {
	ch             *amqp091.Channel
	queueName      string
	packageManager *sqlmanager.PackageManager
	s3             storage.S3
}

func NewUploadConsumer(ch *amqp091.Channel, pm *sqlmanager.PackageManager, s3 storage.S3) (*UploadConsumer, error) {
	q, err := ch.QueueDeclare(
		"tiktok_upload_queue", true, false, false, false, nil,
	)
	if err != nil {
		return nil, err
	}
	return &UploadConsumer{ch: ch, queueName: q.Name, packageManager: pm, s3: s3}, nil
}

func (c *UploadConsumer) Start(ctx context.Context) error {
	msgs, err := c.ch.Consume(
		c.queueName, "", true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var m rabbitmq.UploadMessage
			if err := json.Unmarshal(d.Body, &m); err != nil {
				log.Printf("Upload consumer: invalid message: %v", err)
				continue
			}

			log.Printf("Upload consumer: processing package %d", m.PackageID)

			pkg, err := c.packageManager.GetPackageByPackageID(m.PackageID)
			if err != nil {
				log.Printf("Upload consumer: get package error: %v", err)
				continue
			}

			if strings.HasPrefix(pkg.Label, "https") {
				filePath, err := order.StoreLabelS3(
					c.s3,
					*pkg.CustomTiktokBarcode,
					"pdf",
					fmt.Sprintf("tiktok_%s_%03d", pkg.OrderNumber, rand.Intn(1000)),
				)
				if err != nil {
					pkg.Label = *pkg.CustomTiktokBarcode
				} else {
					pkg.Label = filePath
				}

				if saveErr := c.packageManager.UpdatePackage(pkg, pkg.ID); saveErr != nil {
					log.Printf("Upload consumer: update package error: %v", saveErr)
					continue
				}
			}
		}
	}()

	return nil
}
