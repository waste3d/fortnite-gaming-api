package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenCache struct {
	client *redis.Client
}

func NewTokenCache(client *redis.Client) *TokenCache {
	return &TokenCache{client: client}
}

// SaveRefresh сохраняет токен с TTL 7 дней
func (c *TokenCache) SaveRefresh(ctx context.Context, userID string, refreshToken string) error {
	// Ключ: "refresh_token:<jwt_string>" -> Значение: "user_id"
	// Это позволяет проверить существование конкретного токена
	return c.client.Set(ctx, "refresh_token:"+refreshToken, userID, 7*24*time.Hour).Err()
}

// CheckRefresh проверяет, есть ли токен в белом списке
func (c *TokenCache) CheckRefresh(ctx context.Context, refreshToken string) (string, error) {
	val, err := c.client.Get(ctx, "refresh_token:"+refreshToken).Result()
	if err != nil {
		return "", err
	}
	return val, nil // возвращаем userID
}

// DeleteRefresh удаляет токен (Logout)
func (c *TokenCache) DeleteRefresh(ctx context.Context, refreshToken string) error {
	return c.client.Del(ctx, "refresh_token:"+refreshToken).Err()
}
