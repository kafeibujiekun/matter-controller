let ws;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
let reconnectInterval;

function connect() {
    ws = new WebSocket('ws://' + window.location.host + '/ws');
    
    ws.onmessage = function(event) {
        const message = JSON.parse(event.data);
        console.log('收到消息:', message);
        
        switch(message.type) {
            case 'matter_server_status':
                console.log('处理 Matter Server 状态更新:', message.data);
                window.matterServerStatus.handleUpdate(message.data);
                // window.matterServerInfo.handleUpdate(message.data);
                break;
            case 'matter_server_info':
                console.log('处理 Matter Server 信息更新:', message.data);
                window.matterServerInfo.handleUpdate(message.data);
                break;
            case 'device_list':
                console.log('处理设备列表更新:', message.data);
                updateDeviceList(message.data);
                break;
            case 'device_detail':
                updateDeviceDetail(message.data);
                break;
            case 'log':
                appendLog(message.data);
                break;
            case 'device_added':
                // 处理设备添加成功的响应
                alert('设备添加成功');
                updateDeviceList(message.data);
                break;
            case 'device_add_failed':
                // 处理设备添加失败的响应
                alert('设备添加失败：' + message.data.error);
                break;
            case 'commission_error':
                alert(message.data.error);
                break;
            case 'commission_sent':
                console.log(message.data.message);
                break;
            case 'device_event':
                console.log('处理设备事件:', message.data);
                window.deviceList.handleDeviceEvent(message.data);
                break;
            case 'error':
                console.error('收到错误消息:', message.data);
                break;
            default:
                console.warn('收到未知消息类型:', message.type, message);
        }
    };

    ws.onclose = function() {
        console.log('WebSocket连接关闭，尝试重连...');
        clearInterval(reconnectInterval);
        reconnectInterval = setInterval(function() {
            if (ws.readyState === WebSocket.CLOSED) {
                connect();
            }
        }, 5000);
    };

    ws.onerror = function(err) {
        console.error('WebSocket错误:', err);
    };
}

function appendLog(logData) {
    const logsContent = document.getElementById('logs-content');
    if (!logsContent) return;

    const logEntry = document.createElement('div');
    logEntry.className = 'log-entry';
    
    const time = new Date(logData.time).toLocaleTimeString();
    logEntry.innerHTML = `
        <span class="log-time">[${time}]</span>
        <span class="log-message">${logData.message}</span>
    `;
    
    logsContent.appendChild(logEntry);
    
    // 滚动到最新日志
    logsContent.scrollTop = logsContent.scrollHeight;
}

function updateDeviceList(devices) {
    const deviceList = document.getElementById('device-list');
    if (!deviceList) return;

    deviceList.innerHTML = '';
    
    for (const device of Object.values(devices)) {
        const li = document.createElement('li');
        li.className = 'device-item';
        li.innerHTML = `
            <div class="device-name">${device.name}</div>
            <div class="device-status">
                <span class="status-indicator" style="background-color: ${device.status === 'online' ? '#4CAF50' : '#999'}"></span>
                <span class="status-text">${device.status === 'online' ? '在线' : '离线'}</span>
            </div>
        `;
        
        // 修改点击事件
        li.addEventListener('click', () => {
            window.location.href = `/device-detail.html?id=${device.ID}`;
        });
        
        deviceList.appendChild(li);
    }
}

// 获取URL参数
function getUrlParam(param) {
    const urlParams = new URLSearchParams(window.location.search);
    return urlParams.get(param);
}

// 加载设备详情
function loadDeviceDetail() {
    const deviceId = getUrlParam('id');
    if (!deviceId) return;

    // 请求设备详情
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            type: 'get_device_detail',
            data: {
                device_id: deviceId
            }
        }));
    }
}

// 更新设备详情
function updateDeviceDetail(deviceData) {
    const deviceName = document.getElementById('device-name');
    const deviceStatus = document.getElementById('device-status');
    const deviceInfo = document.getElementById('device-info');
    
    if (deviceName) {
        deviceName.textContent = deviceData.name;
    }

    if (deviceStatus) {
        deviceStatus.innerHTML = `
            <span class="status-indicator" style="background-color: ${deviceData.status === 'online' ? '#4CAF50' : '#999'}"></span>
            <span class="status-text">${deviceData.status === 'online' ? '在线' : '离线'}</span>
        `;
    }
    
    if (deviceInfo) {
        deviceInfo.innerHTML = `
            <div class="info-item">
                <label>设备ID</label>
                <span>${deviceData.ID}</span>
            </div>
            <div class="info-item">
                <label>设备名称</label>
                <span>${deviceData.name}</span>
            </div>
            <!-- 可以添加更多设备信息 -->
        `;
    }
}

// 显示添加设备弹窗
function showAddDeviceModal() {
    const modal = document.getElementById('addDeviceModal');
    modal.classList.add('show');
}

// 隐藏添加设备弹窗
function hideAddDeviceModal() {
    const modal = document.getElementById('addDeviceModal');
    modal.classList.remove('show');
    // 清空输入框
    document.getElementById('network-code').value = '';
}

// 添加设备
function addDevice() {
    const networkCode = document.getElementById('network-code').value;
    if (!networkCode) {
        alert('请输入配网码');
        return;
    }

    console.log('发送配网请求，配网码:', networkCode); // 添加日志

    // 发送配网请求
    ws.send(JSON.stringify({
        type: 'add_device',
        data: {
            code: networkCode
        }
    }));

    // 隐藏弹窗
    hideAddDeviceModal();
}

// 点击遮罩层关闭弹窗
document.querySelector('.modal-overlay').addEventListener('click', hideAddDeviceModal);

// 阻止弹窗内容点击事件冒泡
document.querySelector('.modal-content').addEventListener('click', function(e) {
    e.stopPropagation();
});

// ESC键关闭弹窗
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        hideAddDeviceModal();
    }
});

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    console.log('页面加载完成，初始化 WebSocket 连接');
    // 初始化WebSocket连接
    connect();

    // 如果在设备详情页面，加载设备详情
    if (window.location.pathname === '/device-detail.html') {
        loadDeviceDetail();
    }

    // 如果在主页面，绑定添加设备按钮事件
    const addDeviceBtn = document.getElementById('addDeviceBtn');
    if (addDeviceBtn) {
        addDeviceBtn.addEventListener('click', showAddDeviceModal);
    }

    // 将函数暴露到全局作用域
    window.hideAddDeviceModal = hideAddDeviceModal;
    window.addDevice = addDevice;
}); 