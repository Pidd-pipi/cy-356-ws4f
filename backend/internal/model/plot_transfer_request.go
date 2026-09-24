package model

import "time"

// PlotTransferRequest 地块转交申请实体（认养人申请释放，管理员核准后清空认养关系）。
type PlotTransferRequest struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	PlotID         uint       `gorm:"not null;index" json:"plot_id"`
	Plot           *Plot      `gorm:"foreignKey:PlotID" json:"plot"`
	ApplicantID    uint       `gorm:"not null;index" json:"applicant_id"`
	Applicant      *User      `gorm:"foreignKey:ApplicantID" json:"applicant"`
	Reason         string     `gorm:"size:500;not null" json:"reason"`
	Status         string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	PrevPlotStatus string     `gorm:"size:32;not null" json:"prev_plot_status"`
	ReviewComment  string     `gorm:"size:500" json:"review_comment"`
	ReviewerID     *uint      `gorm:"index" json:"reviewer_id"`
	Reviewer       *User      `gorm:"foreignKey:ReviewerID" json:"reviewer"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
