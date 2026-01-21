package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	// 读取配置文件
	cfg, err := loadConfig("analysis_backend/config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 查询某个交易对的时间范围
	symbol := "BTCUSDT"
	var minTime, maxTime time.Time

	gdb, _ := db.DB()

	// 查询最早时间
	err = gdb.Model(&pdb.MarketKline{}).
		Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Select("MIN(open_time) as min_time").
		Scan(&minTime).Error

	if err != nil {
		log.Fatalf("查询最早时间失败: %v", err)
	}

	// 查询最晚时间
	err = gdb.Model(&pdb.MarketKline{}).
		Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Select("MAX(open_time) as max_time").
		Scan(&maxTime).Error

	if err != nil {
		log.Fatalf("查询最晚时间失败: %v", err)
	}

	fmt.Printf("交易对: %s\n", symbol)
	fmt.Printf("最早时间: %s\n", minTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("最晚时间: %s\n", maxTime.Format("2006-01-02 15:04:05"))

	duration := maxTime.Sub(minTime)
	days := duration.Hours() / 24
	fmt.Printf("总时间跨度: %.1f 天\n", days)

	// 查询最近200条记录的时间范围
	var recentKlines []struct {
		OpenTime time.Time
	}

	err = gdb.Model(&pdb.MarketKline{}).
		Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(200).
		Select("open_time").
		Find(&recentKlines).Error

	if err != nil {
		log.Fatalf("查询最近200条记录失败: %v", err)
	}

	if len(recentKlines) > 0 {
		latest := recentKlines[0].OpenTime
		oldest := recentKlines[len(recentKlines)-1].OpenTime

		fmt.Printf("\n最近200条记录时间范围:\n")
		fmt.Printf("最新: %s\n", latest.Format("2006-01-02 15:04:05"))
		fmt.Printf("最早: %s\n", oldest.Format("2006-01-02 15:04:05"))

		recentDuration := latest.Sub(oldest)
		recentDays := recentDuration.Hours() / 24
		fmt.Printf("时间跨度: %.1f 天 (%.0f 小时)\n", recentDays, recentDuration.Hours())
	}
}

func loadConfig(configPath string) (*config.Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var cfg config.Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}

func connectDatabase(dbConfig struct {
	DSN          string `yaml:"dsn"`
	Automigrate  bool   `yaml:"automigrate"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}) (pdb.Database, error) {
	options := pdb.Options{
		DSN:          dbConfig.DSN,
		Automigrate:  false,
		MaxOpenConns: dbConfig.MaxOpenConns,
		MaxIdleConns: dbConfig.MaxIdleConns,
	}

	return pdb.OpenMySQL(options)
}
