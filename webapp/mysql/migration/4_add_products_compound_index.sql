-- 既存インデックスを削除（重複回避）
DROP INDEX IF EXISTS idx_products_name_value ON products;

-- 複合インデックス追加
CREATE INDEX idx_products_name_value ON products(name, value);
