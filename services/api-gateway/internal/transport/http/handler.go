package handlers

import (
	"api-gateway/internal/client"
	authpb "api-gateway/pkg/authpb/proto/auth"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	client *client.AuthClient
}

func NewAuthHandler(client *client.AuthClient) *AuthHandler {
	return &AuthHandler{client: client}
}

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.client.Client.Register(c, &authpb.RegisterRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user_id": res.UserId})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.client.Client.Login(c, &authpb.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.SetCookie("refresh_token", res.RefreshToken, 7*24*3600, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token": res.AccessToken,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token not found"})
		return
	}

	res, err := h.client.Client.RefreshToken(c, &authpb.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.SetCookie("refresh_token", res.RefreshToken, 7*24*3600, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{
		"access_token": res.AccessToken,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.Status(http.StatusOK)
		return
	}

	_, _ = h.client.Client.Logout(c, &authpb.LogoutRequest{RefreshToken: refreshToken})

	c.SetCookie("refresh_token", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}
