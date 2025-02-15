// 将需要的变量声明在全局作用域
let navItems;
let pageTitle;
let contentArea;

// 页面配置
const pages = {
    overview: {
        title: '概览',
        content: '<ul id="device-list" class="device-list"></ul>'
    },
    logs: {
        title: '日志',
        content: `
            <div class="logs-container">
                <div id="logs-content" class="logs-content"></div>
            </div>
        `
    },
    settings: {
        title: '设置',
        content: '<div class="settings-container">设置内容</div>'
    }
};

// 将updatePage函数定义在全局作用域
function updatePage(pageId, data = {}) {
    // 确保DOM元素已经获取
    if (!pageTitle) pageTitle = document.querySelector('.page-title');
    if (!contentArea) contentArea = document.querySelector('.content-area');
    if (!navItems) navItems = document.querySelectorAll('.nav-item');

    // 更新标题
    pageTitle.textContent = pages[pageId].title;
    
    // 更新内容区域
    contentArea.innerHTML = pages[pageId].content;
    
    // 更新导航项active状态
    navItems.forEach(item => {
        item.classList.remove('active');
        if (item.dataset.page === pageId) {
            item.classList.add('active');
        }
    });

    // 控制添加设备按钮的显示
    const headerActions = document.querySelector('.header-actions');
    if (headerActions) {
        headerActions.style.display = pageId === 'overview' ? 'flex' : 'none';
    }

    // 如果是概览页面，初始化设备列表
    if (pageId === 'overview') {
        if (typeof initDeviceList === 'function') {
            initDeviceList();
        }
    }
}

// 初始化
document.addEventListener('DOMContentLoaded', function() {
    // 初始化DOM元素引用
    navItems = document.querySelectorAll('.nav-item');
    pageTitle = document.querySelector('.page-title');
    contentArea = document.querySelector('.content-area');

    // 为导航项添加点击事件
    navItems.forEach(item => {
        item.addEventListener('click', function() {
            const pageId = this.dataset.page;
            updatePage(pageId);
        });
    });

    // 初始化显示概览页面
    updatePage('overview');
});

// 确保updatePage在全局可用
window.updatePage = updatePage; 