package main

import (
	"fmt"
	"log"
	"os"

	"analysis/analysis_backend/internal/db"
)

func main() {
	fmt.Println("=== 测试数据库迁移 ===")

	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	}

	fmt.Printf("连接数据库: %s\n", dsn)

	// 连接数据库
	database, err := db.OpenMySQL(db.Options{
		DSN:         dsn,
		Automigrate: true, // 启用自动迁移
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()

	fmt.Println("✅ 数据库连接成功")

	// 获取数据库实例
	gdb, err := database.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	// 检查新表是否存在
	tables := []string{"external_operations", "operation_logs", "audit_trails"}

	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", table)
		err := gdb.Raw(query).Scan(&count).Error
		if err != nil {
			log.Printf("检查表 %s 失败: %v", table, err)
			continue
		}

		if count > 0 {
			fmt.Printf("✅ 表 %s 已存在\n", table)
		} else {
			fmt.Printf("❌ 表 %s 不存在\n", table)
		}
	}

	// 测试插入数据到新表
	fmt.Println("\n=== 测试插入数据 ===")

	// 测试插入ExternalOperation
	externalOp := &db.ExternalOperation{
		Symbol:       "TESTUSDT",
		OperationType: "external_test",
		OldAmount:    "100",
		NewAmount:    "150",
		Confidence:   0.95,
		Status:       "detected",
		Notes:        "测试外部操作记录",
	}

	err = gdb.Create(externalOp).Error
	if err != nil {
		log.Printf("插入ExternalOperation失败: %v", err)
	} else {
		fmt.Printf("✅ 成功插入ExternalOperation记录，ID: %d\n", externalOp.ID)
	}

	// 测试插入OperationLog
	opLog := &db.OperationLog{
		UserID:     1,
		EntityType: "test",
		EntityID:   1,
		Action:     "test_action",
		Description: "测试操作日志",
		Source:     "system",
		Level:      "info",
	}

	err = gdb.Create(opLog).Error
	if err != nil {
		log.Printf("插入OperationLog失败: %v", err)
	} else {
		fmt.Printf("✅ 成功插入OperationLog记录，ID: %d\n", opLog.ID)
	}

	// 测试插入AuditTrail
	auditTrail := &db.AuditTrail{
		UserID:       1,
		Action:       "test_action",
		ResourceType: "test_resource",
		ResourceID:   "test_123",
		Details:      "测试审计追踪",
		Success:      true,
	}

	err = gdb.Create(auditTrail).Error
	if err != nil {
		log.Printf("插入AuditTrail失败: %v", err)
	} else {
		fmt.Printf("✅ 成功插入AuditTrail记录，ID: %d\n", auditTrail.ID)
	}

	fmt.Println("\n=== 测试完成 ===")
}