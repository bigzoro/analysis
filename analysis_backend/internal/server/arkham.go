package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	pdb "analysis/internal/db"

	"github.com/gin-gonic/gin"
)

// ArkhamClient 封装 Arkham API 请求
type ArkhamClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// ArkhamAddressSnapshot Arkham API 返回的地址状态
type ArkhamAddressSnapshot struct {
	Address      string                 `json:"address"`
	Label        string                 `json:"label"`
	Chain        string                 `json:"chain"`
	Entity       string                 `json:"entity"`
	BalanceUSD   string                 `json:"balance_usd"`
	LastActiveAt time.Time              `json:"last_active_at"`
	SnapshotAt   time.Time              `json:"snapshot_at"`
	Events       []ArkhamTransfer       `json:"events"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ArkhamTransfer 表示 Arkham 返回的单笔事件
type ArkhamTransfer struct {
	TxHash     string    `json:"tx_hash"`
	Direction  string    `json:"direction"`
	Amount     string    `json:"amount"`
	Symbol     string    `json:"symbol"`
	OccurredAt time.Time `json:"occurred_at"`
}

// NewArkhamClient 构建 Arkham 客户端
func NewArkhamClient(baseURL, apiKey string, httpClient *http.Client) *ArkhamClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &ArkhamClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  httpClient,
	}
}

// FetchAddressSnapshot 拉取地址快照
func (c *ArkhamClient) FetchAddressSnapshot(ctx context.Context, address string) (*ArkhamAddressSnapshot, error) {
	if c == nil || c.baseURL == "" {
		return nil, errors.New("arkham client not configured")
	}
	url := fmt.Sprintf("%s/api/v1/addresses/%s", c.baseURL, address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("arkham error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var snapshot ArkhamAddressSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// Server 增加 Arkham 客户端
func (s *Server) SetArkhamClient(client *ArkhamClient) {
	s.arkhamClient = client
}

// SyncArkhamData 主动同步 Arkham 数据
func (s *Server) SyncArkhamData(ctx context.Context) error {
	if s.arkhamClient == nil {
		return errors.New("arkham client not configured")
	}
	watches, err := s.db.ListArkhamWatches()
	if err != nil {
		return err
	}

	for _, item := range watches {
		snapshot, err := s.arkhamClient.FetchAddressSnapshot(ctx, item.Address)
		if err != nil {
			log.Printf("[WARN] Arkham sync failed for %s: %v", item.Address, err)
			continue
		}

		eventsJSON, _ := json.Marshal(snapshot.Events)
		metadataJSON, _ := json.Marshal(snapshot.Metadata)

		updated := item
		if snapshot.Label != "" {
			updated.Label = snapshot.Label
		}
		if snapshot.Chain != "" {
			updated.Chain = snapshot.Chain
		}
		if snapshot.Entity != "" {
			updated.Entity = snapshot.Entity
		}
		if snapshot.BalanceUSD != "" {
			updated.BalanceUSD = snapshot.BalanceUSD
		}
		updated.LastActiveAt = snapshot.LastActiveAt
		updated.LastSnapshotAt = snapshot.SnapshotAt
		updated.EventsJSON = eventsJSON
		updated.MetadataJSON = metadataJSON

		if err := s.db.CreateOrUpdateArkhamWatch(&updated); err != nil {
			log.Printf("[ERROR] Failed to save Arkham watch %s: %v", item.Address, err)
			continue
		}
	}
	return nil
}

// ListArkhamWatches GET /whales/arkham
func ListArkhamWatches(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := s.db.ListArkhamWatches()
		if err != nil {
			s.DatabaseError(c, "查询 Arkham 监控", err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

// CreateArkhamWatch POST /whales/arkham
func CreateArkhamWatch(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			Address string `json:"address"`
			Label   string `json:"label"`
			Chain   string `json:"chain"`
			Entity  string `json:"entity"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			s.JSONBindError(c, err)
			return
		}
		payload.Address = strings.TrimSpace(payload.Address)
		if payload.Address == "" {
			s.ValidationError(c, "address", "地址不能为空")
			return
		}

		watch := &pdb.ArkhamWatch{
			Address: payload.Address,
			Label:   payload.Label,
			Chain:   payload.Chain,
			Entity:  payload.Entity,
		}
		if err := s.db.CreateOrUpdateArkhamWatch(watch); err != nil {
			s.DatabaseError(c, "保存 Arkham 监控", err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"watch": watch})
	}
}

// DeleteArkhamWatch DELETE /whales/arkham/:address
func DeleteArkhamWatch(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := strings.TrimSpace(c.Param("address"))
		if address == "" {
			s.ValidationError(c, "address", "地址不能为空")
			return
		}
		if err := s.db.DeleteArkhamWatchByAddress(address); err != nil {
			s.DatabaseError(c, "删除 Arkham 监控", err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// QueryArkhamAddress POST /whales/arkham/query
func QueryArkhamAddress(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload struct {
			Address string `json:"address"`
			Chain   string `json:"chain"`
			Entity  string `json:"entity"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			s.JSONBindError(c, err)
			return
		}
		payload.Address = strings.TrimSpace(payload.Address)
		if payload.Address == "" {
			s.ValidationError(c, "address", "地址不能为空")
			return
		}

		if s.arkhamClient == nil {
			s.ValidationError(c, "arkham", "Arkham 客户端未配置")
			return
		}

		ctx := c.Request.Context()

		// 获取地址快照
		snapshot, err := s.arkhamClient.FetchAddressSnapshot(ctx, payload.Address)
		if err != nil {
			log.Printf("[WARN] Arkham API call failed for %s: %v", payload.Address, err)
			// 返回基本信息，即使API调用失败
			result := gin.H{
				"address":        payload.Address,
				"label":          payload.Address, // 使用地址作为标签
				"chain":          payload.Chain,
				"entity":         payload.Entity,
				"balance_usd":    "",
				"last_active_at": nil,
				"queried_at":     time.Now(),
				"transactions":   []ArkhamTransfer{},
				"api_error":      true,
				"error_message":  "Arkham API 访问失败。可能原因：1) API Key无效或过期 2) 账户无API权限 3) API服务暂时不可用。请访问 https://platform.arkhamintelligence.com 检查账户状态",
			}
			c.JSON(http.StatusOK, result)
			return
		}

		result := gin.H{
			"address":        snapshot.Address,
			"label":          snapshot.Label,
			"chain":          snapshot.Chain,
			"entity":         snapshot.Entity,
			"balance_usd":    snapshot.BalanceUSD,
			"last_active_at": snapshot.LastActiveAt,
			"queried_at":     time.Now(),
			"transactions":   snapshot.Events,
		}

		c.JSON(http.StatusOK, result)
	}
}

// TriggerArkhamSync POST /whales/arkham/sync
func TriggerArkhamSync(s *Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.arkhamClient == nil {
			s.ValidationError(c, "arkham", "Arkham 客户端未配置")
			return
		}
		ctx := c.Request.Context()
		go func() {
			if err := s.SyncArkhamData(ctx); err != nil {
				log.Printf("[ERROR] Arkham sync job failed: %v", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"status": "syncing"})
	}
}
