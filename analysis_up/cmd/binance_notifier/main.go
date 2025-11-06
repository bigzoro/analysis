package main

import (
	"analysis/internal/config"
	"analysis/internal/db"
	"context"
	"flag"
	"log"
	"strings"
	"time"
)

func main() {
	cfgPath := flag.String("config", "./config.yaml", "config file path")
	interval := flag.Duration("interval", 5*time.Minute, "poll interval, e.g. 5m")
	once := flag.Bool("once", false, "run once without sending notifications")
	lang := flag.String("lang", "en", "language code for Binance CMS, e.g. en, zh-CN")
	pageSize := flag.Int("page-size", 20, "number of articles to fetch per catalog")

	spotCatalog := flag.Int("spot-catalog", 48, "catalog ID for spot listing announcements (0 to disable)")
	earnCatalog := flag.Int("earn-catalog", 107, "catalog ID for Earn/financial announcements (0 to disable)")
	spotKeywords := flag.String("spot-keywords", "will list,listing,上架,上线", "comma separated keywords for spot announcements")
	earnKeywords := flag.String("earn-keywords", "earn,理财,simple earn,staking,质押", "comma separated keywords for earn announcements")

	tz := flag.String("tz", "Asia/Taipei", "timezone for email display, e.g. Asia/Taipei")

	pmServerToken := flag.String("pm-server", "", "Postmark server token (required)")
	pmAccountToken := flag.String("pm-account", "", "Postmark account token (optional)")
	pmFrom := flag.String("pm-from", "", "Postmark from address (required)")
	pmTo := flag.String("pm-to", "", "Postmark comma-separated recipients (required)")
	pmStream := flag.String("pm-stream", "outbound", "Postmark message stream")

	flag.Parse()

	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)
	config.ApplyProxy(&cfg)

	loc, err := time.LoadLocation(*tz)
	if err != nil {
		log.Fatalf("invalid timezone %s: %v", *tz, err)
	}

	if *pmServerToken == "" || *pmFrom == "" || *pmTo == "" {
		log.Fatalf("postmark configuration missing: server token/from/to are required")
	}

	opts := db.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  cfg.Database.Automigrate,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	}

	gdb, err := db.OpenMySQL(opts)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	httpClient := newHTTPClient()
	fetcher := NewAnnouncementFetcher(httpClient, *lang, *pageSize)

	watches := buildWatches(*spotCatalog, *spotKeywords, *earnCatalog, *earnKeywords)
	if len(watches) == 0 {
		log.Fatalf("no watch rules configured (spot-catalog/earn-catalog disabled)")
	}

	notifier := NewNotifier(gdb, fetcher, watches, PostmarkConfig{
		ServerToken:  *pmServerToken,
		AccountToken: *pmAccountToken,
		Stream:       *pmStream,
		From:         *pmFrom,
		To:           parseRecipients(*pmTo),
	}, loc)

	ctx := context.Background()

	log.Printf("[binance_notifier] priming caches (no email on first run)...")
	if err := notifier.Prime(ctx); err != nil {
		log.Printf("[binance_notifier] prime failed: %v", err)
	}

	if *once {
		log.Printf("[binance_notifier] once mode enabled, exit after prime")
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	log.Printf("[binance_notifier] started. interval=%s lang=%s pageSize=%d", interval.String(), *lang, *pageSize)

	for {
		select {
		case <-ticker.C:
			if err := notifier.Tick(ctx); err != nil {
				log.Printf("[binance_notifier] tick error: %v", err)
			}
		}
	}
}

func buildWatches(spotCatalog int, spotKeywordCSV string, earnCatalog int, earnKeywordCSV string) []Watch {
	watches := make([]Watch, 0, 2)
	if spotCatalog > 0 {
		watches = append(watches, Watch{
			Name:      "spot",
			CatalogID: spotCatalog,
			Keywords:  splitKeywords(spotKeywordCSV),
		})
	}
	if earnCatalog > 0 {
		watches = append(watches, Watch{
			Name:      "earn",
			CatalogID: earnCatalog,
			Keywords:  splitKeywords(earnKeywordCSV),
		})
	}
	return watches
}

func splitKeywords(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseRecipients(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		addr := strings.TrimSpace(p)
		if addr != "" {
			out = append(out, addr)
		}
	}
	return out
}
