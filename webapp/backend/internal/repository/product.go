package repository

import (
	"backend/internal/model"
	"context"
)

type ProductRepository struct {
	db DBTX
}

func NewProductRepository(db DBTX) *ProductRepository {
	return &ProductRepository{db: db}
}

// 商品一覧をDBレベルでページング処理を行う（効率化）
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	var products []model.Product
	var total int
	
	// 総件数取得用のクエリ
	countQuery := "SELECT COUNT(*) FROM products"
	countArgs := []interface{}{}
	
	// データ取得用のクエリ
	dataQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	`
	dataArgs := []interface{}{}

	// 検索条件がある場合
	if req.Search != "" {
		whereClause := " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		
		countQuery += whereClause
		countArgs = append(countArgs, searchPattern, searchPattern)
		
		dataQuery += whereClause
		dataArgs = append(dataArgs, searchPattern, searchPattern)
	}

	// 総件数を取得
	err := r.db.GetContext(ctx, &total, countQuery, countArgs...)
	if err != nil {
		return nil, 0, err
	}

	// ソート条件とページング条件を追加
	dataQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + ", product_id ASC"
	dataQuery += " LIMIT ? OFFSET ?"
	dataArgs = append(dataArgs, req.PageSize, req.Offset)

	// データを取得
	err = r.db.SelectContext(ctx, &products, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
