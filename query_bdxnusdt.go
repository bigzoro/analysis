package main

import (
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
)

func main() {
	// 获取数据库连接
	gdb, err := pdb.GetDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer gdb.Close()

	fmt.Println("=== BDXNUSDT 交易对分析 ===")

	// 查询基本信息
	info, err := db.GetExchangeInfo(gdb, "BDXNUSDT")
	if err != nil {
		fmt.Printf("❌ 查询BDXNUSDT信息失败: %v\n", err)
		return
	}

	fmt.Printf("📊 基本信息:\n")
	fmt.Printf("  交易对: %s\n", info.Symbol)
	fmt.Printf("  状态: %s\n", info.Status)
	fmt.Printf("  市场类型: %s\n", info.MarketType)
	fmt.Printf("  基础资产: %s\n", info.BaseAsset)
	fmt.Printf("  计价资产: %s\n", info.QuoteAsset)
	fmt.Printf("  活跃状态: %v\n", info.IsActive)

	if info.DeactivatedAt != nil {
		fmt.Printf("  下架时间: %v\n", info.DeactivatedAt.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  下架时间: 未下架\n")
	}

	if info.LastSeenActive != nil {
		fmt.Printf("  最后活跃时间: %v\n", info.LastSeenActive.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  最后活跃时间: 无记录\n")
	}

	fmt.Printf("  创建时间: %v\n", info.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("  更新时间: %v\n", info.UpdatedAt.Format("2006-01-02 15:04:05"))

	// 查询活跃状态统计
	stats, err := db.GetExchangeInfoStats(gdb)
	if err != nil {
		fmt.Printf("❌ 查询统计信息失败: %v\n", err)
	} else {
		fmt.Printf("\n📈 整体统计:\n")
		fmt.Printf("  总交易对数: %d\n", stats["total"])
		fmt.Printf("  活跃交易对数: %d\n", stats["active"])
		fmt.Printf("  非活跃交易对数: %d\n", stats["inactive"])
	}

	// 查询最近下架的交易对
	recentlyDeactivated, err := db.GetRecentlyDeactivatedSymbols(gdb, "spot", time.Now().Add(-24*time.Hour))
	if err != nil {
		fmt.Printf("❌ 查询最近下架交易对失败: %v\n", err)
	} else {
		fmt.Printf("\n🗑️  最近24小时下架的交易对:\n")
		for _, symbol := range recentlyDeactivated {
			if symbol.Symbol == "BDXNUSDT" {
				fmt.Printf("  ✅ BDXNUSDT 于 %v 下架\n", symbol.DeactivatedAt.Format("2006-01-02 15:04:05"))
				break
			}
		}

		found := false
		for _, symbol := range recentlyDeactivated {
			if symbol.Symbol == "BDXNUSDT" {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("  ℹ️  BDXNUSDT 不在最近下架列表中\n")
		}
	}

	// 检查是否在活跃交易对列表中
	activeSymbols, err := db.GetUSDTTradingPairs(gdb)
	if err != nil {
		fmt.Printf("❌ 查询活跃交易对失败: %v\n", err)
	} else {
		fmt.Printf("\n🎯 活跃状态检查:\n")
		isActive := false
		for _, symbol := range activeSymbols {
			if symbol == "BDXNUSDT" {
				isActive = true
				break
			}
		}

		if isActive {
			fmt.Printf("  ✅ BDXNUSDT 在活跃交易对列表中\n")
		} else {
			fmt.Printf("  ❌ BDXNUSDT 不在活跃交易对列表中\n")
		}

		fmt.Printf("  当前活跃USDT交易对总数: %d\n", len(activeSymbols))
	}

	fmt.Println("\n=== 分析完成 ===")
}