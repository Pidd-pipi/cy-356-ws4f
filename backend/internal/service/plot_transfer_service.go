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

// PlotTransferService 地块转交申请服务：认养人提交申请，管理员核准后才清空原认养关系。
// 所有多步写操作均在事务内执行，并对地块行与申请行使用 SELECT ... FOR UPDATE。
type PlotTransferService struct {
	transferRepo repository.PlotTransferRepository
	plotRepo     repository.PlotRepository
	db           *gorm.DB
	logger       *slog.Logger
}

// NewPlotTransferService 构造转交申请服务。
func NewPlotTransferService(transferRepo repository.PlotTransferRepository, plotRepo repository.PlotRepository, db *gorm.DB, logger *slog.Logger) *PlotTransferService {
	return &PlotTransferService{transferRepo: transferRepo, plotRepo: plotRepo, db: db, logger: logger}
}

// Submit 认养人在地块列表提交转交申请（adopted/harvested -> pending_transfer）。
func (s *PlotTransferService) Submit(plotID, applicantID uint, role, username, reason string) (*model.PlotTransfer, error) {
	var created *model.PlotTransfer
	err := s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, plotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 仅当前认养人本人可发起转交申请。
		if plot.AdopterID == nil || *plot.AdopterID != applicantID {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 不是地块 %s 的认养人，无权提交转交申请", util.RoleText(role), plot.Code))
		}
		// 存在待处理申请时禁止重复提交（部分唯一索引兜底，这里给出明确业务报错）。
		if pending, perr := s.transferRepo.FindPendingByPlotForUpdate(tx, plotID); perr == nil && pending != nil {
			return util.NewAppError(constants.CodeTransferPending, 409, fmt.Sprintf("地块 %s 已有待处理转交申请（申请单 id=%d），请勿重复提交", plot.Code, pending.ID))
		} else if perr != nil && !errors.Is(perr, repository.ErrNotFound) {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(perr)
		}
		if plot.Status != string(constants.PlotStatusAdopted) && plot.Status != string(constants.PlotStatusHarvested) {
			return util.NewAppError(constants.CodeTransferNotAllowed, 409, fmt.Sprintf("地块 %s 当前状态为 %s，不可提交转交申请", plot.Code, util.PlotStatusText(plot.Status)))
		}

		transfer := &model.PlotTransfer{
			PlotID:         plotID,
			ApplicantID:    applicantID,
			Reason:         reason,
			Status:         string(constants.TransferPending),
			PreviousStatus: plot.Status,
		}
		if err := s.transferRepo.CreateWithTx(tx, transfer); err != nil {
			return util.NewAppError(constants.CodeConflict, 409, fmt.Sprintf("地块 %s 的转交申请已存在或数据冲突", plot.Code)).Wrap(err)
		}
		// 申请期间锁定地块：其他居民不能认养，原认养关系保留。
		plot.Status = string(constants.PlotStatusPendingTransfer)
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		created = transfer
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferSubmitted, "transfer_id", created.ID, "plot_id", plotID, "applicant", applicantID, "role", role, "username", username)
	return s.transferRepo.FindByID(created.ID)
}

// Approve 管理员核准转交申请（pending -> approved，清空原认养关系，地块回到共享池）。
// 与认养人撤回并发时只接受先完成的一次：申请行已非 pending 则返回 CodeTransferAlreadyDone。
func (s *PlotTransferService) Approve(transferID, reviewerID uint, reviewerName, reviewerRole, comment string) (*model.PlotTransfer, error) {
	return s.review(transferID, reviewerID, reviewerName, reviewerRole, true, comment)
}

// Reject 管理员写明意见驳回转交申请（pending -> rejected，地块恢复申请前状态）。
func (s *PlotTransferService) Reject(transferID, reviewerID uint, reviewerName, reviewerRole, comment string) (*model.PlotTransfer, error) {
	return s.review(transferID, reviewerID, reviewerName, reviewerRole, false, comment)
}

// review 核准/驳回共用的事务实现（复用同一状态机入口）。
func (s *PlotTransferService) review(transferID, reviewerID uint, reviewerName, reviewerRole string, approve bool, comment string) (*model.PlotTransfer, error) {
	target := constants.TransferRejected
	if approve {
		target = constants.TransferApproved
	}
	var result *model.PlotTransfer
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 统一加锁顺序 plot -> transfer，避免与 Submit 的 plot->transfer 顺序形成等待环。
		// 先无锁读取申请拿到 plot_id，随后锁地块，最后 FOR UPDATE 重读申请做状态判定。
		pre, err := s.transferRepo.FindByIDTx(tx, transferID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", transferID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		plot, err := s.plotRepo.FindByIDForUpdate(tx, pre.PlotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", pre.PlotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		transfer, err := s.transferRepo.FindByIDForUpdate(tx, transferID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", transferID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 核准与撤回同时到来只接受先完成的一次，后到的提示已经处理。
		if !constants.CanTransferTo(constants.TransferStatus(transfer.Status), target) {
			s.logger.Warn(constants.LogTransferConflict, "transfer_id", transferID, "plot_id", transfer.PlotID, "expected", string(constants.TransferPending), "actual", transfer.Status, "operator", reviewerName)
			return util.NewAppError(constants.CodeTransferAlreadyDone, 409, fmt.Sprintf("转交申请 id=%d 当前状态为 %s，%s", transferID, util.TransferStatusText(transfer.Status), constants.ErrorText[constants.CodeTransferAlreadyDone]))
		}

		now := time.Now()
		transfer.Status = string(target)
		transfer.ReviewerID = &reviewerID
		transfer.ReviewComment = comment
		transfer.ReviewedAt = &now
		if err := s.transferRepo.UpdateWithTx(tx, transfer); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}

		if approve {
			// 管理员同意后才清空原认养关系，地块回到共享池。
			plot.Status = string(constants.PlotStatusAvailable)
			plot.AdopterID = nil
		} else {
			// 驳回：原认养关系保留，地块恢复申请前状态。
			plot.Status = transfer.PreviousStatus
		}
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		result = transfer
		return nil
	})
	if err != nil {
		return nil, err
	}
	logTpl := constants.LogTransferRejected
	if approve {
		logTpl = constants.LogTransferApproved
	}
	s.logger.Info(logTpl, "transfer_id", transferID, "plot_id", result.PlotID, "reviewer", reviewerName, "role", reviewerRole)
	return s.transferRepo.FindByID(result.ID)
}

// Withdraw 认养人本人撤回申请（pending -> withdrawn，地块恢复申请前状态）。
// 与管理员核准并发时，后完成的一次收到 CodeTransferAlreadyDone（"已经处理"）。
func (s *PlotTransferService) Withdraw(transferID, operatorID uint, role string) (*model.PlotTransfer, error) {
	var result *model.PlotTransfer
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 统一加锁顺序 plot -> transfer（与 review / submit 一致，避免并发死锁）。
		pre, err := s.transferRepo.FindByIDTx(tx, transferID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", transferID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if pre.ApplicantID != operatorID {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权撤回转交申请 id=%d，仅申请人本人可撤回", util.RoleText(role), transferID))
		}
		plot, err := s.plotRepo.FindByIDForUpdate(tx, pre.PlotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", pre.PlotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		transfer, err := s.transferRepo.FindByIDForUpdate(tx, transferID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", transferID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if !constants.CanTransferTo(constants.TransferStatus(transfer.Status), constants.TransferWithdrawn) {
			s.logger.Warn(constants.LogTransferConflict, "transfer_id", transferID, "plot_id", transfer.PlotID, "expected", string(constants.TransferPending), "actual", transfer.Status, "operator", operatorID)
			return util.NewAppError(constants.CodeTransferAlreadyDone, 409, fmt.Sprintf("转交申请 id=%d 当前状态为 %s，%s", transferID, util.TransferStatusText(transfer.Status), constants.ErrorText[constants.CodeTransferAlreadyDone]))
		}
		transfer.Status = string(constants.TransferWithdrawn)
		now := time.Now()
		transfer.ReviewedAt = &now
		if err := s.transferRepo.UpdateWithTx(tx, transfer); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 撤回后地块恢复申请前状态，认养关系保留。
		plot.Status = transfer.PreviousStatus
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		result = transfer
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogTransferWithdrawn, "transfer_id", transferID, "plot_id", result.PlotID, "applicant", operatorID, "role", role)
	return s.transferRepo.FindByID(result.ID)
}

// GetByID 查询申请详情（含申请人/审批人/地块）。
func (s *PlotTransferService) GetByID(id uint) (*model.PlotTransfer, error) {
	t, err := s.transferRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("转交申请实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return t, nil
}

// List 管理员分页查询申请（可按状态过滤）。
func (s *PlotTransferService) List(pq util.PageQuery, status string) ([]model.PlotTransfer, int64, error) {
	transfers, total, err := s.transferRepo.List(pq, status)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return transfers, total, nil
}

// AttachLatestToPlot 给单个地块挂载最近一条转交申请（地块详情复用）。
func (s *PlotTransferService) AttachLatestToPlot(p *model.Plot) error {
	t, err := s.transferRepo.FindLatestByPlot(p.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	p.LatestTransfer = t
	return nil
}

// AttachLatestToPlots 批量给地块列表挂载各自最近一条转交申请（列表复用，避免 N+1）。
func (s *PlotTransferService) AttachLatestToPlots(plots []model.Plot) error {
	ids := make([]uint, 0, len(plots))
	for i := range plots {
		ids = append(ids, plots[i].ID)
	}
	latest, err := s.transferRepo.FindLatestByPlotIDs(ids)
	if err != nil {
		return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	for i := range plots {
		if t, ok := latest[plots[i].ID]; ok {
			plots[i].LatestTransfer = t
		}
	}
	return nil
}
