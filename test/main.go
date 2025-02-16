// main.go
package main

import (
	"encoding/json"
	"fmt"
)

// Config 定义配置结构体
type Config struct {
	FabricID                  int    `json:"fabric_id"`
	CompressedFabricID        uint64 `json:"compressed_fabric_id"`
	SchemaVersion             int    `json:"schema_version"`
	MinSupportedSchemaVersion int    `json:"min_supported_schema_version"`
	SDKVersion                string `json:"sdk_version"`
	WifiCredentialsSet        bool   `json:"wifi_credentials_set"`
	ThreadCredentialsSet      bool   `json:"thread_credentials_set"`
	BluetoothEnabled          bool   `json:"bluetooth_enabled"`
}

func main() {
	// JSON数据
	jsonData := `{
		"fabric_id": 1,
		"compressed_fabric_id": 10904920470098451262,
		"schema_version": 11,
		"min_supported_schema_version": 9,
		"sdk_version": "2024.11.4",
		"wifi_credentials_set": false,
		"thread_credentials_set": false,
		"bluetooth_enabled": true
	}`

	// 创建Config实例
	var config Config

	// 解析JSON
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		fmt.Printf("解析JSON出错: %v\n", err)
		return
	}

	// 打印解析结果
	fmt.Printf("Fabric ID: %d\n", config.FabricID)
	fmt.Printf("Compressed Fabric ID: %d\n", config.CompressedFabricID)
	fmt.Printf("Schema Version: %d\n", config.SchemaVersion)
	fmt.Printf("Min Supported Schema Version: %d\n", config.MinSupportedSchemaVersion)
	fmt.Printf("SDK Version: %s\n", config.SDKVersion)
	fmt.Printf("WiFi Credentials Set: %v\n", config.WifiCredentialsSet)
	fmt.Printf("Thread Credentials Set: %v\n", config.ThreadCredentialsSet)
	fmt.Printf("Bluetooth Enabled: %v\n", config.BluetoothEnabled)
}
