package repository

import (
	"backend/internal/model"
	"context"
)

type DbProductRepository struct {
	db DBTX
}

func NewDbProductRepository(db DBTX) IProductRepository {
	return &DbProductRepository{db: db}
}

// 商品一覧を全件取得し、アプリケーション側でページング処理を行う
func (r *DbProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	var products []model.Product
	var total int

	// 件数取得
	countQuery := "SELECT COUNT(*) FROM products"
	countArgs := []interface{}{}
	whereClause := ""
	if req.Search != "" {
		whereClause = " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}
	err := r.db.GetContext(ctx, &total, countQuery+whereClause, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	// ページング取得
	baseQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	`
	args := []interface{}{}
	if req.Search != "" {
		baseQuery += whereClause
		args = append(args, "%"+req.Search+"%", "%"+req.Search+"%")
	}
	baseQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + " , product_id ASC"
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, req.PageSize, req.Offset)

	err = r.db.SelectContext(ctx, &products, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
