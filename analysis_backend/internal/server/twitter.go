package server

import (
	pdb "analysis/internal/db"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type twitterUserResp struct {
	Data struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	} `json:"data"`
}
type twitterTweetsResp struct {
	Data []struct {
		ID        string    `json:"id"`
		Text      string    `json:"text"`
		CreatedAt time.Time `json:"created_at"`
	} `json:"data"`
	Meta struct {
		ResultCount   int    `json:"result_count"`
		NextToken     string `json:"next_token"`
		PreviousToken string `json:"previous_token"`
	} `json:"meta"`
}

func (s *Server) getTwitterUserID(ctx context.Context, username string) (string, error) {
	if s.XBearer == "" {
		return "", errors.New("twitter bearer token not configured")
	}
	u := "https://api.twitter.com/2/users/by/username/" + url.PathEscape(username) + "?user.fields=username"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+s.XBearer)

	// 优化：使用复用的 HTTP 客户端
	resp, err := TwitterHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", errors.New("twitter user lookup failed: " + resp.Status)
	}
	var out twitterUserResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Data.ID == "" {
		return "", errors.New("user not found")
	}
	return out.Data.ID, nil
}

func (s *Server) fetchTweets(ctx context.Context, uid, username string, limit int, paginationToken string) ([]pdb.TwitterPost, string, error) {
	// Twitter API v2 限制：max_results 最大为 100
	if limit <= 0 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}

	params := url.Values{}
	params.Set("max_results", strconv.Itoa(limit))
	params.Set("tweet.fields", "created_at")
	params.Set("exclude", "retweets,replies")
	if paginationToken != "" {
		params.Set("pagination_token", paginationToken)
	}
	u := "https://api.twitter.com/2/users/" + uid + "/tweets?" + params.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	req.Header.Set("Authorization", "Bearer "+s.XBearer)

	// 优化：使用复用的 HTTP 客户端
	resp, err := TwitterHTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, "", errors.New("twitter fetch failed: " + resp.Status)
	}
	var out twitterTweetsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, "", err
	}
	var items []pdb.TwitterPost
	lower := strings.ToLower(username)
	for _, t := range out.Data {
		items = append(items, pdb.TwitterPost{
			Username:  lower,
			TweetID:   t.ID,
			Text:      t.Text,
			URL:       "https://x.com/" + username + "/status/" + t.ID,
			TweetTime: t.CreatedAt.UTC(),
		})
	}
	return items, out.Meta.NextToken, nil
}

// GET /twitter/fetch?username={name}&limit=50&store=1
func (s *Server) FetchTwitterUserPosts(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	if username == "" {
		s.ValidationError(c, "username", "用户名不能为空")
		return
	}
	// 默认拉取 10 条，最大支持 100 条（Twitter API 限制）
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	store := c.DefaultQuery("store", "1") != "0"
	paginationToken := strings.TrimSpace(c.Query("pagination_token"))

	ctx := c.Request.Context()
	uid, err := s.getTwitterUserID(ctx, username)
	if err != nil {
		s.BadRequest(c, "获取 Twitter 用户信息失败", err)
		return
	}
	items, nextToken, err := s.fetchTweets(ctx, uid, username, limit, paginationToken)
	if err != nil {
		s.BadRequest(c, "获取推文失败", err)
		return
	}
	if store {
		if _, err := pdb.SaveTwitterPosts(s.db.DB(), items); err != nil {
			s.DatabaseError(c, "保存推文", err)
			return
		}
		// 失效 Twitter 相关缓存，使新数据立即生效
		_ = s.InvalidateTwitterCache(c.Request.Context())
	}

	response := gin.H{
		"items": items,
		"count": len(items),
	}
	if nextToken != "" {
		response["next_token"] = nextToken
		response["has_more"] = true
	} else {
		response["has_more"] = false
	}
	c.JSON(http.StatusOK, response)
}

// GET /twitter/posts?username={name}&page=1&page_size=5
// 如果 username 为空，返回所有用户的最近推文
// 支持分页：page(>=1), page_size(<=200, 默认5)
// 支持搜索和过滤：keyword, start_date, end_date
func (s *Server) ListTwitterPosts(c *gin.Context) {
	username := strings.TrimSpace(c.Query("username"))
	keyword := strings.TrimSpace(c.Query("keyword"))
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	// 分页参数
	pagination := ParsePaginationParams(
		c.Query("page"),
		c.Query("page_size"),
		5,   // 默认每页数量
		200, // 最大每页数量
	)
	page := pagination.Page
	pageSize := pagination.PageSize

	// 构建查询（用于计数和查询数据）
	q := s.db.DB().Model(&pdb.TwitterPost{})
	// 优化：如果数据存储时已统一大小写，直接查询，避免使用函数导致索引失效
	if username != "" {
		q = q.Where("username = ?", strings.ToLower(username))
	}
	if keyword != "" {
		q = q.Where("text LIKE ?", "%"+keyword+"%")
	}
	// 优化：使用范围查询替代 DATE() 函数，可以利用索引
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			q = q.Where("tweet_time >= ?", t.UTC())
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含当天，所以加一天并减1秒
			endTime := t.UTC().Add(24*time.Hour).Add(-time.Second)
			q = q.Where("tweet_time <= ?", endTime)
		}
	}

	// 计算总数
	var total int64
	if err := q.Count(&total).Error; err != nil {
		s.DatabaseError(c, "查询推文总数", err)
		return
	}

	// 分页查询：按时间倒序（最新的在最前面）
	var list []pdb.TwitterPost
	if err := q.
		Order("tweet_time desc").
		Offset(pagination.Offset).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		s.DatabaseError(c, "查询推文列表", err)
		return
	}

	// 计算总页数
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       list,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	})
}
