// 複数注文をバルクインサートし、生成された注文IDを返す

// 複数注文をバルクインサートし、生成された注文IDを返す
package repository

import (
	"backend/internal/model"
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type OrderRepository struct {
	db DBTX
}

// 複数注文をバルクインサートし、生成された注文IDを返す
func (r *OrderRepository) CreateBulk(ctx context.Context, orders []model.Order) ([]string, error) {
	if len(orders) == 0 {
		return nil, nil
	}
	query := "INSERT INTO orders (user_id, product_id, shipped_status, created_at) VALUES "
	args := []interface{}{}
	placeholders := []string{}
	for _, order := range orders {
		placeholders = append(placeholders, "(?, ?, 'shipping', NOW())")
		args = append(args, order.UserID, order.ProductID)
	}
	query += strings.Join(placeholders, ",")
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	// 生成された注文IDを返す（MySQLの場合、最初のIDのみ取得可能）
	firstID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for i := 0; i < len(orders); i++ {
		ids = append(ids, fmt.Sprintf("%d", firstID+int64(i)))
	}
	return ids, nil
}

func NewOrderRepository(db DBTX) *OrderRepository {
	return &OrderRepository{db: db}
}

// 注文を作成し、生成された注文IDを返す
func (r *OrderRepository) Create(ctx context.Context, order *model.Order) (string, error) {
	query := `INSERT INTO orders (user_id, product_id, shipped_status, created_at) VALUES (?, ?, 'shipping', NOW())`
	result, err := r.db.ExecContext(ctx, query, order.UserID, order.ProductID)
	if err != nil {
		return "", err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

// 複数の注文IDのステータスを一括で更新
// 主に配送ロボットが注文を引き受けた際に一括更新をするために使用
func (r *OrderRepository) UpdateStatuses(ctx context.Context, orderIDs []int64, newStatus string) error {
	if len(orderIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In("UPDATE orders SET shipped_status = ? WHERE order_id IN (?)", newStatus, orderIDs)
	if err != nil {
		return err
	}
	query = r.db.Rebind(query)
	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// 配送中(shipped_status:shipping)の注文一覧を取得
func (r *OrderRepository) GetShippingOrders(ctx context.Context) ([]model.Order, error) {
	var orders []model.Order
	query := `
        SELECT
            o.order_id,
            p.weight,
            p.value
        FROM orders o
        JOIN products p ON o.product_id = p.product_id
        WHERE o.shipped_status = 'shipping'
    `
	err := r.db.SelectContext(ctx, &orders, query)
	return orders, err
}

// 注文履歴一覧を取得
func (r *OrderRepository) ListOrders(ctx context.Context, userID int, req model.ListRequest) ([]model.Order, int, error) {
	// SQL JOIN・検索・ソート・ページングで一括取得
	var conditions []string
	var args []interface{}
	conditions = append(conditions, "o.user_id = ?")
	args = append(args, userID)
	if req.Search != "" {
		if req.Type == "prefix" {
			conditions = append(conditions, "p.name LIKE ?")
			args = append(args, req.Search+"%")
		} else {
			conditions = append(conditions, "p.name LIKE ?")
			args = append(args, "%"+req.Search+"%")
		}
	}
	whereClause := " WHERE " + strings.Join(conditions, " AND ")

	// 件数取得
	countQuery := `SELECT COUNT(*) FROM orders o JOIN products p ON o.product_id = p.product_id` + whereClause
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// ソート
	orderClause := " ORDER BY "
	switch req.SortField {
	case "product_name":
		orderClause += "p.name"
	case "created_at":
		orderClause += "o.created_at"
	case "shipped_status":
		orderClause += "o.shipped_status"
	case "arrived_at":
		orderClause += "o.arrived_at"
	case "order_id":
		fallthrough
	default:
		orderClause += "o.order_id"
	}
	if strings.ToUpper(req.SortOrder) == "DESC" {
		orderClause += " DESC"
	} else {
		orderClause += " ASC"
	}

	// ページング
	dataQuery := `SELECT o.order_id, o.product_id, p.name AS product_name, o.shipped_status, o.created_at, o.arrived_at FROM orders o JOIN products p ON o.product_id = p.product_id` + whereClause + orderClause + " LIMIT ? OFFSET ?"
	argsData := append(args, req.PageSize, req.Offset)

	type orderRow struct {
		OrderID       int          `db:"order_id"`
		ProductID     int          `db:"product_id"`
		ProductName   string       `db:"product_name"`
		ShippedStatus string       `db:"shipped_status"`
		CreatedAt     sql.NullTime `db:"created_at"`
		ArrivedAt     sql.NullTime `db:"arrived_at"`
	}
	var rows []orderRow
	if err := r.db.SelectContext(ctx, &rows, dataQuery, argsData...); err != nil {
		return nil, 0, err
	}

	var orders []model.Order
	for _, o := range rows {
		orders = append(orders, model.Order{
			OrderID:       int64(o.OrderID),
			ProductID:     o.ProductID,
			ProductName:   o.ProductName,
			ShippedStatus: o.ShippedStatus,
			CreatedAt:     o.CreatedAt.Time,
			ArrivedAt:     o.ArrivedAt,
		})
	}

	return orders, total, nil
}
