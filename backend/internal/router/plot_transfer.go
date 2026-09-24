package router

import (
	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/middleware"
)

// registerPlotTransfers 地块转交申请路由。
func (r *Router) registerPlotTransfers(g *gin.RouterGroup) {
	// 申请单集合：管理员查看列表
	transfers := g.Group("/plot-transfers")
	adminList := transfers.Group("")
	adminList.Use(middleware.Auth(r.cfg, r.logger), middleware.RequireRoles(string(constants.RoleAdmin)))
	{
		adminList.GET("", r.plotTransferHandler.List)
	}

	// 认养人在地块列表提交转交申请（登录即可，service 内校验认养人身份）
	plots := g.Group("/plots")
	submit := plots.Group("")
	submit.Use(middleware.Auth(r.cfg, r.logger))
	{
		submit.POST("/:id/transfers", r.plotTransferHandler.Submit)
	}

	// 单条申请：本人可撤回，管理员可核准/驳回
	single := transfers.Group("/:id")
	single.Use(middleware.Auth(r.cfg, r.logger))
	{
		single.GET("", r.plotTransferHandler.Get)
		single.POST("/withdraw", r.plotTransferHandler.Withdraw)
	}
	review := transfers.Group("/:id")
	review.Use(middleware.Auth(r.cfg, r.logger), middleware.RequireRoles(string(constants.RoleAdmin)))
	{
		review.POST("/review", r.plotTransferHandler.Review)
	}
}
