package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetToolUsage(c *gin.Context) {
	if h.usage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mysql is not configured"})
		return
	}
	count, err := h.usage.Get(c.Request.Context(), c.Param("key"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tool":  c.Param("key"),
		"count": count,
	})
}

func (h *Handler) IncrementToolUsage(c *gin.Context) {
	if h.usage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mysql is not configured"})
		return
	}
	count, err := h.usage.Increment(c.Request.Context(), c.Param("key"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"tool":  c.Param("key"),
		"count": count,
	})
}
