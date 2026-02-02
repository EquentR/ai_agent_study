// Common utility functions for Phase1 applications

// API配置
const API_BASE_URL = '/api/v1';

/**
 * 转义HTML特殊字符，防止XSS攻击
 * @param {string} str - 需要转义的字符串
 * @returns {string} 转义后的字符串
 */
function escapeHtml(str) {
    if (!str) return '';
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

/**
 * 转义JavaScript字符串中的特殊字符
 * @param {string} str - 需要转义的字符串
 * @returns {string} 转义后的字符串
 */
function escapeJs(str) {
    if (!str) return '';
    return str.replace(/\\/g, '\\\\')
              .replace(/'/g, "\\'")
              .replace(/"/g, '\\"')
              .replace(/\n/g, '\\n')
              .replace(/\r/g, '\\r');
}

/**
 * 格式化日期时间
 * @param {string} dateStr - 日期字符串
 * @returns {string} 格式化后的日期时间
 */
function formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

/**
 * 显示Toast消息
 * @param {string} message - 消息内容
 * @param {string} type - 消息类型: success, error, info
 */
function showToast(message, type = 'success') {
    // 查找或创建toast元素
    let toast = document.getElementById('toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'toast';
        toast.className = 'toast';
        document.body.appendChild(toast);
    }

    toast.textContent = message;
    toast.className = `toast ${type} show`;

    setTimeout(() => {
        toast.classList.remove('show');
    }, 3000);
}

/**
 * API请求封装
 * @param {string} url - 请求URL
 * @param {object} options - fetch选项
 * @returns {Promise<object>} API响应数据
 */
async function apiRequest(url, options = {}) {
    try {
        const response = await fetch(url, {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            },
            ...options
        });

        const result = await response.json();
        return result;
    } catch (error) {
        console.error('API request error:', error);
        throw error;
    }
}

/**
 * GET请求
 * @param {string} endpoint - API端点
 * @returns {Promise<object>} API响应数据
 */
async function apiGet(endpoint) {
    return apiRequest(`${API_BASE_URL}${endpoint}`, {
        method: 'GET'
    });
}

/**
 * POST请求
 * @param {string} endpoint - API端点
 * @param {object} data - 请求数据
 * @returns {Promise<object>} API响应数据
 */
async function apiPost(endpoint, data) {
    return apiRequest(`${API_BASE_URL}${endpoint}`, {
        method: 'POST',
        body: JSON.stringify(data)
    });
}

/**
 * PUT请求
 * @param {string} endpoint - API端点
 * @param {object} data - 请求数据
 * @returns {Promise<object>} API响应数据
 */
async function apiPut(endpoint, data) {
    return apiRequest(`${API_BASE_URL}${endpoint}`, {
        method: 'PUT',
        body: JSON.stringify(data)
    });
}

/**
 * DELETE请求
 * @param {string} endpoint - API端点
 * @returns {Promise<object>} API响应数据
 */
async function apiDelete(endpoint) {
    return apiRequest(`${API_BASE_URL}${endpoint}`, {
        method: 'DELETE'
    });
}

/**
 * 格式化Token数量，添加千位分隔符
 * @param {number} num - Token数量
 * @returns {string} 格式化后的数字
 */
function formatTokenCount(num) {
    if (!num && num !== 0) return '-';
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

/**
 * 格式化延迟时间
 * @param {number} ms - 毫秒数
 * @returns {string} 格式化后的时间字符串
 */
function formatLatency(ms) {
    if (!ms && ms !== 0) return '-';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}

/**
 * 复制文本到剪贴板
 * @param {string} text - 要复制的文本
 * @returns {Promise<void>}
 */
async function copyToClipboard(text) {
    try {
        await navigator.clipboard.writeText(text);
        showToast('已复制到剪贴板', 'success');
    } catch (error) {
        // 降级方案
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
            showToast('已复制到剪贴板', 'success');
        } catch (err) {
            showToast('复制失败', 'error');
        }
        document.body.removeChild(textarea);
    }
}
