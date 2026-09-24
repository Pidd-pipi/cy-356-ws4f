package router

import (
	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/middleware"
)

// registerPlotTransfers 地块转交申请路由。
func (r *Router) registerPlotTransfers(g *gin.RouterGroup) {
	auth := g.Group("")
	auth.Use(middleware.Auth(r.cfg, r.logger))
	{
		// 认养人在地块下提交转交申请
		auth.POST("/plots/:id/transfer-requests", r.plotTransferHandler.Submit)
		// 申请列表与详情（管理员全部 / 本人）
		auth.GET("/transfer-requests", r.plotTransferHandler.List)
		auth.GET("/transfer-requests/:id", r.plotTransferHandler.Get)
		// 申请人撤回待处理申请
		auth.POST("/transfer-requests/:id/cancel", r.plotTransferHandler.Cancel)
	}

	admin := g.Group("/transfer-requests")
	admin.Use(middleware.Auth(r.cfg, r.logger), middleware.RequireRoles(string(constants.RoleAdmin)))
	{
		// 管理员核准 / 驳回
		admin.POST("/:id/approve", r.plotTransferHandler.Approve)
		admin.POST("/:id/reject", r.plotTransferHandler.Reject)
	}
}
