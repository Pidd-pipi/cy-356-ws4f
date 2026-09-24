package service

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/util"
)

// PlotService 地块服务（认养使用事务 + SELECT FOR UPDATE）。
type PlotService struct {
	plotRepo     repository.PlotRepository
	transferRepo repository.PlotTransferRepository
	db           *gorm.DB
	logger       *slog.Logger
}

// NewPlotService 构造地块服务。
func NewPlotService(plotRepo repository.PlotRepository, transferRepo repository.PlotTransferRepository, db *gorm.DB, logger *slog.Logger) *PlotService {
	return &PlotService{plotRepo: plotRepo, transferRepo: transferRepo, db: db, logger: logger}
}

// GetByID 查询地块详情（被地块 handler 与种植计划 service 复用），附带最近一条转交申请。
func (s *PlotService) GetByID(id uint) (*model.Plot, error) {
	p, err := s.plotRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	if latest, lerr := s.transferRepo.FindLatestByPlot(id); lerr == nil {
		p.LatestTransfer = latest
	} else if !errors.Is(lerr, repository.ErrNotFound) {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(lerr)
	}
	return p, nil
}

// Create 创建地块（管理员）。
func (s *PlotService) Create(req *dto.CreatePlotRequest, operator string) (*model.Plot, error) {
	if _, err := s.plotRepo.FindByCode(req.Code); err == nil {
		return nil, util.NewAppError(constants.CodeConflict, 409, fmt.Sprintf("地块编号 %s 已存在", req.Code))
	}
	p := &model.Plot{
		Name:        req.Name,
		Code:        req.Code,
		Area:        req.Area,
		SoilType:    req.SoilType,
		Sunlight:    req.Sunlight,
		Latitude:    req.Latitude,
		Longitude:   req.Longitude,
		Status:      string(constants.PlotStatusAvailable),
		Description: req.Description,
	}
	if err := s.plotRepo.Create(p); err != nil {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	s.logger.Info(constants.LogPlotCreated, "plot_id", p.ID, "code", p.Code, "operator", operator)
	return p, nil
}

// Update 更新地块（管理员）。
func (s *PlotService) Update(id uint, req *dto.UpdatePlotRequest, operator string) (*model.Plot, error) {
	p, err := s.plotRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", id))
		}
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Code != nil {
		p.Code = *req.Code
	}
	if req.Area != nil {
		p.Area = *req.Area
	}
	if req.SoilType != nil {
		p.SoilType = *req.SoilType
	}
	if req.Sunlight != nil {
		p.Sunlight = *req.Sunlight
	}
	if req.Latitude != nil {
		p.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		p.Longitude = *req.Longitude
	}
	if req.Description != nil {
		p.Description = *req.Description
	}
	if err := s.plotRepo.Update(p); err != nil {
		return nil, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	return p, nil
}

// List 分页查询地块（可过滤状态），并批量挂载每个地块最近一条转交申请。
func (s *PlotService) List(pq util.PageQuery, status string) ([]model.Plot, int64, error) {
	plots, total, err := s.plotRepo.List(pq, status)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	ids := make([]uint, 0, len(plots))
	for i := range plots {
		ids = append(ids, plots[i].ID)
	}
	latest, err := s.transferRepo.FindLatestByPlotIDs(ids)
	if err != nil {
		return nil, 0, util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
	}
	for i := range plots {
		if t, ok := latest[plots[i].ID]; ok {
			plots[i].LatestTransfer = t
		}
	}
	return plots, total, nil
}

// Adopt 认养地块（事务 + 行锁，available -> adopted）。
func (s *PlotService) Adopt(plotID, userID uint, role, username string) (*model.Plot, error) {
	var adopted *model.Plot
	err := s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, plotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		// 转交申请待核准期间，其他居民不能认养该地块。
		if plot.Status == string(constants.PlotStatusPendingTransfer) {
			return util.NewAppError(constants.CodeTransferPending, 409, fmt.Sprintf("地块 %s 转交申请待核准，暂不可认养", plot.Code))
		}
		if plot.Status != string(constants.PlotStatusAvailable) {
			return util.NewAppError(constants.CodePlotNotAvailable, 409, fmt.Sprintf("地块 %s 当前状态为 %s，不可认养", plot.Code, util.PlotStatusText(plot.Status)))
		}
		plot.Status = string(constants.PlotStatusAdopted)
		plot.AdopterID = &userID
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		adopted = plot
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogPlotAdopted, "plot_id", adopted.ID, "code", adopted.Code, "user_id", userID, "role", role)
	return adopted, nil
}

// Release 强制释放地块（仅管理员；认养人须走转交申请流程，见 PlotTransferService.Submit）。
func (s *PlotService) Release(plotID, operatorID uint, operatorRole string) (*model.Plot, error) {
	var released *model.Plot
	err := s.db.Transaction(func(tx *gorm.DB) error {
		plot, err := s.plotRepo.FindByIDForUpdate(tx, plotID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.CodeNotFound, 404, fmt.Sprintf("地块实体 id=%d 不存在", plotID))
			}
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		if operatorRole != string(constants.RoleAdmin) {
			return util.NewAppError(constants.CodeForbidden, 403, fmt.Sprintf("角色 %s 无权直接释放地块 %s，请提交转交申请由管理员核准", util.RoleText(operatorRole), plot.Code))
		}
		if plot.Status != string(constants.PlotStatusHarvested) {
			return util.NewAppError(constants.CodePlotNotAvailable, 409, fmt.Sprintf("地块 %s 当前状态为 %s，仅待释放状态可释放", plot.Code, util.PlotStatusText(plot.Status)))
		}
		plot.Status = string(constants.PlotStatusAvailable)
		plot.AdopterID = nil
		if err := s.plotRepo.UpdateWithTx(tx, plot); err != nil {
			return util.NewAppError(constants.CodeInternalError, 500, constants.ErrorText[constants.CodeInternalError]).Wrap(err)
		}
		released = plot
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logger.Info(constants.LogPlotReleased, "plot_id", released.ID, "code", released.Code, "operator", operatorID)
	return released, nil
}

// MarkHarvested 种植计划完成后将地块置为待释放（harvested）。
// 若地块正处于转交申请待核准状态，则保持 pending_transfer，避免打断审批流程。
func (s *PlotService) MarkHarvested(tx *gorm.DB, plotID uint) error {
	plot, err := s.plotRepo.FindByIDForUpdate(tx, plotID)
	if err != nil {
		return err
	}
	if plot.Status == string(constants.PlotStatusPendingTransfer) {
		return nil
	}
	plot.Status = string(constants.PlotStatusHarvested)
	return s.plotRepo.UpdateWithTx(tx, plot)
}

// CountByStatus 地块状态统计（仪表盘复用）。
func (s *PlotService) CountByStatus() (map[string]int64, error) {
	return s.plotRepo.CountByStatus()
}
