package middleware

import (
	"context"
	"log"
	"net/http"
	"time"

	"backend/internal/repository"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const userContextKey contextKey = "user"

func UserAuthMiddleware(sessionRepo *repository.SessionRepository, redisClient *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_id")
			if err != nil {
				log.Printf("Error retrieving session cookie: %v", err)
				http.Error(w, "Unauthorized: No session cookie", http.StatusUnauthorized)
				return
			}
			sessionID := cookie.Value

			ctx := r.Context()
			var userID int

			// 1. Redisからセッション情報の取得を試みる
			if redisClient != nil {
				cacheKey := "session:" + sessionID
				cachedUserID, err := redisClient.HGet(ctx, cacheKey, "user_id").Int()

				if err == nil {
					// キャッシュヒット - セッション有効期限を確認
					expiresAt, err := redisClient.HGet(ctx, cacheKey, "expires_at").Int64()
					if err == nil && time.Now().Unix() < expiresAt {
						// キャッシュからユーザーIDを取得して処理を続行
						userID = cachedUserID
						r = r.WithContext(context.WithValue(ctx, userContextKey, userID))
						next.ServeHTTP(w, r)
						return
					}

					// 有効期限切れの場合はキャッシュを削除
					if err == nil && time.Now().Unix() >= expiresAt {
						redisClient.Del(ctx, cacheKey)
					}
				}
			}

			// 2. キャッシュミスまたはRedisが使えない場合、DBから直接取得
			dbUserID, err := sessionRepo.FindUserBySessionID(ctx, sessionID)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID = dbUserID

			// 3. セッション情報をキャッシュに保存 (次回のためのキャッシュ再構築)
			if redisClient != nil {
				// DBからセッションの有効期限を取得
				sessionInfo, err := sessionRepo.GetSessionInfo(ctx, sessionID)
				if err == nil && sessionInfo.ExpiresAt.After(time.Now()) {
					// TODO: 共通化できそう
					cacheKey := "session:" + sessionID
					sessionData := map[string]interface{}{
						"user_id":    userID,
						"expires_at": sessionInfo.ExpiresAt.Unix(),
					}

					// セッションと同じ期間キャッシュを保持
					if err := redisClient.HSet(ctx, cacheKey, sessionData).Err(); err != nil {
						log.Printf("[middleware] セッションキャッシュ保存失敗: %v", err)
						// キャッシュ失敗はエラーとして扱わない（アプリケーション続行可能）
					} else {
						redisClient.Expire(ctx, cacheKey, time.Until(sessionInfo.ExpiresAt))
						log.Printf("[middleware] セッションキャッシュ保存成功: %s", sessionID)
					}
				}
			}

			ctx = context.WithValue(ctx, userContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RobotAuthMiddleware(validAPIKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-API-KEY")

			if apiKey == "" || apiKey != validAPIKey {
				http.Error(w, "Forbidden: Invalid or missing API key", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// コンテキストからユーザー情報を取得
// ユーザ情報はUserAuthMiddleware
func GetUserFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userContextKey).(int)
	return userID, ok
}
