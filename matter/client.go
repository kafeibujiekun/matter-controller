package matter

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MatterMessage 定义 Matter Server 消息结构
type MatterMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// Client Matter Server 客户端
type Client struct {
	conn                 *websocket.Conn
	url                  string
	status               string
	statusChan           chan string
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
		statusChan:           make(chan string, 10),
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

// GetStatus 获取当前连接状态
func (c *Client) GetStatus() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

// StatusChan 返回状态更新通道
func (c *Client) StatusChan() <-chan string {
	return c.statusChan
}

// ReceiveChan 返回消息接收通道
func (c *Client) ReceiveChan() <-chan MatterMessage {
	return c.receiveChan
}

// Send 发送消息到 Matter Server
func (c *Client) Send(msg MatterMessage) error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return fmt.Errorf("未连接到 Matter Server")
	}
	c.mu.Unlock()

	c.sendChan <- msg
	return nil
}

// 更新状态
func (c *Client) updateStatus(status string) {
	c.mu.Lock()
	if c.status != status {
		c.status = status
		c.mu.Unlock()
		c.statusChan <- status
		c.logger.Printf("Matter Server 状态更新: %s", status)
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

		var msg MatterMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.logger.Printf("解析消息错误: %v", err)
			continue
		}

		c.logger.Printf("收到消息: %+v", msg)
		c.receiveChan <- msg
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
