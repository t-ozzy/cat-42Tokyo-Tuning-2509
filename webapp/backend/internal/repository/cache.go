// cache.go (新規作成)

package repository

import (
	"backend/internal/model"
	"context"
	"fmt"
	"sync"
)

// キャッシュに保存するデータ構造
type productCacheEntry struct {
	Products []model.Product
	Total    int
}

// ProductRepositoryの振る舞いを定義するインターフェース
type IProductRepository interface {
	ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error)
}

// キャッシュ機能を持つリポジトリ
type CachingProductRepository struct {
	next   IProductRepository // 次のRepository（DBアクセス担当）
	cache  map[string]productCacheEntry
	rwLock sync.RWMutex
}

func NewCachingProductRepository(next IProductRepository) *CachingProductRepository {
	return &CachingProductRepository{
		next:  next,
		cache: make(map[string]productCacheEntry),
	}
}


// cache.go (続き)

// ListProductsはまずキャッシュを確認し、なければDBに問い合わせる
func (r *CachingProductRepository) ListProducts(ctx context.Context, userID int, req model.ListRequest) ([]model.Product, int, error) {
	// 1. リクエスト情報からユニークなキャッシュキーを生成
	key := fmt.Sprintf("products:%s:%s:%s:%d:%d", req.Search, req.SortField, req.SortOrder, req.PageSize, req.Offset)

	// 2. キャッシュの読み取りロック
	r.rwLock.RLock()
	entry, found := r.cache[key]
	r.rwLock.RUnlock()

	// 3. キャッシュヒットした場合、その値を返す
	if found {
		return entry.Products, entry.Total, nil
	}

	// 4. キャッシュミスした場合、DBに問い合わせる
	products, total, err := r.next.ListProducts(ctx, userID, req)
	if err != nil {
		return nil, 0, err
	}

	// 5. 結果をキャッシュに書き込む（書き込みロック）
	r.rwLock.Lock()
	defer r.rwLock.Unlock()
	r.cache[key] = productCacheEntry{
		Products: products,
		Total:    total,
	}

	return products, total, nil
}