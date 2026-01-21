package main

import (
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	log.Println("🧪 测试 AutoMigrate 是否正常工作...")

	// 加载配置
	var cfg config.Config
	config.MustLoad("config.yaml", &cfg)
	config.ApplyProxy(&cfg)

	// 连接数据库（启用AutoMigrate）
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  true, // 启用AutoMigrate
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer gdb.Close()

	log.Println("✅ AutoMigrate 完成，没有出现索引冲突错误！")
	log.Println("🎉 问题已修复！")
}
