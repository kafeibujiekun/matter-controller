// main.go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"matter-controller/matter"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	MatterServer MatterServerConfig `yaml:"matter_server"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type MatterServerConfig struct {
	Address              string `yaml:"address"`
	MaxReconnectAttempts int    `yaml:"max_reconnect_attempts"`
	ReconnectDelay       int    `yaml:"reconnect_delay"`
}

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ClientMessage 定义前端发送的消息结构
type ClientMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// AddDeviceData 定义添加设备消息的数据结构
type AddDeviceData struct {
	Code string `json:"code"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求
	},
}

var (
	devices       = make(map[string]Device)
	clients       = make(map[*websocket.Conn]bool)
	clientsMu     sync.Mutex
	logger        *log.Logger
	matterClient  *matter.Client
	matterStarted bool
	cfg           *Config
)

// 加载配置文件
func loadConfig(filename string) (*Config, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	cfg := &Config{}
	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return cfg, nil
}

// 初始化日志
func initLogger() error {
	// 创建日志目录
	if err := os.MkdirAll("logs", 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开日志文件
	logFile, err := os.OpenFile(
		filepath.Join("logs", "app.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建多输出
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// 创建不同模块的日志记录器
	logger = log.New(multiWriter, "[Main] ", log.LstdFlags)

	return nil
}

// 启动 Matter Server 连接
func startMatterServer() {
	clientsMu.Lock()
	if !matterStarted {
		logger.Println("检测到首个客户端连接，启动 Matter Server 连接...")
		matterClient = matter.NewClient(
			cfg.MatterServer.Address,
			log.New(logger.Writer(), "[Matter] ", log.LstdFlags),
		)

		matterClient.SetReconnectParams(
			cfg.MatterServer.MaxReconnectAttempts,
			time.Duration(cfg.MatterServer.ReconnectDelay)*time.Millisecond,
		)

		matterClient.Start()
		matterStarted = true

		// 处理 Matter Server 状态更新
		go func() {
			for status := range matterClient.StatusChan() {
				broadcastToClients(BroadcastMessage{
					Type: "matter_server_status",
					Data: status,
				})
			}
		}()

		// 处理 Matter Server 消息
		go func() {
			for msg := range matterClient.ReceiveChan() {
				logger.Printf("收到 Matter Server 消息: %+v", msg)
				// TODO: 处理接收到的消息
			}
		}()
	}
	clientsMu.Unlock()
}

// 停止 Matter Server 连接
func stopMatterServer() {
	clientsMu.Lock()
	if matterStarted && len(clients) == 0 {
		logger.Println("所有客户端已断开，停止 Matter Server 连接...")
		matterClient.Stop()
		matterStarted = false
	}
	clientsMu.Unlock()
}

// broadcastToClients 向所有连接的客户端发送消息
func broadcastToClients(message BroadcastMessage) {
	clientsMu.Lock()
	clientCount := len(clients)
	logger.Printf("开始广播消息到 %d 个客户端: %+v", clientCount, message)

	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			logger.Printf("发送消息到客户端 %s 失败: %v", client.RemoteAddr(), err)
			client.Close()
			delete(clients, client)
		} else {
			logger.Printf("成功发送消息到客户端 %s", client.RemoteAddr())
		}
	}
	clientsMu.Unlock()
}

// handleWebSocket 处理 WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("升级 WebSocket 连接失败: %v", err)
		return
	}

	// 添加到 clients map
	clientsMu.Lock()
	clients[conn] = true
	clientCount := len(clients)
	clientsMu.Unlock()

	logger.Printf("新的 WebSocket 连接: %s, 当前连接数: %d", conn.RemoteAddr(), clientCount)

	// 发送当前状态
	status := matterClient.GetCurrentStatus()
	logger.Printf("向客户端 %s 发送当前状态: %+v", conn.RemoteAddr(), status)
	err = conn.WriteJSON(BroadcastMessage{
		Type: status.GetType(),
		Data: status,
	})
	if err != nil {
		logger.Printf("发送状态失败: %v", err)
		conn.Close()
		return
	}

	// 如果有服务器信息，发送服务器信息
	if info, exists := matterClient.GetCurrentInfo(); exists {
		logger.Printf("向客户端 %s 发送服务器信息: %+v", conn.RemoteAddr(), info)
		err = conn.WriteJSON(BroadcastMessage{
			Type: info.GetType(),
			Data: info,
		})
		if err != nil {
			logger.Printf("发送服务器信息失败: %v", err)
			conn.Close()
			return
		}
	} else {
		logger.Printf("客户端 %s: 当前没有服务器信息", conn.RemoteAddr())
	}

	// 清理连接
	defer func() {
		clientsMu.Lock()
		delete(clients, conn)
		remainingClients := len(clients)
		clientsMu.Unlock()
		conn.Close()
		logger.Printf("客户端 %s 断开连接, 剩余连接数: %d", conn.RemoteAddr(), remainingClients)
	}()

	// 保持连接并处理接收到的消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Printf("读取客户端 %s 消息错误: %v", conn.RemoteAddr(), err)
			}
			break
		}

		// 打印接收到的消息
		logger.Printf("收到客户端 %s 消息内容: %s", conn.RemoteAddr(), string(message))

		// 解析消息
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			logger.Printf("解析客户端消息失败: %v", err)
			continue
		}

		logger.Printf("解析的消息: 类型=%s, 数据=%+v", clientMsg.Type, clientMsg.Data)

		// 处理不同类型的消息
		switch clientMsg.Type {
		case "add_device":
			// 将 interface{} 转换为 map
			dataMap, ok := clientMsg.Data.(map[string]interface{})
			if !ok {
				logger.Printf("无效的 add_device 数据格式")
				continue
			}

			// 获取配网码
			code, ok := dataMap["code"].(string)
			if !ok {
				logger.Printf("无效的配网码格式")
				continue
			}

			logger.Printf("收到配网请求，配网码: %s", code)

			// 发送配网请求
			err := matterClient.SendCommissionRequest(code)
			if err != nil {
				logger.Printf("发送配网请求失败: %v", err)
				// 可以在这里向前端发送错误消息
				conn.WriteJSON(BroadcastMessage{
					Type: "error",
					Data: map[string]string{
						"message": "配网请求失败: " + err.Error(),
					},
				})
			} else {
				logger.Printf("配网请求已发送")
			}

		default:
			logger.Printf("未知的消息类型: %s", clientMsg.Type)
		}
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// 只处理根路径的请求
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	logger.Printf("处理首页请求: %s", r.URL.Path)
	http.ServeFile(w, r, "web/templates/index.html")
}

func main() {
	// 加载配置
	var err error
	cfg, err = loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志系统
	if err := initLogger(); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}

	// 初始化 Matter Client
	matterClient = matter.NewClient(cfg.MatterServer.Address, logger)

	// 启动 Matter Client
	matterClient.Start()

	// 设置路由
	mux := http.NewServeMux()

	// 首页路由
	mux.HandleFunc("/", handleIndex)

	// WebSocket 路由
	mux.HandleFunc("/ws", handleWebSocket)

	// 静态文件服务
	// 处理 /static/ 路径下的所有静态资源（css、pics）
	staticFS := http.FileServer(http.Dir("web"))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// 处理 JavaScript 文件
	mux.Handle("/js/", staticFS)

	// 启动服务器
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	logger.Printf("服务器启动在 http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal("ListenAndServe: ", err)
	}
}
