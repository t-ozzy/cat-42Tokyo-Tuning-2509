package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"backend/internal/model"
	"backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type ProductService struct {
	store       *repository.Store
	redisClient *redis.Client
}

func NewProductService(store *repository.Store, redisClient *redis.Client) *ProductService {
	return &ProductService{
		store:       store,
		redisClient: redisClient,
	}
}

func (s *ProductService) CreateOrders(ctx context.Context, userID int, items []model.RequestItem) ([]string, error) {
	var insertedOrderIDs []string

	err := s.store.ExecTx(ctx, func(txStore *repository.Store) error {
		// 注文リストを事前に構築
		var orders []model.Order
		for _, item := range items {
			if item.Quantity > 0 {
				for i := 0; i < item.Quantity; i++ {
					orders = append(orders, model.Order{
						UserID:    userID,
						ProductID: item.ProductID,
					})
				}
			}
		}

		if len(orders) == 0 {
			return nil
		}

		// バルクインサートで一括作成
		ids, err := txStore.OrderRepo.CreateBulk(ctx, orders)
		if err != nil {
			return err
		}
		insertedOrderIDs = ids
		return nil
	})

	if err != nil {
		return nil, err
	}
	log.Printf("Created %d orders for user %d", len(insertedOrderIDs), userID)
	return insertedOrderIDs, nil
}

func (s *ProductService) FetchProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// キャッシュが無効な場合はDBから取得
	if s.redisClient == nil {
		return s.store.ProductRepo.ListProducts(ctx, userID, req)
	}

	// キャッシュキーの作成
	cacheKey := fmt.Sprintf("products:user:%d:page:%d:size:%d:sort:%s:%s:search:%s",
		userID, req.Page, req.PageSize, req.SortField, req.SortOrder, req.Search)

	// キャッシュから取得を試みる
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		// キャッシュヒット
		var result struct {
			Products []model.Product `json:"products"`
			Total    int             `json:"total"`
		}
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			log.Printf("Cache hit for products: %s", cacheKey)
			return result.Products, result.Total, nil
		}
		// 解析エラーがあっても通常のフローで続行
	}

	// DBから取得
	products, total, err := s.store.ProductRepo.ListProducts(ctx, userID, req)
	if err != nil {
		return nil, 0, err
	}

	// キャッシュに結果を保存
	result := struct {
		Products []model.Product `json:"products"`
		Total    int             `json:"total"`
	}{
		Products: products,
		Total:    total,
	}

	jsonData, err := json.Marshal(result)
	if err == nil {
		// キャッシュに2分間保存
		expiration := 2 * time.Minute
		if err := s.redisClient.Set(ctx, cacheKey, jsonData, expiration).Err(); err != nil {
			log.Printf("Failed to cache products: %v", err)
		} else {
			log.Printf("Cached products with key: %s", cacheKey)
		}
	}

	return products, total, nil
}
