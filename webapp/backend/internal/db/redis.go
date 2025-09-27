package db

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// InitRedisClient はRedisクライアントを初期化します
func InitRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379", // Dockerネットワーク内のアドレス
		Password: "",           // パスワードなし
		DB:       0,            // デフォルトDB
	})

	// 接続テスト
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Redisへの接続に失敗しました: %v", err)
		return nil, err
	}

	log.Println("Redisに接続しました")
	redisClient = client
	return client, nil
}

// GetRedisClient は初期化済みのRedisクライアントを返します
func GetRedisClient() *redis.Client {
	return redisClient
}
