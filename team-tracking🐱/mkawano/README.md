# 性能改善レポート

Update: 2025/09/27 17:46
- ベースラインスコア: 267点
- 最終スコア: 631点
- 改善率: 2.36倍（+364点）
- E2Eテスト: 全て通過

## 実施した最適化

### 1. データベースインデックス最適化
```

  実施した最適化

  1. データベースインデックス最適化

  ファイル: webapp/mysql/migration/1_add_indexes.sql

  変更内容:
  -- 商品検索用インデックス
  CREATE INDEX idx_products_name ON products(name);
  CREATE INDEX idx_products_description ON products(description(100));

  -- ソート用インデックス  
  CREATE INDEX idx_products_value ON products(value);
  CREATE INDEX idx_products_weight ON products(weight);

  -- 複合インデックス（ソート + ID）
  CREATE INDEX idx_products_value_id ON products(value, product_id);
  CREATE INDEX idx_products_weight_id ON products(weight, product_id);

  -- 注文管理用インデックス
  CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);
  CREATE INDEX idx_orders_user_id_created_at ON orders(user_id, created_at DESC);
  CREATE INDEX idx_orders_product_id ON orders(product_id);
```
効果: 商品検索・ソート処理の高速化

### 2. DBレベルページング実装

ファイル: webapp/backend/internal/repository/product.go

- 変更前: アプリケーション層でページング（全データ取得後フィルタリング）
- 変更後: DB層でページング（LIMIT/OFFSET使用）

主要な変更:
```
// 総件数を先に取得
err := r.db.GetContext(ctx, &total, countQuery, countArgs...)

// データ取得でLIMIT/OFFSETを使用
dataQuery += " ORDER BY " + req.SortField + " " + req.SortOrder + ", product_id ASC"
dataQuery += " LIMIT ? OFFSET ?"
dataArgs = append(dataArgs, req.PageSize, req.Offset)
```

効果: データ転送量削減（~100k件 → ~20件/リクエスト）

### 3. ナップサック問題アルゴリズム最適化

ファイル: webapp/backend/internal/service/robot.go

変更前: DFS（深さ優先探索）- O(2^n)
- 処理時間: 39.36秒
- 完全解探索だが指数的計算量

変更後: 動的プログラミング（DP）- O(n×W)
- 処理時間: ミリ秒レベル
- 最適解保証かつ効率的

アルゴリズム概要:
```go
// DP配列: dp[i][w] = i番目までの注文を使って重量wまでの最大価値
  dp := make([][]int, n+1)
  for i := range dp {
      dp[i] = make([]int, robotCapacity+1)
  }

  // DPテーブル構築
  for i := 1; i <= n; i++ {
      order := validOrders[i-1]
      for w := 0; w <= robotCapacity; w++ {
          dp[i][w] = dp[i-1][w] // 選ばない場合
          if w >= order.Weight {
              dp[i][w] = max(dp[i][w], dp[i-1][w-order.Weight]+order.Value)
          }
      }
  }
```

効果: 39秒 → ミリ秒の劇的な高速化

