package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SessionRepository struct {
	db DBTX
}

func NewSessionRepository(db DBTX) *SessionRepository {
	return &SessionRepository{db: db}
}

// セッションを作成し、セッションIDと有効期限を返す
func (r *SessionRepository) Create(ctx context.Context, userBusinessID int, duration time.Duration) (string, time.Time, error) {
	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(duration)
	sessionIDStr := sessionUUID.String()

	query := "INSERT INTO user_sessions (session_uuid, user_id, expires_at) VALUES (?, ?, ?)"
	_, err = r.db.ExecContext(ctx, query, sessionIDStr, userBusinessID, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return sessionIDStr, expiresAt, nil
}

// セッションIDからユーザーIDを取得
func (r *SessionRepository) FindUserBySessionID(ctx context.Context, sessionID string) (int, error) {
	var userID int
	query := `
		SELECT 
			u.user_id
		FROM users u
		JOIN user_sessions s ON u.user_id = s.user_id
		WHERE s.session_uuid = ? AND s.expires_at > ?`
	err := r.db.GetContext(ctx, &userID, query, sessionID, time.Now())
	if err != nil {
		return 0, err
	}
	return userID, nil
}

// GetSessionInfo はセッションIDに基づいてセッション情報を取得する
type SessionInfo struct {
	UserID    int
	ExpiresAt time.Time
}

func (r *SessionRepository) GetSessionInfo(ctx context.Context, sessionID string) (*SessionInfo, error) {
	var info SessionInfo
	query := `SELECT user_id, expires_at FROM user_sessions WHERE session_uuid = ?`
	err := r.db.GetContext(ctx, &info, query, sessionID)
	return &info, err
}
