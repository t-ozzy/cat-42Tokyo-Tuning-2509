package db

import (
	"backend/internal/telemetry"
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func InitDBConnection() (*sqlx.DB, error) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "user:password@tcp(db:4306)/42Tokyo2508-db"
	}
	dsn := fmt.Sprintf("%s?charset=utf8mb4&parseTime=True&loc=Local", dbUrl)
	log.Printf(dsn)

	driverName := telemetry.WrapSQLDriver("mysql")
	dbConn, err := sqlx.Open(driverName, dsn)
	if err != nil {
		log.Printf("Failed to open database connection: %v", err)
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = dbConn.PingContext(ctx)
	if err != nil {
		dbConn.Close()
		log.Printf("Failed to connect to database: %v", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	log.Println("Successfully connected to MySQL!")

	// 接続プール最適化設定
	dbConn.SetMaxOpenConns(15)              // 最大接続数を削減（VM環境対応）
	dbConn.SetMaxIdleConns(5)               // アイドル接続数を削減
	dbConn.SetConnMaxLifetime(5 * time.Minute) // 接続の最大寿命
	dbConn.SetConnMaxIdleTime(30 * time.Second) // アイドル時間後に接続を閉じる

	return dbConn, nil
}
