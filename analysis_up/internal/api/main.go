// internal/api/main.go
package main

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	addr := flag.String("addr", ":8010", "listen addr")
	cfgPath := flag.String("config", "./config.yaml", "config path")
	corsOrigins := flag.String("cors", "*", "cors origins, comma separated")
	flag.Parse()

	var cfg config.Config
	config.MustLoad(*cfgPath, &cfg)

	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  cfg.Database.Automigrate,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		panic(err)
	}

	if err := gdb.AutoMigrate(&pdb.User{}); err != nil {
		panic(err)
	}

	api := server.New(gdb)

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	c := cors.DefaultConfig()
	if *corsOrigins == "*" {
		c.AllowAllOrigins = true
	} else {
		var origins []string
		for _, o := range strings.Split(*corsOrigins, ",") {
			o = strings.TrimSpace(o)
			if o == "" {
				continue
			}
			origins = append(origins, o)
		}
		c.AllowOrigins = origins
	}
	c.AllowCredentials = true
	c.AllowHeaders = []string{"Authorization", "Content-Type", "Origin"}
	c.AllowMethods = []string{http.MethodGet, http.MethodPost, http.MethodOptions}
	r.Use(cors.New(c))

	// 健康检查
	r.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true, "time": time.Now().UTC()})
	})

	// 登录注册
	r.POST("/auth/register", api.Register)
	r.POST("/auth/login", api.Login)
	r.GET("/me", api.JWTAuth(), api.Me)

	// cursor & ingest events
	r.GET("/sync/cursor", server.GetCursor(gdb))
	r.POST("/sync/cursor", server.GetCursor(gdb))
	r.POST("/ingest/events", server.IngestEvents(gdb))

	// ✅ 给 market_scanner 用的采集入口（不强制登录）
	r.POST("/ingest/binance/market", server.IngestBinanceMarket(gdb))

	// 需要鉴权的
	priv := r.Group("/")
	priv.Use(api.JWTAuth())
	{
		priv.GET("/entities", api.ListEntities)
		priv.GET("/runs", api.ListRuns)

		priv.GET("/portfolio/latest", api.GetLatestPortfolio)
		priv.GET("/flows/daily", api.GetDailyFlows)
		priv.GET("/flows/weekly", api.GetWeeklyFlows)
		priv.GET("/flows/daily_by_chain", api.GetDailyFlowsByChain)
		priv.GET("/transfers/recent", server.ListTransfers(gdb))

		// ✅ 前端/你自己查历史用的接口
		priv.GET("/market/binance/top", api.GetBinanceMarket)

		priv.POST("/orders/schedule", api.CreateScheduledOrder)
		priv.GET("/orders/schedule", api.ListScheduledOrders)
		priv.POST("/orders/schedule/:id/cancel", api.CancelScheduledOrder)
	}

	// ws
	r.GET("/ws/transfers", server.WSTransfers)

	fmt.Println("API listening at", *addr)
	if err := r.Run(*addr); err != nil {
		panic(err)
	}
}
