package model

import "time"

type Report struct {
	ID                      string    `json:"id"`
	ReportType              string    `json:"report_type"`
	ReportCode              string    `json:"report_code"`
	OrgFilingNo             string    `json:"org_filing_no"`
	AppraisalOrg            string    `json:"appraisal_org"`
	ProjectName             string    `json:"project_name"`
	BuildingAddress         string    `json:"building_address"`
	Conclusions             []string  `json:"conclusions"`
	ConclusionExplanations  []string  `json:"conclusion_explanations"`
	PersonInCharge          string    `json:"person_in_charge"`
	Reviewer                string    `json:"reviewer"`
	Approver                string    `json:"approver"`
	Appraisers              string    `json:"appraisers"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type CreateReportRequest struct {
	ReportType             string   `json:"report_type"`
	ReportCode             string   `json:"report_code"`
	OrgFilingNo            string   `json:"org_filing_no"`
	AppraisalOrg           string   `json:"appraisal_org"`
	ProjectName            string   `json:"project_name"`
	BuildingAddress        string   `json:"building_address"`
	Conclusions            []string `json:"conclusions"`
	ConclusionExplanations []string `json:"conclusion_explanations"`
	PersonInCharge         string   `json:"person_in_charge"`
	Reviewer               string   `json:"reviewer"`
	Approver               string   `json:"approver"`
	Appraisers             string   `json:"appraisers"`
}
