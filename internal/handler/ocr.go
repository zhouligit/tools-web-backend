package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *Handler) OCRImage(c *gin.Context) {
	data, filename, err := h.readImageUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": imageUploadError(err)})
		return
	}
	lang := normalizeOCRLang(c.DefaultPostForm("lang", "ch"))
	result, err := h.ocr.Recognize(filename, data, lang)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func normalizeOCRLang(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "en", "english":
		return "en"
	case "ch_en", "auto", "mixed":
		return "ch"
	default:
		return "ch"
	}
}
