package handlers

import (
	"api-gateway/internal/client"
	authpb "api-gateway/pkg/authpb/proto/auth"
	userpb "api-gateway/pkg/userpb/proto/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userClient *client.UserClient
	authClient *client.AuthClient // Нужен для смены email
}

func NewUserHandler(uc *client.UserClient, ac *client.AuthClient) *UserHandler {
	return &UserHandler{userClient: uc, authClient: ac}
}

// GET /api/v1/user/profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetString("userId") // Из AuthMiddleware

	res, err := h.userClient.Client.GetProfile(c, &userpb.GetProfileRequest{UserId: userID})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// POST /api/v1/user/profile (Обновление username)
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		Username string `json:"username"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.userClient.Client.UpdateProfile(c, &userpb.UpdateProfileRequest{
		UserId:   userID,
		Username: req.Username,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// POST /api/v1/user/avatar
func (h *UserHandler) SetAvatar(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		AvatarID int32 `json:"avatar_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	res, err := h.userClient.Client.SetAvatar(c, &userpb.SetAvatarRequest{
		UserId:   userID,
		AvatarId: req.AvatarID,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// POST /api/v1/user/email/request-change
func (h *UserHandler) RequestEmailChange(c *gin.Context) {
	userID := c.GetString("userId")
	var req struct {
		NewEmail string `json:"new_email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.authClient.Client.RequestEmailChange(c, &authpb.RequestEmailChangeRequest{
		UserId:   userID,
		NewEmail: req.NewEmail,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Email занят или ошибка
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification link sent"})
}

// GET /api/v1/user/email/confirm
// Или /api/v1/auth/confirm-email-change - зависит от того, нужна ли авторизация
// Обычно подтверждение почты работает без токена доступа, только по токену из ссылки
func (h *UserHandler) ConfirmEmailChange(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token missing"})
		return
	}

	_, err := h.authClient.Client.ConfirmEmailChange(c, &authpb.ConfirmEmailChangeRequest{
		Token: token,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email updated successfully"})
}
