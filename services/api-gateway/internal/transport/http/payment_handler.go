package handlers

import (
	"api-gateway/internal/client"
	paymentpb "api-gateway/pkg/paymentpb/proto/payment"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	client *client.PaymentClient
}

func NewPaymentHandler(client *client.PaymentClient) *PaymentHandler {
	return &PaymentHandler{client: client}
}

func (h *PaymentHandler) GetPlans(c *gin.Context) {
	// Делаем пустой запрос по gRPC
	res, err := h.client.Client.GetPlans(c, &paymentpb.GetPlansRequest{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить список тарифов: " + err.Error()})
		return
	}

	// Возвращаем список планов
	c.JSON(http.StatusOK, res.Plans)
}

func (h *PaymentHandler) Redeem(c *gin.Context) {
	// 1. Достаем ID пользователя из контекста (положил AuthMiddleware)
	userId := c.GetString("userId")
	if userId == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Парсим JSON тело запроса
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поле 'code' обязательно"})
		return
	}

	// 3. Отправляем gRPC запрос в Payment Service
	res, err := h.client.Client.RedeemPromo(c, &paymentpb.RedeemPromoRequest{
		UserId: userId,
		Code:   req.Code,
	})

	// 4. Обработка ошибок (например, код не найден или истек)
	if err != nil {
		// status.Code(err) можно использовать для более точного маппинга,
		// но пока вернем 400 и текст ошибки
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 5. Успех
	c.JSON(http.StatusOK, gin.H{
		"success":    res.Success,
		"message":    res.Message,
		"plan_name":  res.PlanName,
		"expires_at": res.ExpiresAt,
	})
}
