package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redisClient *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{redisClient: client}
}

func (rl *RateLimiter) Limit(keySuffix string, limit int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		key := fmt.Sprintf("rate_limit:%s:%s", keySuffix, ip)

		count, err := rl.redisClient.Incr(c, key).Result()
		if err != nil {
			c.Next()
			return
		}

		// Если это первый запрос (count == 1), ставим время жизни ключу
		if count == 1 {
			rl.redisClient.Expire(c, key, window)
		}

		if count > int64(limit) {
			ttl, _ := rl.redisClient.TTL(c, key).Result()

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests",
				"retry_after": fmt.Sprintf("%.0f minutes", ttl.Minutes()),
			})
			return
		}
		c.Next()
	}
}
