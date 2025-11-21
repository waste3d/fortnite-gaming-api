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

func (c *TokenCache) SaveRefresh(ctx context.Context, userID string, refreshToken string) error {
	// Храним 7 дней
	return c.client.Set(ctx, "refresh_token:"+refreshToken, userID, 7*24*time.Hour).Err()
}

func (c *TokenCache) CheckRefresh(ctx context.Context, refreshToken string) (string, error) {
	val, err := c.client.Get(ctx, "refresh_token:"+refreshToken).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *TokenCache) DeleteRefresh(ctx context.Context, refreshToken string) error {
	return c.client.Del(ctx, "refresh_token:"+refreshToken).Err()
}
