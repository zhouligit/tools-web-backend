package handler

import (
	"net/http"

	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateReport(c *gin.Context) {
	if h.reports == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mysql is not configured"})
		return
	}
	var req model.CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	report, err := h.reports.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, report)
}

func (h *Handler) GetReport(c *gin.Context) {
	if h.reports == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mysql is not configured"})
		return
	}
	report, err := h.reports.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	c.JSON(http.StatusOK, report)
}
