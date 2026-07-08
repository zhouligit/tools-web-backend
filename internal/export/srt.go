package export

import (
	"fmt"
	"strings"

	"github.com/find-work/tools-web-backend/internal/model"
)

func ToSRT(segments []model.Segment) string {
	if len(segments) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range segments {
		text := strings.TrimSpace(seg.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&b, "%d\n", i+1)
		fmt.Fprintf(&b, "%s --> %s\n", formatSRTTime(seg.Start), formatSRTTime(seg.End))
		fmt.Fprintf(&b, "%s\n\n", text)
	}
	return b.String()
}

func formatSRTTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	totalMs := int(sec * 1000)
	ms := totalMs % 1000
	totalSec := totalMs / 1000
	s := totalSec % 60
	totalMin := totalSec / 60
	m := totalMin % 60
	h := totalMin / 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
