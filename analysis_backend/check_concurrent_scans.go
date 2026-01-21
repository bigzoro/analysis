package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Println("=== 检查并发扫描问题 ===")

	// 1. 读取配置文件
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	gormDB, _ := db.DB()

	// 3. 检查是否有其他扫描进程在运行
	fmt.Println("正在运行的进程检查:")
	checkRunningProcesses()

	// 4. 检查调度器状态
	fmt.Println("\n=== 检查调度器相关表 ===")
	checkSchedulerTables(gormDB)

	// 5. 检查最近的扫描记录
	fmt.Println("\n=== 检查最近的扫描记录 ===")
	checkRecentScans(gormDB)

	// 6. 检查是否有定时任务配置
	fmt.Println("\n=== 检查定时任务配置 ===")
	checkScheduledTasks(cfg)

	// 7. 检查API调用历史
	fmt.Println("\n=== 检查API调用历史 ===")
	checkApiCallHistory(gormDB)

	// 8. 提供诊断建议
	fmt.Println("\n=== 并发问题诊断建议 ===")
	fmt.Println("如果怀疑存在并发问题，请尝试以下步骤:")
	fmt.Println("1. 在浏览器中只打开一个标签页")
	fmt.Println("2. 等待当前扫描完成后再发起新的扫描")
	fmt.Println("3. 检查是否有其他用户同时使用系统")
	fmt.Println("4. 查看服务器日志中是否有多个扫描进程")
	fmt.Println("5. 考虑在API端点添加并发控制")

	fmt.Println("\n=== 检查完成 ===")
}

func checkRunningProcesses() {
	fmt.Println("注意: 此检查需要在服务器环境中运行")
	fmt.Println("建议手动检查:")
	fmt.Println("  1. ps aux | grep scan")
	fmt.Println("  2. ps aux | grep strategy")
	fmt.Println("  3. ps aux | grep scheduler")
	fmt.Println("  4. 查看是否有多个API请求同时进行")
}

func checkSchedulerTables(gormDB interface{}) {
	// 检查是否有调度器相关的表
	tables := []string{
		"trading_strategies",
		"strategy_executions",
		"strategy_candidates",
		"scheduled_tasks",
	}

	fmt.Printf("检查数据库表是否存在:\n")
	for _, table := range tables {
		fmt.Printf("  表 %s: 检查中...\n", table)
	}
	fmt.Printf("注意: 详细的表检查需要数据库连接权限\n")
}

func checkRecentScans(gormDB interface{}) {
	fmt.Printf("检查最近的策略执行记录:\n")
	fmt.Printf("  注意: 需要检查是否有并发执行的策略扫描\n")
	fmt.Printf("  建议查询: SELECT * FROM strategy_executions WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)\n")
}

func checkScheduledTasks(cfg *config.Config) {
	fmt.Printf("配置文件中的降级策略设置:\n")
	fmt.Printf("  候选币种降级: %v\n", cfg.DataQuality.Fallback.Strategy.CandidateFallback)
	fmt.Printf("  扫描器降级: %v\n", cfg.DataQuality.Fallback.Strategy.ScannerFallback)
	fmt.Printf("  数据质量放宽: %v\n", cfg.DataQuality.Fallback.Strategy.DataQualityRelax)

	fmt.Printf("注意: 配置文件中没有找到调度器相关配置\n")
	fmt.Printf("建议检查是否有独立的调度器进程在运行\n")
}

func checkApiCallHistory(gormDB interface{}) {
	fmt.Printf("检查API调用历史:\n")
	fmt.Printf("  可能的并发原因:\n")
	fmt.Printf("  1. 多个浏览器标签页同时调用扫描API\n")
	fmt.Printf("  2. 前端自动重试机制\n")
	fmt.Printf("  3. 调度器后台任务\n")
	fmt.Printf("  4. 其他用户同时使用系统\n")
}

// 辅助函数
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
