package media

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Processor struct {
	tempDir    string
	ffmpegPath string
	ytdlpPath  string
}

func NewProcessor(tempDir, ffmpegPath, ytdlpPath string) *Processor {
	return &Processor{
		tempDir:    tempDir,
		ffmpegPath: ffmpegPath,
		ytdlpPath:  ytdlpPath,
	}
}

func (p *Processor) EnsureTempDir(taskID string) (string, error) {
	dir := filepath.Join(p.tempDir, taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (p *Processor) DownloadURL(ctx context.Context, taskDir, sourceURL string) (string, error) {
	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		outTpl := filepath.Join(taskDir, "source.%(ext)s")
		cmd := exec.CommandContext(ctx, p.ytdlpPath,
			"-f", "bestaudio/best",
			"--no-playlist",
			"-o", outTpl,
			sourceURL,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("yt-dlp failed: %w, output: %s", err, string(output))
		}
		return findFirstFile(taskDir, "source")
	}
	return "", fmt.Errorf("unsupported url: %s", sourceURL)
}

func (p *Processor) ExtractAudio(ctx context.Context, taskDir, inputPath string) (string, error) {
	wavPath := filepath.Join(taskDir, "audio_16k.wav")
	cmd := exec.CommandContext(ctx, p.ffmpegPath,
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		wavPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}
	return wavPath, nil
}

func (p *Processor) SaveUpload(taskDir, originalName string, data []byte) (string, error) {
	ext := filepath.Ext(originalName)
	if ext == "" {
		ext = ".bin"
	}
	path := filepath.Join(taskDir, "upload"+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (p *Processor) Cleanup(taskID string) {
	dir := filepath.Join(p.tempDir, taskID)
	_ = os.RemoveAll(dir)
}

func findFirstFile(dir, prefix string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("downloaded file not found in %s", dir)
}

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func WaitContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
