package service

import (
	"testing"

	"gorm.io/gorm"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/repository"
	"github.com/communitygarden/server/internal/util"
)

// newTransferService 构造转交申请服务（测试辅助）。
func newTransferService(t *testing.T, db *gorm.DB) *PlotTransferService {
	t.Helper()
	return NewPlotTransferService(repository.NewPlotTransferRepository(db), repository.NewPlotRepository(db), db, testLogger())
}

func TestPlotTransferService_Submit(t *testing.T) {
	db := newTestServiceDB(t)
	svc := newTransferService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	other := newTestUser(t, db, "citizen", "citizen")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-TR-SUB", "adopted", &uid)

	// 非认养人不能提交
	if _, err := svc.Submit(plot.ID, other.ID, "想换一块地"); err == nil {
		t.Fatalf("expected forbidden error for non-adopter")
	}
	// 认养人提交成功，地块进入转交审核中
	req, err := svc.Submit(plot.ID, owner.ID, "农忙无暇打理，申请转交")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if req.Status != string(constants.TransferPending) || req.PrevPlotStatus != "adopted" {
		t.Errorf("submit result invalid: status=%s prev=%s", req.Status, req.PrevPlotStatus)
	}
	// 待处理期间其他居民不能认养（地块状态非 available）
	plotSvc, _ := newPlotService(t, db)
	if _, err := plotSvc.Adopt(plot.ID, other.ID, "citizen", "citizen"); err == nil {
		t.Fatalf("expected adopt conflict while transfer pending")
	}
	// 重复提交被拒
	if _, err := svc.Submit(plot.ID, owner.ID, "再次提交"); err == nil {
		t.Fatalf("expected duplicate submit error")
	}
	// 空闲地块不能提交转交申请
	free := newTestPlot(t, db, "P-TR-FREE", "available", nil)
	if _, err := svc.Submit(free.ID, owner.ID, "空闲地块"); err == nil {
		t.Fatalf("expected plot state error for available plot")
	}
}

func TestPlotTransferService_Approve(t *testing.T) {
	db := newTestServiceDB(t)
	svc := newTransferService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	admin := newTestUser(t, db, "admin", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-TR-APR", "harvested", &uid)

	req, err := svc.Submit(plot.ID, owner.ID, "收成结束，申请转交")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	got, err := svc.Approve(req.ID, admin.ID, "admin", "同意，感谢照料")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if got.Status != string(constants.TransferApproved) || got.ReviewComment != "同意，感谢照料" || got.ReviewedAt == nil {
		t.Errorf("approve result invalid: %+v", got)
	}
	// 核准后地块回到共享池且认养关系清空
	plotSvc, _ := newPlotService(t, db)
	p, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p.Status != string(constants.PlotStatusAvailable) || p.AdopterID != nil {
		t.Errorf("plot not released: status=%s adopter=%v", p.Status, p.AdopterID)
	}
	// 核准与撤回竞争：后到的撤回提示已处理
	if _, err := svc.Cancel(req.ID, owner.ID); err == nil {
		t.Fatalf("expected already-handled error for late cancel")
	} else if ae, ok := err.(*util.AppError); !ok || ae.Code != constants.CodeTransferNotPending {
		t.Errorf("want CodeTransferNotPending, got %v", err)
	}
	// 重复核准同样被拒
	if _, err := svc.Approve(req.ID, admin.ID, "admin", "重复核准"); err == nil {
		t.Fatalf("expected already-handled error for repeated approve")
	}
}

func TestPlotTransferService_Reject(t *testing.T) {
	db := newTestServiceDB(t)
	svc := newTransferService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	admin := newTestUser(t, db, "admin", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-TR-REJ", "adopted", &uid)

	req, err := svc.Submit(plot.ID, owner.ID, "想退租")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	got, err := svc.Reject(req.ID, admin.ID, "admin", "作物即将成熟，建议本季结束后再申请")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if got.Status != string(constants.TransferRejected) || got.ReviewComment == "" {
		t.Errorf("reject result invalid: %+v", got)
	}
	// 驳回后地块恢复申请前状态，认养关系保留
	plotSvc, _ := newPlotService(t, db)
	p, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p.Status != "adopted" || p.AdopterID == nil || *p.AdopterID != owner.ID {
		t.Errorf("plot not restored: status=%s adopter=%v", p.Status, p.AdopterID)
	}
}

func TestPlotTransferService_Cancel(t *testing.T) {
	db := newTestServiceDB(t)
	svc := newTransferService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	other := newTestUser(t, db, "citizen", "citizen")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-TR-CXL", "adopted", &uid)

	req, err := svc.Submit(plot.ID, owner.ID, "临时有事")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	// 非申请人不能撤回
	if _, err := svc.Cancel(req.ID, other.ID); err == nil {
		t.Fatalf("expected forbidden error for non-applicant cancel")
	}
	got, err := svc.Cancel(req.ID, owner.ID)
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if got.Status != string(constants.TransferCancelled) {
		t.Errorf("cancel result invalid: status=%s", got.Status)
	}
	// 撤回后地块恢复申请前状态
	plotSvc, _ := newPlotService(t, db)
	p, err := plotSvc.GetByID(plot.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p.Status != "adopted" || p.AdopterID == nil || *p.AdopterID != owner.ID {
		t.Errorf("plot not restored: status=%s adopter=%v", p.Status, p.AdopterID)
	}
	// 撤回与核准竞争：后到的核准提示已处理
	admin := newTestUser(t, db, "admin", "admin")
	if _, err := svc.Approve(req.ID, admin.ID, "admin", "同意"); err == nil {
		t.Fatalf("expected already-handled error for late approve")
	} else if ae, ok := err.(*util.AppError); !ok || ae.Code != constants.CodeTransferNotPending {
		t.Errorf("want CodeTransferNotPending, got %v", err)
	}
}

func TestPlotTransferService_ListScope(t *testing.T) {
	db := newTestServiceDB(t)
	svc := newTransferService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	other := newTestUser(t, db, "citizen", "citizen")
	admin := newTestUser(t, db, "admin", "admin")
	uid := owner.ID
	oid := other.ID
	p1 := newTestPlot(t, db, "P-TR-L1", "adopted", &uid)
	p2 := newTestPlot(t, db, "P-TR-L2", "adopted", &oid)
	req1, err := svc.Submit(p1.ID, owner.ID, "理由一")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if _, err := svc.Submit(p2.ID, other.ID, "理由二"); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// 普通用户只能看到自己的申请
	_, total, err := svc.List(util.PageQuery{Page: 1, PageSize: 10}, "", owner.ID, "farmer")
	if err != nil || total != 1 {
		t.Errorf("owner list total=%d err=%v, want 1", total, err)
	}
	// 管理员看到全部
	_, total, err = svc.List(util.PageQuery{Page: 1, PageSize: 10}, "", admin.ID, "admin")
	if err != nil || total != 2 {
		t.Errorf("admin list total=%d err=%v, want 2", total, err)
	}
	// 详情越权：非本人非管理员不可见
	if _, err := svc.GetByID(req1.ID, other.ID, "citizen"); err == nil {
		t.Errorf("expected forbidden for other user's request detail")
	}
	// 本人与管理员可见
	if _, err := svc.GetByID(req1.ID, owner.ID, "farmer"); err != nil {
		t.Errorf("applicant should see own request: %v", err)
	}
	if _, err := svc.GetByID(req1.ID, admin.ID, "admin"); err != nil {
		t.Errorf("admin should see any request: %v", err)
	}
}
