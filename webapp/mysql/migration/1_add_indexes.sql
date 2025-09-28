-- パフォーマンス改善のためのインデックス追加

-- 既存インデックスを削除（重複回避）
DROP INDEX IF EXISTS idx_products_name ON products;
DROP INDEX IF EXISTS idx_products_description ON products;
DROP INDEX IF EXISTS idx_products_value ON products;
DROP INDEX IF EXISTS idx_products_weight ON products;
DROP INDEX IF EXISTS idx_products_value_id ON products;
DROP INDEX IF EXISTS idx_products_weight_id ON products;
DROP INDEX IF EXISTS idx_products_name_id ON products;
DROP INDEX IF EXISTS idx_orders_shipped_status ON orders;
DROP INDEX IF EXISTS idx_orders_user_id ON orders;
DROP INDEX IF EXISTS idx_orders_product_id ON orders;
DROP INDEX IF EXISTS idx_orders_created_at ON orders;
DROP INDEX IF EXISTS idx_orders_user_id_created_at ON orders;

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

-- ORDER BY name, product_id用の複合インデックス
CREATE INDEX idx_products_name_id ON products(name, product_id);

-- ordersテーブルのインデックス
-- ロボット配送計画用（shipped_statusでの絞り込み）
CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);

-- JOIN・検索・ソート用インデックス
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_product_id ON orders(product_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);

-- ユーザー別注文履歴用
CREATE INDEX idx_orders_user_id_created_at ON orders(user_id, created_at DESC);