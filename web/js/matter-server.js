// Matter Server 状态管理类
class MatterServerStatus {
    constructor() {
        console.log('初始化 MatterServerStatus');
        this.statusIndicator = document.querySelector('.status-indicator');
        this.statusText = document.querySelector('.status-text');
        this.updateStatus('connecting');
    }

    handleUpdate(data) {
        console.log('MatterServerStatus 收到状态更新:', data);
        if (data.status === 'connected') {
            this.updateStatus('connected');
        } else if (data.status === 'disconnected') {
            this.updateStatus('disconnected');
        } else if (data.status === 'connecting') {
            this.updateStatus('connecting');
        }
    }

    updateStatus(status) {
        console.log('更新连接状态为:', status);
        this.statusIndicator.classList.remove('status-connected', 'status-disconnected', 'status-connecting');
        let statusDisplay = '';
        
        switch(status) {
            case 'connected':
                this.statusIndicator.classList.add('status-connected');
                statusDisplay = 'Matter Server 已连接';
                break;
            case 'disconnected':
                this.statusIndicator.classList.add('status-disconnected');
                statusDisplay = 'Matter Server 已断开';
                break;
            case 'connecting':
                this.statusIndicator.classList.add('status-connecting');
                statusDisplay = 'Matter Server 连接中...';
                break;
        }
        
        console.log('更新状态显示为:', statusDisplay);
        this.statusText.textContent = statusDisplay;
    }
}

// Matter Server 信息管理类
class MatterServerInfo {
    constructor() {
        console.log('初始化 MatterServerInfo');
        this.serverInfo = document.querySelector('.server-info');
        this.clearInfo();
    }

    handleUpdate(data) {
        console.log('MatterServerInfo 收到信息更新:', data);
        if (data.info) {
            this.updateInfo(data.info);
        } else {
            this.clearInfo();
        }
    }

    updateInfo(info) {
        console.log('更新服务器信息:', info);
        // 将 Compressed Fabric ID 转换为十六进制，并保证 16 位
        const compressedFabricIdHex = '0x' + info.compressed_fabric_id.toString(16).toUpperCase().padStart(16, '0');
        
        this.serverInfo.innerHTML = `
            <div>Fabric ID: ${info.fabric_id}</div>
            <div>Compressed Fabric ID: ${compressedFabricIdHex}</div>
            <div>SDK Version: ${info.sdk_version}</div>
            <div>Schema Version: ${info.schema_version}</div>
        `;
    }

    clearInfo() {
        console.log('清除服务器信息');
        this.serverInfo.innerHTML = `
            <div>Fabric ID: --</div>
            <div>Compressed Fabric ID: --</div>
            <div>SDK Version: --</div>
            <div>Schema Version: --</div>
        `;
    }
}

// 创建实例
document.addEventListener('DOMContentLoaded', () => {
    console.log('页面加载完成，初始化 Matter Server 状态和信息管理');
    window.matterServerStatus = new MatterServerStatus();
    window.matterServerInfo = new MatterServerInfo();
});