package server

import (
	"net/http"
	"time"
)

// ==================== HTTP 客户端复用 ====================

// DefaultHTTPClient 默认 HTTP 客户端（优化：复用连接，配置合理的超时）
var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,              // 最大空闲连接数
		MaxIdleConnsPerHost: 10,               // 每个主机的最大空闲连接数
		IdleConnTimeout:     90 * time.Second, // 空闲连接超时
		DisableKeepAlives:   false,            // 启用连接复用
	},
}

// TwitterHTTPClient Twitter API 专用 HTTP 客户端
var TwitterHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     60 * time.Second,
		DisableKeepAlives:   false,
	},
}
