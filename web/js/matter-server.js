class MatterServerStatus {
    constructor() {
        this.statusIndicator = document.querySelector('.status-indicator');
        this.statusText = document.querySelector('.status-text');
        this.updateStatus('connecting');
    }

    updateStatus(status) {
        this.statusIndicator.classList.remove('status-connected', 'status-disconnected', 'status-connecting');
        
        switch(status) {
            case 'connected':
                this.statusIndicator.classList.add('status-connected');
                this.statusText.textContent = 'Matter Server 已连接';
                break;
            case 'disconnected':
                this.statusIndicator.classList.add('status-disconnected');
                this.statusText.textContent = 'Matter Server 已断开';
                break;
            case 'connecting':
                this.statusIndicator.classList.add('status-connecting');
                this.statusText.textContent = 'Matter Server 连接中...';
                break;
        }
    }
}

// 创建状态实例
document.addEventListener('DOMContentLoaded', () => {
    window.matterServerStatus = new MatterServerStatus();
});