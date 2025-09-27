package service

import (
	"context"
	"log"

	"backend/internal/model"
	"backend/internal/repository"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type ProductService struct {
	store *repository.Store
}

func NewProductService(store *repository.Store) *ProductService {
	return &ProductService{store: store}
}

func (s *ProductService) CreateOrders(ctx context.Context, userID int, items []model.RequestItem) ([]string, error) {
	var insertedOrderIDs []string

	err := s.store.ExecTx(ctx, func(txStore *repository.Store) error {
		itemsToProcess := make(map[int]int)
		for _, item := range items {
			if item.Quantity > 0 {
				itemsToProcess[item.ProductID] = item.Quantity
			}
		}
		if len(itemsToProcess) == 0 {
			return nil
		}

		for pID, quantity := range itemsToProcess {
			for i := 0; i < quantity; i++ {
				order := &model.Order{
					UserID:    userID,
					ProductID: pID,
				}
				orderID, err := txStore.OrderRepo.Create(ctx, order)
				if err != nil {
					return err
				}
				insertedOrderIDs = append(insertedOrderIDs, orderID)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	log.Printf("Created %d orders for user %d", len(insertedOrderIDs), userID)
	return insertedOrderIDs, nil
}

func (s *ProductService) FetchProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// トレースのスパンを作成
	ctx, span := otel.Tracer("product-service").Start(ctx, "FetchProducts")
	defer span.End()

	// リクエスト情報をスパンに記録
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("request.page", req.Page),
		attribute.Int("request.page_size", req.PageSize),
		attribute.String("request.search", req.Search),
	)

	products, total, err := s.store.ProductRepo.ListProducts(ctx, userID, req)

	if err != nil {
		// エラー情報をスパンに記録
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to fetch products")
		return nil, 0, err
	}

	// レスポンス情報をスパンに記録
	span.SetAttributes(
		attribute.Int("response.products_count", len(products)),
		attribute.Int("response.total", total),
	)

	return products, total, err
}
