-- パフォーマンス改善のためのインデックス追加

-- productsテーブルのインデックス作成（エラー無視）

-- productsテーブルのインデックス（重複エラー無視）
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_description ON products(description(100));
CREATE INDEX idx_products_value ON products(value);
CREATE INDEX idx_products_weight ON products(weight);
CREATE INDEX idx_products_value_id ON products(value, product_id);
CREATE INDEX idx_products_weight_id ON products(weight, product_id);
CREATE INDEX idx_products_name_id ON products(name, product_id);

-- ordersテーブルのインデックス（重複エラー無視）
CREATE INDEX idx_orders_shipped_status ON orders(shipped_status);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_product_id ON orders(product_id);
CREATE INDEX idx_orders_created_at ON orders(created_at);
CREATE INDEX idx_orders_user_id_created_at ON orders(user_id, created_at DESC);