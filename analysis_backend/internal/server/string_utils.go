package server

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"
)

// ==================== 字符串处理优化工具 ====================

// BuildCacheKey 构建缓存键（优化：使用 strings.Builder）
func BuildCacheKey(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	var builder strings.Builder
	// 预估大小：每个部分平均10个字符 + 分隔符
	builder.Grow(len(parts) * 12)

	for i, part := range parts {
		if i > 0 {
			builder.WriteString(":")
		}
		builder.WriteString(part)
	}

	return builder.String()
}

// BuildCacheKeyWithHash 构建带哈希的缓存键（优化：使用 strings.Builder）
func BuildCacheKeyWithHash(prefix string, key string) string {
	hash := md5.Sum([]byte(key))

	var builder strings.Builder
	// 预估大小：prefix + hash (32 chars) + 分隔符
	builder.Grow(len(prefix) + 35)
	builder.WriteString(prefix)
	builder.WriteString(":")
	builder.WriteString(fmt.Sprintf("%x", hash))

	return builder.String()
}

// BuildLogMessage 构建日志消息（优化：使用 strings.Builder）
func BuildLogMessage(parts ...string) string {
	if len(parts) == 0 {
		return ""
	}

	var builder strings.Builder
	// 预估大小
	totalLen := 0
	for _, part := range parts {
		totalLen += len(part)
	}
	builder.Grow(totalLen + len(parts)*2) // 额外空间用于分隔符

	for i, part := range parts {
		if i > 0 {
			builder.WriteString(" ")
		}
		builder.WriteString(part)
	}

	return builder.String()
}

// FormatErrorLog 格式化错误日志（优化：使用 strings.Builder）
func FormatErrorLog(traceID string, statusCode int, message string, err error, method, path, query string) string {
	var builder strings.Builder
	// 预估大小
	builder.Grow(100 + len(traceID) + len(message) + len(method) + len(path) + len(query))

	builder.WriteString("[ERROR] [")
	builder.WriteString(traceID)
	builder.WriteString("] HTTP ")
	builder.WriteString(strconv.Itoa(statusCode))
	builder.WriteString(": ")
	builder.WriteString(message)

	if err != nil {
		builder.WriteString(" - ")
		builder.WriteString(err.Error())
	}

	if method != "" || path != "" {
		builder.WriteString(" | Method: ")
		builder.WriteString(method)
		builder.WriteString(", Path: ")
		builder.WriteString(path)
		if query != "" {
			builder.WriteString("?")
			builder.WriteString(query)
		}
	}

	return builder.String()
}
