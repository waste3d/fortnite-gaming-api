package handlers

import (
	"api-gateway/internal/client"
	authpb "api-gateway/pkg/authpb/proto/auth"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type registerReq struct {
	Email    string `json:"email" binding:"required,email"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginReq struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	DeviceId   string `json:"device_id" binding:"required"`
	DeviceName string `json:"device_name"`
}

type forgotReq struct {
	Email string `json:"email" binding:"required,email"`
}

type resetReq struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

type logoutReq struct {
	DeviceId string `json:"device_id"`
}

type AuthHandler struct {
	client *client.AuthClient
}

func NewAuthHandler(client *client.AuthClient) *AuthHandler {
	return &AuthHandler{client: client}
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
		Email:      req.Email,
		Password:   req.Password,
		DeviceId:   req.DeviceId,
		DeviceName: req.DeviceName,
	})

	if err != nil {
		println("LOGIN ERROR FROM GRPC:", err.Error())
		if strings.Contains(err.Error(), "limit reached") {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Device limit reached for your subscription"})
			return
		}
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

	var req logoutReq
	_ = c.ShouldBindJSON(&req)

	_, _ = h.client.Client.Logout(c, &authpb.LogoutRequest{
		RefreshToken: refreshToken,
		DeviceId:     req.DeviceId,
	})

	c.SetCookie("refresh_token", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.client.Client.ForgotPassword(c, &authpb.ForgotPasswordRequest{
		Email: req.Email,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If email exists, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.client.Client.ResetPassword(c, &authpb.ResetPasswordRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

func (h *AuthHandler) GetDevices(c *gin.Context) {
	userID := c.GetString("userId")

	res, err := h.client.Client.GetDevices(c, &authpb.GetDevicesRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, res.Devices)
}

func (h *AuthHandler) RemoveDevice(c *gin.Context) {
	userID := c.GetString("userId")
	deviceID := c.Param("id")

	_, err := h.client.Client.RemoveDevice(c, &authpb.RemoveDeviceRequest{
		UserId:   userID,
		DeviceId: deviceID,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
