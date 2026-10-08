package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func OpenMySQL(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("MYSQL_DSN is empty")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS reports (
  id VARCHAR(36) NOT NULL PRIMARY KEY,
  report_type VARCHAR(255) NOT NULL DEFAULT '',
  report_code VARCHAR(128) NOT NULL DEFAULT '',
  org_filing_no VARCHAR(64) NOT NULL DEFAULT '',
  appraisal_org VARCHAR(255) NOT NULL DEFAULT '',
  project_name VARCHAR(512) NOT NULL DEFAULT '',
  building_address VARCHAR(512) NOT NULL DEFAULT '',
  conclusions JSON NOT NULL,
  conclusion_explanations JSON NOT NULL,
  person_in_charge VARCHAR(128) NOT NULL DEFAULT '',
  reviewer VARCHAR(128) NOT NULL DEFAULT '',
  approver VARCHAR(128) NOT NULL DEFAULT '',
  appraisers VARCHAR(255) NOT NULL DEFAULT '',
  created_at DATETIME(3) NOT NULL,
  updated_at DATETIME(3) NOT NULL,
  KEY idx_reports_code (report_code),
  KEY idx_reports_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	return err
}
