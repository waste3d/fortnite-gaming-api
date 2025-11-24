package middleware

import (
	"net/http"
	"strings"

	"api-gateway/internal/client"
	authpb "api-gateway/pkg/authpb/proto/auth"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(authClient *client.AuthClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		accessToken := parts[1]

		res, err := authClient.Client.Validate(c, &authpb.ValidateRequest{
			AccessToken: accessToken,
		})

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set("userId", res.UserId)

		c.Next()
	}
}
