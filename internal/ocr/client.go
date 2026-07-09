package ocr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/find-work/tools-web-backend/internal/model"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type recognizeResponse struct {
	Text       string           `json:"text"`
	Lines      []model.OCRLine  `json:"lines"`
	LineCount  int              `json:"line_count"`
	DurationMS int              `json:"duration_ms"`
}

func (c *Client) Recognize(filename string, data []byte, lang string) (*model.OCRResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return nil, err
	}
	_ = writer.WriteField("lang", lang)
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/recognize", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ocr service error %d: %s", resp.StatusCode, string(b))
	}

	var result recognizeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &model.OCRResult{
		Text:       result.Text,
		Lines:      result.Lines,
		LineCount:  result.LineCount,
		DurationMS: result.DurationMS,
	}, nil
}

func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ocr unhealthy: %d", resp.StatusCode)
	}
	return nil
}
