package handlers

import (
	"api-gateway/internal/client"
	coursepb "api-gateway/pkg/coursepb/proto/course"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CourseHandler struct {
	client *client.CourseClient
}

func NewCourseHandler(client *client.CourseClient) *CourseHandler {
	return &CourseHandler{client: client}
}

// GET /api/v1/courses
func (h *CourseHandler) List(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	res, err := h.client.Client.GetCourses(c, &coursepb.GetCoursesRequest{
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
	userID := c.GetString("userId") // Будет пустой, если нет токена (зависит от middleware)
	courseID := c.Param("id")

	res, err := h.client.Client.GetCourse(c, &coursepb.GetCourseRequest{
		CourseId: courseID,
		UserId:   userID,
	})

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}

	c.JSON(http.StatusOK, res)
}

// POST /api/v1/courses
func (h *CourseHandler) Create(c *gin.Context) {
	var req coursepb.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.client.Client.CreateCourse(c, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// DELETE /api/v1/courses/:id
func (h *CourseHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	_, err := h.client.Client.DeleteCourse(c, &coursepb.DeleteCourseRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
