package db

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

// redisClientの初期化
func InitRedisClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     "tuning-redis:6379", // Dockerネットワーク内のアドレス
		Password: "",                  // パスワードなし
		DB:       0,                   // デフォルトDB
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

	// グローバル変数のredisClientにセット
	redisClient = client
	return client, nil
}

// グローバル変数のredisClientをこの関数経由で渡すことで一括管理ができる
func GetRedisClient() *redis.Client {
	return redisClient
}
