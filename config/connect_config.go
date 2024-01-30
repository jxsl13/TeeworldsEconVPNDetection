package config

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
)

type ConnectConfig struct {
	RedisAddress  string `koanf:"redis.address" validate:"required"`
	RedisPassword string `koanf:"redis.password"`
	RedisDB       int    `koanf:"redis.db.vpn"`
}

func NewConnect() *ConnectConfig {
	return &ConnectConfig{
		RedisAddress: "localhost:6379",
		RedisDB:      15,
	}
}

func (c *ConnectConfig) Validate() error {
	err := validator.New().Struct(c)
	if err != nil {
		return err
	}

	options := redis.Options{
		Addr:     c.RedisAddress,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
	}

	redisClient := redis.NewClient(&options)
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil || pong != "PONG" {
		return fmt.Errorf("%w: %v", errRedisDatabaseNotFound, err)
	}

	return nil
}
