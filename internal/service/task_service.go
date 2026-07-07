package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/find-work/tools-web-backend/internal/asr"
	"github.com/find-work/tools-web-backend/internal/bos"
	"github.com/find-work/tools-web-backend/internal/config"
	"github.com/find-work/tools-web-backend/internal/media"
	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/store"
	"github.com/google/uuid"
)

type TaskService struct {
	cfg       config.Config
	store     *store.TaskStore
	media     *media.Processor
	asr       *asr.Client
	bos       *bos.Client
}

func NewTaskService(cfg config.Config, st *store.TaskStore, bosClient *bos.Client) *TaskService {
	return &TaskService{
		cfg:   cfg,
		store: st,
		media: media.NewProcessor(cfg.TempDir, cfg.FFmpegPath, cfg.YtDlpPath),
		asr:   asr.NewClient(cfg.ASRServiceURL),
		bos:   bosClient,
	}
}

func (s *TaskService) CreateFromURL(sourceURL, language string) (*model.Task, error) {
	if strings.TrimSpace(sourceURL) == "" {
		return nil, fmt.Errorf("source_url is required")
	}
	if language == "" {
		language = "zh"
	}
	task := s.newTask(model.TaskMediaToText)
	task.SourceURL = sourceURL
	task.Language = language
	s.store.Save(task)
	go s.processTask(task.ID)
	return task, nil
}

func (s *TaskService) CreateFromUpload(filename string, data []byte, language string) (*model.Task, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty file")
	}
	maxBytes := s.cfg.MaxUploadMB * 1024 * 1024
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file too large, max %d MB", s.cfg.MaxUploadMB)
	}
	if language == "" {
		language = "zh"
	}
	task := s.newTask(model.TaskMediaToText)
	task.SourceFile = filename
	task.Language = language
	s.store.Save(task)

	taskDir, err := s.media.EnsureTempDir(task.ID)
	if err != nil {
		return nil, err
	}
	localPath, err := s.media.SaveUpload(taskDir, filename, data)
	if err != nil {
		return nil, err
	}
	if s.bos.Enabled() {
		key := s.bos.GenerateKey("uploads/"+task.ID, filepath.Base(localPath))
		if _, err := s.bos.UploadLocal(key, localPath); err != nil {
			log.Printf("bos upload warning: %v", err)
		} else {
			task.SourceFile = key
			s.store.Save(task)
		}
	}
	go s.processTask(task.ID)
	return task, nil
}

func (s *TaskService) Get(id string) (*model.Task, bool) {
	return s.store.Get(id)
}

func (s *TaskService) List(limit int) []model.Task {
	return s.store.List(limit)
}

func (s *TaskService) newTask(taskType model.TaskType) *model.Task {
	now := time.Now()
	return &model.Task{
		ID:        uuid.NewString(),
		Type:      taskType,
		Status:    model.TaskPending,
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *TaskService) processTask(taskID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	s.update(taskID, func(t *model.Task) {
		t.Status = model.TaskProcessing
		t.Progress = 5
	})

	task, ok := s.store.Get(taskID)
	if !ok {
		return
	}

	taskDir, err := s.media.EnsureTempDir(taskID)
	if err != nil {
		s.fail(taskID, err)
		return
	}
	defer s.media.Cleanup(taskID)

	var mediaPath string
	switch {
	case task.SourceURL != "":
		s.update(taskID, func(t *model.Task) { t.Progress = 15 })
		mediaPath, err = s.media.DownloadURL(ctx, taskDir, task.SourceURL)
	case task.SourceFile != "":
		s.update(taskID, func(t *model.Task) { t.Progress = 15 })
		localName := "upload" + filepath.Ext(filepath.Base(task.SourceFile))
		mediaPath = filepath.Join(taskDir, localName)
		if _, err := os.Stat(mediaPath); err != nil {
			if s.bos.Enabled() {
				if err = s.bos.DownloadToFile(task.SourceFile, mediaPath); err != nil {
					s.fail(taskID, err)
					return
				}
			} else {
				s.fail(taskID, fmt.Errorf("upload file not found: %s", mediaPath))
				return
			}
		}
	default:
		s.fail(taskID, fmt.Errorf("no media source"))
		return
	}
	if err != nil {
		s.fail(taskID, err)
		return
	}

	s.update(taskID, func(t *model.Task) { t.Progress = 40 })
	wavPath, err := s.media.ExtractAudio(ctx, taskDir, mediaPath)
	if err != nil {
		s.fail(taskID, err)
		return
	}

	if s.bos.Enabled() {
		key := s.bos.GenerateKey("audio/"+taskID, "audio_16k.wav")
		if _, err := s.bos.UploadLocal(key, wavPath); err != nil {
			log.Printf("bos audio upload warning: %v", err)
		}
	}

	s.update(taskID, func(t *model.Task) { t.Progress = 60 })
	text, segments, err := s.asr.Transcribe(wavPath, task.Language)
	if err != nil {
		s.fail(taskID, err)
		return
	}

	s.update(taskID, func(t *model.Task) {
		t.Status = model.TaskCompleted
		t.Progress = 100
		t.FullText = text
		t.Segments = segments
	})
}

func (s *TaskService) update(id string, fn func(*model.Task)) {
	s.store.Update(id, fn)
}

func (s *TaskService) fail(id string, err error) {
	log.Printf("task %s failed: %v", id, err)
	s.store.Update(id, func(t *model.Task) {
		t.Status = model.TaskFailed
		t.ErrorMessage = err.Error()
	})
}

func (s *TaskService) HealthChecks() map[string]string {
	status := map[string]string{
		"api": "ok",
		"asr": "unknown",
	}
	if err := s.asr.Health(); err != nil {
		status["asr"] = err.Error()
	} else {
		status["asr"] = "ok"
	}
	if !media.CommandExists(s.cfg.FFmpegPath) {
		status["ffmpeg"] = "missing"
	} else {
		status["ffmpeg"] = "ok"
	}
	return status
}
