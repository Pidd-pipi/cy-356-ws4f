package model

import "time"

// PlotTransfer 地块转交（释放）申请：认养人发起，管理员核准后地块才回到共享池。
// 状态取值与 constants.TransferStatus 保持一致：
// pending（待处理）/ approved（已核准）/ rejected（已驳回）/ withdrawn（已撤回）。
type PlotTransfer struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	PlotID      uint   `gorm:"not null;index:idx_plot_transfer_plot,priority:1" json:"plot_id"`
	Plot        *Plot  `gorm:"foreignKey:PlotID" json:"plot"`
	ApplicantID uint   `gorm:"not null;index" json:"applicant_id"`
	Applicant   *User  `gorm:"foreignKey:ApplicantID" json:"applicant"`
	Reason      string `gorm:"size:512;not null" json:"reason"`
	// 部分唯一索引（仅 pending 行）由迁移脚本与 service 事务共同保证：同一地块同时只允许一条待处理申请。
	Status         string     `gorm:"size:32;not null;default:pending;index:idx_plot_transfer_plot,priority:2" json:"status"`
	PreviousStatus string     `gorm:"size:32;not null" json:"previous_status"`
	ReviewerID     *uint      `gorm:"index" json:"reviewer_id"`
	Reviewer       *User      `gorm:"foreignKey:ReviewerID" json:"reviewer"`
	ReviewComment  string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
