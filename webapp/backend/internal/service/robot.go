package service

import (
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/service/utils"
	"context"
	"log"
	"sort"
	
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
	// 貪欲法: 価値密度（value/weight）でソートして、容量内で最も効率の良い順に選択
	
	// 価値密度を計算して構造体に格納
	type orderWithDensity struct {
		order   model.Order
		density float64
	}
	
	var ordersWithDensity []orderWithDensity
	for _, order := range orders {
		if order.Weight <= 0 {
			continue // 重量0以下は除外
		}
		density := float64(order.Value) / float64(order.Weight)
		ordersWithDensity = append(ordersWithDensity, orderWithDensity{
			order:   order,
			density: density,
		})
	}
	
	// 価値密度の高い順にソート（降順）
	sort.Slice(ordersWithDensity, func(i, j int) bool {
		return ordersWithDensity[i].density > ordersWithDensity[j].density
	})
	
	// 貪欲に選択
	var selectedOrders []model.Order
	totalWeight := 0
	totalValue := 0
	
	for _, item := range ordersWithDensity {
		// コンテキストのキャンセルをチェック
		select {
		case <-ctx.Done():
			return model.DeliveryPlan{}, ctx.Err()
		default:
		}
		
		if totalWeight+item.order.Weight <= robotCapacity {
			selectedOrders = append(selectedOrders, item.order)
			totalWeight += item.order.Weight
			totalValue += item.order.Value
		}
	}

	return model.DeliveryPlan{
		RobotID:     robotID,
		TotalWeight: totalWeight,
		TotalValue:  totalValue,
		Orders:      selectedOrders,
	}, nil
}
