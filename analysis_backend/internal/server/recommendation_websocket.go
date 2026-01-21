package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// WSRecommendations WebSocket实时推荐
func (s *Server) WSRecommendations(c *gin.Context) {
	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] 升级失败: %v", err)
		return
	}
	defer ws.Close()

	// 获取客户端信息
	clientIP := c.ClientIP()
	log.Printf("[WebSocket] 新连接建立: %s", clientIP)

	// 读取客户端的订阅消息
	_, message, err := ws.ReadMessage()
	if err != nil {
		log.Printf("[WebSocket] 读取订阅消息失败: %v", err)
		return
	}

	// 解析订阅请求
	var subscription struct {
		Action          string   `json:"action"`
		Symbols         []string `json:"symbols"`
		UpdateFrequency string   `json:"update_frequency"`
	}

	if err := json.Unmarshal(message, &subscription); err != nil {
		log.Printf("[WebSocket] 解析订阅消息失败: %v", err)
		ws.WriteJSON(gin.H{"error": "无效的订阅格式"})
		return
	}

	if subscription.Action != "subscribe" {
		ws.WriteJSON(gin.H{"error": "不支持的操作"})
		return
	}

	// 验证订阅参数
	if err := s.validateWebSocketSubscription(&subscription); err != nil {
		log.Printf("[WebSocket] 订阅参数验证失败: %v", err)
		ws.WriteJSON(gin.H{"error": err.Error()})
		return
	}

	log.Printf("[WebSocket] 客户端订阅: symbols=%v, frequency=%s", subscription.Symbols, subscription.UpdateFrequency)

	// 解析并优化更新频率
	updateInterval := s.optimizeUpdateInterval(subscription.UpdateFrequency, subscription.Symbols)

	log.Printf("[WebSocket] 优化后的更新频率: %v (原始请求: %s)", updateInterval, subscription.UpdateFrequency)

	// 发送确认消息
	ws.WriteJSON(gin.H{
		"type":    "subscription_confirmed",
		"message": "实时推荐订阅成功",
		"config": gin.H{
			"update_interval": updateInterval.String(),
			"symbols":         subscription.Symbols,
		},
	})

	// 创建定时器发送实时更新
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	// 模拟实时数据更新
	for {
		select {
		case <-ticker.C:
			// 生成真实的实时推荐更新
			realtimeUpdates := s.generateRealtimeUpdates(subscription.Symbols)

			// 发送更新
			response := gin.H{
				"type":            "recommendation_update",
				"timestamp":       time.Now().Unix(),
				"recommendations": realtimeUpdates,
			}

			if err := ws.WriteJSON(response); err != nil {
				log.Printf("[WebSocket] 发送更新失败: %v", err)
				return
			}

			log.Printf("[WebSocket] 发送实时更新: %d 条推荐", len(realtimeUpdates))

		case <-time.After(100 * time.Millisecond):
			// 短暂延迟，避免CPU占用过高
			// 这里可以添加连接健康检查，但不主动读取消息
			continue
		}
	}
}

// validateWebSocketSubscription 验证WebSocket订阅参数
func (s *Server) validateWebSocketSubscription(sub *struct {
	Action          string   `json:"action"`
	Symbols         []string `json:"symbols"`
	UpdateFrequency string   `json:"update_frequency"`
}) error {
	// 验证币种
	if len(sub.Symbols) == 0 {
		return fmt.Errorf("至少需要指定一个币种")
	}

	if len(sub.Symbols) > 20 {
		return fmt.Errorf("订阅币种数量不能超过20个")
	}

	// 验证并清理币种名称
	validSymbols := make([]string, 0, len(sub.Symbols))
	symbolSet := make(map[string]bool)

	for _, symbol := range sub.Symbols {
		cleanSymbol := strings.TrimSpace(strings.ToUpper(symbol))
		if cleanSymbol == "" {
			continue
		}

		// 检查格式
		if !regexp.MustCompile(`^[A-Z0-9]{1,10}$`).MatchString(cleanSymbol) {
			return fmt.Errorf("无效的币种名称: %s", symbol)
		}

		// 去重
		if !symbolSet[cleanSymbol] {
			symbolSet[cleanSymbol] = true
			validSymbols = append(validSymbols, cleanSymbol)
		}
	}

	if len(validSymbols) == 0 {
		return fmt.Errorf("没有有效的币种名称")
	}

	sub.Symbols = validSymbols
	return nil
}

// generateRealtimeUpdates 生成真实的实时推荐更新
func (s *Server) generateRealtimeUpdates(symbols []string) []gin.H {
	updates := make([]gin.H, 0, len(symbols))

	for _, symbol := range symbols {
		// 获取最新的推荐数据
		ctx := context.Background()
		recommendation, err := s.generateSingleAIRecommendation(ctx, symbol)
		if err != nil {
			log.Printf("[WebSocket] 生成推荐更新失败 %s: %v", symbol, err)
			continue
		}

		// 只返回关键的更新字段，避免发送完整推荐数据
		update := gin.H{
			"symbol":        symbol,
			"overall_score": recommendation["overall_score"],
			"ml_prediction": recommendation["ml_prediction"],
			"ml_confidence": recommendation["ml_confidence"],
			"price":         recommendation["price"],
			"risk_score":    recommendation["risk_score"],
			"update_time":   time.Now().Unix(),
		}

		updates = append(updates, update)
	}

	return updates
}

// getRandomMarketState 获取随机市场状态
func (s *Server) getRandomMarketState() string {
	states := []string{"bull", "bear", "sideways"}
	return states[rand.Intn(len(states))]
}

// optimizeUpdateInterval 根据多种因素智能优化WebSocket更新频率
//
// 优化策略:
// 1. 基础频率限制: 10秒-5分钟之间
// 2. 币种数量影响: 币种越多，更新频率越低（减少服务器负载）
// 3. 时间因素: 交易高峰期提高频率，非交易时段降低频率
// 4. 系统负载: 高负载时降低频率避免过载
//
// 参数:
//   - requestedFrequency: 客户端请求的频率（如"60s"）
//   - symbols: 订阅的币种列表
//
// 返回:
//   - 优化后的更新间隔时间
func (s *Server) optimizeUpdateInterval(requestedFrequency string, symbols []string) time.Duration {
	// 解析请求的频率
	baseInterval := 60 * time.Second // 默认60秒
	if requestedFrequency != "" {
		if strings.HasSuffix(requestedFrequency, "s") {
			if sec, err := strconv.Atoi(strings.TrimSuffix(requestedFrequency, "s")); err == nil && sec > 0 {
				baseInterval = time.Duration(sec) * time.Second
			}
		}
	}

	// 应用频率限制
	minInterval := 10 * time.Second  // 最快10秒
	maxInterval := 300 * time.Second // 最慢5分钟

	if baseInterval < minInterval {
		baseInterval = minInterval
	} else if baseInterval > maxInterval {
		baseInterval = maxInterval
	}

	// 根据币种数量调整频率（越多币种，更新频率越低）
	symbolCount := len(symbols)
	if symbolCount > 10 {
		// 超过10个币种，每增加2个币种增加5秒延迟
		extraDelay := time.Duration((symbolCount-10)/2) * 5 * time.Second
		baseInterval += extraDelay
		if baseInterval > maxInterval {
			baseInterval = maxInterval
		}
	}

	// 根据当前时间调整频率（交易高峰期更频繁更新）
	now := time.Now().UTC()
	hour := now.Hour()

	// 亚洲交易时段 (0:00-8:00 UTC) 和欧美交易时段 (12:00-20:00 UTC) 提高频率
	if (hour >= 0 && hour <= 8) || (hour >= 12 && hour <= 20) {
		// 交易高峰期减少20%的更新间隔
		baseInterval = time.Duration(float64(baseInterval) * 0.8)
		if baseInterval < minInterval {
			baseInterval = minInterval
		}
	} else {
		// 非交易时段增加20%的更新间隔
		baseInterval = time.Duration(float64(baseInterval) * 1.2)
		if baseInterval > maxInterval {
			baseInterval = maxInterval
		}
	}

	// 根据系统负载调整频率
	loadFactor := s.getSystemLoadFactor()
	if loadFactor > 0.8 {
		// 高负载时增加更新间隔
		baseInterval = time.Duration(float64(baseInterval) * 1.5)
		if baseInterval > maxInterval {
			baseInterval = maxInterval
		}
	}

	log.Printf("[WebSocket] 频率优化: 币种数=%d, 时间=%02d:00 UTC, 负载=%.2f, 最终间隔=%v",
		symbolCount, hour, loadFactor, baseInterval)

	return baseInterval
}

// getSystemLoadFactor 获取系统负载因子 (0.0-1.0)
func (s *Server) getSystemLoadFactor() float64 {
	// 简化的负载估算
	// 在实际实现中，可以基于CPU使用率、内存使用率、活跃连接数等计算

	// 这里使用随机值模拟，实际应该基于系统指标
	return 0.3 + rand.Float64()*0.4 // 0.3-0.7之间的随机值
}
