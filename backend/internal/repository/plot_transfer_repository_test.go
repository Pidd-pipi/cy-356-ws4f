package repository

import (
	"testing"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

func TestPlotTransferRepository_LatestAndList(t *testing.T) {
	db := newTestDB(t)
	plotRepo := NewPlotRepository(db)
	repo := NewPlotTransferRepository(db)
	user := seedUser(t, db, "owner", "citizen")
	p := &model.Plot{Name: "P-T1", Code: "P-T1", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31.0, Longitude: 121.0, Status: "adopted", AdopterID: &user.ID}
	if err := plotRepo.Create(p); err != nil {
		t.Fatalf("create plot: %v", err)
	}

	mk := func(reason, status string) *model.PlotTransfer {
		tr := &model.PlotTransfer{PlotID: p.ID, ApplicantID: user.ID, Reason: reason, Status: status, PreviousStatus: "adopted"}
		if err := db.Create(tr).Error; err != nil {
			t.Fatalf("create transfer: %v", err)
		}
		return tr
	}
	first := mk("第一次申请", string(constants.TransferRejected))
	second := mk("第二次申请", string(constants.TransferPending))

	// FindLatestByPlot 返回最新一条
	got, err := repo.FindLatestByPlot(p.ID)
	if err != nil || got.ID != second.ID {
		t.Fatalf("FindLatestByPlot id=%v err=%v, want %d", got, err, second.ID)
	}

	// 批量挂载
	m, err := repo.FindLatestByPlotIDs([]uint{p.ID})
	if err != nil || m[p.ID].ID != second.ID {
		t.Fatalf("FindLatestByPlotIDs = %v err=%v", m, err)
	}

	// 列表按状态过滤
	pending, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, string(constants.TransferPending))
	if err != nil || total != 1 || len(pending) != 1 || pending[0].ID != second.ID {
		t.Fatalf("List pending total=%d len=%d err=%v", total, len(pending), err)
	}
	all, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, "")
	if err != nil || total != 2 || len(all) != 2 || all[0].ID != second.ID {
		t.Fatalf("List all total=%d len=%d err=%v", total, len(all), err)
	}
	_ = first
}

func TestPlotTransferRepository_PartialUniqueIndex(t *testing.T) {
	db := newTestDB(t)
	user := seedUser(t, db, "owner2", "citizen")
	p := &model.Plot{Name: "P-T2", Code: "P-T2", Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31.0, Longitude: 121.0, Status: "adopted", AdopterID: &user.ID}
	if err := db.Create(p).Error; err != nil {
		t.Fatalf("create plot: %v", err)
	}
	tr1 := &model.PlotTransfer{PlotID: p.ID, ApplicantID: user.ID, Reason: "第一条待处理", Status: string(constants.TransferPending), PreviousStatus: "adopted"}
	if err := db.Create(tr1).Error; err != nil {
		t.Fatalf("create tr1: %v", err)
	}
	tr2 := &model.PlotTransfer{PlotID: p.ID, ApplicantID: user.ID, Reason: "第二条待处理", Status: string(constants.TransferPending), PreviousStatus: "adopted"}
	if err := db.Create(tr2).Error; err == nil {
		t.Fatalf("expected unique index violation for two pending transfers of same plot")
	}
	// 第一条改为已驳回后，允许再次存在 pending
	if err := db.Model(tr1).Update("status", string(constants.TransferRejected)).Error; err != nil {
		t.Fatalf("update tr1: %v", err)
	}
	if err := db.Create(tr2).Error; err != nil {
		t.Fatalf("create tr2 after first resolved should succeed: %v", err)
	}
}
