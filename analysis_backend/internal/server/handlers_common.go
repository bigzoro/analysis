package server

import (
	"net/http"
	"strconv"
	"strings"

	pdb "analysis/internal/db"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

// PaginationParams 分页参数
type PaginationParams struct {
	Page     int
	PageSize int
	Offset   int
}

// ParsePaginationParams 解析分页参数（统一处理）
// defaultPageSize: 默认每页数量
// maxPageSize: 最大每页数量（防止过大查询）
func ParsePaginationParams(pageStr, pageSizeStr string, defaultPageSize, maxPageSize int) PaginationParams {
	params := PaginationParams{
		Page:     1,
		PageSize: defaultPageSize,
	}

	// 解析页码
	if s := strings.TrimSpace(pageStr); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 1 {
			params.Page = v
		}
	}

	// 解析每页数量
	if s := strings.TrimSpace(pageSizeStr); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			if v > maxPageSize {
				v = maxPageSize
			}
			params.PageSize = v
		}
	}

	// 计算偏移量
	params.Offset = (params.Page - 1) * params.PageSize

	return params
}

// parseCoinsParam 解析币种参数（从逗号分隔的字符串）
func parseCoinsParam(coinsStr string) []string {
	if coinsStr == "" {
		return nil
	}
	var coins []string
	for _, x := range strings.Split(coinsStr, ",") {
		x = strings.ToUpper(strings.TrimSpace(x))
		if x != "" {
			coins = append(coins, x)
		}
	}
	return coins
}

// flowQueryParams 资金流查询参数
type flowQueryParams struct {
	Entity string
	Coins  []string
	Latest bool
	RunID  string
	Start  string
	End    string
}

// buildFlowQuery 构建资金流查询（支持 DailyFlow 和 WeeklyFlow）
func buildFlowQuery(db *gorm.DB, model interface{}, params flowQueryParams) *gorm.DB {
	q := db.Model(model).Where("entity = ?", params.Entity)

	if params.Latest && params.RunID != "" {
		q = q.Where("run_id = ?", params.RunID)
	}

	if len(params.Coins) > 0 {
		q = q.Where("coin IN ?", params.Coins)
	}

	if params.Start != "" {
		q = q.Where("day >= ?", params.Start)
	}

	if params.End != "" {
		q = q.Where("day <= ?", params.End)
	}

	return q
}

// buildWeeklyFlowQuery 构建周度资金流查询
func buildWeeklyFlowQuery(db *gorm.DB, params flowQueryParams) *gorm.DB {
	q := db.Model(&pdb.WeeklyFlow{}).Where("entity = ?", params.Entity)

	if params.Latest && params.RunID != "" {
		q = q.Where("run_id = ?", params.RunID)
	}

	if len(params.Coins) > 0 {
		q = q.Where("coin IN ?", params.Coins)
	}

	return q
}

// flowRow 资金流行数据（日度）
type flowRow struct {
	Day string  `json:"day"`
	In  float64 `json:"in"`
	Out float64 `json:"out"`
	Net float64 `json:"net"`
}

// weeklyFlowRow 资金流行数据（周度）
type weeklyFlowRow struct {
	Week string  `json:"week"`
	In   float64 `json:"in"`
	Out  float64 `json:"out"`
	Net  float64 `json:"net"`
}

// HoldingDTO 持仓数据传输对象
type HoldingDTO struct {
	Chain    string  `json:"chain"`
	Symbol   string  `json:"symbol"`
	Decimals int     `json:"decimals"`
	Amount   string  `json:"amount"`
	ValueUSD float64 `json:"value_usd"`
}

// WebSocket upgrader for all WebSocket connections
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}
