package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/find-work/tools-web-backend/internal/export"
	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/ocr"
	"github.com/find-work/tools-web-backend/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	tasks         *service.TaskService
	reports       *service.ReportService
	usage         *service.UsageService
	auth          *service.AuthService
	images        *imageproc.Processor
	ocr           *ocr.Client
	maxImageBytes int64
	cookieSecure  bool
}

func New(
	tasks *service.TaskService,
	reports *service.ReportService,
	usage *service.UsageService,
	auth *service.AuthService,
	images *imageproc.Processor,
	ocrClient *ocr.Client,
	maxImageMB int64,
	cookieSecure bool,
) *Handler {
	return &Handler{
		tasks:         tasks,
		reports:       reports,
		usage:         usage,
		auth:          auth,
		images:        images,
		ocr:           ocrClient,
		maxImageBytes: maxImageMB * 1024 * 1024,
		cookieSecure:  cookieSecure,
	}
}

func (h *Handler) Health(c *gin.Context) {
	checks := h.tasks.HealthChecks()
	if err := h.ocr.Health(); err != nil {
		checks["ocr"] = err.Error()
	} else {
		checks["ocr"] = "ok"
	}
	if h.reports == nil {
		checks["mysql"] = "disabled"
	} else {
		checks["mysql"] = "ok"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"checks":  checks,
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

	// 公开：登录注册、扫码看报告（微信）
	api.POST("/auth/register", h.RegisterUser)
	api.POST("/auth/login", h.Login)
	api.POST("/auth/logout", h.Logout)
	api.GET("/auth/me", h.Me)
	api.GET("/reports/:id", h.GetReport)

	authed := api.Group("")
	authed.Use(h.RequireAuth())
	{
		authed.POST("/tasks/media-to-text", h.CreateTask)
		authed.POST("/tasks/media-to-text/upload", h.UploadTask)
		authed.GET("/tasks/:id/srt", h.GetTaskSRT)
		authed.GET("/tasks/:id", h.GetTask)
		authed.GET("/tasks", h.ListTasks)

		authed.POST("/tools/image/convert", h.ConvertImage)
		authed.POST("/tools/image/compress", h.CompressImage)
		authed.POST("/tools/image/ocr", h.OCRImage)

		authed.POST("/reports", h.CreateReport)

		authed.GET("/tools/:key/usage", h.GetToolUsage)
		authed.POST("/tools/:key/usage", h.IncrementToolUsage)
	}
}
