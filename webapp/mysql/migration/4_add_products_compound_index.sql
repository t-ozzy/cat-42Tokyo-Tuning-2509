-- 複合インデックス追加（重複エラー無視）
CREATE INDEX idx_products_name_value ON products(name, value);
