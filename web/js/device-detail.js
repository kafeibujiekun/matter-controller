let ws;
const deviceId = new URLSearchParams(window.location.search).get('id');

function connectWebSocket() {
    ws = new WebSocket('ws://localhost:8080/ws');

    ws.onopen = () => {
        console.log('WebSocket连接已建立');
        requestDeviceDetail();
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        if (message.type === 'device_detail' && message.data.id === deviceId) {
            updateDeviceDetail(message.data);
        }
    };

    ws.onclose = () => {
        console.log('WebSocket连接已关闭');
        setTimeout(connectWebSocket, 3000);
    };
}

function requestDeviceDetail() {
    ws.send(JSON.stringify({
        type: 'get_device_detail',
        data: { id: deviceId }
    }));
}

function updateDeviceDetail(device) {
    // 更新设备基本信息
    document.getElementById('device-name').textContent = device.name;
    document.getElementById('manufacturer').textContent = 'Zemismart Technology Limited';
    document.getElementById('firmware').textContent = '1.10';
    document.getElementById('hardware').textContent = '1.0';

    // 更新设备详细信息
    document.getElementById('node-id').textContent = '4';
    document.getElementById('network-type').textContent = 'Wi-Fi';
    document.getElementById('device-type').textContent = '终端设备';
    document.getElementById('network-name').textContent = 'HONOR-103W6W';
    document.getElementById('mac-address').textContent = '38:01:8c:00:05:46';
    document.getElementById('ip-address').textContent = device.ip || '未知';

    // 更新开关状态
    document.getElementById('device-switch').checked = device.status === '在线';
}

// 开关状态改变事件
document.getElementById('device-switch').addEventListener('change', function(e) {
    ws.send(JSON.stringify({
        type: 'update_device_status',
        data: {
            id: deviceId,
            status: e.target.checked ? '在线' : '离线'
        }
    }));
});

window.onload = connectWebSocket; 