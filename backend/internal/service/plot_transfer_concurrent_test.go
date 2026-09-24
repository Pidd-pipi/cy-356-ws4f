package service

import (
	"errors"
	"testing"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/util"
)

// TestPlotTransferService_RaceApproveFirst 核准与撤回同时到来、核准先完成：后到的撤回应提示已经处理。
func TestPlotTransferService_RaceApproveFirst(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner-ra", "citizen")
	admin := newTestUser(t, db, "admin-ra", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-RACE-A", "adopted", &uid)

	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner-ra", "农忙")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := transferSvc.Approve(tr.ID, admin.ID, "admin-ra", "admin", "同意"); err != nil {
		t.Fatalf("Approve: %v", err)
	}
	// 撤回后到
	_, err = transferSvc.Withdraw(tr.ID, owner.ID, "citizen")
	if !isAppErrCode(err, constants.CodeTransferAlreadyDone) {
		t.Fatalf("late withdraw should be already-done, got %v", err)
	}
	// 地块已在共享池
	got, _ := plotSvc.GetByID(plot.ID)
	if got.Status != string(constants.PlotStatusAvailable) || got.AdopterID != nil {
		t.Fatalf("plot should be available without adopter, status=%s adopter=%v", got.Status, got.AdopterID)
	}
}

// TestPlotTransferService_RaceWithdrawFirst 核准与撤回同时到来、撤回先完成：后到的核准应提示已经处理。
func TestPlotTransferService_RaceWithdrawFirst(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner-rw", "citizen")
	admin := newTestUser(t, db, "admin-rw", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-RACE-W", "adopted", &uid)

	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner-rw", "误点")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := transferSvc.Withdraw(tr.ID, owner.ID, "citizen"); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	// 核准后到
	_, err = transferSvc.Approve(tr.ID, admin.ID, "admin-rw", "admin", "迟到的核准")
	if !isAppErrCode(err, constants.CodeTransferAlreadyDone) {
		t.Fatalf("late approve should be already-done, got %v", err)
	}
	got, _ := plotSvc.GetByID(plot.ID)
	if got.Status != string(constants.PlotStatusAdopted) || got.AdopterID == nil || *got.AdopterID != uid {
		t.Fatalf("plot should remain adopted by owner, status=%s", got.Status)
	}
}

// TestPlotTransferService_DoubleReviewGuarded 两个核准请求顺序到来时只成功一次。
func TestPlotTransferService_DoubleReviewGuarded(t *testing.T) {
	db := newTestServiceDB(t)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner-dr", "citizen")
	admin := newTestUser(t, db, "admin-dr", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-DOUBLE", "adopted", &uid)
	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner-dr", "x")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := transferSvc.Approve(tr.ID, admin.ID, "admin-dr", "admin", "first"); err != nil {
		t.Fatalf("first approve: %v", err)
	}
	_, err = transferSvc.Reject(tr.ID, admin.ID, "admin-dr", "admin", "second")
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != constants.CodeTransferAlreadyDone {
		t.Fatalf("second review should be already-done, got %v", err)
	}
}

func isAppErrCode(err error, code int) bool {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}
