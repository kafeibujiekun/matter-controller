// main.go
package main

import (
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

var (
	devices   = make(map[string]Device)
	clients   = make(map[*websocket.Conn]bool)
	clientsMu sync.Mutex
	broadcast = make(chan Message)
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有来源
		},
	}
	logger        *log.Logger
	matterClient  *matter.Client
	matterStarted bool
	cfg           *Config
)

// 初始化日志
func initLogger() error {
	// 创建logs目录
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建日志文件（按日期）
	logFile := filepath.Join(logsDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	// 设置日志输出到文件和控制台
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger = log.New(multiWriter, "", log.Ldate|log.Ltime)

	logger.Printf("日志系统初始化完成，日志文件: %s", logFile)
	return nil
}

// 启动 Matter Server 连接
func startMatterServer() {
	clientsMu.Lock()
	if !matterStarted {
		logger.Println("检测到首个客户端连接，启动 Matter Server 连接...")
		matterClient = matter.NewClient(
			cfg.MatterServer.Address,
			logger,
		)

		// 设置重连参数
		matterClient.SetReconnectParams(
			cfg.MatterServer.MaxReconnectAttempts,
			time.Duration(cfg.MatterServer.ReconnectDelay)*time.Millisecond,
		)

		matterClient.Start()
		matterStarted = true

		// 处理 Matter Server 状态更新
		go func() {
			for status := range matterClient.StatusChan() {
				broadcast <- Message{
					Type: "matter_server_status",
					Data: map[string]string{
						"status": status,
					},
				}
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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Printf("升级WebSocket失败: %v", err)
		return
	}
	defer conn.Close()

	// 记录客户端连接信息
	clientAddr := conn.RemoteAddr().String()
	logger.Printf("新客户端连接: %s", clientAddr)

	clientsMu.Lock()
	clients[conn] = true
	clientCount := len(clients)
	clientsMu.Unlock()

	// 如果是第一个客户端，启动 Matter Server
	if clientCount == 1 {
		startMatterServer()
	}

	// 发送当前设备列表
	deviceList := Message{
		Type: "device_list",
		Data: devices,
	}
	clientsMu.Lock()
	if err := conn.WriteJSON(deviceList); err != nil {
		logger.Printf("发送设备列表失败: %v", err)
	}
	clientsMu.Unlock()

	// 发送当前 Matter Server 状态
	if matterStarted {
		status := matterClient.GetStatus()
		conn.WriteJSON(Message{
			Type: "matter_server_status",
			Data: map[string]string{
				"status": status,
			},
		})
	}

	// 处理客户端消息
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Printf("WebSocket错误 [%s]: %v", clientAddr, err)
			} else {
				logger.Printf("客户端断开连接: %s", clientAddr)
			}

			clientsMu.Lock()
			delete(clients, conn)
			clientCount := len(clients)
			clientsMu.Unlock()

			// 如果没有客户端连接了，停止 Matter Server
			if clientCount == 0 {
				stopMatterServer()
			}
			break
		}

		logger.Printf("收到消息 [%s] - 类型: %s, 内容: %+v", clientAddr, msg.Type, msg.Data)

		switch msg.Type {
		case "add_device":
			if deviceData, ok := msg.Data.(map[string]interface{}); ok {
				if networkCode, ok := deviceData["network_code"].(string); ok {
					// 生成随机设备ID
					deviceID := fmt.Sprintf("device_%d", time.Now().Unix())

					// 创建新设备
					newDevice := Device{
						ID:     deviceID,
						Name:   fmt.Sprintf("设备_%s", networkCode),
						Status: "online",
					}

					// 添加设备到设备列表
					clientsMu.Lock()
					devices[deviceID] = newDevice
					clientsMu.Unlock()

					// 广播设备更新消息
					broadcast <- Message{
						Type: "device_added",
						Data: devices,
					}

					logger.Printf("添加新设备成功: %+v", newDevice)
				} else {
					// 发送添加失败消息
					conn.WriteJSON(Message{
						Type: "device_add_failed",
						Data: map[string]string{
							"error": "无效的配网码格式",
						},
					})
					logger.Printf("添加设备失败: 无效的配网码格式")
				}
			} else {
				// 发送添加失败消息
				conn.WriteJSON(Message{
					Type: "device_add_failed",
					Data: map[string]string{
						"error": "无效的消息格式",
					},
				})
				logger.Printf("添加设备失败: 无效的消息格式")
			}
		case "get_device_detail":
			if deviceData, ok := msg.Data.(map[string]interface{}); ok {
				if deviceID, ok := deviceData["device_id"].(string); ok {
					clientsMu.Lock()
					if device, exists := devices[deviceID]; exists {
						conn.WriteJSON(Message{
							Type: "device_detail",
							Data: device,
						})
						logger.Printf("发送设备详情: %+v", device)
					} else {
						conn.WriteJSON(Message{
							Type: "device_detail_error",
							Data: map[string]string{
								"error": "设备不存在",
							},
						})
						logger.Printf("设备详情请求失败: 设备不存在 (ID: %s)", deviceID)
					}
					clientsMu.Unlock()
				}
			}
		}
	}
}

func handleBroadcast() {
	for msg := range broadcast {
		clientsMu.Lock()
		logger.Printf("广播消息 - 类型: %s, 内容: %+v", msg.Type, msg.Data)

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					logger.Printf("广播消息错误: %v", err)
				}
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
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

	// 启动广播处理
	go handleBroadcast()

	// 静态文件服务
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	http.Handle("/js/", http.FileServer(http.Dir("web")))

	// 路由处理
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			// 主页
			http.ServeFile(w, r, "web/templates/index.html")
		case "/device-detail.html":
			// 设备详情页
			http.ServeFile(w, r, "web/templates/device-detail.html")
		default:
			// 404 处理
			http.NotFound(w, r)
		}
	})

	// WebSocket 处理
	http.HandleFunc("/ws", handleWebSocket)

	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	logger.Printf("服务器启动在 http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
