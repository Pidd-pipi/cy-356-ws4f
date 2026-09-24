package service

import (
	"testing"

	"github.com/communitygarden/server/internal/constants"
)

// TestPlotTransferService_FullLifecycle 覆盖：提交 -> 核准 -> 地块回到共享池。
func TestPlotTransferService_FullLifecycle(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner1", "citizen")
	admin := newTestUser(t, db, "admin1", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-T-APPROVE", "adopted", &uid)

	// 非认养人不能提交
	if _, err := transferSvc.Submit(plot.ID, admin.ID, "admin", "admin1", "代提交"); err == nil {
		t.Fatalf("expected forbidden for non-adopter submit")
	}

	// 认养人提交
	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner1", "农忙，无暇打理")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if tr.Status != string(constants.TransferPending) {
		t.Fatalf("transfer status=%s, want pending", tr.Status)
	}
	// 地块进入待核准状态且认养关系保留
	locked, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if locked.Status != string(constants.PlotStatusPendingTransfer) || locked.AdopterID == nil || *locked.AdopterID != uid {
		t.Fatalf("plot status=%s adopter=%v, want pending_transfer with adopter kept", locked.Status, locked.AdopterID)
	}
	if locked.LatestTransfer == nil || locked.LatestTransfer.Reason != "农忙，无暇打理" {
		t.Fatalf("latest transfer reason missing on plot detail")
	}

	// 重复提交应被拒绝
	if _, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner1", "再提一次"); err == nil {
		t.Fatalf("expected conflict for duplicate pending submit")
	}

	// 非申请人不能撤回
	if _, err := transferSvc.Withdraw(tr.ID, admin.ID, "admin"); err == nil {
		t.Fatalf("expected forbidden for non-applicant withdraw")
	}

	// 管理员核准
	approved, err := transferSvc.Approve(tr.ID, admin.ID, "admin1", "admin", "同意转交")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != string(constants.TransferApproved) || approved.ReviewComment != "同意转交" {
		t.Fatalf("approved transfer invalid: %+v", approved)
	}
	// 核准后原认养关系清空，地块回到共享池
	done, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID after approve: %v", err)
	}
	if done.Status != string(constants.PlotStatusAvailable) || done.AdopterID != nil {
		t.Fatalf("plot after approve status=%s adopter=%v", done.Status, done.AdopterID)
	}

	// 核准后再撤回应提示已经处理
	if _, err := transferSvc.Withdraw(tr.ID, owner.ID, "citizen"); err == nil {
		t.Fatalf("expected already-done error for late withdraw")
	}
	// 核准后再次核准同样被拒绝
	if _, err := transferSvc.Approve(tr.ID, admin.ID, "admin1", "admin", ""); err == nil {
		t.Fatalf("expected already-done error for duplicate approve")
	}
}

// TestPlotTransferService_Reject 覆盖：提交 -> 驳回（必须带意见）-> 地块恢复已认养。
func TestPlotTransferService_Reject(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner2", "citizen")
	admin := newTestUser(t, db, "admin2", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-T-REJECT", "adopted", &uid)

	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner2", "要搬家")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	rejected, err := transferSvc.Reject(tr.ID, admin.ID, "admin2", "admin", "当前轮作未结束，请收获后再申请")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != string(constants.TransferRejected) || rejected.ReviewerID == nil {
		t.Fatalf("rejected transfer invalid: %+v", rejected)
	}
	restored, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if restored.Status != string(constants.PlotStatusAdopted) || restored.AdopterID == nil || *restored.AdopterID != uid {
		t.Fatalf("plot after reject status=%s adopter=%v, want adopted with same adopter", restored.Status, restored.AdopterID)
	}

	// 驳回后认养人可以重新提交
	if _, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner2", "已收获，再次申请"); err != nil {
		t.Fatalf("resubmit after reject should be allowed: %v", err)
	}
}

// TestPlotTransferService_Withdraw 覆盖：提交 -> 本人撤回 -> 地块恢复，可再次申请。
func TestPlotTransferService_Withdraw(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner3", "citizen")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-T-WITHDRAW", "harvested", &uid)

	tr, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner3", "误点，先撤回")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	withdrawn, err := transferSvc.Withdraw(tr.ID, owner.ID, "citizen")
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if withdrawn.Status != string(constants.TransferWithdrawn) {
		t.Fatalf("status=%s, want withdrawn", withdrawn.Status)
	}
	restored, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// previous_status 为 harvested，撤回后恢复
	if restored.Status != string(constants.PlotStatusHarvested) || restored.AdopterID == nil {
		t.Fatalf("plot after withdraw status=%s", restored.Status)
	}
	// 撤回后管理员再核准应提示已经处理
	admin := newTestUser(t, db, "admin3", "admin")
	if _, err := transferSvc.Approve(tr.ID, admin.ID, "admin3", "admin", "迟到的核准"); err == nil {
		t.Fatalf("expected already-done error for approve after withdraw")
	}
}
