package asr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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
			Timeout: 30 * time.Minute,
		},
	}
}

type transcribeResponse struct {
	Language string          `json:"language"`
	Text     string          `json:"text"`
	Segments []model.Segment `json:"segments"`
}

func (c *Client) Transcribe(wavPath, language string) (string, []model.Segment, error) {
	file, err := os.Open(wavPath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(wavPath))
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", nil, err
	}
	_ = writer.WriteField("language", language)
	if err := writer.Close(); err != nil {
		return "", nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/transcribe", body)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("asr service error %d: %s", resp.StatusCode, string(b))
	}

	var result transcribeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}
	return result.Text, result.Segments, nil
}

func (c *Client) Health() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("asr unhealthy: %d", resp.StatusCode)
	}
	return nil
}
