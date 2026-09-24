package dto

import (
	"github.com/communitygarden/server/internal/model"
)

// CreateTransferRequest 认养人提交转交（释放）申请。
type CreateTransferRequest struct {
	Reason string `json:"reason" binding:"required,max=512"`
}

// ReviewTransferRequest 管理员核准/驳回转交申请（approve 必填；reject 时意见必填）。
type ReviewTransferRequest struct {
	// 指针类型：区分"未传"与"显式 false"，漏传时返回校验错误而非误走驳回。
	Approve *bool  `json:"approve" binding:"required"`
	Comment string `json:"comment" binding:"omitempty,max=512"`
}

// PlotBriefDTO 申请单中内嵌的地块摘要（避免与 PlotOutDTO 循环引用）。
type PlotBriefDTO struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// TransferApplicationOutDTO 转交申请输出（列表/详情共用）。
type TransferApplicationOutDTO struct {
	ID            uint          `json:"id"`
	PlotID        uint          `json:"plot_id"`
	Plot          *PlotBriefDTO `json:"plot"`
	ApplicantID   uint          `json:"applicant_id"`
	Applicant     *UserOutDTO   `json:"applicant"`
	Reason        string        `json:"reason"`
	Status        string        `json:"status"`
	ReviewerID    *uint         `json:"reviewer_id"`
	Reviewer      *UserOutDTO   `json:"reviewer"`
	ReviewComment string        `json:"review_comment"`
	ReviewedAt    string        `json:"reviewed_at"`
	CreatedAt     string        `json:"created_at"`
}

// ToTransferApplicationOutDTO 模型转 DTO。
func ToTransferApplicationOutDTO(t *model.PlotTransfer) *TransferApplicationOutDTO {
	out := &TransferApplicationOutDTO{
		ID:            t.ID,
		PlotID:        t.PlotID,
		ApplicantID:   t.ApplicantID,
		Reason:        t.Reason,
		Status:        t.Status,
		ReviewerID:    t.ReviewerID,
		ReviewComment: t.ReviewComment,
		CreatedAt:     t.CreatedAt.Format("2006-01-02 15:04:05"),
	}
	if t.Plot != nil {
		out.Plot = &PlotBriefDTO{ID: t.Plot.ID, Name: t.Plot.Name, Code: t.Plot.Code}
	}
	if t.Applicant != nil {
		out.Applicant = ToUserOutDTO(t.Applicant)
	}
	if t.Reviewer != nil {
		out.Reviewer = ToUserOutDTO(t.Reviewer)
	}
	if t.ReviewedAt != nil {
		out.ReviewedAt = t.ReviewedAt.Format("2006-01-02 15:04:05")
	}
	return out
}
