-- 既存インデックスを削除（重複回避）
DROP INDEX IF EXISTS idx_products_name ON products;
DROP INDEX IF EXISTS idx_products_description ON products;
DROP INDEX IF EXISTS idx_products_value ON products;
DROP INDEX IF EXISTS idx_products_weight ON products;

-- productsテーブルのインデックス追加
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_description ON products(description);
CREATE INDEX idx_products_value ON products(value);
CREATE INDEX idx_products_weight ON products(weight);
