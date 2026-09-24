package constants

// 业务枚举定义：同一枚举同时出现在 model / dto / service 状态机 / handler 校验 /
// 前端 constants / 筛选徽标 / 错误码 / 日志模板 / formatters 中。
// 新增一个枚举值必须同步修改至少 10 处文件（牵一发动全身）。

// RoleType 角色类型
type RoleType string

const (
	RoleAdmin   RoleType = "admin"   // 管理员
	RoleFarmer  RoleType = "farmer"  // 农场主
	RoleCitizen RoleType = "citizen" // 城市居民
)

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// PlotStatus 地块状态
type PlotStatus string

const (
	PlotStatusAvailable       PlotStatus = "available"        // 空闲可认养
	PlotStatusAdopted         PlotStatus = "adopted"          // 已认养
	PlotStatusHarvested       PlotStatus = "harvested"        // 已收成待释放
	PlotStatusPendingTransfer PlotStatus = "pending_transfer" // 转交申请待核准（其他居民不可认养）
)

// TransferStatus 地块转交申请状态机：pending -> approved / rejected / withdrawn
type TransferStatus string

const (
	TransferPending   TransferStatus = "pending"   // 待管理员处理
	TransferApproved  TransferStatus = "approved"  // 管理员核准，地块回到共享池
	TransferRejected  TransferStatus = "rejected"  // 管理员驳回，认养关系保留
	TransferWithdrawn TransferStatus = "withdrawn" // 认养人撤回，认养关系保留
)

// SoilType 土壤类型
type SoilType string

const (
	SoilLoam  SoilType = "loam"  // 壤土
	SoilClay  SoilType = "clay"  // 黏土
	SoilSand  SoilType = "sand"  // 沙土
	SoilBlack SoilType = "black" // 黑土
)

// Sunlight 日照条件
type Sunlight string

const (
	SunlightFull    Sunlight = "full"    // 全日照
	SunlightPartial Sunlight = "partial" // 半日照
	SunlightShade   Sunlight = "shade"   // 遮阴
)

// PlanStatus 种植计划状态机：planned -> planting -> growing -> harvesting -> completed
type PlanStatus string

const (
	PlanStatusPlanned    PlanStatus = "planned"
	PlanStatusPlanting   PlanStatus = "planting"
	PlanStatusGrowing    PlanStatus = "growing"
	PlanStatusHarvesting PlanStatus = "harvesting"
	PlanStatusCompleted  PlanStatus = "completed"
)

// CropType 作物类型
type CropType string

const (
	CropVegetable CropType = "vegetable" // 蔬菜
	CropFruit     CropType = "fruit"     // 水果
	CropHerb      CropType = "herb"      // 香草
)

// Season 季节
type Season string

const (
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
	SeasonWinter Season = "winter"
)

// DiaryAction 种植日记动作类型
type DiaryAction string

const (
	DiarySowing      DiaryAction = "sowing"
	DiaryWatering    DiaryAction = "watering"
	DiaryFertilizing DiaryAction = "fertilizing"
	DiaryPestControl DiaryAction = "pest_control"
	DiaryHarvest     DiaryAction = "harvest"
	DiaryOther       DiaryAction = "other"
)

// HarvestQuality 收成品质
type HarvestQuality string

const (
	QualityExcellent HarvestQuality = "excellent" // 优
	QualityGood      HarvestQuality = "good"      // 良
	QualityFair      HarvestQuality = "fair"      // 一般
)

// PostType 社区帖子类型
type PostType string

const (
	PostExperience PostType = "experience" // 种植经验
	PostPest       PostType = "pest"       // 病虫害防治
	PostRecipe     PostType = "recipe"     // 食谱创意
	PostActivity   PostType = "activity"   // 线下农耕活动
)

// PostStatus 社区帖子状态
type PostStatus string

const (
	PostStatusPublished PostStatus = "published"
	PostStatusRemoved   PostStatus = "removed"
)

// 季节推荐作物表（静态推荐数据，服务层读取）
var SeasonCrops = map[Season][]string{
	SeasonSpring: {"菠菜", "生菜", "豌豆", "草莓", "香葱", "萝卜"},
	SeasonSummer: {"番茄", "黄瓜", "茄子", "辣椒", "西瓜", "罗勒"},
	SeasonAutumn: {"白菜", "胡萝卜", "南瓜", "苹果", "迷迭香", "大蒜"},
	SeasonWinter: {"羽衣甘蓝", "韭菜", "芹菜", "薄荷", "卷心菜", "洋葱"},
}

// TransferStatusTransitions 转交申请状态机（service 核准/撤回并发判定 +
// 前端按钮显隐 + 日志模板 + formatters + 错误码多处定义）。
var TransferStatusTransitions = map[TransferStatus][]TransferStatus{
	TransferPending:   {TransferApproved, TransferRejected, TransferWithdrawn},
	TransferApproved:  {},
	TransferRejected:  {},
	TransferWithdrawn: {},
}

// CanTransferTo 判断当前申请状态是否允许流转到目标状态。
func CanTransferTo(from, to TransferStatus) bool {
	for _, s := range TransferStatusTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}
