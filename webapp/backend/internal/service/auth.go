package service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
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

	// キャッシュキーを設定（ユーザー名とパスワードのハッシュ）
	// userNameだけにしてしまうと、間違ったパスワードでログイン失敗した場合でも正しいパスワードでログインしたときと同じキャッシュキーになってしまう
	// つまり、間違ったパスワードでログイン失敗した場合でもキャッシュヒットしてしまう
	// そのため、userNameとpasswordのハッシュを組み合わせたキーにする
	// 例: auth:login:alice:5f4dcc3b5aa765d61d8327deb882cf99
	cacheKey := "auth:login:" + userName + ":" + utils.HashString(password)

	// Redisからキャッシュをチェック
	if s.redisClient != nil {
		cachedUserID, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			// キャッシュヒット！
			span.AddEvent("auth_cache_hit")
			log.Printf("Auth cache hit for user: %s", userName)

			// キャッシュからユーザーIDを取得
			userID, _ := strconv.Atoi(cachedUserID)

			// セッション作成のみ実行（DB検証をスキップ）
			sessionDuration := 60 * time.Second
			sessionID, expiresAt, err := s.store.SessionRepo.Create(ctx, userID, sessionDuration)
			if err != nil {
				log.Printf("[Login] セッション生成失敗: %v", err)
				return "", time.Time{}, ErrInternalServer
			}

			return sessionID, expiresAt, nil
		} else if err != redis.Nil {
			// エラーログ（redis.Nilは「キーが存在しない」エラーなのでログ不要）
			log.Printf("Redis error: %v", err)
		}

		// キャッシュミス
		span.AddEvent("auth_cache_miss")
	}

	// 通常の認証フロー（既存のコード）
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

		err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if err != nil {
			log.Printf("[Login] パスワード検証失敗: %v", err)
			span.RecordError(err)
			return ErrInvalidPassword
		}

		// 認証成功したらRedisにキャッシュする
		if s.redisClient != nil {
			// 5分間キャッシュ
			s.redisClient.Set(ctx, cacheKey, user.UserID, 5*time.Minute)
		}

		sessionDuration := 24 * time.Hour
		sessionID, expiresAt, err = s.store.SessionRepo.Create(ctx, user.UserID, sessionDuration)
		if err != nil {
			log.Printf("[Login] セッション生成失敗: %v", err)
			return ErrInternalServer
		}
		return nil
	})

	if err != nil {
		return "", time.Time{}, err
	}

	log.Printf("Login successful for UserName '%s', session created.", userName)
	return sessionID, expiresAt, nil
}
