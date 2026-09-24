package constants

// messages.go 同时包含接口返回文案、日志文案、错误提示文案（多处耦合的“屎山”设计点）。
const (
	MsgSuccess             = "ok"
	MsgWelcome             = "欢迎使用城市共享菜园管理平台"
	MsgLoginRequired       = "请先登录后再继续操作"
	MsgPermissionDenied    = "当前角色 %s 无权执行该操作，需要角色: %s"
	MsgPlotAdoptSuccess    = "地块认养成功，开始你的都市农夫之旅"
	MsgPlotReleaseOK       = "地块已释放，重新回到共享池"
	MsgTransferSubmitOK    = "转交申请已提交，等待管理员核准"
	MsgTransferApproveOK   = "转交申请已核准，地块已回到共享池"
	MsgTransferRejectOK    = "转交申请已驳回，原认养关系保留"
	MsgTransferWithdrawOK  = "转交申请已撤回"
	MsgTransferAlreadyDone = "该申请已经处理，请勿重复操作"
	MsgPlanCreatedOK       = "种植计划创建成功，系统已生成预期收获时间线"
	MsgPlanStatusOK        = "种植计划状态已更新为 %s"
	MsgHarvestRecordOK     = "收成记录已保存，年度统计已更新"
	MsgDiaryCreatedOK      = "种植日记已发布"
	MsgPostCreatedOK       = "社区帖子发布成功"
	MsgRegisterOK          = "注册成功，请登录"
	MsgLoginOK             = "登录成功"
	MsgAuditListOK         = "审计日志查询成功"
	MsgRecommendOK         = "季节作物推荐获取成功"
	MsgStatsOK             = "年度收成统计获取成功"
	MsgHealthOK            = "服务运行正常"
)
