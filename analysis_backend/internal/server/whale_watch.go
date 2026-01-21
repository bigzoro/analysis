package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const watchEventLimit = 4

type watchListItem struct {
	ID            uint          `json:"id"`
	Label         string        `json:"label"`
	Address       string        `json:"address"`
	Chain         string        `json:"chain"`
	Entity        string        `json:"entity"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	LatestEventAt *time.Time    `json:"latest_event_at,omitempty"`
	Events        []transferDTO `json:"events"`
}

type watchSummary struct {
	TotalWatchers  int    `json:"total_watchers"`
	ActiveWatchers int    `json:"active_watchers"`
	TotalEvents    int    `json:"total_events"`
	LargestEvent   string `json:"largest_event,omitempty"`
}

type createWhaleWatchRequest struct {
	Label   string `json:"label"`
	Address string `json:"address"`
	Chain   string `json:"chain"`
	Entity  string `json:"entity"`
}

// ListWhaleWatches GET /whales/watchlist
func ListWhaleWatches(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		watches, err := s.db.ListWhaleWatches()
		if err != nil {
			s.DatabaseError(c, "查询大户监控列表", err)
			return
		}

		var (
			items        = make([]watchListItem, 0, len(watches))
			summary      = watchSummary{TotalWatchers: len(watches)}
			maxAmount    float64
			largestEvent string
		)

		for _, watch := range watches {
			events, latestAt, err := s.fetchWatchEvents(watch, watchEventLimit)
			if err != nil {
				s.DatabaseError(c, "获取监控事件", err)
				return
			}

			if len(events) > 0 {
				summary.ActiveWatchers++
				summary.TotalEvents += len(events)
				for _, ev := range events {
					amount, err := strconv.ParseFloat(ev.Amount, 64)
					if err != nil {
						continue
					}
					if amount > maxAmount {
						maxAmount = amount
						largestEvent = fmt.Sprintf("%s %s", ev.Coin, ev.Amount)
					}
				}
			}

			items = append(items, watchListItem{
				ID:            watch.ID,
				Label:         watch.Label,
				Address:       watch.Address,
				Chain:         watch.Chain,
				Entity:        watch.Entity,
				CreatedAt:     watch.CreatedAt,
				UpdatedAt:     watch.UpdatedAt,
				LatestEventAt: latestAt,
				Events:        events,
			})
		}

		if largestEvent != "" {
			summary.LargestEvent = largestEvent
		}

		c.JSON(http.StatusOK, gin.H{
			"watchlist": items,
			"summary":   summary,
		})
	}
}

// CreateWhaleWatch POST /whales/watchlist
func CreateWhaleWatch(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload createWhaleWatchRequest
		if err := c.ShouldBindJSON(&payload); err != nil {
			s.JSONBindError(c, err)
			return
		}

		address := normalizeWatchAddress(payload.Address)
		if address == "" {
			s.ValidationError(c, "address", "地址不能为空")
			return
		}

		if _, err := s.db.GetWhaleWatchByAddress(address); err == nil {
			s.ValidationError(c, "address", "该地址已在监控列表中")
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.DatabaseError(c, "检查重复地址", err)
			return
		}

		entry := &pdb.WhaleWatch{
			Label:   strings.TrimSpace(payload.Label),
			Address: address,
			Chain:   strings.ToLower(strings.TrimSpace(payload.Chain)),
			Entity:  strings.TrimSpace(payload.Entity),
		}

		if err := s.db.CreateWhaleWatch(entry); err != nil {
			s.DatabaseError(c, "保存监控地址", err)
			return
		}

		c.JSON(http.StatusCreated, gin.H{"watch": entry})
	}
}

// DeleteWhaleWatch DELETE /whales/watchlist/:address
func DeleteWhaleWatch(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := normalizeWatchAddress(c.Param("address"))
		if address == "" {
			s.ValidationError(c, "address", "地址不能为空")
			return
		}

		if _, err := s.db.GetWhaleWatchByAddress(address); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				s.NotFound(c, "监控地址不存在")
				return
			}
			s.DatabaseError(c, "读取监控地址", err)
			return
		}

		if err := s.db.DeleteWhaleWatchByAddress(address); err != nil {
			s.DatabaseError(c, "删除监控地址", err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) fetchWatchEvents(watch pdb.WhaleWatch, limit int) ([]transferDTO, *time.Time, error) {
	q := s.db.DB().Model(&pdb.TransferEvent{}).Where("address = ?", watch.Address)

	if watch.Entity != "" {
		q = q.Where("entity = ?", watch.Entity)
	}
	if watch.Chain != "" {
		q = q.Where("chain = ?", watch.Chain)
	}

	var rows []pdb.TransferEvent
	if err := q.Order("occurred_at DESC").Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, nil, err
	}

	if len(rows) == 0 {
		return nil, nil, nil
	}

	dtos := make([]transferDTO, 0, len(rows))
	for _, row := range rows {
		dtos = append(dtos, transferDTO{
			ID:         row.ID,
			Entity:     row.Entity,
			Chain:      row.Chain,
			Coin:       row.Coin,
			Direction:  row.Direction,
			Amount:     row.Amount,
			TxID:       row.TxID,
			Address:    row.Address,
			From:       row.From,
			To:         row.To,
			OccurredAt: row.OccurredAt,
			CreatedAt:  row.CreatedAt,
		})
	}

	latest := rows[0].OccurredAt
	return dtos, &latest, nil
}

func normalizeWatchAddress(addr string) string {
	return strings.ToLower(strings.TrimSpace(addr))
}
