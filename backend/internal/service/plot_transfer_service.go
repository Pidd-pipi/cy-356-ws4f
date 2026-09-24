package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/util"
)

// PlotTransferService 地块转交申请服务（多步写操作全部在事务内，并发用 SELECT FOR UPDATE）。
type PlotTransferService struct {
	transferRepo repository.PlotTransferRepository
	plotRepo     repository.PlotRepository
	db           *gorm.DB
	logger       *slog.Logger
}

// NewPlotTransferService 构造地块转交申请服务。
func NewPlotTransferService(transferRepo repository.PlotTransferRepository, plotRepo repository.PlotRepository, db *gorm.DB, logger *slog.Logger) *PlotTransferService {
	return &PlotTransferService{transferRepo: transferRepo, plotRepo: plotRepo, db: db, logger: logger}
}

// Submit 认养人提交转交申请（adopted/harvested -> transfer_pending，期间其他居民不可认养）。
func (s *PlotTransferService) Submit(plotID, applicantID uint, reason string) (*model.PlotTransferRequest, error) {
	var created *model.PlotTransferRequest
	err := s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, plotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if plot.Status == string(constants.PlotStatusTransferPending) {
			return util.NewAppError(constants.CodeTransferDuplicate, 409, fmt.Sprintf("地块 %s 已有待处理的转交申请，请勿重复提交", plot.Code))
		}
		if plot.Status != string(constants.PlotStatusAdopted) && plot.Status != string(constants.PlotStatusHarvested) {
			return util.NewAppError(constants.CodeTransferPlotState, 409, fmt.Sprintf("地块 %s 当前状态为 %s，仅已认养或待释放状态可提交转交申请", plot.Code, util.PlotStatusText(plot.Status)))
		}
		if plot.AdopterID == nil || *plot.AdopterID != applicantID {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("用户 id=%d 不是地块 %s 的认养人，无权提交转交申请", applicantID, plot.Code))
		}
		exists, err := s.transferRepo.ExistsPendingByPlot(tx, plotID)
		if err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if exists {
			return util.NewAppError(constants.CodeTransferDuplicate, 409, fmt.Sprintf("地块 %s 已有待处理的转交申请，请勿重复提交", plot.Code))
		}
		req := &model.PlotTransferRequest{
			PlotID:         plot.ID,
			ApplicantID:    applicantID,
			Reason:         reason,
			Status:         string(constants.TransferPending),
			PrevPlotStatus: plot.Status,
		}
		if err := s.transferRepo.CreateWithTx(tx, req); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		plot.Status = string(constants.PlotStatusTransferPending)
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		req.Plot = plot
		created = req
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferRequested, "request_id", created.ID, "plot_id", created.PlotID, "code", created.Plot.Code, "applicant", applicantID, "prev_status", created.PrevPlotStatus)
	return s.findFullByID(created.ID)
}

// findFullByID 事务提交后重新查询完整关联（申请人/处理人/地块）的申请记录。
func (s *PlotTransferService) findFullByID(id uint) (*model.PlotTransferRequest, error) {
	req, err := s.transferRepo.FindByID(id)
	if err != nil {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return req, nil
}

// Approve 管理员核准：清空原认养关系，地块回到共享池（pending -> approved，plot -> available）。
func (s *PlotTransferService) Approve(requestID, reviewerID uint, reviewerName, comment string) (*model.PlotTransferRequest, error) {
	req, plot, err := s.review(requestID, reviewerID, comment, string(constants.TransferApproved))
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferApproved, "request_id", req.ID, "plot_id", plot.ID, "code", plot.Code, "reviewer", reviewerName)
	return s.findFullByID(req.ID)
}

// Reject 管理员驳回：写明处理意见，地块恢复申请前状态（pending -> rejected）。
func (s *PlotTransferService) Reject(requestID, reviewerID uint, reviewerName, comment string) (*model.PlotTransferRequest, error) {
	req, plot, err := s.review(requestID, reviewerID, comment, string(constants.TransferRejected))
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferRejected, "request_id", req.ID, "plot_id", plot.ID, "code", plot.Code, "reviewer", reviewerName)
	return s.findFullByID(req.ID)
}

// review 核准/驳回共用的事务骨架：行锁申请 + 行锁地块，只接受先完成的一次状态流转。
func (s *PlotTransferService) review(requestID, reviewerID uint, comment, target string) (*model.PlotTransferRequest, *model.Plot, error) {
	var updated *model.PlotTransferRequest
	var plot *model.Plot
	err := s.db.Transaction(func(tx *gorm.DB) error {
		req, err := s.transferRepo.FindByIDForUpdate(tx, requestID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", requestID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if req.Status != string(constants.TransferPending) {
			return util.NewAppError(constants.CodeTransferNotPending, 409, fmt.Sprintf("转交申请 id=%d 已被处理（当前状态 %s），请勿重复操作", req.ID, util.TransferStatusText(req.Status)))
		}
		plot, err = s.plotRepo.FindByIDForUpdate(tx, req.PlotID)
		if err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		now := time.Now()
		req.Status = target
		req.ReviewComment = comment
		req.ReviewerID = &reviewerID
		req.ReviewedAt = &now
		if target == string(constants.TransferApproved) {
			plot.Status = string(constants.PlotStatusAvailable)
			plot.AdopterID = nil
		} else {
			plot.Status = req.PrevPlotStatus
		}
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.transferRepo.UpdateWithTx(tx, req); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		req.Plot = plot
		updated = req
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return updated, plot, nil
}

// Cancel 申请人撤回待处理申请（pending -> cancelled，地块恢复申请前状态）。
func (s *PlotTransferService) Cancel(requestID, operatorID uint) (*model.PlotTransferRequest, error) {
	var updated *model.PlotTransferRequest
	var plot *model.Plot
	err := s.db.Transaction(func(tx *gorm.DB) error {
		req, err := s.transferRepo.FindByIDForUpdate(tx, requestID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", requestID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if req.ApplicantID != operatorID {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("用户 id=%d 不是转交申请 id=%d 的申请人，无权撤回", operatorID, req.ID))
		}
		if req.Status != string(constants.TransferPending) {
			return util.NewAppError(constants.CodeTransferNotPending, 409, fmt.Sprintf("转交申请 id=%d 已被处理（当前状态 %s），无法撤回", req.ID, util.TransferStatusText(req.Status)))
		}
		plot, err = s.plotRepo.FindByIDForUpdate(tx, req.PlotID)
		if err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		now := time.Now()
		req.Status = string(constants.TransferCancelled)
		req.ReviewedAt = &now
		plot.Status = req.PrevPlotStatus
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if err := s.transferRepo.UpdateWithTx(tx, req); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		req.Plot = plot
		updated = req
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferCancelled, "request_id", updated.ID, "plot_id", plot.ID, "code", plot.Code, "applicant", operatorID)
	return s.findFullByID(updated.ID)
}

// List 分页查询申请（管理员看全部，普通用户强制只看自己）。
func (s *PlotTransferService) List(pq util.PageQuery, status string, operatorID uint, operatorRole string) ([]model.PlotTransferRequest, int64, error) {
	var applicantID *uint
	if operatorRole != string(constants.RoleAdmin) {
		applicantID = &operatorID
	}
	reqs, total, err := s.transferRepo.List(pq, status, applicantID)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return reqs, total, nil
}

// GetByID 查询申请详情（仅申请人本人或管理员可见）。
func (s *PlotTransferService) GetByID(id, operatorID uint, operatorRole string) (*model.PlotTransferRequest, error) {
	req, err := s.transferRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	if operatorRole != string(constants.RoleAdmin) && req.ApplicantID != operatorID {
		return nil, util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权查看转交申请 id=%d", util.RoleText(operatorRole), id))
	}
	return req, nil
}
