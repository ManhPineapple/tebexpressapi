package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

type OcrMessage struct {
	PackageID int64 `json:"package_id"`
}

type UploadMessage struct {
	PackageID int64 `json:"package_id"`
}

type RabbitMqOpts struct {
	User     string
	Password string
	Host     string
	Port     int
	VHost    string
}

func DefaultRabbitMQFromConfig(opts *RabbitMqOpts) *RabbitMqOpts {
	if opts == nil {
		opts = &RabbitMqOpts{
			User:     viper.GetString("rabbitmq.user"),
			Password: viper.GetString("rabbitmq.password"),
			Host:     viper.GetString("rabbitmq.host"),
			Port:     viper.GetInt("rabbitmq.port"),
			VHost:    viper.GetString("rabbitmq.vhost"),
		}
	}
	return opts
}

func NewConnection(opts *RabbitMqOpts) (*amqp091.Connection, error) {
	opts = DefaultRabbitMQFromConfig(opts)

	// build DSN
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		opts.User,
		opts.Password,
		opts.Host,
		opts.Port,
		opts.VHost,
	)

	return amqp091.Dial(dsn)
}

func NewChannel(conn *amqp091.Connection) (*amqp091.Channel, error) {
	return conn.Channel()
}

// Producer
type Producer struct {
	ch *amqp091.Channel
}

func NewProducer(ch *amqp091.Channel) *Producer {
	return &Producer{ch: ch}
}

func (p *Producer) Publish(ctx context.Context, queueName string, msg interface{}) error {
	// declare queue if not already
	q, err := p.ch.QueueDeclare(
		queueName, true, false, false, false, nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.ch.PublishWithContext(
		ctx,
		"",
		q.Name,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
