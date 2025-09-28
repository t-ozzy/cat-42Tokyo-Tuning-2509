package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
	
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type RobotService struct {
	store *repository.Store
}

func NewRobotService(store *repository.Store) *RobotService {
	return &RobotService{store: store}
}

func (s *RobotService) GenerateDeliveryPlan(ctx context.Context, robotID string, capacity int) (*model.DeliveryPlan, error) {
	tracer := otel.Tracer("service.robot")
	ctx, span := tracer.Start(ctx, "RobotService.GenerateDeliveryPlan")
	defer span.End()
	span.SetAttributes(attribute.String("robot.id", robotID), attribute.Int("robot.capacity", capacity))

	var plan model.DeliveryPlan

	err := utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.ExecTx(ctx, func(txStore *repository.Store) error {
			_, ordersSpan := tracer.Start(ctx, "GetShippingOrders")
			orders, err := txStore.OrderRepo.GetShippingOrders(ctx)
			ordersSpan.SetAttributes(attribute.Int("orders.count", len(orders)))
			ordersSpan.End()
			if err != nil {
				return err
			}
			
			_, planSpan := tracer.Start(ctx, "SelectOrdersForDelivery")
			plan, err = selectOrdersForDelivery(ctx, orders, robotID, capacity)
			planSpan.SetAttributes(
				attribute.Int("plan.orders_count", len(plan.Orders)),
				attribute.Int("plan.total_weight", plan.TotalWeight),
				attribute.Int("plan.total_value", plan.TotalValue),
			)
			planSpan.End()
			if err != nil {
				return err
			}
			
			if len(plan.Orders) > 0 {
				orderIDs := make([]int64, len(plan.Orders))
				for i, order := range plan.Orders {
					orderIDs[i] = order.OrderID
				}

				_, updateSpan := tracer.Start(ctx, "UpdateOrderStatuses")
				if err := txStore.OrderRepo.UpdateStatuses(ctx, orderIDs, "delivering"); err != nil {
					updateSpan.End()
					return err
				}
				updateSpan.SetAttributes(attribute.Int("updated.orders_count", len(orderIDs)))
				updateSpan.End()
				log.Printf("Updated status to 'delivering' for %d orders", len(orderIDs))
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *RobotService) UpdateOrderStatus(ctx context.Context, orderID int64, newStatus string) error {
	return utils.WithTimeout(ctx, func(ctx context.Context) error {
		return s.store.OrderRepo.UpdateStatuses(ctx, []int64{orderID}, newStatus)
	})
}

func selectOrdersForDelivery(ctx context.Context, orders []model.Order, robotID string, robotCapacity int) (model.DeliveryPlan, error) {
	// 動的プログラミングによるナップサック解法（最適解を保証）
	
	// 重量0以下の注文を除外
	var validOrders []model.Order
	for _, order := range orders {
		if order.Weight > 0 && order.Weight <= robotCapacity {
			validOrders = append(validOrders, order)
		}
	}
	
	n := len(validOrders)
	if n == 0 {
		return model.DeliveryPlan{
			RobotID:     robotID,
			TotalWeight: 0,
			TotalValue:  0,
			Orders:      []model.Order{},
		}, nil
	}
	
	// DP配列: dp[i][w] = i番目までの注文を使って重量wまでの最大価値
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, robotCapacity+1)
	}
	
	// DPテーブルを構築
	for i := 1; i <= n; i++ {
		// コンテキストのキャンセルをチェック
		select {
		case <-ctx.Done():
			return model.DeliveryPlan{}, ctx.Err()
		default:
		}
		
		order := validOrders[i-1]
		for w := 0; w <= robotCapacity; w++ {
			// この注文を選ばない場合
			dp[i][w] = dp[i-1][w]
			
			// この注文を選ぶ場合（重量が許す場合）
			if w >= order.Weight {
				dp[i][w] = max(dp[i][w], dp[i-1][w-order.Weight]+order.Value)
			}
		}
	}
	
	// 最適解の復元
	var selectedOrders []model.Order
	totalWeight := 0
	totalValue := dp[n][robotCapacity]
	
	w := robotCapacity
	for i := n; i > 0 && w > 0; i-- {
		// この注文が選ばれているかチェック
		if dp[i][w] != dp[i-1][w] {
			order := validOrders[i-1]
			selectedOrders = append(selectedOrders, order)
			totalWeight += order.Weight
			w -= order.Weight
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  totalValue,
		Orders:      selectedOrders,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
