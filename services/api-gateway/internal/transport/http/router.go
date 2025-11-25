package handlers

import (
	"api-gateway/internal/client"
	"api-gateway/internal/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(authHandler *AuthHandler, userHandler *UserHandler, limiter *middleware.RateLimiter, authClient *client.AuthClient, courseHandler *CourseHandler) *gin.Engine {
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"https://bazakursov.ru", "http://bazakursov.ru", "https://www.bazakursov.ru"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}
	r.Use(cors.New(config))

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", limiter.Limit("login", 5, 1*time.Minute), authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			auth.POST("/forgot-password", limiter.Limit("forgot_pass", 1, 5*time.Minute), authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
		}
		api.GET("/user/email/confirm", userHandler.ConfirmEmailChange)
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware(authClient))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/avatar", userHandler.SetAvatar)
			user.POST("/email/change", userHandler.RequestEmailChange)
			user.GET("/devices", authHandler.GetDevices)
			user.DELETE("/devices/:id", authHandler.RemoveDevice)
		}
		course := api.Group("/courses")
		course.Use(middleware.AuthMiddleware(authClient))
		{
			course.GET("", courseHandler.List)
			course.GET("/:id", courseHandler.GetOne)
			course.POST("", courseHandler.Create)
			course.DELETE("/:id", courseHandler.Delete)
		}
	}

	return r
}
