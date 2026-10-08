package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/find-work/tools-web-backend/internal/model"
	"github.com/find-work/tools-web-backend/internal/store"
	"github.com/google/uuid"
)

type ReportService struct {
	store *store.ReportStore
}

func NewReportService(st *store.ReportStore) *ReportService {
	return &ReportService{store: st}
}

func (s *ReportService) Create(ctx context.Context, req model.CreateReportRequest) (*model.Report, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("mysql is not configured")
	}
	req.ReportType = strings.TrimSpace(req.ReportType)
	req.ReportCode = strings.TrimSpace(req.ReportCode)
	if req.ReportType == "" {
		return nil, fmt.Errorf("report_type is required")
	}
	if req.ReportCode == "" {
		return nil, fmt.Errorf("report_code is required")
	}

	now := time.Now()
	report := &model.Report{
		ID:                     uuid.NewString(),
		ReportType:             req.ReportType,
		ReportCode:             req.ReportCode,
		OrgFilingNo:            strings.TrimSpace(req.OrgFilingNo),
		AppraisalOrg:           strings.TrimSpace(req.AppraisalOrg),
		ProjectName:            strings.TrimSpace(req.ProjectName),
		BuildingAddress:        strings.TrimSpace(req.BuildingAddress),
		Conclusions:            req.Conclusions,
		ConclusionExplanations: req.ConclusionExplanations,
		PersonInCharge:         strings.TrimSpace(req.PersonInCharge),
		Reviewer:               strings.TrimSpace(req.Reviewer),
		Approver:               strings.TrimSpace(req.Approver),
		Appraisers:             strings.TrimSpace(req.Appraisers),
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.store.Create(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

func (s *ReportService) Get(ctx context.Context, id string) (*model.Report, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("mysql is not configured")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	return s.store.Get(ctx, id)
}
