package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	FrontendOrigins  []string
	TempDir          string
	ASRServiceURL   string
	FFmpegPath      string
	YtDlpPath       string
	MaxUploadMB     int64
	MaxDurationSec  int
	MaxImageMB      int64
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
	maxDuration, _ := strconv.Atoi(getEnv("MAX_DURATION_SEC", "7200"))
	maxImageMB, _ := strconv.ParseInt(getEnv("MAX_IMAGE_MB", "20"), 10, 64)
	return Config{
		Addr:            getEnv("ADDR", ":18080"),
		FrontendOrigins: parseOrigins(getEnv("FRONTEND_ORIGIN", "http://localhost:5173")),
		TempDir:        getEnv("TEMP_DIR", "/tmp/tools-web"),
		ASRServiceURL:  getEnv("ASR_SERVICE_URL", "http://127.0.0.1:18081"),
		FFmpegPath:     getEnv("FFMPEG_PATH", "ffmpeg"),
		YtDlpPath:      getEnv("YTDLP_PATH", "yt-dlp"),
		MaxUploadMB:    maxMB,
		MaxDurationSec: maxDuration,
		MaxImageMB:     maxImageMB,
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

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
