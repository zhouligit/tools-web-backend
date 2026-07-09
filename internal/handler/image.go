package handler

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/gin-gonic/gin"
)

func (h *Handler) readImageUpload(c *gin.Context) ([]byte, string, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return nil, "", err
	}
	src, err := file.Open()
	if err != nil {
		return nil, "", err
	}
	defer src.Close()
	data, err := imageproc.ReadImageFile(src, h.maxImageBytes)
	if err != nil {
		return nil, "", err
	}
	return data, file.Filename, nil
}

func (h *Handler) writeImageResult(c *gin.Context, originalName string, result *imageproc.Result) {
	filename := imageproc.OutputFilename(originalName, result.Ext)
	c.Header("Content-Type", result.MIME)
	c.Header("Content-Disposition", attachmentDisposition(filename))
	c.Header("X-Original-Size", strconv.Itoa(result.OriginalSize))
	c.Header("X-Output-Size", strconv.Itoa(result.OutputSize))
	c.Header("Access-Control-Expose-Headers", "X-Original-Size, X-Output-Size, Content-Disposition")
	c.Data(http.StatusOK, result.MIME, result.Data)
}

func attachmentDisposition(filename string) string {
	return mime.FormatMediaType("attachment", map[string]string{
		"filename":  asciiFilename(filename),
		"filename*": "UTF-8''" + url.PathEscape(filename),
	})
}

func asciiFilename(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 128 && r != '"' && r != '\\' {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) {
			b.WriteByte('_')
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "image." + strings.TrimPrefix(extWithDot(name), ".")
	}
	return b.String()
}

func extWithDot(filename string) string {
	idx := strings.LastIndex(filename, ".")
	if idx <= 0 {
		return ""
	}
	return filename[idx:]
}

func (h *Handler) ConvertImage(c *gin.Context) {
	data, filename, err := h.readImageUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": imageUploadError(err)})
		return
	}
	format, err := imageproc.ParseFormat(c.DefaultPostForm("format", "jpg"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	quality, _ := strconv.Atoi(c.DefaultPostForm("quality", "85"))
	result, err := h.images.Convert(data, filename, format, quality)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.writeImageResult(c, filename, result)
}

func (h *Handler) CompressImage(c *gin.Context) {
	data, filename, err := h.readImageUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": imageUploadError(err)})
		return
	}
	quality, _ := strconv.Atoi(c.DefaultPostForm("quality", "85"))
	maxEdge, _ := strconv.Atoi(c.DefaultPostForm("max_edge", "0"))
	outputFormat := c.DefaultPostForm("output_format", "keep")
	result, err := h.images.Compress(data, quality, maxEdge, outputFormat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.writeImageResult(c, filename, result)
}

func imageUploadError(err error) string {
	if err == io.EOF || strings.Contains(err.Error(), "file is required") {
		return "file is required"
	}
	return err.Error()
}
