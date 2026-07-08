package model

import "time"

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskProcessing TaskStatus = "processing"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
)

type TaskType string

const (
	TaskMediaToText TaskType = "media_to_text"
)

type Segment struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

type Task struct {
	ID           string       `json:"id"`
	Type         TaskType     `json:"type"`
	Status       TaskStatus   `json:"status"`
	Progress     int          `json:"progress"`
	Stage        string       `json:"stage,omitempty"`
	DurationSec  float64      `json:"duration_sec,omitempty"`
	SourceURL    string       `json:"source_url,omitempty"`
	SourceFile   string       `json:"source_file,omitempty"`
	Language     string       `json:"language,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
	FullText     string       `json:"full_text,omitempty"`
	Segments     []Segment    `json:"segments,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

type CreateTaskRequest struct {
	SourceURL  string `json:"source_url"`
	Language   string `json:"language"`
}

type TaskListResponse struct {
	Items []Task `json:"items"`
	Total int    `json:"total"`
}
