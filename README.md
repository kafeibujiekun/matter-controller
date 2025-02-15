# Matter Controller

Matter Controller 是一个用于管理和控制 Matter 设备的 Web 应用程序。它提供了一个直观的用户界面，用于监控和控制支持 Matter 协议的智能设备。

## 功能特性

- 实时显示 Matter Server 连接状态
- 设备管理和监控
- 支持设备配网
- 实时状态更新
- 响应式 Web 界面

## 技术栈

- 后端：Go
- 前端：HTML, CSS, JavaScript
- WebSocket 用于实时通信
- Matter Server 集成

## 安装和运行

1. 克隆仓库：
```bash
git clone [repository-url]
cd matter-controller
```

2. 安装依赖：

```bash
go mod download
```

3. 配置：

```bash
# 复制配置文件模板：
cp config.yaml.example config.yaml

# 根据需要修改 `config.yaml` 中的配置
```
* 配置说明

配置文件 `config.yaml` 包含以下配置项：

```yaml
server:
  port: "8080" # Web 服务器端口
  host: "0.0.0.0" # Web 服务器监听地址
matter_server:
  address: "ws://ip-address:5580/ws" # Matter Server WebSocket 地址
  max_reconnect_attempts: 5 # 最大重连次数
  reconnect_delay: 3000 # 重连延迟（毫秒）
```

4. 运行：

```bash
go run main.go
```

5. 访问应用：
打开浏览器访问 `http://localhost:8080`

## 开发
### 项目结构
├── main.go # 主程序入口
├── config.yaml # 配置文件
├── matter/
│ └── client.go # Matter Server 客户端
├── web/
│ ├── static/ # 静态资源
│ │ ├── css/ # 样式文件
│ │ ├── js/ # JavaScript 文件
│ │ └── pics/ # 图片资源
│ └── templates/ # HTML 模板
└── README.md

### 构建
```bash
go build
```
