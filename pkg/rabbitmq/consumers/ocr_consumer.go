package consumers

import (
	"context"
	"encoding/json"
	"log"
	"tebexpressapi/pkg/constant"
	"tebexpressapi/pkg/models/entity"
	"tebexpressapi/pkg/rabbitmq"
	"tebexpressapi/pkg/sqlmanager"
	"tebexpressapi/pkg/utils"

	"github.com/rabbitmq/amqp091-go"
)

type OcrConsumer struct {
	ch        *amqp091.Channel
	queueName string

	packageManager   *sqlmanager.PackageManager
	trackingManager  *sqlmanager.TrackingManager
	warehouseManager *sqlmanager.WareHouseManager
}

func NewOcrConsumer(
	ch *amqp091.Channel,
	pm *sqlmanager.PackageManager,
	tm *sqlmanager.TrackingManager,
	wm *sqlmanager.WareHouseManager,
) (*OcrConsumer, error) {
	q, err := ch.QueueDeclare(
		"tiktok_ocr_queue",
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, err
	}

	return &OcrConsumer{
		ch:               ch,
		queueName:        q.Name,
		packageManager:   pm,
		trackingManager:  tm,
		warehouseManager: wm,
	}, nil
}

func (c *OcrConsumer) Start(ctx context.Context) error {
	msgs, err := c.ch.Consume(
		c.queueName, "", true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var m rabbitmq.OcrMessage
			if err := json.Unmarshal(d.Body, &m); err != nil {
				log.Printf("OCR consumer: invalid message: %v", err)
				continue
			}

			pkg, err := c.packageManager.GetPackageByPackageID(m.PackageID)
			if err != nil {
				log.Printf("Upload consumer: get package error: %v", err)
				continue
			}

			if pkg.CustomTiktokBarcode == nil {
				continue
			}

			trackingNumber, mapRecipientChange, err := utils.GetNslogOcrOutput(*pkg.CustomTiktokBarcode)
			if err != nil {
				log.Printf("Failed to OCR for package %d: %v", pkg.ID, err)

				err := c.packageManager.SaveUpdatePackage2(
					pkg.ID,
					pkg.UserID,
					map[string]interface{}{"recipient": "N/A"},
					[]entity.PackageAuditLog{},
					0,
					[]entity.ExtraFee{},
					pkg.Status,
				)
				if err != nil {
					log.Printf("Failed to mark OCR-failed package %d as N/A: %v", pkg.ID, err)
				}

				continue
			}

			err = c.packageManager.SaveUpdatePackage2(
				pkg.ID,
				pkg.UserID,
				mapRecipientChange,
				[]entity.PackageAuditLog{},
				0,
				[]entity.ExtraFee{},
				pkg.Status,
			)
			if err != nil {
				log.Printf("Failed to update package %d: %v", pkg.ID, err)
				continue
			}

			texasWarehouse, err := c.warehouseManager.GetWareHouse(sqlmanager.OptionWareHouse{
				Status: 1,
				State:  "TX",
			})
			if err != nil {
				log.Printf("Error get Warehouse: %v", err)
				continue
			}

			trackings := []entity.Tracking{{
				PackageID:      pkg.ID,
				TrackingNumber: trackingNumber,
				LabelURL:       pkg.Label,
				CarrierID:      1, //hard-coded
				Status:         constant.TrackingStatusSuccess,
				Weight:         pkg.Weight,
				Width:          pkg.Width,
				Length:         pkg.Length,
				Height:         pkg.Height,
				ShipmentCost:   pkg.ShippingFee,
				HubID:          &texasWarehouse.ID,
				UserID:         pkg.UserID,
				CarrierService: "FirstClass",
			}}

			oldTracking, err := c.trackingManager.GetTracking(sqlmanager.TrackingOption{
				PackageID: pkg.ID,
				Status:    constant.TrackingStatusSuccess,
			})
			if err != nil {
				log.Printf("Failed to get tracking %s: %v", trackingNumber, err)
			}
			if oldTracking == nil {
				log.Printf("Tracking not found: %s", trackingNumber)
			} else {
				oldTracking.Status = constant.TrackingStatusCanceled
				if err := c.trackingManager.Update(oldTracking); err != nil {
					log.Printf("Failed to update tracking %s: %v", trackingNumber, err)
					continue
				}
			}

			err = c.trackingManager.CreateTrackingTiktok(trackings)
			if err != nil {
				log.Printf("Error create tiktok tracking: %v", err)
				continue
			}

			log.Printf("OCR success: pkg %d → tracking %s", pkg.ID, trackingNumber)
		}
	}()

	return nil
}
