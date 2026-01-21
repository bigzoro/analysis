package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	// 连接到WebSocket
	url := "ws://127.0.0.1:8010/ws/realtime-gainers"
	fmt.Printf("连接到: %s\n", url)

	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close()

	// 发送futures订阅请求
	subscription := map[string]interface{}{
		"action":   "subscribe",
		"kind":     "futures",
		"category": "trading",
		"limit":    15,
		"interval": 20,
	}

	fmt.Printf("发送订阅请求: %+v\n", subscription)
	if err := conn.WriteJSON(subscription); err != nil {
		log.Fatal("发送订阅失败:", err)
	}

	// 读取响应
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	for i := 0; i < 3; i++ {
		var response map[string]interface{}
		err := conn.ReadJSON(&response)
		if err != nil {
			fmt.Printf("读取响应失败: %v\n", err)
			break
		}

		fmt.Printf("收到响应 %d: %+v\n", i+1, response)

		// 如果是数据更新，打印详细信息
		if response["type"] == "gainers_update" {
			gainers := response["gainers"].([]interface{})
			fmt.Printf("数据条数: %d\n", len(gainers))
			if len(gainers) > 0 {
				first := gainers[0].(map[string]interface{})
				fmt.Printf("第一条数据: %s = %.2f%%\n",
					first["symbol"], first["price_change_24h"])
			}
		}
	}

	fmt.Println("测试完成")
}