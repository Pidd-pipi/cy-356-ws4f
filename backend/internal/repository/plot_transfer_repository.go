package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

// PlotTransferRepository 地块转交申请仓储接口。
type PlotTransferRepository interface {
	CreateWithTx(tx *gorm.DB, t *model.PlotTransfer) error
	UpdateWithTx(tx *gorm.DB, t *model.PlotTransfer) error
	FindByID(id uint) (*model.PlotTransfer, error)
	FindByIDTx(tx *gorm.DB, id uint) (*model.PlotTransfer, error)
	FindByIDForUpdate(tx *gorm.DB, id uint) (*model.PlotTransfer, error)
	FindPendingByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.PlotTransfer, error)
	FindLatestByPlot(plotID uint) (*model.PlotTransfer, error)
	FindLatestByPlotIDs(plotIDs []uint) (map[uint]*model.PlotTransfer, error)
	List(pq util.PageQuery, status string) ([]model.PlotTransfer, int64, error)
}

type plotTransferRepository struct {
	db *gorm.DB
}

// NewPlotTransferRepository 构造转交申请仓储。
func NewPlotTransferRepository(db *gorm.DB) PlotTransferRepository {
	return &plotTransferRepository{db: db}
}

func (r *plotTransferRepository) CreateWithTx(tx *gorm.DB, t *model.PlotTransfer) error {
	return tx.Create(t).Error
}

func (r *plotTransferRepository) UpdateWithTx(tx *gorm.DB, t *model.PlotTransfer) error {
	return tx.Save(t).Error
}

func (r *plotTransferRepository) FindByID(id uint) (*model.PlotTransfer, error) {
	var t model.PlotTransfer
	if err := r.db.Preload("Applicant").Preload("Reviewer").Preload("Plot").First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindByIDForUpdate 核准/撤回事务内使用 SELECT ... FOR UPDATE 锁定申请行。
func (r *plotTransferRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.PlotTransfer, error) {
	var t model.PlotTransfer
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindByIDTx 事务内无锁查询（用于先取得 plot_id 以统一加锁顺序，随后再用 FOR UPDATE 重读）。
func (r *plotTransferRepository) FindByIDTx(tx *gorm.DB, id uint) (*model.PlotTransfer, error) {
	var t model.PlotTransfer
	if err := tx.First(&t, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindPendingByPlotForUpdate 提交申请前在事务内确认该地块没有待处理申请（同时锁定地块行）。
func (r *plotTransferRepository) FindPendingByPlotForUpdate(tx *gorm.DB, plotID uint) (*model.PlotTransfer, error) {
	var t model.PlotTransfer
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("plot_id = ? AND status = ?", plotID, string(constants.TransferPending)).
		Order("id DESC").First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindLatestByPlot 查询某地块最近一条申请（任意状态），供地块详情展示。
func (r *plotTransferRepository) FindLatestByPlot(plotID uint) (*model.PlotTransfer, error) {
	var t model.PlotTransfer
	err := r.db.Preload("Applicant").Preload("Reviewer").
		Where("plot_id = ?", plotID).Order("id DESC").First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// FindLatestByPlotIDs 批量查询多个地块各自最近一条申请（地块列表一次性挂载，避免 N+1）。
func (r *plotTransferRepository) FindLatestByPlotIDs(plotIDs []uint) (map[uint]*model.PlotTransfer, error) {
	out := make(map[uint]*model.PlotTransfer)
	if len(plotIDs) == 0 {
		return out, nil
	}
	var transfers []model.PlotTransfer
	// 相关子查询取每个 plot_id 下最大 id（PostgreSQL / SQLite 均可执行）。
	subQuery := r.db.Model(&model.PlotTransfer{}).
		Select("MAX(id)").Where("plot_id IN ?", plotIDs).Group("plot_id")
	if err := r.db.Preload("Applicant").Preload("Reviewer").
		Where("id IN (?)", subQuery).Find(&transfers).Error; err != nil {
		return nil, err
	}
	for i := range transfers {
		out[transfers[i].PlotID] = &transfers[i]
	}
	return out, nil
}

// List 管理员分页查询申请（可按状态过滤，默认按申请时间倒序）。
func (r *plotTransferRepository) List(pq util.PageQuery, status string) ([]model.PlotTransfer, int64, error) {
	var transfers []model.PlotTransfer
	var total int64
	q := r.db.Model(&model.PlotTransfer{}).Preload("Applicant").Preload("Reviewer").Preload("Plot")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := util.Paginate(q.Order("id DESC"), pq).Find(&transfers).Error; err != nil {
		return nil, 0, err
	}
	return transfers, total, nil
}
