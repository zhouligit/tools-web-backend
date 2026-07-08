package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	FrontendOrigin  string
	TempDir         string
	ASRServiceURL   string
	FFmpegPath      string
	YtDlpPath       string
	MaxUploadMB     int64
	TaskRetention   time.Duration
	BOS             BOSConfig
}

type BOSConfig struct {
	Enabled   bool
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

func Load() Config {
	maxMB, _ := strconv.ParseInt(getEnv("MAX_UPLOAD_MB", "500"), 10, 64)
	return Config{
		Addr:           getEnv("ADDR", ":18080"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:5173"),
		TempDir:        getEnv("TEMP_DIR", "/tmp/tools-web"),
		ASRServiceURL:  getEnv("ASR_SERVICE_URL", "http://127.0.0.1:18081"),
		FFmpegPath:     getEnv("FFMPEG_PATH", "ffmpeg"),
		YtDlpPath:      getEnv("YTDLP_PATH", "yt-dlp"),
		MaxUploadMB:    maxMB,
		TaskRetention:  24 * time.Hour,
		BOS: BOSConfig{
			Enabled:   getEnv("BOS_ENABLED", "false") == "true",
			Endpoint:  getEnv("BOS_ENDPOINT", "https://bj.bcebos.com"),
			Bucket:    getEnv("BOS_BUCKET", ""),
			AccessKey: getEnv("BOS_ACCESS_KEY", ""),
			SecretKey: getEnv("BOS_SECRET_KEY", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
