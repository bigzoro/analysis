package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	pdb "analysis/internal/db"
)

func main() {
	// 数据库连接
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	fmt.Println("Connected to database successfully")

	// 查询所有订单
	var orders []pdb.ScheduledOrder
	if err := db.Find(&orders).Error; err != nil {
		log.Fatalf("Failed to query orders: %v", err)
	}

	fmt.Printf("Found %d orders\n", len(orders))

	// 查找开仓和平仓订单对
	for _, order := range orders {
		if order.ReduceOnly {
			// 这是平仓订单，设置parent_order_id
			if order.ParentOrderId == 0 {
				// 查找对应的开仓订单（相同的symbol和用户，reduce_only=false）
				var parentOrder pdb.ScheduledOrder
				if err := db.Where("user_id = ? AND symbol = ? AND reduce_only = ? AND status = 'filled'",
					order.UserID, order.Symbol, false).First(&parentOrder).Error; err == nil {

					// 更新平仓订单的parent_order_id
					db.Model(&pdb.ScheduledOrder{}).Where("id = ?", order.ID).Update("parent_order_id", parentOrder.ID)
					fmt.Printf("Updated close order %d with parent_order_id %d\n", order.ID, parentOrder.ID)

					// 更新开仓订单的close_order_ids
					var closeOrderIds []uint
					if parentOrder.CloseOrderIds != "" {
						parts := strings.Split(parentOrder.CloseOrderIds, ",")
						for _, part := range parts {
							if id, err := strconv.ParseUint(strings.TrimSpace(part), 10, 32); err == nil {
								closeOrderIds = append(closeOrderIds, uint(id))
							}
						}
					}

					// 检查是否已经包含了这个ID
					found := false
					for _, id := range closeOrderIds {
						if id == order.ID {
							found = true
							break
						}
					}

					if !found {
						closeOrderIds = append(closeOrderIds, order.ID)
						var idsStr []string
						for _, id := range closeOrderIds {
							idsStr = append(idsStr, strconv.FormatUint(uint64(id), 10))
						}

						db.Model(&pdb.ScheduledOrder{}).Where("id = ?", parentOrder.ID).Update("close_order_ids", strings.Join(idsStr, ","))
						fmt.Printf("Updated parent order %d with close_order_ids %s\n", parentOrder.ID, strings.Join(idsStr, ","))
					}
				}
			}
		}
	}

	fmt.Println("Order associations setup completed!")
}
