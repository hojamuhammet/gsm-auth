package database

import (
	"fmt"
	"gsm-auth/internal/config"

	"github.com/go-redis/redis/v8"
)

type Database struct {
	client *redis.Client
}

func InitDB(cfg *config.Config) (*Database, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &Database{client: rdb}, nil
}

func (d *Database) Close() error {
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

func (d *Database) GetDB() *redis.Client {
	return d.client
}
