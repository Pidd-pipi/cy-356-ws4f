package service

import (
	"fmt"
	"testing"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/util"
)

func TestPlotService_Adopt(t *testing.T) {
	db := newTestServiceDB(t)
	svc, _ := newPlotService(t, db)
	user := newTestUser(t, db, "citizen", "citizen")
	plot := newTestPlot(t, db, "P-ADOPT", "available", nil)

	tests := []struct {
		name    string
		plotID  uint
		userID  uint
		wantErr bool
	}{
		{name: "adopt available", plotID: plot.ID, userID: user.ID, wantErr: false},
		{name: "adopt again conflicts", plotID: plot.ID, userID: user.ID, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.Adopt(tt.plotID, tt.userID, "citizen", "citizen")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Adopt: %v", err)
			}
			if got.Status != string(constants.PlotStatusAdopted) || got.AdopterID == nil || *got.AdopterID != tt.userID {
				t.Errorf("adopt result invalid: status=%s adopter=%v", got.Status, got.AdopterID)
			}
		})
	}
}

func TestPlotService_Release(t *testing.T) {
	db := newTestServiceDB(t)
	svc, _ := newPlotService(t, db)
	owner := newTestUser(t, db, "farmer", "farmer")
	other := newTestUser(t, db, "citizen2", "citizen")
	admin := newTestUser(t, db, "admin1", "admin")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-REL", "harvested", &uid)

	// 认养人不能再直接释放（须走转交申请流程）
	if _, err := svc.Release(plot.ID, owner.ID, "farmer"); err == nil {
		t.Fatalf("expected forbidden error for adopter direct release")
	}
	// 其他居民同样无权直接释放
	if _, err := svc.Release(plot.ID, other.ID, "citizen"); err == nil {
		t.Fatalf("expected forbidden error for non-admin release")
	}
	// 管理员可直接强制释放
	got, err := svc.Release(plot.ID, admin.ID, "admin")
	if err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got.Status != string(constants.PlotStatusAvailable) || got.AdopterID != nil {
		t.Errorf("release result invalid: status=%s adopter=%v", got.Status, got.AdopterID)
	}
}

func TestPlotService_AdoptBlockedByPendingTransfer(t *testing.T) {
	db := newTestServiceDB(t)
	plotSvc, _ := newPlotService(t, db)
	transferSvc, _ := newPlotTransferService(t, db)
	owner := newTestUser(t, db, "owner", "citizen")
	other := newTestUser(t, db, "other", "citizen")
	uid := owner.ID
	plot := newTestPlot(t, db, "P-PENDING", "adopted", &uid)

	// 认养人提交转交申请
	if _, err := transferSvc.Submit(plot.ID, owner.ID, "citizen", "owner", "农忙无暇打理"); err != nil {
		// 走 SQLite 测试时若部分唯一索引缺失，下面的断言仍能验证主流程
		t.Fatalf("Submit: %v", err)
	}
	// 待处理期间其他居民不能认养
	if _, err := plotSvc.Adopt(plot.ID, other.ID, "citizen", "other"); err == nil {
		t.Fatalf("expected adopt blocked while transfer pending")
	}
}

func TestPlotService_AdoptUsesPageQuery(t *testing.T) {
	// 验证 List 分页复用
	db := newTestServiceDB(t)
	svc, _ := newPlotService(t, db)
	for i := 0; i < 3; i++ {
		newTestPlot(t, db, fmt.Sprintf("P-LIST-%d", i), "available", nil)
	}
	plots, total, err := svc.List(util.PageQuery{Page: 1, PageSize: 2}, "")
	if err != nil || total != 3 {
		t.Errorf("List total=%d err=%v", total, err)
	}
	if len(plots) != 2 {
		t.Errorf("List len=%d, want 2", len(plots))
	}
}
