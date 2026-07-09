package model

type OCRLine struct {
	Text       string      `json:"text"`
	Confidence float64     `json:"confidence"`
	Box        [][]float64 `json:"box"`
}

type OCRResult struct {
	Text       string    `json:"text"`
	Lines      []OCRLine `json:"lines"`
	LineCount  int       `json:"line_count"`
	DurationMS int       `json:"duration_ms"`
}
