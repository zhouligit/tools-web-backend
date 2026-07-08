package main

import (
	"log"
	"os"

	"github.com/find-work/tools-web-backend/internal/bos"
	"github.com/find-work/tools-web-backend/internal/config"
	"github.com/find-work/tools-web-backend/internal/handler"
	"github.com/find-work/tools-web-backend/internal/service"
	"github.com/find-work/tools-web-backend/internal/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(cfg.TempDir, 0o755); err != nil {
		log.Fatalf("create temp dir: %v", err)
	}

	bosClient, err := bos.NewClient(cfg.BOS)
	if err != nil {
		log.Fatalf("init bos: %v", err)
	}

	st := store.NewTaskStore()
	taskSvc := service.NewTaskService(cfg, st, bosClient)
	h := handler.New(taskSvc)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))
	h.Register(r)

	log.Printf("tools-web-backend listening on %s", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
