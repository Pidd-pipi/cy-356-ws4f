package dto

import (
	"github.com/communitygarden/server/internal/model"
)

// CreateTransferRequestRequest 认养人提交地块转交申请（填写接管理由）。
type CreateTransferRequestRequest struct {
	Reason string `json:"reason" binding:"required,min=2,max=500"`
}

// ApproveTransferRequestRequest 管理员核准转交申请（处理意见可选）。
type ApproveTransferRequestRequest struct {
	ReviewComment string `json:"review_comment" binding:"omitempty,max=500"`
}

// RejectTransferRequestRequest 管理员驳回转交申请（必须写明处理意见）。
type RejectTransferRequestRequest struct {
	ReviewComment string `json:"review_comment" binding:"required,min=2,max=500"`
}

// TransferRequestOutDTO 转交申请输出（列表与详情保留申请原因、处理意见及当前状态）。
type TransferRequestOutDTO struct {
	ID             uint        `json:"id"`
	PlotID         uint        `json:"plot_id"`
	Plot           *PlotOutDTO `json:"plot"`
	ApplicantID    uint        `json:"applicant_id"`
	Applicant      *UserOutDTO `json:"applicant"`
	Reason         string      `json:"reason"`
	Status         string      `json:"status"`
	PrevPlotStatus string      `json:"prev_plot_status"`
	ReviewComment  string      `json:"review_comment"`
	ReviewerID     *uint       `json:"reviewer_id"`
	Reviewer       *UserOutDTO `json:"reviewer"`
	ReviewedAt     string      `json:"reviewed_at"`
	CreatedAt      string      `json:"created_at"`
}

// ToTransferRequestOutDTO 模型转 DTO。
func ToTransferRequestOutDTO(r *model.PlotTransferRequest) *TransferRequestOutDTO {
	out := &TransferRequestOutDTO{
		ID:             r.ID,
		PlotID:         r.PlotID,
		ApplicantID:    r.ApplicantID,
		Reason:         r.Reason,
		Status:         r.Status,
		PrevPlotStatus: r.PrevPlotStatus,
		ReviewComment:  r.ReviewComment,
		ReviewerID:     r.ReviewerID,
		CreatedAt:      r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if r.Plot != nil {
		out.Plot = ToPlotOutDTO(r.Plot)
	}
	if r.Applicant != nil {
		out.Applicant = ToUserOutDTO(r.Applicant)
	}
	if r.Reviewer != nil {
		out.Reviewer = ToUserOutDTO(r.Reviewer)
	}
	if r.ReviewedAt != nil {
		out.ReviewedAt = r.ReviewedAt.Format("2006-01-02 15:04:05")
	}
	return out
}
