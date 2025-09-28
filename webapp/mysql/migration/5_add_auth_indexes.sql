-- 既存インデックスを削除（重複回避）
DROP INDEX IF EXISTS idx_users_user_name ON users;
DROP INDEX IF EXISTS idx_user_sessions_user_id ON user_sessions;
DROP INDEX IF EXISTS idx_user_sessions_expires_at ON user_sessions;

-- 認証関連インデックス追加
CREATE INDEX idx_users_user_name ON users(user_name);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX idx_user_sessions_expires_at ON user_sessions(expires_at);
