package service

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"backend/internal/repository"
	"backend/internal/service/utils"

	"github.com/redis/go-redis/v9"

	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
	ErrInternalServer  = errors.New("internal server error")
)

type AuthService struct {
	store       *repository.Store
	redisClient *redis.Client
}

func NewAuthService(store *repository.Store, redisClient *redis.Client) *AuthService {
	return &AuthService{
		store:       store,
		redisClient: redisClient,
	}
}

// verifyPassword はハッシュ化されたパスワードを検証します
// bcryptとPBKDF2の両方のフォーマットをサポートします
func verifyPassword(storedHash, password string) (bool, error) {
	// PBKDF2ハッシュのフォーマット：$pbkdf2-sha256$i=10000$salt_base64$hash_base64
	if strings.HasPrefix(storedHash, "$pbkdf2-sha256$") {
		parts := strings.Split(storedHash, "$")
		if len(parts) != 4 {
			return false, errors.New("invalid hash format")
		}

		// イテレーション回数を解析
		var iterations int
		if _, err := fmt.Sscanf(parts[1], "i=%d", &iterations); err != nil {
			return false, errors.New("invalid iteration format")
		}

		// ソルトとハッシュをデコード
		salt, err := base64.StdEncoding.DecodeString(parts[2])
		if err != nil {
			return false, err
		}

		storedHashBytes, err := base64.StdEncoding.DecodeString(parts[3])
		if err != nil {
			return false, err
		}

		// PBKDF2でパスワードをハッシュ化
		keyLen := len(storedHashBytes)
		computedHash := pbkdf2.Key([]byte(password), salt, iterations, keyLen, sha256.New)

		// タイミング攻撃を防ぐため、constant-timeな比較を使用
		return subtle.ConstantTimeCompare(storedHashBytes, computedHash) == 1, nil
	}

	// bcryptフォーマットの場合（既存のハッシュをサポートするため）
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	return err == nil, nil
}

func (s *AuthService) Login(ctx context.Context, userName, password string) (string, time.Time, error) {
	ctx, span := otel.Tracer("service.auth").Start(ctx, "AuthService.Login")
	defer span.End()

	// 通常の認証フロー
	var sessionID string
	var expiresAt time.Time
	err := utils.WithTimeout(ctx, func(ctx context.Context) error {
		user, err := s.store.UserRepo.FindByUserName(ctx, userName)
		if err != nil {
			log.Printf("[Login] ユーザー検索失敗(userName: %s): %v", userName, err)
			if errors.Is(err, sql.ErrNoRows) {
				return ErrUserNotFound
			}
			return ErrInternalServer
		}

		// パスワード検証
		valid, err := verifyPassword(user.PasswordHash, password)
		if err != nil {
			log.Printf("[Login] パスワード検証エラー: %v", err)
			span.RecordError(err)
			return ErrInternalServer
		}
		if !valid {
			log.Printf("[Login] パスワード検証失敗")
			return ErrInvalidPassword
		}

		sessionDuration := 24 * time.Hour
		sessionID, expiresAt, err = s.store.SessionRepo.Create(ctx, user.UserID, sessionDuration)
		if err != nil {
			log.Printf("[Login] セッション生成失敗: %v", err)
			return ErrInternalServer
		}

		// 認証成功後、Redisにセッション情報をキャッシュする
		if s.redisClient != nil {
			// TODO: 共通化できそう
			cacheKey := "session:" + sessionID
			sessionData := map[string]interface{}{
				"user_id":    user.UserID,
				"expires_at": expiresAt.Unix(),
			}

			// セッションと同じ期間キャッシュを保持
			if err := s.redisClient.HSet(ctx, cacheKey, sessionData).Err(); err != nil {
				log.Printf("[Login] セッションキャッシュ保存失敗: %v", err)
				// キャッシュ失敗はエラーとして扱わない（アプリケーション続行可能）
			} else {
				s.redisClient.Expire(ctx, cacheKey, time.Until(expiresAt))
				log.Printf("[Login] セッションキャッシュ保存成功: %s", sessionID)
			}
		}

		return nil
	})

	if err != nil {
		return "", time.Time{}, err
	}

	log.Printf("Login successful for UserName '%s', session created.", userName)
	return sessionID, expiresAt, nil
}
