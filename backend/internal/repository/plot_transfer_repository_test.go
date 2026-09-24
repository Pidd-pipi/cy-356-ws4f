package repository

import (
	"testing"

	"github.com/communitygarden/server/internal/model"
	"github.com/communitygarden/server/internal/util"
)

func TestPlotTransferRepository_ListAndPending(t *testing.T) {
	db := newTestDB(t)
	repo := NewPlotTransferRepository(db)
	plotRepo := NewPlotRepository(db)
	owner := seedUser(t, db, "farmer", "farmer")
	other := seedUser(t, db, "citizen", "citizen")

	mkPlot := func(code string) *model.Plot {
		p := &model.Plot{Name: code, Code: code, Area: 10, SoilType: "loam", Sunlight: "full", Latitude: 31.0, Longitude: 121.0, Status: "transfer_pending", AdopterID: &owner.ID}
		if err := plotRepo.Create(p); err != nil {
			t.Fatalf("create plot: %v", err)
		}
		return p
	}
	p1, p2 := mkPlot("P-TR-R1"), mkPlot("P-TR-R2")

	seed := func(plotID, applicantID uint, status string) *model.PlotTransferRequest {
		r := &model.PlotTransferRequest{PlotID: plotID, ApplicantID: applicantID, Reason: "测试理由", Status: status, PrevPlotStatus: "adopted"}
		if err := repo.CreateWithTx(db, r); err != nil {
			t.Fatalf("create transfer request: %v", err)
		}
		return r
	}
	r1 := seed(p1.ID, owner.ID, "pending")
	seed(p2.ID, other.ID, "approved")

	// ExistsPendingByPlot：p1 有待处理申请，p2 没有
	exists, err := repo.ExistsPendingByPlot(db, p1.ID)
	if err != nil || !exists {
		t.Errorf("ExistsPendingByPlot(p1) = %v, %v; want true", exists, err)
	}
	exists, err = repo.ExistsPendingByPlot(db, p2.ID)
	if err != nil || exists {
		t.Errorf("ExistsPendingByPlot(p2) = %v, %v; want false", exists, err)
	}

	// 状态过滤
	_, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, "pending", nil)
	if err != nil || total != 1 {
		t.Errorf("list pending: total=%d err=%v, want 1", total, err)
	}
	// 申请人过滤
	_, total, err = repo.List(util.PageQuery{Page: 1, PageSize: 10}, "", &other.ID)
	if err != nil || total != 1 {
		t.Errorf("list by applicant: total=%d err=%v, want 1", total, err)
	}
	// 全部
	all, total, err := repo.List(util.PageQuery{Page: 1, PageSize: 10}, "", nil)
	if err != nil || total != 2 || len(all) != 2 {
		t.Errorf("list all: total=%d len=%d err=%v, want 2", total, len(all), err)
	}
	if len(all) > 0 && all[0].Plot == nil {
		t.Errorf("expected Plot preloaded in list")
	}

	// FindByID 预加载关联
	got, err := repo.FindByID(r1.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Applicant == nil || got.Applicant.Username != "farmer" || got.Plot == nil || got.Plot.Code != "P-TR-R1" {
		t.Errorf("FindByID preloads invalid: applicant=%+v plot=%+v", got.Applicant, got.Plot)
	}
}
