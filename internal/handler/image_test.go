package handler

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/gin-gonic/gin"
)

func makePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 120, G: 80, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func imageMultipart(t *testing.T, filename string, data []byte, fields map[string]string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, w.FormDataContentType()
}

func TestConvertImageHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{images: imageproc.NewProcessor(), maxImageBytes: 20 * 1024 * 1024}
	r := gin.New()
	r.POST("/convert", h.ConvertImage)

	body, ctype := imageMultipart(t, "photo.png", makePNG(200, 150), map[string]string{
		"format":  "jpg",
		"quality": "85",
	})
	req := httptest.NewRequest(http.MethodPost, "/convert", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("X-Original-Size") == "" || w.Header().Get("X-Output-Size") == "" {
		t.Fatal("missing size headers")
	}
	if len(w.Body.Bytes()) == 0 {
		t.Fatal("empty body")
	}
}

func TestCompressImageHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{images: imageproc.NewProcessor(), maxImageBytes: 20 * 1024 * 1024}
	r := gin.New()
	r.POST("/compress", h.CompressImage)

	body, ctype := imageMultipart(t, "big.png", makePNG(800, 600), map[string]string{
		"quality":        "80",
		"max_edge":       "300",
		"output_format":  "webp",
	})
	req := httptest.NewRequest(http.MethodPost, "/compress", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "image/webp" {
		t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
	}
}

func TestConvertImageMissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &Handler{images: imageproc.NewProcessor(), maxImageBytes: 1024}
	r := gin.New()
	r.POST("/convert", h.ConvertImage)

	req := httptest.NewRequest(http.MethodPost, "/convert", bytes.NewReader(nil))
	req.Header.Set("Content-Type", "multipart/form-data")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
