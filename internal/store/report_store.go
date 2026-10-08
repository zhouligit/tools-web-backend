package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/find-work/tools-web-backend/internal/model"
)

type ReportStore struct {
	db *sql.DB
}

func NewReportStore(db *sql.DB) *ReportStore {
	return &ReportStore{db: db}
}

func (s *ReportStore) Create(ctx context.Context, report *model.Report) error {
	conclusions, err := json.Marshal(normalizeSlots(report.Conclusions))
	if err != nil {
		return err
	}
	explanations, err := json.Marshal(normalizeSlots(report.ConclusionExplanations))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO reports (
  id, report_type, report_code, org_filing_no, appraisal_org,
  project_name, building_address, conclusions, conclusion_explanations,
  person_in_charge, reviewer, approver, appraisers, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID,
		report.ReportType,
		report.ReportCode,
		report.OrgFilingNo,
		report.AppraisalOrg,
		report.ProjectName,
		report.BuildingAddress,
		conclusions,
		explanations,
		report.PersonInCharge,
		report.Reviewer,
		report.Approver,
		report.Appraisers,
		report.CreatedAt,
		report.UpdatedAt,
	)
	return err
}

func (s *ReportStore) Get(ctx context.Context, id string) (*model.Report, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, report_type, report_code, org_filing_no, appraisal_org,
       project_name, building_address, conclusions, conclusion_explanations,
       person_in_charge, reviewer, approver, appraisers, created_at, updated_at
FROM reports WHERE id = ?`, id)

	var report model.Report
	var conclusionsRaw, explanationsRaw []byte
	err := row.Scan(
		&report.ID,
		&report.ReportType,
		&report.ReportCode,
		&report.OrgFilingNo,
		&report.AppraisalOrg,
		&report.ProjectName,
		&report.BuildingAddress,
		&conclusionsRaw,
		&explanationsRaw,
		&report.PersonInCharge,
		&report.Reviewer,
		&report.Approver,
		&report.Appraisers,
		&report.CreatedAt,
		&report.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	report.Conclusions, err = decodeSlots(conclusionsRaw)
	if err != nil {
		return nil, fmt.Errorf("decode conclusions: %w", err)
	}
	report.ConclusionExplanations, err = decodeSlots(explanationsRaw)
	if err != nil {
		return nil, fmt.Errorf("decode conclusion_explanations: %w", err)
	}
	return &report, nil
}

func normalizeSlots(items []string) []string {
	out := make([]string, 3)
	for i := 0; i < 3 && i < len(items); i++ {
		out[i] = items[i]
	}
	return out
}

func decodeSlots(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{"", "", ""}, nil
	}
	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	return normalizeSlots(items), nil
}
