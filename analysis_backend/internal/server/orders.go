package server

import (
	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 合约交易操作类型定义
type FuturesOperationType struct {
	Type        string `json:"type"`        // 开多/开空/平多/平空
	Description string `json:"description"` // 详细描述
	Class       string `json:"class"`       // CSS类名
}

// 获取合约交易操作类型
func getFuturesOperationType(side string, reduceOnly bool) FuturesOperationType {
	if reduceOnly {
		// 平仓操作
		if side == "BUY" {
			return FuturesOperationType{
				Type:        "平空",
				Description: "平空头仓位",
				Class:       "close-short",
			}
		} else {
			return FuturesOperationType{
				Type:        "平多",
				Description: "平多头仓位",
				Class:       "close-long",
			}
		}
	} else {
		// 开仓操作
		if side == "BUY" {
			return FuturesOperationType{
				Type:        "开多",
				Description: "开多头仓位",
				Class:       "open-long",
			}
		} else {
			return FuturesOperationType{
				Type:        "开空",
				Description: "开空头仓位",
				Class:       "open-short",
			}
		}
	}
}

type scheduleReq struct {
	// Bracket 扩展参数
	BracketEnabled bool    `json:"bracket_enabled"`
	TPPercent      float64 `json:"tp_percent"`
	SLPercent      float64 `json:"sl_percent"`
	TPPrice        string  `json:"tp_price"`
	SLPrice        string  `json:"sl_price"`
	WorkingType    string  `json:"working_type"`

	Exchange    string `json:"exchange"`     // "binance_futures"
	Testnet     bool   `json:"testnet"`      // true=走测试网
	Symbol      string `json:"symbol"`       // e.g. BTCUSDT
	Side        string `json:"side"`         // BUY/SELL
	OrderType   string `json:"order_type"`   // MARKET/LIMIT
	Quantity    string `json:"quantity"`     // 下单数量(合约张/币数，按交易所规则)
	Price       string `json:"price"`        // 限价单需要
	Leverage    int    `json:"leverage"`     // 0=不设置
	ReduceOnly  bool   `json:"reduce_only"`  // 可选
	StrategyID  *uint  `json:"strategy_id"`  // 可选策略ID
	TriggerTime string `json:"trigger_time"` // ISO8601，本地前端传 UTC 或带时区
}

func (s *Server) CreateScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req scheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}
	if req.Exchange == "" || req.Symbol == "" || req.Side == "" || req.OrderType == "" || req.Quantity == "" || req.TriggerTime == "" {
		s.ValidationError(c, "", "缺少必填字段：exchange, symbol, side, order_type, quantity, trigger_time")
		return
	}
	tt, err := time.Parse(time.RFC3339, req.TriggerTime)
	if err != nil {
		s.ValidationError(c, "trigger_time", "触发时间格式错误，应为 RFC3339 格式")
		return
	}

	// 如果是合约交易，验证交易对是否支持
	if strings.ToLower(req.Exchange) == "binance_futures" {
		// 使用配置的环境设置，而不是请求的Testnet字段
		useTestnet := s.cfg.Exchange.Binance.IsTestnet
		client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
		supported, err := client.IsSymbolSupported(strings.ToUpper(req.Symbol))
		if err != nil {
			s.ValidationError(c, "symbol", fmt.Sprintf("无法验证交易对支持情况: %v", err))
			return
		}
		if !supported {
			s.ValidationError(c, "symbol", fmt.Sprintf("交易对 %s 不支持期货交易", req.Symbol))
			return
		}
	}
	ord := &pdb.ScheduledOrder{
		UserID:           uid,
		Exchange:         strings.ToLower(req.Exchange),
		Testnet:          req.Testnet,
		Symbol:           strings.ToUpper(req.Symbol),
		Side:             strings.ToUpper(req.Side),
		OrderType:        strings.ToUpper(req.OrderType),
		Quantity:         strings.TrimSpace(req.Quantity),
		AdjustedQuantity: "", // 创建时为空，执行时更新
		Price:            strings.TrimSpace(req.Price),
		Leverage:         req.Leverage,
		ReduceOnly:       req.ReduceOnly,
		StrategyID:       req.StrategyID,

		// === 保存 Bracket 参数 ===
		BracketEnabled: req.BracketEnabled,
		TPPercent:      req.TPPercent,
		SLPercent:      req.SLPercent,
		TPPrice:        strings.TrimSpace(req.TPPrice),
		SLPrice:        strings.TrimSpace(req.SLPrice),
		WorkingType:    strings.ToUpper(strings.TrimSpace(req.WorkingType)),

		TriggerTime: tt.UTC(),
		Status:      "pending",
	}
	if err := s.db.CreateScheduledOrder(ord); err != nil {
		s.DatabaseError(c, "创建定时订单", err)
		return
	}

	// 异步预设保证金模式（不阻塞API响应）
	if req.StrategyID != nil && strings.ToLower(req.Exchange) == "binance_futures" {
		go s.trySetMarginModeForScheduledOrder(ord.ID, *req.StrategyID, req.Symbol)
	}

	// 返回完整的订单信息，包括操作类型
	operationType := getFuturesOperationType(ord.Side, ord.ReduceOnly)
	c.JSON(http.StatusOK, gin.H{
		"id":              ord.ID,
		"operation_type":  operationType.Type,
		"operation_desc":  operationType.Description,
		"operation_class": operationType.Class,
		"side":            ord.Side,
		"reduce_only":     ord.ReduceOnly,
		"symbol":          ord.Symbol,
		"trigger_time":    ord.TriggerTime,
	})
}

// 批量创建定时订单
type batchScheduleReq struct {
	Orders []scheduleReq `json:"orders" binding:"required,min=1,max=10"`
}

func (s *Server) CreateBatchScheduledOrders(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	var req batchScheduleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		s.JSONBindError(c, err)
		return
	}

	if len(req.Orders) == 0 {
		s.ValidationError(c, "orders", "订单列表不能为空")
		return
	}

	if len(req.Orders) > 10 {
		s.ValidationError(c, "orders", "一次最多只能创建10个订单")
		return
	}

	results := make([]gin.H, 0, len(req.Orders))
	successCount := 0
	failCount := 0

	// 处理每个订单
	for i, orderReq := range req.Orders {
		// 验证单个订单的必填字段
		if orderReq.Exchange == "" || orderReq.Symbol == "" || orderReq.Side == "" ||
			orderReq.OrderType == "" || orderReq.Quantity == "" || orderReq.TriggerTime == "" {
			results = append(results, gin.H{
				"index":   i,
				"symbol":  orderReq.Symbol,
				"success": false,
				"error":   "缺少必填字段：exchange, symbol, side, order_type, quantity, trigger_time",
			})
			failCount++
			continue
		}

		// 解析触发时间
		tt, err := time.Parse(time.RFC3339, orderReq.TriggerTime)
		if err != nil {
			results = append(results, gin.H{
				"index":   i,
				"symbol":  orderReq.Symbol,
				"success": false,
				"error":   "触发时间格式错误，应为 RFC3339 格式",
			})
			failCount++
			continue
		}

		// 如果是合约交易，验证交易对是否支持
		if strings.ToLower(orderReq.Exchange) == "binance_futures" {
			// 使用配置的环境设置，而不是请求的Testnet字段
			useTestnet := s.cfg.Exchange.Binance.IsTestnet
			client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
			supported, err := client.IsSymbolSupported(strings.ToUpper(orderReq.Symbol))
			if err != nil {
				results = append(results, gin.H{
					"index":   i,
					"symbol":  orderReq.Symbol,
					"success": false,
					"error":   fmt.Sprintf("无法验证交易对支持情况: %v", err),
				})
				failCount++
				continue
			}
			if !supported {
				results = append(results, gin.H{
					"index":   i,
					"symbol":  orderReq.Symbol,
					"success": false,
					"error":   fmt.Sprintf("交易对 %s 不支持期货交易", orderReq.Symbol),
				})
				failCount++
				continue
			}
		}

		// 创建订单
		ord := &pdb.ScheduledOrder{
			UserID:           uid,
			Exchange:         strings.ToLower(orderReq.Exchange),
			Testnet:          orderReq.Testnet,
			Symbol:           strings.ToUpper(orderReq.Symbol),
			Side:             strings.ToUpper(orderReq.Side),
			OrderType:        strings.ToUpper(orderReq.OrderType),
			Quantity:         strings.TrimSpace(orderReq.Quantity),
			AdjustedQuantity: "", // 创建时为空，执行时更新
			Price:            strings.TrimSpace(orderReq.Price),
			Leverage:         orderReq.Leverage,
			ReduceOnly:       orderReq.ReduceOnly,
			StrategyID:       orderReq.StrategyID,

			// Bracket 参数
			BracketEnabled: orderReq.BracketEnabled,
			TPPercent:      orderReq.TPPercent,
			SLPercent:      orderReq.SLPercent,
			TPPrice:        strings.TrimSpace(orderReq.TPPrice),
			SLPrice:        strings.TrimSpace(orderReq.SLPrice),
			WorkingType:    strings.ToUpper(strings.TrimSpace(orderReq.WorkingType)),

			TriggerTime: tt.UTC(),
			Status:      "pending",
		}

		if err := s.db.CreateScheduledOrder(ord); err != nil {
			results = append(results, gin.H{
				"index":   i,
				"symbol":  orderReq.Symbol,
				"success": false,
				"error":   fmt.Sprintf("创建订单失败: %v", err),
			})
			failCount++
			continue
		}

		// 在保存订单后，立即尝试异步设置保证金模式
		if ord.StrategyID != nil {
			go s.trySetMarginModeForScheduledOrder(ord.ID, *ord.StrategyID, ord.Symbol)
		}

		// 添加操作类型信息
		operationType := getFuturesOperationType(ord.Side, ord.ReduceOnly)
		results = append(results, gin.H{
			"index":           i,
			"symbol":          orderReq.Symbol,
			"success":         true,
			"id":              ord.ID,
			"operation_type":  operationType.Type,
			"operation_desc":  operationType.Description,
			"operation_class": operationType.Class,
		})
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"total":         len(req.Orders),
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       results,
	})
}

// GET /orders/schedule?page=1&page_size=50
func (s *Server) ListScheduledOrders(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	// 分页参数
	pagination := ParsePaginationParams(
		c.Query("page"),
		c.Query("page_size"),
		50,  // 默认每页数量
		200, // 最大每页数量
	)

	// 调试日志
	log.Printf("[Order-API] ListScheduledOrders: user_id=%d, page=%d, page_size=%d, offset=%d",
		uid, pagination.Page, pagination.PageSize, pagination.Offset)

	// 使用接口方法查询
	orders, total, err := s.db.ListScheduledOrders(uid, pagination)
	if err != nil {
		s.DatabaseError(c, "查询定时订单列表", err)
		return
	}

	// 为每个订单添加操作类型信息和关联订单信息
	enhancedOrders := make([]gin.H, len(orders))
	for i, order := range orders {
		operationType := getFuturesOperationType(order.Side, order.ReduceOnly)
		// 获取关联订单信息
		relatedOrders := s.getRelatedOrdersSummary(&order)
		enhancedOrders[i] = gin.H{
			"id":                order.ID,
			"user_id":           order.UserID,
			"exchange":          order.Exchange,
			"testnet":           order.Testnet,
			"symbol":            order.Symbol,
			"side":              order.Side,
			"order_type":        order.OrderType,
			"quantity":          order.Quantity,
			"adjusted_quantity": order.AdjustedQuantity,
			"price":             order.Price,
			"leverage":          order.Leverage,
			"reduce_only":       order.ReduceOnly,
			"strategy_id":       order.StrategyID,
			"bracket_enabled":   order.BracketEnabled,
			"tp_percent":        order.TPPercent,
			"sl_percent":        order.SLPercent,
			"tp_price":          order.TPPrice,
			"sl_price":          order.SLPrice,
			"working_type":      order.WorkingType,
			"trigger_time":      order.TriggerTime,
			"status":            order.Status,
			"result":            order.Result,
			"created_at":        order.CreatedAt,
			"updated_at":        order.UpdatedAt,
			// 新增的操作类型信息
			"operation_type":  operationType.Type,
			"operation_desc":  operationType.Description,
			"operation_class": operationType.Class,
			// 新增的关联订单摘要信息
			"related_orders": relatedOrders,
		}
	}

	// 计算总页数
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       enhancedOrders,
		"total":       total,
		"page":        pagination.Page,
		"page_size":   pagination.PageSize,
		"total_pages": totalPages,
	})
}

func (s *Server) CancelScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	idStr := c.Param("id")

	// 解析 ID
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		s.ValidationError(c, "id", "无效的订单ID")
		return
	}

	// 获取订单
	order, err := s.db.GetScheduledOrderByID(id)
	if err != nil {
		s.NotFound(c, "订单不存在")
		return
	}

	// 检查权限
	if order.UserID != uid {
		s.Forbidden(c, "无权操作此订单")
		return
	}

	// 检查状态
	if order.Status != "pending" && order.Status != "processing" {
		s.ValidationError(c, "status", "只能取消待执行或处理中的订单")
		return
	}

	// 更新状态
	order.Status = "canceled"
	if err := s.db.UpdateScheduledOrder(order); err != nil {
		s.DatabaseError(c, "取消定时订单", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"updated": 1})
}

// GET /orders/schedule/:id - 获取单个定时订单详情
func (s *Server) GetScheduledOrderDetail(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))

	orderIDStr := c.Param("id")
	orderID, err := strconv.ParseUint(orderIDStr, 10, 32)
	if err != nil {
		s.BadRequest(c, "订单ID格式错误", err)
		return
	}

	// 获取订单详情
	order, err := s.db.GetScheduledOrderByID(uint(orderID))
	if err != nil {
		s.DatabaseError(c, "获取订单详情", err)
		return
	}

	// 检查权限
	if order.UserID != uid {
		s.Forbidden(c, "无权访问此订单")
		return
	}

	// 为订单添加操作类型信息
	operationType := getFuturesOperationType(order.Side, order.ReduceOnly)

	// 查询订单状态（如果有clientOrderId）
	var orderStatus gin.H
	// 对于已结束的订单，不查询币安API状态，避免状态被更新为已完成
	isFinished := s.isOrderFinished(order)
	if order.ClientOrderId != "" && (order.Status == "success" || order.Status == "filled" || order.Status == "processing") && !isFinished {
		orderStatus = s.queryOrderStatus(order)
	}

	// 查询账户持仓信息
	var accountPositions gin.H
	if order.Exchange == "binance_futures" {
		accountPositions = s.queryAccountPositions(order)
	}

	// 查询关联订单信息
	relatedOrders := s.queryRelatedOrders(order)

	// 计算利润信息（对有成交信息的订单）
	var profitInfo gin.H
	hasQuantityInfo := order.AdjustedQuantity != ""
	if orderStatus != nil {
		if executedQty, ok := orderStatus["executed_qty"].(string); ok && executedQty != "" && executedQty != "0" {
			hasQuantityInfo = true
		}
	}

	if (order.Status == "filled" || order.Status == "success") && hasQuantityInfo {
		profitInfo = s.calculateOrderProfit(order, orderStatus, accountPositions)
	}

	// 计算名义价值和保证金信息
	var nominalValue, marginAmount, dealAmount float64
	var calculationNote string

	if order.AdjustedQuantity != "" && order.AvgPrice != "" {
		// 使用调整后的数量和平均价格计算
		if qty, err := strconv.ParseFloat(order.AdjustedQuantity, 64); err == nil {
			if price, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
				nominalValue = qty * price                            // 名义价值 = 数量 × 价格
				marginAmount = nominalValue / float64(order.Leverage) // 保证金 = 名义价值 ÷ 杠杆
				dealAmount = nominalValue                             // 成交金额 = 名义价值
				calculationNote = "基于实际成交数据计算"
			}
		}
	} else if order.Quantity != "" && order.AvgPrice != "" {
		// 使用原始数量和平均价格计算
		if qty, err := strconv.ParseFloat(order.Quantity, 64); err == nil {
			if price, err := strconv.ParseFloat(order.AvgPrice, 64); err == nil {
				nominalValue = qty * price                            // 名义价值 = 数量 × 价格
				marginAmount = nominalValue / float64(order.Leverage) // 保证金 = 名义价值 ÷ 杠杆
				dealAmount = nominalValue                             // 成交金额 = 名义价值
				calculationNote = "基于原始数量估算"
			}
		}
	} else {
		calculationNote = "暂无成交数据"
	}

	// 构建响应数据
	response := gin.H{
		"id":                order.ID,
		"user_id":           order.UserID,
		"exchange":          order.Exchange,
		"testnet":           order.Testnet,
		"symbol":            order.Symbol,
		"side":              order.Side,
		"order_type":        order.OrderType,
		"quantity":          order.Quantity,
		"adjusted_quantity": order.AdjustedQuantity,
		"price":             order.Price,
		"leverage":          order.Leverage,
		"reduce_only":       order.ReduceOnly,
		"strategy_id":       order.StrategyID,
		"bracket_enabled":   order.BracketEnabled,
		"tp_percent":        order.TPPercent,       // 用户设置的止盈百分比
		"sl_percent":        order.SLPercent,       // 用户设置的止损百分比
		"actual_tp_percent": order.ActualTPPercent, // 实际使用的止盈百分比
		"actual_sl_percent": order.ActualSLPercent, // 实际使用的止损百分比
		"tp_price":          order.TPPrice,
		"sl_price":          order.SLPrice,
		"working_type":      order.WorkingType,
		"trigger_time":      order.TriggerTime,
		"status":            order.Status,
		"result":            order.Result,
		"created_at":        order.CreatedAt,
		"updated_at":        order.UpdatedAt,
		// 新增的订单跟踪信息
		"client_order_id":   order.ClientOrderId,
		"exchange_order_id": order.ExchangeOrderId,
		"executed_quantity": order.ExecutedQty,
		"avg_price":         order.AvgPrice,
		// 新增的金额计算信息
		"nominal_value":    nominalValue,    // 名义价值（合约总价值）
		"margin_amount":    marginAmount,    // 保证金金额
		"deal_amount":      dealAmount,      // 成交金额（名义价值）
		"calculation_note": calculationNote, // 计算说明
		// 新增的操作类型信息
		"operation_type":  operationType.Type,
		"operation_desc":  operationType.Description,
		"operation_class": operationType.Class,
		// 关联订单信息
		"related_orders": relatedOrders,
	}

	// 添加订单状态信息
	if orderStatus != nil {
		response["order_status"] = orderStatus
	}

	// 添加利润信息
	if profitInfo != nil {
		response["profit_info"] = profitInfo
	}

	c.JSON(http.StatusOK, response)
}

// queryOrderStatus 查询订单在交易所的状态
func (s *Server) queryOrderStatus(order *pdb.ScheduledOrder) gin.H {
	if order.ClientOrderId == "" {
		return gin.H{
			"error": "订单缺少clientOrderId，无法查询状态",
		}
	}

	// 只查询币安期货订单
	if strings.ToLower(order.Exchange) != "binance_futures" {
		return gin.H{
			"error": "只支持查询币安期货订单状态",
		}
	}

	// 使用配置的环境设置，而不是订单的Testnet字段
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	orderStatus, err := client.QueryOrder(order.Symbol, order.ClientOrderId)
	if err != nil {
		log.Printf("[订单状态查询] 查询失败 %s: %v", order.ClientOrderId, err)
		return gin.H{
			"error": fmt.Sprintf("查询订单状态失败: %v", err),
		}
	}

	// 返回查询结果
	result := gin.H{
		"client_order_id": orderStatus.ClientOrderId,
		"order_id":        strconv.FormatInt(orderStatus.OrderId, 10),
		"status":          orderStatus.Status,
		"executed_qty":    orderStatus.ExecutedQty,
		"avg_price":       orderStatus.AvgPrice,
		"side":            orderStatus.Side,
		"symbol":          orderStatus.Symbol,
	}

	// 如果查询到的状态比数据库中的状态更新，则更新数据库
	shouldUpdate := false
	updateData := make(map[string]interface{})

	if order.ExchangeOrderId == "" && orderStatus.OrderId > 0 {
		updateData["exchange_order_id"] = strconv.FormatInt(orderStatus.OrderId, 10)
		shouldUpdate = true
	}

	if order.ExecutedQty == "" && orderStatus.ExecutedQty != "" && orderStatus.ExecutedQty != "0" {
		updateData["executed_quantity"] = orderStatus.ExecutedQty
		shouldUpdate = true
	}

	if order.AvgPrice == "" && orderStatus.AvgPrice != "" && orderStatus.AvgPrice != "0" {
		updateData["avg_price"] = orderStatus.AvgPrice
		shouldUpdate = true
	}

	// 如果订单已经完全成交，更新状态
	if orderStatus.Status == "FILLED" && order.Status != "filled" {
		updateData["status"] = "filled"
		shouldUpdate = true
	} else if orderStatus.Status == "CANCELED" && order.Status != "canceled" {
		updateData["status"] = "canceled"
		shouldUpdate = true
	} else if orderStatus.Status == "REJECTED" && order.Status != "failed" {
		updateData["status"] = "failed"
		shouldUpdate = true
	}

	if shouldUpdate {
		err := s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Updates(updateData).Error
		if err != nil {
			log.Printf("[订单状态更新] 更新失败 %d: %v", order.ID, err)
		} else {
			log.Printf("[订单状态更新] 成功更新订单 %d: %v", order.ID, updateData)
		}
	}

	return result
}

// queryAccountPositions 查询账户持仓信息
func (s *Server) queryAccountPositions(order *pdb.ScheduledOrder) gin.H {
	if order.Exchange != "binance_futures" {
		return gin.H{
			"error": "只支持查询币安期货持仓信息",
		}
	}

	// 使用配置的环境设置，而不是订单的Testnet字段
	// 这样可以确保所有查询都使用统一的配置环境，避免混淆
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)

	positions, err := client.GetPositions()
	if err != nil {
		log.Printf("[持仓查询] 查询失败 %s: %v", order.Symbol, err)
		return gin.H{
			"error": fmt.Sprintf("查询持仓信息失败: %v", err),
		}
	}

	// 查找当前交易对的持仓
	for _, pos := range positions {
		if pos.Symbol == order.Symbol {
			return gin.H{
				"symbol":            pos.Symbol,
				"position_amt":      pos.PositionAmt,
				"entry_price":       pos.EntryPrice,
				"mark_price":        pos.MarkPrice,
				"unrealized_profit": pos.UnRealizedProfit,
				"leverage":          pos.Leverage,
				"liquidation_price": pos.LiquidationPrice,
				"margin_type":       pos.MarginType,
				"position_side":     pos.PositionSide,
				"update_time":       pos.UpdateTime,
			}
		}
	}

	// 如果没有找到持仓，返回空持仓信息
	return gin.H{
		"symbol":       order.Symbol,
		"position_amt": "0",
		"has_position": false,
	}
}

// hasMatchingCloseOrder 检查是否存在匹配的平仓订单
func (s *Server) hasMatchingCloseOrder(order *pdb.ScheduledOrder) bool {
	// 只检查开仓订单
	if order.ReduceOnly {
		return false
	}

	// 检查是否有关联的平仓订单，并且这些订单已经成交
	if order.CloseOrderIds != "" {
		closeOrderIds := strings.Split(order.CloseOrderIds, ",")
		for _, idStr := range closeOrderIds {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
				closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
				if err == nil && closeOrder.Status == "filled" {
					// 找到已成交的平仓订单
					return true
				}
			}
		}
	}

	return false
}

// isOrderFinished 检查订单是否已结束（开仓订单已被平仓）
func (s *Server) isOrderFinished(order *pdb.ScheduledOrder) bool {
	// 只检查开仓订单
	if order.ReduceOnly {
		return false
	}

	// 检查是否有关联的平仓订单，并且这些订单已经成交
	if order.CloseOrderIds != "" {
		closeOrderIds := strings.Split(order.CloseOrderIds, ",")
		for _, idStr := range closeOrderIds {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
				closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
				if err == nil && closeOrder.Status == "filled" {
					// 找到已成交的平仓订单，说明订单已结束
					return true
				}
			}
		}
	}

	return false
}

// calculateOrderProfit 计算订单的利润信息
func (s *Server) calculateOrderProfit(order *pdb.ScheduledOrder, orderStatus gin.H, accountPositions gin.H) gin.H {
	var quantity float64
	var entryPrice float64
	var err error

	// 获取执行数量
	if orderStatus != nil && orderStatus["executed_qty"] != nil {
		executedQtyInterface := orderStatus["executed_qty"]
		if executedQtyStr, ok := executedQtyInterface.(string); ok && executedQtyStr != "" && executedQtyStr != "0" {
			// 使用交易所返回的实际执行数量
			quantity, err = strconv.ParseFloat(executedQtyStr, 64)
			if err != nil || quantity <= 0 {
				return gin.H{
					"error": "无法解析执行数量",
				}
			}
		}
	}

	// 如果没有从交易所获取到执行数量，使用计划数量
	if quantity == 0 && order.AdjustedQuantity != "" {
		quantity, err = strconv.ParseFloat(order.AdjustedQuantity, 64)
		if err != nil || quantity <= 0 {
			return nil
		}
	}

	if quantity == 0 {
		return nil
	}

	// 获取开仓价格
	if orderStatus != nil && orderStatus["avg_price"] != nil && orderStatus["avg_price"] != "" {
		// 使用交易所返回的平均成交价
		avgPriceStr := orderStatus["avg_price"].(string)
		entryPrice, err = strconv.ParseFloat(avgPriceStr, 64)
		if err != nil || entryPrice <= 0 {
			return gin.H{
				"error": "无法解析平均成交价",
			}
		}
	} else {
		// 估算开仓价格（对于未成交订单）
		ctx := context.Background()
		currentPrice, err := s.getCurrentPrice(ctx, order.Symbol, "futures")
		if err != nil {
			return gin.H{
				"error": "无法获取当前价格",
			}
		}
		entryPrice = currentPrice // 简单估算
	}

	// 获取当前价格
	ctx := context.Background()
	currentPrice, err := s.getCurrentPrice(ctx, order.Symbol, "futures")
	if err != nil {
		log.Printf("[利润计算] 获取当前价格失败 %s: %v", order.Symbol, err)
		return gin.H{
			"error": "无法获取当前价格",
		}
	}

	// 计算持仓价值
	positionValue := quantity * currentPrice

	// 计算利润
	var unrealizedPnL float64
	var totalPnL float64
	var realizedPnL float64

	// 计算已实现利润（先计算，因为这会影响未实现利润的计算）
	if order.ReduceOnly {
		// 平仓订单：计算本次平仓相对于开仓价格的利润
		if order.ParentOrderId > 0 {
			parentOrder, err := s.db.GetScheduledOrderByID(order.ParentOrderId)
			if err == nil && parentOrder.Status == "filled" && parentOrder.AvgPrice != "" {
				if parentEntryPrice, err := strconv.ParseFloat(parentOrder.AvgPrice, 64); err == nil {
					// 计算平仓订单的利润：相对于开仓价格的收益
					if order.Side == "BUY" {
						// 买入平仓（空头平仓）：利润 = 开仓价格 - 平仓价格
						realizedPnL = (parentEntryPrice - entryPrice) * quantity
					} else {
						// 卖出平仓（多头平仓）：利润 = 平仓价格 - 开仓价格
						realizedPnL = (entryPrice - parentEntryPrice) * quantity
					}
				}
			}
		}
		// 平仓订单已完成，未实现利润为0
		unrealizedPnL = 0
	} else {
		// 开仓订单：计算所有平仓订单的已实现利润
		if order.CloseOrderIds != "" {
			realizedPnL = s.calculateRealizedPnL(order, entryPrice)

			// 检查是否已被完全平仓
			totalClosedQty := s.getTotalClosedQuantity(order)
			if totalClosedQty >= quantity {
				// 已被完全平仓，未实现利润为0
				unrealizedPnL = 0
			} else {
				// 部分持仓：计算剩余持仓的未实现利润
				remainingQty := quantity - totalClosedQty
				if order.Side == "BUY" {
					// 多头持仓
					unrealizedPnL = remainingQty * (currentPrice - entryPrice)
				} else {
					// 空头持仓
					unrealizedPnL = remainingQty * (entryPrice - currentPrice)
				}
			}
		} else {
			// 没有平仓订单：计算全部持仓的未实现利润
			if order.Side == "BUY" {
				// 多头持仓
				unrealizedPnL = quantity * (currentPrice - entryPrice)
			} else {
				// 空头持仓
				unrealizedPnL = quantity * (entryPrice - currentPrice)
			}
		}
	}

	// 总利润 = 未实现利润 + 已实现利润
	totalPnL = unrealizedPnL + realizedPnL

	// 计算利润率
	var pnlPercentage float64
	if entryPrice > 0 {
		pnlPercentage = (totalPnL / (quantity * entryPrice)) * 100
	}

	// 确定持仓类型
	positionType := "short"
	if order.Side == "BUY" {
		positionType = "long"
	}

	// 确定数据来源
	dataSource := "estimated"
	if orderStatus != nil {
		dataSource = "exchange"
	}

	// 检查实际持仓状态
	actualPositionStatus := "unknown"
	var actualPositionAmt string

	if accountPositions != nil && accountPositions["error"] == nil {
		if positionAmt, ok := accountPositions["position_amt"].(string); ok {
			actualPositionAmt = positionAmt
			if positionAmt == "0" || positionAmt == "0.0" || positionAmt == "" {
				// 如果是平仓订单且账户无持仓，则已平仓
				if order.ReduceOnly {
					actualPositionStatus = "closed"
				} else {
					actualPositionStatus = "no_position"
				}
			} else {
				// 有持仓
				if order.ReduceOnly {
					// 平仓订单但仍有持仓，说明平仓未完成
					actualPositionStatus = "partially_closed"
				} else {
					// 开仓订单且有持仓，检查是否存在对应的平仓订单
					if hasMatchingCloseOrder := s.hasMatchingCloseOrder(order); hasMatchingCloseOrder {
						// 如果存在对应的平仓订单，说明用户已经手动平仓
						actualPositionStatus = "closed"
					} else {
						// 开仓订单且有持仓，说明开仓成功
						actualPositionStatus = "position_held"
					}
				}
			}

			// 如果有真实的持仓信息，优先使用
			if entryPriceStr, ok := accountPositions["entry_price"].(string); ok && entryPriceStr != "" {
				if realEntryPrice, err := strconv.ParseFloat(entryPriceStr, 64); err == nil {
					entryPrice = realEntryPrice
					dataSource = "account" // 账户持仓数据
				}
			}
		}
	}

	// 重新计算利润（使用更准确的入场价格）
	if entryPrice > 0 {
		// 对于已平仓的交易，未实现利润为0
		if actualPositionStatus == "closed" {
			unrealizedPnL = 0
		} else {
			// 计算未实现利润
			if order.Side == "BUY" {
				unrealizedPnL = quantity * (currentPrice - entryPrice)
			} else {
				unrealizedPnL = quantity * (entryPrice - currentPrice)
			}
		}

		// 重新计算总利润：已实现利润 + 未实现利润
		if !order.ReduceOnly && order.CloseOrderIds != "" {
			// 对于有平仓订单的开仓订单，需要重新计算已实现利润
			realizedPnL = s.calculateRealizedPnL(order, entryPrice)
		}
		totalPnL = unrealizedPnL + realizedPnL
		pnlPercentage = (totalPnL / (quantity * entryPrice)) * 100
	}

	// 计算名义价值和保证金
	nominalValue := quantity * entryPrice                  // 名义价值 = 数量 × 入场价格
	marginAmount := nominalValue / float64(order.Leverage) // 保证金 = 名义价值 ÷ 杠杆
	dealAmount := quantity * entryPrice                    // 成交金额（名义价值）

	// 构建利润信息
	profitInfo := gin.H{
		"current_price":          currentPrice,
		"entry_price":            entryPrice,
		"quantity":               quantity,
		"position_value":         positionValue,  // 当前持仓价值
		"nominal_value":          nominalValue,   // 名义价值（合约总价值）
		"margin_amount":          marginAmount,   // 保证金金额
		"deal_amount":            dealAmount,     // 成交金额（名义价值）
		"leverage":               order.Leverage, // 杠杆倍数
		"unrealized_pnl":         unrealizedPnL,
		"realized_pnl":           realizedPnL,
		"total_pnl":              totalPnL,
		"pnl_percentage":         pnlPercentage,
		"position_type":          positionType,
		"data_source":            dataSource,
		"actual_position_status": actualPositionStatus,
		"actual_position_amt":    actualPositionAmt,
	}

	// 添加状态说明
	if order.Status == "filled" && orderStatus != nil {
		profitInfo["note"] = "基于交易所成交数据的准确利润计算"
	} else if orderStatus != nil && orderStatus["executed_qty"] != nil {
		profitInfo["note"] = "基于部分成交数据的利润估算"
	} else {
		profitInfo["note"] = "基于当前价格的利润估算，实际利润以交易所成交记录为准"
	}

	return profitInfo
}

// POST /orders/schedule/:id/close-position 手动平仓
func (s *Server) ClosePosition(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	idStr := c.Param("id")

	// 解析 ID
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		s.ValidationError(c, "id", "无效的订单ID")
		return
	}

	// 获取订单（检查是否存在和权限）
	order, err := s.db.GetScheduledOrderByID(id)
	if err != nil {
		s.NotFound(c, "订单不存在")
		return
	}

	// 检查权限
	if order.UserID != uid {
		s.Forbidden(c, "无权操作此订单")
		return
	}

	// 检查订单是否可以平仓
	if order.ReduceOnly {
		s.BadRequest(c, "这是一个平仓订单，无法再次平仓", nil)
		return
	}

	if order.Status != "filled" && order.Status != "completed" {
		s.BadRequest(c, "只有已成交的开仓订单才能手动平仓", nil)
		return
	}

	// 获取当前持仓信息
	// 使用配置的环境设置，而不是订单的Testnet字段
	useTestnet := s.cfg.Exchange.Binance.IsTestnet
	client := bf.New(useTestnet, s.cfg.Exchange.Binance.APIKey, s.cfg.Exchange.Binance.SecretKey)
	positions, err := client.GetPositions()
	if err != nil {
		s.DatabaseError(c, "获取持仓信息失败", err)
		return
	}

	// 查找对应交易对的持仓
	var positionAmount float64
	for _, pos := range positions {
		if pos.Symbol == order.Symbol {
			positionAmount, _ = strconv.ParseFloat(pos.PositionAmt, 64)
			break
		}
	}

	if positionAmount == 0 {
		s.BadRequest(c, "该交易对当前无持仓", nil)
		return
	}

	// 创建平仓订单
	closeSide := "SELL"
	if positionAmount < 0 {
		closeSide = "BUY"
		positionAmount = -positionAmount // 取绝对值
	}

	closeOrder := &pdb.ScheduledOrder{
		UserID:         uid,
		Exchange:       order.Exchange,
		Testnet:        order.Testnet,
		Symbol:         order.Symbol,
		Side:           closeSide,
		OrderType:      "MARKET",
		Quantity:       strconv.FormatFloat(positionAmount, 'f', -1, 64),
		Price:          "",
		Leverage:       order.Leverage,
		ReduceOnly:     true,
		BracketEnabled: false,
		TPPercent:      0,
		SLPercent:      0,
		TPPrice:        "",
		SLPrice:        "",
		WorkingType:    "MARK_PRICE",
		TriggerTime:    time.Now().UTC(),
		Status:         "pending",
		ParentOrderId:  order.ID, // 关联到开仓订单
	}

	if err := s.db.CreateScheduledOrder(closeOrder); err != nil {
		s.DatabaseError(c, "创建平仓订单失败", err)
		return
	}

	// 更新原订单的关联字段
	if err := s.updateOrderAssociations(order, closeOrder.ID); err != nil {
		log.Printf("[平仓关联] 更新订单关联失败 %d: %v", order.ID, err)
		// 不影响平仓操作，继续执行
	}

	log.Printf("[平仓] 用户%d为订单%d创建平仓订单%d: %s %s %s",
		uid, id, closeOrder.ID, order.Symbol, closeSide, strconv.FormatFloat(positionAmount, 'f', -1, 64))

	c.JSON(http.StatusOK, gin.H{
		"close_order_id": closeOrder.ID,
		"symbol":         order.Symbol,
		"side":           closeSide,
		"quantity":       positionAmount,
		"message":        "平仓订单已创建",
	})
}

// updateOrderAssociations 更新订单关联关系
func (s *Server) updateOrderAssociations(parentOrder *pdb.ScheduledOrder, closeOrderID uint) error {
	// 解析现有的close_order_ids
	var closeOrderIds []uint
	if parentOrder.CloseOrderIds != "" {
		// 这里应该解析JSON，但为了简化，我们暂时使用逗号分隔的字符串
		// 实际实现中应该使用JSON
		parts := strings.Split(parentOrder.CloseOrderIds, ",")
		for _, part := range parts {
			if id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32); err == nil {
				closeOrderIds = append(closeOrderIds, uint(id))
			}
		}
	}

	// 添加新的close_order_id
	closeOrderIds = append(closeOrderIds, closeOrderID)

	// 更新为逗号分隔的字符串
	var idsStr []string
	for _, id := range closeOrderIds {
		idsStr = append(idsStr, strconv.FormatUint(uint64(id), 10))
	}

	// 更新数据库
	return s.db.DB().Model(&pdb.ScheduledOrder{}).Where("id = ?", parentOrder.ID).Update("close_order_ids", strings.Join(idsStr, ",")).Error
}

// queryRelatedOrders 查询关联订单信息
func (s *Server) queryRelatedOrders(order *pdb.ScheduledOrder) gin.H {
	result := gin.H{
		"parent_order": nil,
		"close_orders": []gin.H{},
	}

	// 如果是平仓订单，查询父订单（开仓订单）
	if order.ParentOrderId > 0 {
		parentOrder, err := s.db.GetScheduledOrderByID(order.ParentOrderId)
		if err == nil && parentOrder.UserID == order.UserID {
			operationType := getFuturesOperationType(parentOrder.Side, parentOrder.ReduceOnly)
			result["parent_order"] = gin.H{
				"id":             parentOrder.ID,
				"symbol":         parentOrder.Symbol,
				"side":           parentOrder.Side,
				"status":         parentOrder.Status,
				"operation_type": operationType.Type,
				"trigger_time":   parentOrder.TriggerTime,
				"executed_qty":   parentOrder.ExecutedQty,
				"avg_price":      parentOrder.AvgPrice,
			}
		}
	}

	// 如果是开仓订单，查询关联的平仓订单
	if !order.ReduceOnly && order.CloseOrderIds != "" {
		closeOrderIds := strings.Split(order.CloseOrderIds, ",")
		var closeOrders []gin.H

		for _, idStr := range closeOrderIds {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
				closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
				if err == nil && closeOrder.UserID == order.UserID {
					operationType := getFuturesOperationType(closeOrder.Side, closeOrder.ReduceOnly)
					closeOrders = append(closeOrders, gin.H{
						"id":             closeOrder.ID,
						"symbol":         closeOrder.Symbol,
						"side":           closeOrder.Side,
						"status":         closeOrder.Status,
						"operation_type": operationType.Type,
						"trigger_time":   closeOrder.TriggerTime,
						"executed_qty":   closeOrder.ExecutedQty,
						"avg_price":      closeOrder.AvgPrice,
					})
				}
			}
		}

		result["close_orders"] = closeOrders
	}

	// 查询Bracket订单的TP/SL订单
	bracketOrders := s.queryBracketOrders(order)
	result["bracket_orders"] = bracketOrders

	return result
}

// queryBracketOrders 查询Bracket订单的TP/SL订单信息
func (s *Server) queryBracketOrders(order *pdb.ScheduledOrder) gin.H {
	result := gin.H{
		"tp_order":    nil,
		"sl_order":    nil,
		"has_bracket": false,
	}

	// 查询BracketLink记录
	var bracketLink pdb.BracketLink
	err := s.db.DB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracketLink).Error
	if err != nil {
		// 如果没找到，尝试作为TP/SL订单查询
		err = s.db.DB().Where("tp_client_id = ? OR sl_client_id = ?", order.ClientOrderId, order.ClientOrderId).First(&bracketLink).Error
		if err != nil {
			return result
		}
	}

	result["has_bracket"] = true
	result["bracket_status"] = bracketLink.Status
	result["bracket_group_id"] = bracketLink.GroupID

	// 查询TP订单
	if bracketLink.TPClientID != "" {
		var tpOrder pdb.ScheduledOrder
		err := s.db.DB().Where("client_order_id = ?", bracketLink.TPClientID).First(&tpOrder).Error
		if err == nil {
			operationType := getFuturesOperationType(tpOrder.Side, tpOrder.ReduceOnly)
			result["tp_order"] = gin.H{
				"id":             tpOrder.ID,
				"symbol":         tpOrder.Symbol,
				"side":           tpOrder.Side,
				"status":         tpOrder.Status,
				"operation_type": operationType.Type,
				"trigger_time":   tpOrder.TriggerTime,
				"executed_qty":   tpOrder.ExecutedQty,
				"avg_price":      tpOrder.AvgPrice,
				"trigger_price":  tpOrder.TPPrice,
			}
		}
	}

	// 查询SL订单
	if bracketLink.SLClientID != "" {
		var slOrder pdb.ScheduledOrder
		err := s.db.DB().Where("client_order_id = ?", bracketLink.SLClientID).First(&slOrder).Error
		if err == nil {
			operationType := getFuturesOperationType(slOrder.Side, slOrder.ReduceOnly)
			result["sl_order"] = gin.H{
				"id":             slOrder.ID,
				"symbol":         slOrder.Symbol,
				"side":           slOrder.Side,
				"status":         slOrder.Status,
				"operation_type": operationType.Type,
				"trigger_time":   slOrder.TriggerTime,
				"executed_qty":   slOrder.ExecutedQty,
				"avg_price":      slOrder.AvgPrice,
				"trigger_price":  slOrder.SLPrice,
			}
		}
	}

	return result
}

// getRelatedOrdersSummary 获取关联订单的摘要信息（用于订单列表）
func (s *Server) getRelatedOrdersSummary(order *pdb.ScheduledOrder) gin.H {
	result := gin.H{
		"has_parent":   false,
		"has_close":    false,
		"parent_count": 0,
		"close_count":  0,
		"trade_chain":  "",
	}

	// 如果是平仓订单，检查是否有父订单
	if order.ParentOrderId > 0 {
		parentOrder, err := s.db.GetScheduledOrderByID(order.ParentOrderId)
		if err == nil && parentOrder.UserID == order.UserID {
			result["has_parent"] = true
			result["parent_count"] = 1
			result["parent_id"] = parentOrder.ID
			result["parent_status"] = parentOrder.Status
			result["parent_operation"] = getFuturesOperationType(parentOrder.Side, parentOrder.ReduceOnly).Type
			result["trade_chain"] = fmt.Sprintf("交易链 #%d", parentOrder.ID)
		} else if err != nil {
			// 父订单已被删除，清理无效引用
			result["has_parent"] = false
			result["parent_count"] = 0
			result["parent_deleted"] = true
			// 可选：自动清理无效的ParentOrderId引用
			// go s.db.DB().Model(order).Update("parent_order_id", 0)
		}
	}

	// 如果是开仓订单，检查是否有平仓订单和平仓订单
	if !order.ReduceOnly {
		var closeOrders []gin.H
		var scalingOrdersIds []uint

		// 检查CloseOrderIds中的平仓订单
		if order.CloseOrderIds != "" {
			// 处理逗号分隔的格式，如"1450"或"1450,1451"
			closeOrderIdsStr := strings.TrimSpace(order.CloseOrderIds)

			// 按逗号分割
			closeOrderIds := strings.Split(closeOrderIdsStr, ",")
			for _, idStr := range closeOrderIds {
				if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
					closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
					if err == nil && closeOrder.UserID == order.UserID {
						// 为平仓订单添加操作类型信息
						closeOpType := getFuturesOperationType(closeOrder.Side, closeOrder.ReduceOnly)
						closeOrders = append(closeOrders, gin.H{
							"id":                closeOrder.ID,
							"symbol":            closeOrder.Symbol,
							"side":              closeOrder.Side,
							"status":            closeOrder.Status,
							"operation_type":    closeOpType.Type,
							"operation_desc":    closeOpType.Description,
							"operation_class":   closeOpType.Class,
							"trigger_time":      closeOrder.TriggerTime,
							"executed_qty":      closeOrder.ExecutedQty,
							"avg_price":         closeOrder.AvgPrice,
							"quantity":          closeOrder.Quantity,
							"adjusted_quantity": closeOrder.AdjustedQuantity,
							"price":             closeOrder.Price,
						})
					}
					// 如果子订单已被删除，跳过（不记录错误）
				}
			}
		}

		// 检查通过ParentOrderId关联的子订单（包括加仓订单）
		var scalingOrders []pdb.ScheduledOrder
		s.db.DB().Where("user_id = ? AND parent_order_id = ? AND reduce_only = false",
			order.UserID, order.ID).Find(&scalingOrders)

		for _, scalingOrder := range scalingOrders {
			scalingOrdersIds = append(scalingOrdersIds, scalingOrder.ID)
		}

		// 设置平仓订单信息
		if len(closeOrders) > 0 {
			result["has_close"] = true
			result["close_count"] = len(closeOrders)
			result["close_orders"] = closeOrders
		}

		// 设置加仓订单信息
		if len(scalingOrdersIds) > 0 {
			result["has_scaling"] = true
			result["scaling_count"] = len(scalingOrdersIds)
			result["scaling_ids"] = scalingOrdersIds
			log.Printf("[RelatedOrders] 订单 %d 找到 %d 个加仓订单: %v",
				order.ID, len(scalingOrdersIds), scalingOrdersIds)
		}

		// 设置总的子订单信息
		totalChildren := len(closeOrders) + len(scalingOrdersIds)
		if totalChildren > 0 {
			result["has_children"] = true
			result["children_count"] = totalChildren
			result["trade_chain"] = fmt.Sprintf("交易链 #%d", order.ID)
		}

		// 检查Bracket订单（TP/SL订单）
		var bracketLink pdb.BracketLink
		err := s.db.DB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracketLink).Error
		if err == nil {
			// 找到了Bracket链接，检查TP和SL订单是否存在
			bracketInfo := gin.H{
				"has_bracket": true,
				"bracket_id":  bracketLink.ID,
				"group_id":    bracketLink.GroupID,
				"status":      bracketLink.Status,
			}

			tpOrder := (*pdb.ScheduledOrder)(nil)
			slOrder := (*pdb.ScheduledOrder)(nil)

			// 检查TP订单
			if bracketLink.TPClientID != "" {
				var tpOrderTemp pdb.ScheduledOrder
				err := s.db.DB().Where("client_order_id = ?", bracketLink.TPClientID).First(&tpOrderTemp).Error
				if err == nil {
					tpOrder = &tpOrderTemp
				}
			}

			// 检查SL订单
			if bracketLink.SLClientID != "" {
				var slOrderTemp pdb.ScheduledOrder
				err := s.db.DB().Where("client_order_id = ?", bracketLink.SLClientID).First(&slOrderTemp).Error
				if err == nil {
					slOrder = &slOrderTemp
				}
			}

			// 统计Bracket订单数量
			bracketCount := 0
			if tpOrder != nil {
				bracketCount++
			}
			if slOrder != nil {
				bracketCount++
			}

			if bracketCount > 0 {
				bracketInfo["bracket_count"] = bracketCount
				result["has_bracket"] = true
				result["bracket_count"] = bracketCount
				result["bracket_info"] = bracketInfo

				// 如果还没有交易链标识，创建一个
				if result["trade_chain"] == "" {
					result["trade_chain"] = fmt.Sprintf("交易链 #%d", order.ID)
				}
			}
		}
	}

	return result
}

// cleanupOrderReferences 清理订单引用关系
func (s *Server) cleanupOrderReferences(userID, orderID uint) error {
	// 1. 清理子订单对该订单的引用（如果该订单是父订单）
	err := s.db.DB().Model(&pdb.ScheduledOrder{}).
		Where("user_id = ? AND parent_order_id = ?", userID, orderID).
		Update("parent_order_id", 0).Error
	if err != nil {
		log.Printf("[CleanupReferences] 清理父订单引用失败: %v", err)
	}

	// 2. 从其他订单的close_order_ids中移除该订单ID
	// 查询所有包含该订单ID的close_order_ids字段
	var ordersWithCloseRefs []pdb.ScheduledOrder
	err = s.db.DB().Where("user_id = ? AND close_order_ids LIKE ?", userID, "%"+fmt.Sprintf("%d", orderID)+"%").
		Find(&ordersWithCloseRefs).Error
	if err != nil {
		log.Printf("[CleanupReferences] 查询包含close_order_ids的订单失败: %v", err)
		return err
	}

	// 清理每个订单的close_order_ids
	for _, order := range ordersWithCloseRefs {
		closeOrderIds := strings.Split(order.CloseOrderIds, ",")
		var cleanedIds []string

		for _, idStr := range closeOrderIds {
			if id, parseErr := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); parseErr == nil {
				if uint(id) != orderID {
					cleanedIds = append(cleanedIds, strings.TrimSpace(idStr))
				}
			}
		}

		newCloseOrderIds := strings.Join(cleanedIds, ",")
		err = s.db.DB().Model(&order).Update("close_order_ids", newCloseOrderIds).Error
		if err != nil {
			log.Printf("[CleanupReferences] 更新close_order_ids失败: %v", err)
		}
	}

	// 3. 清理 BracketLink 记录
	// 获取要删除的订单的 client_order_id
	var orderToDelete pdb.ScheduledOrder
	err = s.db.DB().Where("id = ?", orderID).First(&orderToDelete).Error
	if err != nil {
		log.Printf("[CleanupReferences] 获取要删除的订单失败: %v", err)
		return err
	}

	// 检查是否有相关的 BracketLink 记录
	var bracketLink pdb.BracketLink
	err = s.db.DB().Where("entry_client_id = ? OR tp_client_id = ? OR sl_client_id = ?",
		orderToDelete.ClientOrderId, orderToDelete.ClientOrderId, orderToDelete.ClientOrderId).
		First(&bracketLink).Error

	if err == nil {
		// 找到了相关的 BracketLink 记录
		log.Printf("[CleanupReferences] 找到相关的BracketLink记录 (ID:%d, GroupID:%s)，需要清理",
			bracketLink.ID, bracketLink.GroupID)

		// 根据删除的订单类型更新 BracketLink
		updates := make(map[string]interface{})

		if bracketLink.EntryClientID == orderToDelete.ClientOrderId {
			// 删除的是开仓订单，整个BracketLink失效
			log.Printf("[CleanupReferences] 删除开仓订单，标记BracketLink为无效状态")
			updates["status"] = "orphaned"
		} else if bracketLink.TPClientID == orderToDelete.ClientOrderId {
			// 删除的是止盈订单，清空TPClientID
			log.Printf("[CleanupReferences] 删除止盈订单，清空TPClientID")
			updates["tp_client_id"] = ""
		} else if bracketLink.SLClientID == orderToDelete.ClientOrderId {
			// 删除的是止损订单，清空SLClientID
			log.Printf("[CleanupReferences] 删除止损订单，清空SLClientID")
			updates["sl_client_id"] = ""
		}

		// 检查是否所有订单都被删除了
		if updates["tp_client_id"] == "" && updates["sl_client_id"] == "" && bracketLink.EntryClientID != "" {
			// 如果只剩下开仓订单，且开仓订单还存在，则保持BracketLink有效
			log.Printf("[CleanupReferences] BracketLink仍有开仓订单，保持有效状态")
		} else if bracketLink.TPClientID == "" && bracketLink.SLClientID == "" && bracketLink.EntryClientID == "" {
			// 如果所有订单都被删除了，标记为orphaned
			log.Printf("[CleanupReferences] 所有相关订单都被删除，标记BracketLink为orphaned")
			updates["status"] = "orphaned"
		}

		// 应用更新
		if len(updates) > 0 {
			err = s.db.DB().Model(&bracketLink).Updates(updates).Error
			if err != nil {
				log.Printf("[CleanupReferences] 更新BracketLink失败: %v", err)
			} else {
				log.Printf("[CleanupReferences] BracketLink更新成功")
			}
		}
	}

	return nil
}

// calculateRealizedPnL 计算已实现利润（基于关联的平仓订单）
func (s *Server) calculateRealizedPnL(order *pdb.ScheduledOrder, entryPrice float64) float64 {
	if order.CloseOrderIds == "" {
		return 0
	}

	var totalRealizedPnL float64
	closeOrderIds := strings.Split(order.CloseOrderIds, ",")

	for _, idStr := range closeOrderIds {
		if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
			closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
			if err != nil || closeOrder.Status != "filled" || closeOrder.AvgPrice == "" || closeOrder.ExecutedQty == "" {
				continue
			}

			// 获取平仓价格和数量
			closePrice, err1 := strconv.ParseFloat(closeOrder.AvgPrice, 64)
			closeQty, err2 := strconv.ParseFloat(closeOrder.ExecutedQty, 64)
			if err1 != nil || err2 != nil || closeQty <= 0 {
				continue
			}

			// 计算平仓订单的利润
			var closePnL float64
			if order.Side == "BUY" {
				// 多头开仓，对应的平仓是SELL，利润 = (平仓价 - 开仓价) * 数量
				closePnL = (closePrice - entryPrice) * closeQty
			} else {
				// 空头开仓，对应的平仓是BUY，利润 = (开仓价 - 平仓价) * 数量
				closePnL = (entryPrice - closePrice) * closeQty
			}

			totalRealizedPnL += closePnL
		}
	}

	return totalRealizedPnL
}

// getTotalClosedQuantity 计算已平仓的总数量
func (s *Server) getTotalClosedQuantity(order *pdb.ScheduledOrder) float64 {
	if order.CloseOrderIds == "" {
		return 0
	}

	var totalClosedQty float64
	closeOrderIds := strings.Split(order.CloseOrderIds, ",")

	for _, idStr := range closeOrderIds {
		if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
			closeOrder, err := s.db.GetScheduledOrderByID(uint(id))
			if err != nil || closeOrder.Status != "filled" || closeOrder.ExecutedQty == "" {
				continue
			}

			// 获取平仓数量
			closeQty, err := strconv.ParseFloat(closeOrder.ExecutedQty, 64)
			if err != nil || closeQty <= 0 {
				continue
			}

			totalClosedQty += closeQty
		}
	}

	return totalClosedQty
}

// DELETE /orders/schedule/:id
func (s *Server) DeleteScheduledOrder(c *gin.Context) {
	uidVal, _ := c.Get("uid")
	uid := uint(uidVal.(uint))
	idStr := c.Param("id")

	// 解析 ID
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		s.ValidationError(c, "id", "无效的订单ID")
		return
	}

	// 获取删除模式参数（默认为级联删除以保持向后兼容）
	cascadeStr := c.DefaultQuery("cascade", "true")
	cascadeDelete := cascadeStr == "true"

	// 获取要级联删除的平仓订单ID列表（如果前端指定了的话）
	closeOrderIdsStr := c.Query("closeOrderIds")
	var specifiedCloseOrderIds []uint
	if closeOrderIdsStr != "" {
		for _, idStr := range strings.Split(closeOrderIdsStr, ",") {
			if id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 32); err == nil {
				specifiedCloseOrderIds = append(specifiedCloseOrderIds, uint(id))
			}
		}
	}

	log.Printf("[DeleteOrder] 接收到的cascade参数: %s, 解析为: %v, 指定删除的平仓订单: %v",
		cascadeStr, cascadeDelete, specifiedCloseOrderIds)

	// 获取订单（检查是否存在和权限）
	order, err := s.db.GetScheduledOrderByID(id)
	if err != nil {
		s.NotFound(c, "订单不存在")
		return
	}

	// 检查权限
	if order.UserID != uid {
		s.Forbidden(c, "无权操作此订单")
		return
	}

	// 如果是开仓订单且启用级联删除，查找关联的平仓订单并级联删除
	var cascadeDeletedOrders []uint
	log.Printf("[DeleteOrder] 订单 %d: ReduceOnly=%v, CloseOrderIds='%s', cascadeDelete=%v",
		order.ID, order.ReduceOnly, order.CloseOrderIds, cascadeDelete)

	if !order.ReduceOnly && cascadeDelete {
		var ordersToCheck []uint

		// 优先使用前端指定的订单ID列表
		if len(specifiedCloseOrderIds) > 0 {
			ordersToCheck = specifiedCloseOrderIds
			log.Printf("[DeleteOrder] 使用前端指定的 %d 个平仓订单ID: %v", len(ordersToCheck), ordersToCheck)
		} else if order.CloseOrderIds != "" {
			// 回退到使用数据库中的关联关系
			closeOrderIds := strings.Split(order.CloseOrderIds, ",")
			log.Printf("[DeleteOrder] 解析到 %d 个关联订单ID: %v", len(closeOrderIds), closeOrderIds)

			for _, closeIdStr := range closeOrderIds {
				if closeId, err := strconv.ParseUint(strings.TrimSpace(closeIdStr), 10, 32); err == nil {
					ordersToCheck = append(ordersToCheck, uint(closeId))
				}
			}
		}

		// 检查并删除指定的订单
		for _, closeId := range ordersToCheck {
			closeOrder, err := s.db.GetScheduledOrderByID(closeId)
			if err == nil && closeOrder.UserID == uid {
				log.Printf("[CascadeDelete] 将删除平仓订单 %d (状态: %s)", closeId, closeOrder.Status)
				cascadeDeletedOrders = append(cascadeDeletedOrders, closeId)
			} else {
				log.Printf("[CascadeDelete] 无法获取平仓订单 %d: %v", closeId, err)
			}
		}
	}

	// 级联删除关联的平仓订单
	for _, cascadeId := range cascadeDeletedOrders {
		log.Printf("[CascadeDelete] 开始删除关联平仓订单 %d", cascadeId)

		// 为每个要删除的订单清理引用关系
		if err := s.cleanupOrderReferences(uid, cascadeId); err != nil {
			log.Printf("[CascadeDelete] 清理平仓订单 %d 引用关系失败: %v", cascadeId, err)
			// 继续删除，不因清理失败而停止
		}

		// 删除平仓订单
		if err := s.db.DeleteScheduledOrder(uid, cascadeId); err != nil {
			log.Printf("[CascadeDelete] 删除平仓订单 %d 失败: %v", cascadeId, err)
			// 继续处理其他订单
		} else {
			log.Printf("[CascadeDelete] 成功删除平仓订单 %d", cascadeId)
		}
	}

	// 如果是开仓订单且启用级联删除，查找并删除关联的Bracket TP/SL订单
	var bracketDeletedOrders []uint
	if !order.ReduceOnly && cascadeDelete {
		// 查询BracketLink记录
		var bracketLink pdb.BracketLink
		err := s.db.DB().Where("entry_client_id = ?", order.ClientOrderId).First(&bracketLink).Error
		if err == nil && bracketLink.Status != "orphaned" {
			log.Printf("[BracketCascadeDelete] 找到关联的Bracket订单，GroupID: %s", bracketLink.GroupID)

			// 删除TP订单
			if bracketLink.TPClientID != "" {
				var tpOrder pdb.ScheduledOrder
				err := s.db.DB().Where("client_order_id = ?", bracketLink.TPClientID).First(&tpOrder).Error
				if err == nil && tpOrder.UserID == uid {
					log.Printf("[BracketCascadeDelete] 将删除关联的止盈订单 %d (状态: %s)", tpOrder.ID, tpOrder.Status)
					bracketDeletedOrders = append(bracketDeletedOrders, tpOrder.ID)
				}
			}

			// 删除SL订单
			if bracketLink.SLClientID != "" {
				var slOrder pdb.ScheduledOrder
				err := s.db.DB().Where("client_order_id = ?", bracketLink.SLClientID).First(&slOrder).Error
				if err == nil && slOrder.UserID == uid {
					log.Printf("[BracketCascadeDelete] 将删除关联的止损订单 %d (状态: %s)", slOrder.ID, slOrder.Status)
					bracketDeletedOrders = append(bracketDeletedOrders, slOrder.ID)
				}
			}
		} else if err == nil {
			log.Printf("[BracketCascadeDelete] Bracket订单已被标记为orphaned，跳过删除TP/SL订单")
		}
	}

	// 级联删除关联的Bracket TP/SL订单
	for _, bracketId := range bracketDeletedOrders {
		log.Printf("[BracketCascadeDelete] 开始删除关联的Bracket订单 %d", bracketId)

		// 为每个要删除的订单清理引用关系
		if err := s.cleanupOrderReferences(uid, bracketId); err != nil {
			log.Printf("[BracketCascadeDelete] 清理Bracket订单 %d 引用关系失败: %v", bracketId, err)
			// 继续删除，不因清理失败而停止
		}

		// 删除Bracket订单
		if err := s.db.DeleteScheduledOrder(uid, bracketId); err != nil {
			log.Printf("[BracketCascadeDelete] 删除Bracket订单 %d 失败: %v", bracketId, err)
			// 继续处理其他订单
		} else {
			log.Printf("[BracketCascadeDelete] 成功删除Bracket订单 %d", bracketId)
		}
	}

	// 删除订单前，先清理相关的引用关系
	if err := s.cleanupOrderReferences(uid, id); err != nil {
		log.Printf("[DeleteOrder] 清理引用关系失败: %v", err)
		// 不阻止删除操作，继续执行
	}

	// 删除主订单
	if err := s.db.DeleteScheduledOrder(uid, id); err != nil {
		s.DatabaseError(c, "删除定时订单", err)
		return
	}

	// 构建响应消息
	message := "删除成功"
	cascadeDeletedCount := len(cascadeDeletedOrders)
	bracketDeletedCount := len(bracketDeletedOrders)
	totalCascadeCount := cascadeDeletedCount + bracketDeletedCount

	if cascadeDelete && totalCascadeCount > 0 {
		messageParts := []string{}
		if cascadeDeletedCount > 0 {
			messageParts = append(messageParts, fmt.Sprintf("%d 个关联的平仓订单", cascadeDeletedCount))
		}
		if bracketDeletedCount > 0 {
			messageParts = append(messageParts, fmt.Sprintf("%d 个关联的止盈止损订单", bracketDeletedCount))
		}
		message = fmt.Sprintf("删除成功 (同时删除了 %s)", strings.Join(messageParts, " 和 "))
	}

	log.Printf("[DeleteOrder] 成功删除订单 %d%s", id, func() string {
		if totalCascadeCount > 0 {
			parts := []string{}
			if cascadeDeletedCount > 0 {
				parts = append(parts, fmt.Sprintf(" 及 %d 个关联平仓订单", cascadeDeletedCount))
			}
			if bracketDeletedCount > 0 {
				parts = append(parts, fmt.Sprintf(" 及 %d 个关联止盈止损订单", bracketDeletedCount))
			}
			return strings.Join(parts, "")
		}
		return ""
	}())

	c.JSON(http.StatusOK, gin.H{
		"success":               true,
		"message":               message,
		"cascade_deleted_count": cascadeDeletedCount,
		"bracket_deleted_count": bracketDeletedCount,
		"total_cascade_count":   totalCascadeCount,
	})
}

// isOrderCompleted 检查订单是否已完成（已成交）
func isOrderCompleted(status string) bool {
	return status == "filled" || status == "completed"
}

// trySetMarginModeForScheduledOrder 尝试为定时订单预设保证金模式
// 这个函数异步执行，不阻塞订单创建流程
func (s *Server) trySetMarginModeForScheduledOrder(orderID uint, strategyID uint, symbol string) {
	log.Printf("[MarginMode] 开始为定时订单预设保证金模式: orderID=%d, strategyID=%d, symbol=%s", orderID, strategyID, symbol)

	// 获取策略配置
	var strategy pdb.TradingStrategy
	if err := s.db.DB().Where("id = ?", strategyID).First(&strategy).Error; err != nil {
		log.Printf("[MarginMode] 获取策略配置失败: %v", err)
		return
	}

	// 检查是否为期货交易
	if strategy.Conditions.TradingType != "futures" && strategy.Conditions.TradingType != "both" {
		log.Printf("[MarginMode] 策略不是期货交易类型，跳过保证金模式设置: %s", strategy.Conditions.TradingType)
		return
	}

	// 使用优化后的设置函数（复用scheduler的逻辑）
	marginResult := s.trySetMarginModeWithStrategy(&strategy, symbol)

	// 记录设置结果到订单
	var order pdb.ScheduledOrder
	if err := s.db.DB().Where("id = ?", orderID).First(&order).Error; err != nil {
		log.Printf("[MarginMode] 获取订单失败，无法记录设置结果: %v", err)
		return
	}

	// 在订单备注中记录保证金模式设置结果
	// 注意：ScheduledOrder可能没有专门的字段来记录这个，我们可以通过日志和可能的扩展字段来处理

	if marginResult.Success {
		log.Printf("[MarginMode] ✅ 定时订单保证金模式预设成功: orderID=%d, symbol=%s, mode=%s",
			orderID, symbol, marginResult.MarginType)

		// 可以考虑添加一个新的字段来记录设置状态
		// 目前通过日志记录
	} else {
		log.Printf("[MarginMode] ❌ 定时订单保证金模式预设失败: orderID=%d, symbol=%s, error=%v",
			orderID, symbol, marginResult.Error)

		// 记录失败原因，供后续处理
		if strings.Contains(marginResult.Error.Error(), "存在未成交订单") {
			log.Printf("[MarginMode] 💡 此订单将在执行时重试设置保证金模式")
		}
	}
}

// trySetMarginModeWithStrategy 使用策略配置设置保证金模式（复用scheduler的优化逻辑）
func (s *Server) trySetMarginModeWithStrategy(strategy *pdb.TradingStrategy, symbol string) *MarginModeResult {
	// 创建调度器实例来调用 setMarginTypeForStrategy
	scheduler := NewOrderScheduler(s.db.DB(), s.cfg, s)
	return scheduler.setMarginTypeForStrategy(strategy, symbol)
}

// MarginModeResult 使用 scheduler.go 中定义的类型
