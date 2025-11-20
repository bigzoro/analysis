package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

type sseEvent struct {
	TS        time.Time `json:"ts"`
	Chain     string    `json:"chain"`
	Coin      string    `json:"coin"`
	Direction string    `json:"direction"`
	Amount    string    `json:"amount"`
	TxID      string    `json:"txid"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Address   string    `json:"address"`
}

// /stream/transfers?entity=binance&coin=BTC,ETH,USDT,USDC,SOL&since=2025-09-04T00:00:00Z
func (s *Server) StreamTransfers(c *gin.Context) {
	entity := strings.TrimSpace(c.Query("entity"))
	if entity == "" {
		// 优化：使用统一的错误处理
		s.ValidationError(c, "entity", "实体名称不能为空")
		return
	}
	var coins []string
	if q := strings.TrimSpace(c.Query("coin")); q != "" {
		for _, x := range strings.Split(q, ",") {
			x = strings.ToUpper(strings.TrimSpace(x))
			if x != "" {
				coins = append(coins, x)
			}
		}
	}
	sinceStr := strings.TrimSpace(c.Query("since"))
	since := time.Now().UTC().Add(-5 * time.Minute)
	if sinceStr != "" {
		if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = t
		}
	}

	w := c.Writer
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		c.String(http.StatusInternalServerError, "streaming unsupported")
		return
	}

	ctx := c.Request.Context()
	// 预热注释行，避免代理延迟
	fmt.Fprintf(w, ": ok\n\n")
	flusher.Flush()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var rows []pdb.TransferEvent
			q := s.db.DB().Where("entity = ? AND created_at > ?", entity, since).
				Order("created_at asc, id asc").
				Limit(500)
			if len(coins) > 0 {
				q = q.Where("coin IN ?", coins)
			}
			if err := q.Find(&rows).Error; err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"message\": %q}\n\n", err.Error())
				flusher.Flush()
				continue
			}
			if len(rows) == 0 {
				fmt.Fprint(w, ": ping\n\n")
				flusher.Flush()
				continue
			}
			// 优化：使用 JSON 编码器进行流式编码，提高性能
			encoder := json.NewEncoder(w)
			for _, r := range rows {
				ev := sseEvent{
					//TS:        r.TS,
					Chain:     r.Chain,
					Coin:      r.Coin,
					Direction: r.Direction,
					Amount:    r.Amount,
					TxID:      r.TxID,
					From:      r.From,
					To:        r.To,
					Address:   r.Address,
				}
				// 优化：使用流式编码，避免创建临时字符串
				fmt.Fprint(w, "event: transfer\ndata: ")
				if err := encoder.Encode(ev); err != nil {
					// 编码失败，记录日志但继续处理其他数据
					log.Printf("[ERROR] Failed to encode SSE event: %v", err)
					continue
				}
				// encoder.Encode 会自动添加换行，SSE 需要两个换行
				fmt.Fprint(w, "\n")
				since = r.CreatedAt
			}
			flusher.Flush()
		}
	}
}
