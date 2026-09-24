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

// NewPlotTransferHandler 构造地块转交申请接口。
func NewPlotTransferHandler(transferService *service.PlotTransferService, audit middleware.AuditWriter) *PlotTransferHandler {
	return &PlotTransferHandler{transferService: transferService, audit: audit}
}

// Submit 认养人在地块列表提交转交申请（填写接管理由）。
func (h *PlotTransferHandler) Submit(c *gin.Context) {
	plotID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.CreateTransferRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	created, err := h.transferService.Submit(uint(plotID), claims.UserID, req.Reason)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "SUBMIT_TRANSFER_REQUEST", "plot_transfer_request", strconv.FormatUint(uint64(created.ID), 10),
		"提交地块转交申请，地块 "+created.Plot.Code, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferRequestOutDTO(created))
}

// List 转交申请列表（管理员看全部，认养人看自己的）。
func (h *PlotTransferHandler) List(c *gin.Context) {
	pq := util.ParsePageQuery(c)
	status := c.Query("status")
	claims, _ := util.GetClaims(c)
	reqs, total, err := h.transferService.List(pq, status, claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	list := make([]*dto.TransferRequestOutDTO, 0, len(reqs))
	for i := range reqs {
		list = append(list, dto.ToTransferRequestOutDTO(&reqs[i]))
	}
	util.OK(c, util.PageResult{List: list, Total: total, Page: pq.Page, PageSize: pq.PageSize})
}

// Get 转交申请详情（申请人本人或管理员）。
func (h *PlotTransferHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	req, err := h.transferService.GetByID(uint(id), claims.UserID, claims.Role)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	util.OK(c, dto.ToTransferRequestOutDTO(req))
}

// Approve 管理员核准转交申请（清空原认养关系，地块回到共享池）。
func (h *PlotTransferHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.ApproveTransferRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	updated, err := h.transferService.Approve(uint(id), claims.UserID, claims.Username, req.ReviewComment)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "APPROVE_TRANSFER_REQUEST", "plot_transfer_request", strconv.FormatUint(uint64(id), 10),
		"核准地块转交申请，地块 "+updated.Plot.Code+" 回到共享池", c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferRequestOutDTO(updated))
}

// Reject 管理员驳回转交申请（必须写明处理意见）。
func (h *PlotTransferHandler) Reject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	var req dto.RejectTransferRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidationFailed, constants.ErrorText[constants.CodeValidationFailed]+": "+err.Error())
		return
	}
	claims, _ := util.GetClaims(c)
	updated, err := h.transferService.Reject(uint(id), claims.UserID, claims.Username, req.ReviewComment)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "REJECT_TRANSFER_REQUEST", "plot_transfer_request", strconv.FormatUint(uint64(id), 10),
		"驳回地块转交申请，地块 "+updated.Plot.Code, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferRequestOutDTO(updated))
}

// Cancel 申请人撤回待处理的转交申请。
func (h *PlotTransferHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "路径参数 id 必须为正整数")
		return
	}
	claims, _ := util.GetClaims(c)
	updated, err := h.transferService.Cancel(uint(id), claims.UserID)
	if err != nil {
		util.FailWithAppError(c, err)
		return
	}
	_ = h.audit.Write(claims.UserID, claims.Username, claims.Role, "CANCEL_TRANSFER_REQUEST", "plot_transfer_request", strconv.FormatUint(uint64(id), 10),
		"撤回地块转交申请，地块 "+updated.Plot.Code, c.ClientIP(), util.GetRequestID(c))
	util.OK(c, dto.ToTransferRequestOutDTO(updated))
}
