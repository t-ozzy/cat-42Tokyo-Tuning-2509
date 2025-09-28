package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"backend/internal/repository"
	"backend/internal/service/utils"

	"github.com/redis/go-redis/v9"

	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
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

func (s *AuthService) Login(ctx context.Context, userName, password string) (string, time.Time, error) {
	ctx, span := otel.Tracer("service.auth").Start(ctx, "AuthService.Login")
	defer span.End()

	log.Printf("[Debug] ログイン試行: userName=%s", userName)

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
		log.Printf("[Debug] ユーザー検索成功: userID=%d", user.UserID)

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			log.Printf("[Login] パスワード検証失敗: %v", err)
			span.RecordError(err)
			return ErrInvalidPassword
		}
		log.Printf("[Debug] パスワード検証成功")

		sessionDuration := 24 * time.Hour
		sessionID, expiresAt, err = s.store.SessionRepo.Create(ctx, user.UserID, sessionDuration)
		if err != nil {
			log.Printf("[Login] セッション生成失敗: %v", err)
			return ErrInternalServer
		}
		log.Printf("[Debug] セッション生成成功: sessionID=%s", sessionID)

		if s.redisClient != nil {
			cacheKey := "session:" + sessionID
			err := s.redisClient.Set(ctx, cacheKey, user.UserID, sessionDuration).Err()
			if err != nil {
				log.Printf("[Login] Redisキャッシュ保存失敗: %v", err)
			} else {
				log.Printf("[Debug] Redisキャッシュ保存成功")
			}
		} else {
			log.Printf("[Debug] Redisクライアントが利用できません")
		}

		return nil
	})

	if err != nil {
		log.Printf("[Debug] ログイン失敗: %v", err)
		return "", time.Time{}, err
	}
	log.Printf("[Debug] ログイン成功: sessionID=%s", sessionID)

	return sessionID, expiresAt, nil
}
