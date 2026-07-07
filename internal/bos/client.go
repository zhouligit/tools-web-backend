package bos

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"
	"github.com/find-work/tools-web-backend/internal/config"
)

type Client struct {
	enabled bool
	client  *bos.Client
	bucket  string
}

func NewClient(cfg config.BOSConfig) (*Client, error) {
	if !cfg.Enabled {
		return &Client{enabled: false}, nil
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("BOS enabled but bucket/access_key/secret_key missing")
	}
	client, err := bos.NewClient(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{enabled: true, client: client, bucket: cfg.Bucket}, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled
}

func (c *Client) UploadLocal(key, localPath string) (string, error) {
	if !c.enabled {
		return localPath, nil
	}
	_, err := c.client.PutObjectFromFile(c.bucket, key, localPath, nil)
	if err != nil {
		return "", err
	}
	return key, nil
}

func (c *Client) UploadReader(key string, r io.ReadSeeker) error {
	if !c.enabled {
		return nil
	}
	_, err := c.client.PutObjectFromStream(c.bucket, key, r, nil)
	return err
}

func (c *Client) DownloadToFile(key, localPath string) error {
	if !c.enabled {
		if _, err := os.Stat(key); err != nil {
			return fmt.Errorf("local file not found: %s", key)
		}
		return copyFile(key, localPath)
	}
	return c.client.BasicGetObjectToFile(c.bucket, key, localPath)
}

func (c *Client) PublicURL(key string) string {
	if !c.enabled {
		return key
	}
	return fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(c.client.Config.Endpoint, "/"), c.bucket, key)
}

func (c *Client) GenerateKey(prefix, filename string) string {
	return filepath.ToSlash(filepath.Join(prefix, filename))
}

func (c *Client) BasicGetObject(key string) (*api.GetObjectResult, error) {
	return c.client.BasicGetObject(c.bucket, key)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
