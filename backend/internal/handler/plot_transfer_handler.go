package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/communitygarden/server/internal/constants"
	"github.com/communitygarden/server/internal/dto"
	"github.com/communitygarden/server/internal/middleware"
	"github.com/communitygarden/server/internal/service"
	"github.com/communitygarden/server/internal/util"
)

// PlotTransferHandler 地块转交申请接口。
type PlotTransferHandler struct {
	transferService *service.PlotTransferService
	audit           middleware.AuditWriter
}

// NewPlotTransferHandler 构造转交申请接口。
func NewPlotTransferHandler(transferService *service.PlotTransferService, audit middleware.AuditWriter) *PlotTransferHandler {
	return &PlotTransferHandler{transferService: transferService, audit: audit}
}

// Submit 认养人在地块列表提交转交申请（登录）。
func (h *PlotTransferHandler) Submit(c *gin.Context) {
	plotID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	transfer, err := h.transferService.Submit(uint(plotID), claims.UserID, claims.Role, claims.Username, req.Reason)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "SUBMIT_TRANSFER", "plot_transfer", strconv.FormatUint(uint64(transfer.ID), 10),
		"提交地块转交申请 plot_id="+strconv.FormatUint(plotID, 10)+" 理由: "+req.Reason, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferApplicationOutDTO(transfer))
}

// List 管理员分页查询转交申请（可按 status 过滤）。
func (h *PlotTransferHandler) List(c *gin.Context) {
	pq := util.ParsePageQuery(c)
	status := c.Query("status")
	transfers, total, err := h.transferService.List(pq, status)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.TransferApplicationOutDTO, 0, len(transfers))
	for i := range transfers {
		list = append(list, dto.ToTransferApplicationOutDTO(&transfers[i]))
	}
	util.OK(c, util.PageResult{List: list, Total: total, Page: pq.Page, PageSize: pq.PageSize})
}

// Get 查询单条转交申请详情（登录）。
func (h *PlotTransferHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	transfer, err := h.transferService.GetByID(uint(id))
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	util.OK(c, dto.ToTransferApplicationOutDTO(transfer))
}

// Review 管理员核准/驳回转交申请（驳回必须写明意见）。
func (h *PlotTransferHandler) Review(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.ReviewTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	if !*req.Approve && req.Comment == "" {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, "驳回转交申请 id="+c.Param("id")+" 必须填写处理意见 comment 字段")
		return
	}
	claims, _ := util.GetClaims(c)
	if *req.Approve {
		transfer, svcErr := h.transferService.Approve(uint(id), claims.UserID, claims.Username, claims.Role, req.Comment)
		if svcErr != nil {
			util.FailWithAppError(c, svcErr)
			return
		}
		_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "APPROVE_TRANSFER", "plot_transfer", c.Param("id"),
			"核准地块转交申请 id="+c.Param("id")+" 意见: "+req.Comment, c.ClientIP(), util.GetRequestID(c))
		util.OK(c, dto.ToTransferApplicationOutDTO(transfer))
		return
	}
	transfer, svcErr := h.transferService.Reject(uint(id), claims.UserID, claims.Username, claims.Role, req.Comment)
	if svcErr != nil {
		util.FailWithAppError(c, svcErr)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "REJECT_TRANSFER", "plot_transfer", c.Param("id"),
		"驳回地块转交申请 id="+c.Param("id")+" 意见: "+req.Comment, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferApplicationOutDTO(transfer))
}

// Withdraw 认养人本人撤回转交申请。
func (h *PlotTransferHandler) Withdraw(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	transfer, err := h.transferService.Withdraw(uint(id), claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "WITHDRAW_TRANSFER", "plot_transfer", c.Param("id"),
		"撤回地块转交申请 id="+c.Param("id"), c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferApplicationOutDTO(transfer))
}
