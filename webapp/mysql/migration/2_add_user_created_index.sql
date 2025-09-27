-- ユーザーごとの注文履歴取得を高速化する複合インデックス
CREATE INDEX idx_orders_user_created ON orders(user_id, created_at);
