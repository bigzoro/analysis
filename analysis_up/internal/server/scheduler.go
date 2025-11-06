package server

import (
	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type OrderScheduler struct {
	db  *gorm.DB
	ctx context.Context
}

func NewOrderScheduler(db *gorm.DB) *OrderScheduler {
	return &OrderScheduler{db: db, ctx: context.Background()}
}

func (s *OrderScheduler) Start() {
	go s.loop()
}

func (s *OrderScheduler) loop() {
	tk := time.NewTicker(1 * time.Second)
	defer tk.Stop()

	for range tk.C {
		s.tick()
	}
}

func (s *OrderScheduler) tick() {
	now := time.Now().UTC()

	var batch []pdb.ScheduledOrder
	// 取到期且尚未处理的订单
	if err := s.db.
		Where("status = ? AND trigger_time <= ?", "pending", now).
		Order("trigger_time asc").
		Limit(20).
		Find(&batch).Error; err != nil {
		return
	}
	for _, ord := range batch {
		// 乐观推进状态，防止并发重复执行
		res := s.db.Model(&pdb.ScheduledOrder{}).
			Where("id = ? AND status = ?", ord.ID, "pending").
			Update("status", "processing")
		if res.Error != nil || res.RowsAffected == 0 {
			continue
		}
		go s.execute(ord)
	}
}

func (s *OrderScheduler) execute(o pdb.ScheduledOrder) {
	ex := strings.ToLower(o.Exchange)
	var status, result string

	switch ex {
	case "binance_futures":
		c := bf.New(o.Testnet)
		// 可选：设置杠杆
		if o.Leverage > 0 {
			if code, body, err := c.SetLeverage(o.Symbol, o.Leverage); err != nil || code >= 400 {
				result = fmt.Sprintf("set leverage failed: code=%d body=%s err=%v", code, string(body), err)
				s.fail(o.ID, result)
				return
			}
		}
		// 一键三连：若启用则下进场单后挂 TP/SL
		if o.BracketEnabled {
			code, body, err := c.PlaceOrder(o.Symbol, o.Side, o.OrderType, o.Quantity, o.Price, o.ReduceOnly)
			if err != nil || code >= 400 {
				result = fmt.Sprintf("order failed: code=%d body=%s err=%v", code, string(body), err)
				s.fail(o.ID, result)
				return
			}
			// 计算参考入场价
			refPx := ""
			if strings.ToUpper(o.OrderType) == "MARKET" || o.Price == "" {
				if px, e := c.GetMarkPrice(o.Symbol); e == nil && px > 0 {
					refPx = fmt.Sprintf("%.8f", px)
				}
			} else {
				refPx = o.Price
			}
			// 若百分比存在，按参考价计算 TP/SL 绝对值
			var tpPrice, slPrice string
			if o.TPPercent > 0 && refPx != "" {
				f, _ := strconv.ParseFloat(refPx, 64)
				if strings.ToUpper(o.Side) == "BUY" {
					tpPrice = fmt.Sprintf("%.8f", f*(1.0+o.TPPercent/100.0))
				} else {
					tpPrice = fmt.Sprintf("%.8f", f*(1.0-o.TPPercent/100.0))
				}
			}
			if o.SLPercent > 0 && refPx != "" {
				f, _ := strconv.ParseFloat(refPx, 64)
				if strings.ToUpper(o.Side) == "BUY" {
					slPrice = fmt.Sprintf("%.8f", f*(1.0-o.SLPercent/100.0))
				} else {
					slPrice = fmt.Sprintf("%.8f", f*(1.0+o.SLPercent/100.0))
				}
			}
			if tpPrice == "" && strings.TrimSpace(o.TPPrice) != "" {
				tpPrice = strings.TrimSpace(o.TPPrice)
			}
			if slPrice == "" && strings.TrimSpace(o.SLPrice) != "" {
				slPrice = strings.TrimSpace(o.SLPrice)
			}

			// 挂 reduceOnly 的出场单（closePosition=true）
			exitSide := "SELL"
			if strings.ToUpper(o.Side) == "SELL" {
				exitSide = "BUY"
			}
			gid := fmt.Sprintf("sch-%d-%d", o.ID, time.Now().Unix())
			tpCID := gid + "-tp"
			slCID := gid + "-sl"

			if tpPrice != "" {
				if code, body, err := c.PlaceConditionalClose(o.Symbol, exitSide, "TAKE_PROFIT_MARKET",
					tpPrice, o.WorkingType, true, true, tpCID); err != nil || code >= 400 {
					result = fmt.Sprintf("tp failed: code=%d body=%s err=%v", code, string(body), err)
				}
			}
			if slPrice != "" {
				if code, body, err := c.PlaceConditionalClose(o.Symbol, exitSide, "STOP_MARKET",
					slPrice, o.WorkingType, true, true, slCID); err != nil || code >= 400 {
					result += " | " + fmt.Sprintf("sl failed: code=%d body=%s err=%v", code, string(body), err)
				}
			}
			// 保存 BracketLink 记录（忽略错误）
			_ = s.db.Create(&pdb.BracketLink{
				ScheduleID:    o.ID,
				Symbol:        o.Symbol,
				GroupID:       gid,
				EntryClientID: "", // 如需可改造记录 entry 的 clientId
				TPClientID:    tpCID,
				SLClientID:    slCID,
				Status:        "active",
			}).Error
		} else {
			code, body, err := c.PlaceOrder(o.Symbol, o.Side, o.OrderType, o.Quantity, o.Price, o.ReduceOnly)
			if err != nil || code >= 400 {
				result = fmt.Sprintf("order failed: code=%d body=%s err=%v", code, string(body), err)
				s.fail(o.ID, result)
				return
			}
		}
	default:
		status = "failed"
		result = "unsupported exchange: " + o.Exchange
	}

	status = "success"
	_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", o.ID).
		Updates(map[string]any{"status": status, "result": result})
}

func (s *OrderScheduler) fail(id uint, reason string) {
	log.Println("[scheduler] order fail:", reason)
	_ = s.db.Model(&pdb.ScheduledOrder{}).Where("id = ?", id).
		Updates(map[string]any{"status": "failed", "result": reason})
}
