package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type StrategyConditions struct {
	MeanReversionEnabled    bool    `json:"mean_reversion_enabled"`
	MeanReversionMode       string  `json:"mean_reversion_mode"`
	MeanReversionSubMode    string  `json:"mean_reversion_sub_mode"`
	MRBollingerBandsEnabled bool    `json:"mr_bollinger_bands_enabled"`
	MRRSIEnabled            bool    `json:"mr_rsi_enabled"`
	MRPriceChannelEnabled   bool    `json:"mr_price_channel_enabled"`
	MRPeriod                int     `json:"mr_period"`
	MRBollingerMultiplier   float64 `json:"mr_bollinger_multiplier"`
	MRRSIOversold           int     `json:"mr_rsi_oversold"`
	MRRSIOverbought         int     `json:"mr_rsi_overbought"`
	MRChannelPeriod         int     `json:"mr_channel_period"`
	MRMinReversionStrength  float64 `json:"mr_min_reversion_strength"`
	MRSignalMode            string  `json:"mr_signal_mode"`
}

type UpdateStrategyRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Conditions  StrategyConditions `json:"conditions"`
}

func main() {
	fmt.Println("🧪 测试更新策略ID=30为自适应模式")
	fmt.Println("=====================================")

	// 构造更新请求
	request := UpdateStrategyRequest{
		Name:        "优化均值回归",
		Description: "测试自适应模式",
		Conditions: StrategyConditions{
			MeanReversionEnabled:    true,
			MeanReversionMode:       "enhanced",
			MeanReversionSubMode:    "adaptive",
			MRBollingerBandsEnabled: true,
			MRRSIEnabled:            true,
			MRPriceChannelEnabled:   false,
			MRPeriod:                20,
			MRBollingerMultiplier:   2.0,
			MRRSIOversold:           30,
			MRRSIOverbought:         70,
			MRChannelPeriod:         20,
			MRMinReversionStrength:  0.5,
			MRSignalMode:            "ADAPTIVE_OSCILLATION",
		},
	}

	// 序列化JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("❌ JSON序列化失败: %v", err)
	}

	fmt.Printf("📤 发送请求数据:\n%s\n\n", string(jsonData))

	// 发送PUT请求到更新API
	url := "http://localhost:8080/api/strategies/30"
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("❌ 创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token") // 使用测试token

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("❌ 发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("❌ 读取响应失败: %v", err)
	}

	fmt.Printf("📥 响应状态: %s\n", resp.Status)
	fmt.Printf("📥 响应内容:\n%s\n", string(body))

	if resp.StatusCode == 200 {
		fmt.Println("✅ API调用成功")
	} else {
		fmt.Printf("❌ API调用失败，状态码: %d\n", resp.StatusCode)
	}
}
