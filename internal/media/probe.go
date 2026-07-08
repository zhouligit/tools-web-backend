package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type MediaInfo struct {
	HasAudio    bool
	HasVideo    bool
	DurationSec float64
	Format      string
}

type ffprobeOutput struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Format   string `json:"format_name"`
	} `json:"format"`
}

func (p *Processor) ffprobePath() string {
	if strings.Contains(p.ffmpegPath, "ffmpeg") {
		return strings.Replace(p.ffmpegPath, "ffmpeg", "ffprobe", 1)
	}
	return "ffprobe"
}

func (p *Processor) ProbeMedia(ctx context.Context, inputPath string) (*MediaInfo, error) {
	cmd := exec.CommandContext(ctx, p.ffprobePath(),
		"-v", "error",
		"-show_entries", "format=duration,format_name:stream=codec_type",
		"-of", "json",
		inputPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var parsed ffprobeOutput
	if err := json.Unmarshal(output, &parsed); err != nil {
		return nil, fmt.Errorf("ffprobe parse failed: %w", err)
	}

	info := &MediaInfo{Format: parsed.Format.Format}
	for _, stream := range parsed.Streams {
		switch stream.CodecType {
		case "audio":
			info.HasAudio = true
		case "video":
			info.HasVideo = true
		}
	}
	if parsed.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(parsed.Format.Duration, 64); err == nil {
			info.DurationSec = duration
		}
	}
	return info, nil
}

func (p *Processor) ValidateMedia(ctx context.Context, inputPath string, maxDurationSec int) (*MediaInfo, error) {
	info, err := p.ProbeMedia(ctx, inputPath)
	if err != nil {
		return nil, err
	}
	if !info.HasAudio {
		return info, fmt.Errorf("no audio track found in media file")
	}
	if maxDurationSec > 0 && info.DurationSec > float64(maxDurationSec) {
		return info, fmt.Errorf(
			"media too long: %.0fs (max %ds)",
			info.DurationSec,
			maxDurationSec,
		)
	}
	return info, nil
}
