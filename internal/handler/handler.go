package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/find-work/tools-web-backend/internal/export"
	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	tasks          *service.TaskService
	images         *imageproc.Processor
	maxImageBytes  int64
}

func New(tasks *service.TaskService, images *imageproc.Processor, maxImageMB int64) *Handler {
	return &Handler{
		tasks:         tasks,
		images:        images,
		maxImageBytes: maxImageMB * 1024 * 1024,
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"checks":  h.tasks.HealthChecks(),
		"service": "tools-web-backend",
	})
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req model.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := h.tasks.CreateFromURL(req.SourceURL, req.Language)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, task)
}

func (h *Handler) UploadTask(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	language := c.DefaultPostForm("language", "zh")
	task, err := h.tasks.CreateFromUpload(file.Filename, data, language)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, task)
}

func (h *Handler) GetTask(c *gin.Context) {
	task, ok := h.tasks.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) ListTasks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	c.JSON(http.StatusOK, model.TaskListResponse{
		Items: h.tasks.List(limit),
		Total: len(h.tasks.List(0)),
	})
}

func (h *Handler) GetTaskSRT(c *gin.Context) {
	task, ok := h.tasks.Get(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	if task.Status != model.TaskCompleted || len(task.Segments) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task result not ready"})
		return
	}
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="transcript.srt"`)
	c.String(http.StatusOK, export.ToSRT(task.Segments))
}

func (h *Handler) Register(r *gin.Engine) {
	api := r.Group("/api/v1")
	api.GET("/health", h.Health)
	api.POST("/tasks/media-to-text", h.CreateTask)
	api.POST("/tasks/media-to-text/upload", h.UploadTask)
	api.POST("/tools/image/convert", h.ConvertImage)
	api.POST("/tools/image/compress", h.CompressImage)
	api.GET("/tasks/:id/srt", h.GetTaskSRT)
	api.GET("/tasks/:id", h.GetTask)
	api.GET("/tasks", h.ListTasks)
}
