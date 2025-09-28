-- productsテーブルのインデックス追加（重複エラー無視）
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_description ON products(description);
CREATE INDEX idx_products_value ON products(value);
CREATE INDEX idx_products_weight ON products(weight);
