package main

import (
	"analysis/internal/config"
	"context"
	"flag"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	api := flag.String("api", "http://127.0.0.1:8010", "API base, e.g. http://127.0.0.1:8010")
	interval := flag.Duration("interval", 2*time.Hour, "fetch interval, e.g. 2h")
	limit := flag.Int("limit", 20, "top N symbols, e.g. 20")
	spot := flag.Bool("spot", true, "fetch spot top gainers")
	futures := flag.Bool("futures", true, "fetch futures top gainers")
	cfgPath := flag.String("config", "./config.yaml", "config file")
	once := flag.Bool("once", false, "run once and exit")

	// 用于 2 小时段对齐
	tz := flag.String("tz", "Asia/Taipei", "timezone for 2h alignment, e.g. Asia/Taipei, UTC")

	// ===== 回调告警（上一段 vs 当前段）=====
	alertEnable := flag.Bool("alert-pullback", true, "enable pullback alert between slots")
	alertThresholdPct := flag.Float64("alert-threshold", 10.0, "pullback threshold in percent, e.g. 10 = 10%")

	// ===== Postmark（使用 github.com/keighl/postmark）=====
	pmServerToken := flag.String("pm-server-token", "", "Postmark Server Token (can also be set via POSTMARK_SERVER_TOKEN env var)")
	pmAccountToken := flag.String("pm-account-token", "", "Postmark Account Token (optional, can also be set via POSTMARK_ACCOUNT_TOKEN env var)")
	pmFrom := flag.String("pm-from", "", "From email address (required)")
	pmTo := flag.String("pm-to", "", "Comma-separated TO addresses (required)")
	pmStream := flag.String("pm-stream", "outbound", "Postmark MessageStream (default: outbound)")

	// 注意：黑名单不再在同步时使用，前端显示时会过滤黑名单

	flag.Parse()

	log.Printf("[market_scanner] starting ...")
	log.Printf("[market_scanner] api=%s interval=%s limit=%d spot=%v futures=%v config=%s tz=%s",
		*api, interval.String(), *limit, *spot, *futures, *cfgPath, *tz)
	log.Printf("[market_scanner] alert enable=%v threshold=%.2f%% postmark_to=%s stream=%s",
		*alertEnable, *alertThresholdPct, *pmTo, *pmStream)

	// 1) 读配置 & 代理（走你项目现有代理设置）
	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg)
	log.Printf("[market_scanner] loaded config, proxy: http=%s https=%s enabled=%v",
		cfg.Proxy.HTTP, cfg.Proxy.HTTPS, cfg.Proxy.Enable)

	// 验证代理环境变量是否设置成功
	if cfg.Proxy.Enable {
		httpProxy := os.Getenv("HTTP_PROXY")
		httpsProxy := os.Getenv("HTTPS_PROXY")
		log.Printf("[market_scanner] proxy env vars: HTTP_PROXY=%s HTTPS_PROXY=%s", httpProxy, httpsProxy)
	}

	// 2) 时区
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		log.Fatalf("[market_scanner] invalid timezone %s: %v", *tz, err)
	}

	// 3) 规范 API base
	base := strings.TrimRight(*api, "/")

	// 优化：从环境变量获取 Postmark Token（如果命令行参数未提供）
	pmToken := *pmServerToken
	if pmToken == "" {
		pmToken = os.Getenv("POSTMARK_SERVER_TOKEN")
	}
	pmAccToken := *pmAccountToken
	if pmAccToken == "" {
		pmAccToken = os.Getenv("POSTMARK_ACCOUNT_TOKEN")
	}

	// 4) 创建采集器（带 Postmark 告警）
	// 注意：同步时不再使用黑名单，直接保存涨幅前20个，前端显示时会过滤黑名单
	collector := NewBinanceMarketCollector(
		base,
		*interval,
		*limit,
		*spot,
		*futures,
		loc,
		AlertConfig{
			Enable:               *alertEnable,
			Threshold:            *alertThresholdPct / 100.0, // 10 -> 0.10
			PostmarkServerToken:  pmToken,
			PostmarkAccountToken: pmAccToken,
			PostmarkStream:       *pmStream,
			From:                 *pmFrom,
			ToCSV:                *pmTo,
		},
	)

	log.Printf(
		"[market_scanner] collector created, first at local=%s (tz=%s), utc=%s",
		collector.NextTimeLocal().Format(time.RFC3339),
		loc.String(),
		collector.NextTimeUTC().Format(time.RFC3339),
	)

	ctx := context.Background()

	// 5) 启动时先拉一次（作为上一段基线，不触发告警）
	log.Printf("[market_scanner] running initial fetch ...")
	if err := collector.RunOnce(ctx); err != nil {
		log.Printf("[market_scanner] initial fetch failed: %v (will retry on schedule)", err)
	} else {
		log.Printf("[market_scanner] initial fetch done.")
	}

	if *once {
		log.Printf("[market_scanner] run-once mode, exit.")
		return
	}

	// 7) 主循环：到下一个对齐点再拉，会对比并可能发告警
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastSkipLog time.Time // 上次打印 skip 日志的时间

	for {
		select {
		case now := <-ticker.C:
			ok := collector.Tick(ctx, now.UTC())
			if ok {
				log.Printf(
					"[market_scanner] tick finished, next local=%s utc=%s",
					collector.NextTimeLocal().Format(time.RFC3339),
					collector.NextTimeUTC().Format(time.RFC3339),
				)
				lastSkipLog = time.Time{} // 重置，下次 skip 时立即打印
			} else {
				// 只在距离上次打印超过 10 分钟时才打印 skip 日志
				if lastSkipLog.IsZero() || time.Since(lastSkipLog) >= 10*time.Minute {
					nowLocal := now.In(loc)
					log.Printf(
						"[market_scanner] tick skipped, now_local=%s now_utc=%s < next_local=%s next_utc=%s",
						nowLocal.Format(time.RFC3339),
						now.UTC().Format(time.RFC3339),
						collector.NextTimeLocal().Format(time.RFC3339),
						collector.NextTimeUTC().Format(time.RFC3339),
					)
					lastSkipLog = time.Now()
				}
			}
		}
	}
}
