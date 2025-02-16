package matter

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message 消息接口
type Message interface {
	GetType() string
}

// MatterMessage 定义 Matter Server 消息结构
type MatterMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// CommissionRequest 配网请求结构
type CommissionRequest struct {
	MessageID string                `json:"message_id"`
	Command   string                `json:"command"`
	Args      CommissionRequestArgs `json:"args"`
}

// CommissionRequestArgs 配网请求参数
type CommissionRequestArgs struct {
	Code        string `json:"code"`
	NetworkOnly bool   `json:"network_only"`
}

// ServerInfo Matter Server 信息
type ServerInfo struct {
	FabricID                  int    `json:"fabric_id"`
	CompressedFabricID        uint64 `json:"compressed_fabric_id"`
	SchemaVersion             int    `json:"schema_version"`
	MinSupportedSchemaVersion int    `json:"min_supported_schema_version"`
	SDKVersion                string `json:"sdk_version"`
	WifiCredentialsSet        bool   `json:"wifi_credentials_set"`
	ThreadCredentialsSet      bool   `json:"thread_credentials_set"`
	BluetoothEnabled          bool   `json:"bluetooth_enabled"`
}

// StatusMessage 状态消息结构
type StatusMessage struct {
	Status string `json:"status"`
}

// InfoMessage 信息消息结构
type InfoMessage struct {
	Info ServerInfo `json:"info"`
}

// GetType 实现 Message 接口
func (s StatusMessage) GetType() string {
	return "matter_server_status"
}

// GetType 实现 Message 接口
func (i InfoMessage) GetType() string {
	return "matter_server_info"
}

// Client Matter Server 客户端
type Client struct {
	conn                 *websocket.Conn
	url                  string
	status               string
	serverInfo           *ServerInfo // 添加服务器信息存储
	statusChan           chan Message
	sendChan             chan MatterMessage
	receiveChan          chan MatterMessage
	logger               *log.Logger
	mu                   sync.Mutex
	reconnectDelay       time.Duration
	maxReconnectAttempts int
	stop                 chan struct{}
}

// NewClient 创建新的 Matter Server 客户端
func NewClient(url string, logger *log.Logger) *Client {
	return &Client{
		url:                  url,
		status:               "disconnected",
		statusChan:           make(chan Message, 10),
		sendChan:             make(chan MatterMessage, 100),
		receiveChan:          make(chan MatterMessage, 100),
		logger:               logger,
		reconnectDelay:       3 * time.Second,
		maxReconnectAttempts: 5,
		stop:                 make(chan struct{}),
	}
}

// Start 启动客户端
func (c *Client) Start() {
	go c.connectLoop()
	go c.processMessages()
}

// Stop 停止客户端
func (c *Client) Stop() {
	close(c.stop)
	if c.conn != nil {
		c.conn.Close()
	}
}

// GetCurrentStatus 获取当前状态
func (c *Client) GetCurrentStatus() StatusMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	status := StatusMessage{
		Status: c.status,
	}
	c.logger.Printf("获取当前状态: %+v", status)
	return status
}

// GetCurrentInfo 获取当前服务器信息
func (c *Client) GetCurrentInfo() (InfoMessage, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.serverInfo != nil {
		info := InfoMessage{
			Info: *c.serverInfo,
		}
		c.logger.Printf("获取当前服务器信息: %+v", info)
		return info, true
	}
	c.logger.Printf("当前没有服务器信息")
	return InfoMessage{}, false
}

// StatusChan 返回状态更新通道
func (c *Client) StatusChan() <-chan Message {
	return c.statusChan
}

// ReceiveChan 返回消息接收通道
func (c *Client) ReceiveChan() <-chan MatterMessage {
	return c.receiveChan
}

// Send 发送消息到 Matter Server
func (c *Client) Send(msg interface{}) error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("未连接到 Matter Server")
	}

	err := c.conn.WriteJSON(msg)
	c.mu.Unlock()

	if err != nil {
		c.logger.Printf("发送消息错误: %v", err)
		return err
	}

	c.logger.Printf("发送消息成功: %+v", msg)
	return nil
}

// 更新状态
func (c *Client) updateStatus(status string) {
	c.mu.Lock()
	if c.status != status {
		oldStatus := c.status
		c.status = status
		// 如果断开连接，清除服务器信息
		if status == "disconnected" {
			c.serverInfo = nil
			c.logger.Printf("连接断开，清除服务器信息")
		}
		c.mu.Unlock()
		c.logger.Printf("Matter Server 状态从 %s 更新为: %s", oldStatus, status)
	} else {
		c.mu.Unlock()
	}
}

// 连接循环
func (c *Client) connectLoop() {
	attempts := 0
	for {
		select {
		case <-c.stop:
			return
		default:
			if attempts >= c.maxReconnectAttempts {
				c.logger.Printf("达到最大重连次数 (%d), 等待后重试", c.maxReconnectAttempts)
				c.updateStatus("disconnected")
				time.Sleep(c.reconnectDelay * 2)
				attempts = 0
				continue
			}

			c.logger.Printf("尝试连接到 Matter Server: %s (尝试 %d/%d)", c.url, attempts+1, c.maxReconnectAttempts)
			c.updateStatus("connecting")

			conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
			if err != nil {
				c.logger.Printf("连接失败: %v", err)
				attempts++
				time.Sleep(c.reconnectDelay)
				continue
			}

			c.mu.Lock()
			c.conn = conn
			c.mu.Unlock()

			c.updateStatus("connected")
			c.logger.Printf("连接成功")
			attempts = 0

			// 启动消息处理
			go c.readMessages()
			go c.writeMessages()

			// 等待连接断开
			<-c.stop
			c.updateStatus("disconnected")
			conn.Close()

			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()

			time.Sleep(c.reconnectDelay)
		}
	}
}

// 处理接收到的消息
func (c *Client) readMessages() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.logger.Printf("读取消息错误: %v", err)
			c.stop <- struct{}{}
			return
		}

		c.logger.Printf("收到 Matter Server 原始消息: %s", string(message))
		message = []byte(strings.ReplaceAll(string(message), "\n", ""))

		// 首先尝试解析为 ServerInfo
		var serverInfo ServerInfo
		if err := json.Unmarshal(message, &serverInfo); err == nil {
			c.logger.Printf("成功解析为 ServerInfo: %+v", serverInfo)

			// 保存服务器信息
			c.mu.Lock()
			c.serverInfo = &serverInfo
			c.status = "connected"
			c.mu.Unlock()

			continue
		}

		// 如果不是 ServerInfo，则尝试解析为 MatterMessage
		var msg MatterMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Printf("解析为 MatterMessage 失败: %v", err)
			continue
		}

		// 只有当消息类型不为空时才处理
		if msg.Type != "" {
			c.logger.Printf("成功解析为 MatterMessage: %+v", msg)
			c.receiveChan <- msg
		} else {
			c.logger.Printf("收到空消息类型，忽略")
		}
	}
}

// 处理发送消息
func (c *Client) writeMessages() {
	for {
		select {
		case msg := <-c.sendChan:
			c.mu.Lock()
			if c.conn == nil {
				c.mu.Unlock()
				continue
			}
			err := c.conn.WriteJSON(msg)
			c.mu.Unlock()

			if err != nil {
				c.logger.Printf("发送消息错误: %v", err)
				c.stop <- struct{}{}
				return
			}
			c.logger.Printf("发送消息: %+v", msg)
		case <-c.stop:
			return
		}
	}
}

// 处理消息
func (c *Client) processMessages() {
	for {
		select {
		case msg := <-c.receiveChan:
			// 根据消息类型处理
			switch msg.Type {
			case "device_event":
				c.handleDeviceEvent(msg)
			case "command_response":
				c.handleCommandResponse(msg)
			default:
				c.logger.Printf("未知消息类型: %s", msg.Type)
			}
		case <-c.stop:
			return
		}
	}
}

// 处理设备事件
func (c *Client) handleDeviceEvent(msg MatterMessage) {
	c.logger.Printf("处理设备事件: %+v", msg)
	// TODO: 实现设备事件处理逻辑
}

// 处理命令响应
func (c *Client) handleCommandResponse(msg MatterMessage) {
	c.logger.Printf("处理命令响应: %+v", msg)
	// TODO: 实现命令响应处理逻辑
}

// SetReconnectParams 设置重连参数
func (c *Client) SetReconnectParams(maxAttempts int, delay time.Duration) {
	c.maxReconnectAttempts = maxAttempts
	c.reconnectDelay = delay
}

// SendCommissionRequest 发送配网请求
func (c *Client) SendCommissionRequest(code string) error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("未连接到 Matter Server")
	}
	c.mu.Unlock()

	// 生成唯一的消息ID
	messageID := fmt.Sprintf("%d", time.Now().UnixNano())

	request := CommissionRequest{
		MessageID: messageID,
		Command:   "commission_with_code",
		Args: CommissionRequestArgs{
			Code:        code,
			NetworkOnly: true,
		},
	}

	jsonBytes, _ := json.Marshal(request)
	c.logger.Printf("发送配网请求: %s", string(jsonBytes))
	return c.Send(request)
}
