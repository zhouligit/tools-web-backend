package main

import (
	"log"
	"os"

	"github.com/find-work/tools-web-backend/internal/bos"
	"github.com/find-work/tools-web-backend/internal/config"
	"github.com/find-work/tools-web-backend/internal/db"
	"github.com/find-work/tools-web-backend/internal/handler"
	"github.com/find-work/tools-web-backend/internal/imageproc"
	"github.com/find-work/tools-web-backend/internal/ocr"
	"github.com/find-work/tools-web-backend/internal/service"
	"github.com/find-work/tools-web-backend/internal/store"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

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

	var reportSvc *service.ReportService
	if cfg.MySQLDSN != "" {
		sqlDB, err := db.OpenMySQL(cfg.MySQLDSN)
		if err != nil {
			log.Fatalf("init mysql: %v", err)
		}
		defer sqlDB.Close()
		reportSvc = service.NewReportService(store.NewReportStore(sqlDB))
		log.Printf("mysql connected, reports persistence enabled")
	} else {
		log.Printf("MYSQL_DSN empty, report APIs disabled")
	}

	h := handler.New(taskSvc, reportSvc, imageproc.NewProcessor(), ocr.NewClient(cfg.OCRServiceURL), cfg.MaxImageMB)

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendOrigins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"X-Original-Size", "X-Output-Size", "Content-Disposition"},
		AllowCredentials: true,
	}))
	h.Register(r)

	log.Printf("tools-web-backend listening on %s", cfg.Addr)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
