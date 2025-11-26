package handlers

import (
	"api-gateway/internal/client"
	coursepb "api-gateway/pkg/coursepb/proto/course"
	userpb "api-gateway/pkg/userpb/proto/user" // <--- Добавили импорт
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	courseClient *client.CourseClient
	userClient   *client.UserClient // <--- Добавили клиент юзера
}

// Обновили конструктор
func NewCourseHandler(cc *client.CourseClient, uc *client.UserClient) *CourseHandler {
	return &CourseHandler{
		courseClient: cc,
		userClient:   uc,
	}
}

// GET /api/v1/courses
func (h *CourseHandler) List(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	res, err := h.courseClient.Client.GetCourses(c, &coursepb.GetCoursesRequest{
		Search:   search,
		Category: category,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// GET /api/v1/courses/:id
func (h *CourseHandler) GetOne(c *gin.Context) {
	userID := c.GetString("userId")
	courseID := c.Param("id")

	res, err := h.courseClient.Client.GetCourse(c, &coursepb.GetCourseRequest{
		CourseId: courseID,
		UserId:   userID,
	})

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	fmt.Printf("DEBUG GATEWAY: Found %d lessons for course %s\n", len(res.Lessons), courseID)

	c.JSON(http.StatusOK, res)
}

// POST /api/v1/courses
func (h *CourseHandler) Create(c *gin.Context) {
	// 1. Получаем ID текущего юзера
	userID := c.GetString("userId")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 2. Делаем запрос в User Service, чтобы узнать статус подписки
	profile, err := h.userClient.Client.GetProfile(c, &userpb.GetProfileRequest{
		UserId: userID,
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to verify user profile"})
		return
	}

	// 3. ПРОВЕРКА: Только admin может создавать
	if profile.SubscriptionStatus != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admins only"})
		return
	}

	// 4. Если админ, создаем курс
	var req coursepb.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.courseClient.Client.CreateCourse(c, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// DELETE /api/v1/courses/:id
func (h *CourseHandler) Delete(c *gin.Context) {
	// 1. Сюда тоже лучше добавить проверку на админа
	userID := c.GetString("userId")
	profile, err := h.userClient.Client.GetProfile(c, &userpb.GetProfileRequest{UserId: userID})
	if err != nil || profile.SubscriptionStatus != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	id := c.Param("id")
	_, err = h.courseClient.Client.DeleteCourse(c, &coursepb.DeleteCourseRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ... imports (убедись что userpb импортирован)

// POST /api/v1/courses/:id/start
func (h *CourseHandler) StartCourse(c *gin.Context) {
	userID := c.GetString("userId")
	courseID := c.Param("id")

	// Нам нужны Title и CoverURL. Обычно фронтенд их присылает,
	// либо мы можем сделать GetCourse запрос к courseClient, чтобы их достать.
	// Для простоты пусть фронт пришлет их в body, так быстрее (меньше запросов между сервисами)
	var req struct {
		Title    string `json:"title"`
		CoverURL string `json:"cover_url"`
	}
	// Если фронт не прислал, можно сделать запрос к h.courseClient.GetCourse
	// Но давай предположим, что фронт присылает для оптимизации.
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.userClient.Client.StartCourse(c, &userpb.StartCourseRequest{
		UserId:   userID,
		CourseId: courseID,
		Title:    req.Title,
		CoverUrl: req.CoverURL,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// POST /api/v1/courses/:id/progress
func (h *CourseHandler) UpdateProgress(c *gin.Context) {
	userID := c.GetString("userId")
	courseID := c.Param("id")

	var req struct {
		Percent int32 `json:"percent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.userClient.Client.UpdateProgress(c, &userpb.UpdateProgressRequest{
		UserId:          userID,
		CourseId:        courseID,
		ProgressPercent: req.Percent,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "status": res.Status})
}

// POST /api/v1/courses/:id/lessons/:lessonId/complete
func (h *CourseHandler) CompleteLesson(c *gin.Context) {
	userID := c.GetString("userId")
	courseID := c.Param("id")
	lessonID := c.Param("lessonId")

	// Получаем total_lessons из тела запроса (фронт знает длину массива)
	// Или можно спросить CourseService, но проще принять от фронта для скорости
	var req struct {
		TotalLessons int32 `json:"total_lessons"`
	}
	_ = c.ShouldBindJSON(&req)

	res, err := h.userClient.Client.CompleteLesson(c, &userpb.CompleteLessonRequest{
		UserId:       userID,
		CourseId:     courseID,
		LessonId:     lessonID,
		TotalLessons: req.TotalLessons,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"new_percent": res.NewPercent,
		"status":      res.Status,
	})
}
