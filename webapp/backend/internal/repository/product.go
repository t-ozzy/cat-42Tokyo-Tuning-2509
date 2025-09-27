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

// 商品一覧をページング付きで取得（DBレベルでLIMIT/OFFSETを適用）
func (r *ProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// 総件数を取得する処理　start
	// 総件数を取得するクエリ
	countQuery := `SELECT COUNT(*) FROM products`
	countArgs := []interface{}{}

	// 検索条件があれば追加（カウントクエリにも同じ条件を適用）
	whereClause := ""
	if req.Search != "" {
		whereClause = " WHERE (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + req.Search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}

	countQuery += whereClause

	// 総件数を取得
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, countArgs...); err != nil {
		return nil, 0, err
	}
	// 総件数を取得する処理　end

	// 必要なデータのみを取得するクエリ
	var products []model.Product
	selectQuery := `
		SELECT product_id, name, value, weight, image, description
		FROM products
	` + whereClause

	// ソート条件を追加
	selectQuery += " ORDER BY " + req.SortField + " " + req.SortOrder

	// 二次ソートの追加（一貫性のため）
	if req.SortField != "product_id" {
		selectQuery += ", product_id ASC"
	}

	// LIMIT と OFFSET を追加
	selectQuery += " LIMIT ? OFFSET ?"

	// 引数の準備
	selectArgs := []interface{}{}
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		selectArgs = append(selectArgs, searchPattern, searchPattern)
	}

	// LIMIT と OFFSET のパラメータを追加
	selectArgs = append(selectArgs, req.PageSize, req.Offset)

	// ページング済みデータを取得
	if err := r.db.SelectContext(ctx, &products, selectQuery, selectArgs...); err != nil {
		return nil, 0, err
	}

	return products, total, nil
}
