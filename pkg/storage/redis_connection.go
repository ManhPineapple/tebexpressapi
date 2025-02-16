package storage

import (
	"context"
	"crypto/tls"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func DefaultRedisFromConfig(opts *redis.Options) *redis.Options {
	if opts == nil {
		opts = &redis.Options{
			Addr:     viper.GetString("redis.addr"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		}

		if viper.GetBool("redis.is_tls") {
			opts.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}

	return opts
}

func NewRedisConnection(opts *redis.Options) (*redis.Client, error) {
	rdb := redis.NewClient(opts)

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}
	return rdb, nil
}
