package cache

import (
	"context"
	"errors"
	"strings"
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

func (c *TokenCache) SaveResetToken(ctx context.Context, token string, userID string) error {
	return c.client.Set(ctx, "reset_token:"+token, userID, 15*time.Minute).Err()
}

func (c *TokenCache) GetResetToken(ctx context.Context, token string) (string, error) {
	val, err := c.client.Get(ctx, "reset_token:"+token).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *TokenCache) DeleteResetToken(ctx context.Context, token string) error {
	return c.client.Del(ctx, "reset_token:"+token).Err()
}

func (c *TokenCache) SaveEmailChangeToken(ctx context.Context, token, userID, newEmail string) error {
	val := userID + "|" + newEmail
	return c.client.Set(ctx, "email_change:"+token, val, 15*time.Minute).Err()
}

func (c *TokenCache) GetEmailChangeData(ctx context.Context, token string) (string, string, error) {
	val, err := c.client.Get(ctx, "email_change:"+token).Result()
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(val, "|")
	if len(parts) != 2 {
		return "", "", errors.New("invalid token data")
	}
	return parts[0], parts[1], nil
}

func (c *TokenCache) DeleteEmailChangeToken(ctx context.Context, token string) {
	c.client.Del(ctx, "email_change:"+token)
}
