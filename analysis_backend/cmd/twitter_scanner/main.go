package main

import (
	"analysis/internal/config"
	"analysis/internal/netutil"
	"context"
	"flag"
	"log"
	"net/url"
	"strings"
	"time"
)

func main() {
	// === 与项目一致：配置驱动代理 ===
	cfgPath := flag.String("config", "config.yaml", "config file")
	apiBase := flag.String("api", "http://127.0.0.1:8010", "API base")
	interval := flag.Duration("interval", 2*time.Minute, "poll interval")
	// 可选：命令行指定用户名，优先于配置；多用户用逗号
	usersFlag := flag.String("users", "", "comma-separated twitter usernames (override config)")
	flag.Parse()

	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg) // 统一代理

	// 用户列表：flag 优先，其次 config
	var users []string
	if strings.TrimSpace(*usersFlag) != "" {
		for _, u := range strings.Split(*usersFlag, ",") {
			u = strings.TrimSpace(strings.TrimPrefix(u, "@"))
			if u != "" {
				users = append(users, u)
			}
		}
	} else {
		// 按你的 config 结构取值（示例：cfg.Twitter.MonitorUsers）
		for _, u := range cfg.Twitter.MonitorUsers {
			u = strings.TrimSpace(strings.TrimPrefix(u, "@"))
			if u != "" {
				users = append(users, u)
			}
		}
	}
	if len(users) == 0 {
		log.Fatal("no twitter users provided (use -users or config.Twitter.MonitorUsers)")
	}

	if cfg.Twitter.IntervalSeconds > 0 {
		*interval = time.Duration(cfg.Twitter.IntervalSeconds) * time.Second
	}

	log.Printf("[twitter_scanner] start; api=%s interval=%s users=%v (proxy from config)", *apiBase, *interval, users)

	ctx := context.Background()
	//tk := time.NewTicker(*interval)
	//defer tk.Stop()

	//for {
	//	for _, u := range users {
	//		// 调后端 fetch（由后端持 Bearer 调官方 API 并入库）
	//		// /twitter/fetch?username=u&limit=50&store=1
	//		v := url.Values{}
	//		v.Set("username", u)
	//		v.Set("limit", "5")
	//		v.Set("store", "1")
	//		fetchURL := strings.TrimRight(*apiBase, "/") + "/twitter/fetch?" + v.Encode()
	//
	//		var out any
	//		if err := netutil.GetJSON(ctx, fetchURL, &out); err != nil {
	//			log.Printf("fetch %s err: %v", u, err)
	//		} else {
	//			log.Printf("fetched %s ok", u)
	//		}
	//		// 可按需 sleep 少许，避免一瞬间打满
	//		time.Sleep(500 * time.Millisecond)
	//	}
	//	<-tk.C
	//}

	for _, u := range users {
		// 调后端 fetch（由后端持 Bearer 调官方 API 并入库）
		// /twitter/fetch?username=u&limit=50&store=1
		v := url.Values{}
		v.Set("username", u)
		v.Set("limit", "5")
		v.Set("store", "1")
		fetchURL := strings.TrimRight(*apiBase, "/") + "/twitter/fetch?" + v.Encode()

		var out any
		if err := netutil.GetJSON(ctx, fetchURL, &out); err != nil {
			log.Printf("fetch %s err: %v", u, err)
		} else {
			log.Printf("fetched %s ok", u)
		}
		// 可按需 sleep 少许，避免一瞬间打满
		time.Sleep(500 * time.Millisecond)
	}
}
