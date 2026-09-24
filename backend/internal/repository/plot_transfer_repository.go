package repository

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

// PlotTransferRepository 地块转交申请仓储接口。
type PlotTransferRepository interface {
	CreateWithTx(tx *gorm.DB, r *model.PlotTransferRequest) error
	FindByID(id uint) (*model.PlotTransferRequest, error)
	FindByIDForUpdate(tx *gorm.DB, id uint) (*model.PlotTransferRequest, error)
	UpdateWithTx(tx *gorm.DB, r *model.PlotTransferRequest) error
	ExistsPendingByPlot(tx *gorm.DB, plotID uint) (bool, error)
	List(pq util.PageQuery, status string, applicantID *uint) ([]model.PlotTransferRequest, int64, error)
}

type plotTransferRepository struct {
	db *gorm.DB
}

// NewPlotTransferRepository 构造地块转交申请仓储。
func NewPlotTransferRepository(db *gorm.DB) PlotTransferRepository {
	return &plotTransferRepository{db: db}
}

func (r *plotTransferRepository) CreateWithTx(tx *gorm.DB, req *model.PlotTransferRequest) error {
	return tx.Create(req).Error
}

func (r *plotTransferRepository) FindByID(id uint) (*model.PlotTransferRequest, error) {
	var req model.PlotTransferRequest
	if err := r.db.Preload("Plot").Preload("Plot.Adopter").Preload("Applicant").Preload("Reviewer").First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

// FindByIDForUpdate 核准/驳回/撤回并发竞争使用 SELECT ... FOR UPDATE 行锁（事务内执行）。
func (r *plotTransferRepository) FindByIDForUpdate(tx *gorm.DB, id uint) (*model.PlotTransferRequest, error) {
	var req model.PlotTransferRequest
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *plotTransferRepository) UpdateWithTx(tx *gorm.DB, req *model.PlotTransferRequest) error {
	return tx.Save(req).Error
}

// ExistsPendingByPlot 校验同一地块是否已有待处理申请（事务内执行，配合行锁防并发重复提交）。
func (r *plotTransferRepository) ExistsPendingByPlot(tx *gorm.DB, plotID uint) (bool, error) {
	var count int64
	if err := tx.Model(&model.PlotTransferRequest{}).
		Where("plot_id = ? AND status = ?", plotID, "pending").
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// List 分页查询申请（管理员看全部，认养人只看自己；可按状态过滤）。
func (r *plotTransferRepository) List(pq util.PageQuery, status string, applicantID *uint) ([]model.PlotTransferRequest, int64, error) {
	var reqs []model.PlotTransferRequest
	var total int64
	q := r.db.Model(&model.PlotTransferRequest{}).Preload("Plot").Preload("Applicant").Preload("Reviewer")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if applicantID != nil {
		q = q.Where("applicant_id = ?", *applicantID)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := util.Paginate(q.Order("id DESC"), pq).Find(&reqs).Error; err != nil {
		return nil, 0, err
	}
	return reqs, total, nil
}
