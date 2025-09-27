-- パフォーマンス改善のためのインデックス追加

-- productsテーブルのインデックス
-- 商品検索用（name, descriptionでのLIKE検索）
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_description ON products(description(100)); -- TEXTの部分インデックス

-- ソート用インデックス
CREATE INDEX idx_products_value ON products(value);
CREATE INDEX idx_products_weight ON products(weight);

-- 複合インデックス（ソート + ID での効率化）
CREATE INDEX idx_products_value_id ON products(value, product_id);
CREATE INDEX idx_products_weight_id ON products(weight, product_id);

-- ordersテーブルのインデックス
-- ロボット配送計画用（shipped_statusでの絞り込み）
CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);

-- ユーザー別注文履歴用
CREATE INDEX idx_orders_user_id_created_at ON orders(user_id, created_at DESC);

-- 商品別注文集計用
CREATE INDEX idx_orders_product_id ON orders(product_id);