package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/find-work/tools-web-backend/internal/ocr"
	"github.com/gin-gonic/gin"
)

func mockOCRServer(t *testing.T, recognize func(w http.ResponseWriter, r *http.Request)) *ocr.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/v1/recognize":
			recognize(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return ocr.NewClient(srv.URL)
}

func TestOCRImageHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ocrClient := mockOCRServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text":        "你好世界",
			"lines":       []map[string]any{{"text": "你好世界", "confidence": 0.99, "box": [][]float64{{0, 0}}}},
			"line_count":  1,
			"duration_ms": 120,
		})
	})
	h := &Handler{
		images:        imageproc.NewProcessor(),
		ocr:           ocrClient,
		maxImageBytes: 20 * 1024 * 1024,
	}
	r := gin.New()
	r.POST("/ocr", h.OCRImage)

	body, ctype := imageMultipart(t, "note.png", makePNG(120, 80), map[string]string{"lang": "ch"})
	req := httptest.NewRequest(http.MethodPost, "/ocr", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "你好世界") {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestOCRImageHandlerServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ocrClient := mockOCRServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	h := &Handler{
		images:        imageproc.NewProcessor(),
		ocr:           ocrClient,
		maxImageBytes: 20 * 1024 * 1024,
	}
	r := gin.New()
	r.POST("/ocr", h.OCRImage)

	body, ctype := imageMultipart(t, "note.png", makePNG(10, 10), nil)
	req := httptest.NewRequest(http.MethodPost, "/ocr", body)
	req.Header.Set("Content-Type", ctype)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}
