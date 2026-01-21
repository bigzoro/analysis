package main

import (
	"fmt"
	"time"
)

type TestStruct struct {
	Field1 string
	Field2 string
	Field3 string
}

func safeTimestamp() int64 {
	ts := time.Now().Unix()
	if ts > 999999999 {
		ts = ts % 1000000000
	}
	return ts
}

func main() {
	// 测试handlers.go中的语法
	test1 := &TestStruct{
		Field1: "value1",
		Field2: func() string {
			safeEntryID := uint(12345) % 10000000
			safeExternalID := uint(67890) % 1000000
			return fmt.Sprintf("EC_%d_%d", safeEntryID, safeExternalID)
		}(),
		Field3: "value3",
	}

	fmt.Printf("Test1: %+v\n", test1)

	// 测试scheduler.go中的语法
	shortReason := "整体止损"
	switch shortReason {
	case "整体止损":
		shortReason = "STOP_LOSS"
	case "整体止盈":
		shortReason = "TAKE_PROFIT"
	case "整体止损止盈":
		shortReason = "STOP_ALL"
	default:
		if len(shortReason) > 8 {
			shortReason = shortReason[:8]
		}
	}

	test2 := &TestStruct{
		Field1: "value1",
		Field2: fmt.Sprintf("OC_%s_%d_%d", shortReason, uint(123), safeTimestamp()),
		Field3: "value3",
	}

	fmt.Printf("Test2: %+v\n", test2)

	fmt.Println("语法测试通过！")
}
