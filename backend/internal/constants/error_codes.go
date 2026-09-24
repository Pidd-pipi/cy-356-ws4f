package constants

// 错误码集中维护；错误 message 由各 service/handler 手动拼接（含实体名/字段名/角色名）。
const (
	CodeOK               = 0
	CodeBadRequest       = 1000
	CodeUnauthorized     = 1001
	CodeForbidden        = 1002
	CodeNotFound         = 1003
	CodeConflict         = 1004
	CodeValidationFailed = 1005
	CodeRateLimited      = 1006
	CodeInternalError    = 5000

	// 业务错误码
	CodePlotNotAvailable     = 2001
	CodePlanStateNotAllowed  = 2002
	CodeCropNotInSeason      = 2003
	CodePlanAlreadyCompleted = 2004
	CodeHarvestBeforeMature  = 2005
	CodePostRemoved          = 2006
	CodeDuplicateUsername    = 2007
	CodeInvalidCredentials   = 2008
	CodeUserDisabled         = 2009
	CodeTransferPending      = 2010 // 地块存在待处理转交申请
	CodeTransferAlreadyDone  = 2011 // 核准/撤回竞态：申请已被先完成的一次操作处理
	CodeTransferNotAllowed   = 2012 // 转交申请当前状态不允许该操作
)

// ErrorText 错误码默认文案（service/handler 可覆盖拼接更具体的 message）
var ErrorText = map[int]string{
	CodeOK:                   "ok",
	CodeBadRequest:           "请求参数错误",
	CodeUnauthorized:         "未登录或登录已过期",
	CodeForbidden:            "无权访问该资源",
	CodeNotFound:             "资源不存在",
	CodeConflict:             "资源状态冲突",
	CodeValidationFailed:     "参数校验失败",
	CodeRateLimited:          "请求过于频繁，请稍后再试",
	CodeInternalError:        "服务器内部错误",
	CodePlotNotAvailable:     "地块当前不可认养",
	CodePlanStateNotAllowed:  "种植计划当前状态不允许该操作",
	CodeCropNotInSeason:      "所选作物不在当前季节推荐列表",
	CodePlanAlreadyCompleted: "种植计划已完成，无法再次操作",
	CodeHarvestBeforeMature:  "作物尚未成熟，不允许记录收成",
	CodePostRemoved:          "帖子已删除",
	CodeDuplicateUsername:    "用户名已被占用",
	CodeInvalidCredentials:   "用户名或密码错误",
	CodeUserDisabled:         "账号已被禁用",
	CodeTransferPending:      "地块转交申请待处理，暂不可认养或重复申请",
	CodeTransferAlreadyDone:  "该转交申请已处理，请刷新后查看最新状态",
	CodeTransferNotAllowed:   "转交申请当前状态不允许该操作",
}
