import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/router'

// 创建 axios 实例
const instance = axios.create({
    baseURL: '/api',
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json'
    }
})

// 请求拦截器
instance.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// 响应拦截器
// 响应拦截器
instance.interceptors.response.use(
    response => {
        // 🔧 直接返回 data,后端已经返回正确格式
        return response.data
    },
    error => {
        if (error.response) {
            const { status, data } = error.response

            if (status === 401) {
                ElMessage.error('登录已过期，请重新登录')
                localStorage.removeItem('token')
                localStorage.removeItem('userInfo')
                router.push('/login')
            } else if (status === 403) {
                ElMessage.error('权限不足')
            } else if (status === 500) {
                // 🔧 确保显示后端返回的错误信息
                ElMessage.error(data.error || '服务器错误')
            } else {
                ElMessage.error(data.error || '请求失败')
            }
        } else {
            ElMessage.error('网络错误')
        }
        return Promise.reject(error)
    }
)

// API 接口
const api = {
    // 用户认证
    register: (data) => instance.post('/register', data),
    login: (data) => instance.post('/login', data),

    // 用户相关
    getUsers: () => instance.get('/users'),
    getUserInfo: (id) => instance.get(`/users/${id}`),


    // 消息相关
    getMessages: (userId) => instance.get(`/messages/${userId}`),
    sendMessage: (data) => instance.post('/messages', data),
    checkMessageBeforeSend: (data) => instance.post('/messages/check', data),  // 🔧 添加这行
    deleteMessage: (id) => instance.delete(`/messages/${id}`),

    clearChatHistory: (userId) => instance.delete(`/messages/clear/${userId}`),
    deleteMyMessages: (userId) => instance.delete(`/messages/mine/${userId}`),

    // 语法错误相关
    getGrammarErrors: (errorType = 'all') => {
        const params = errorType && errorType !== 'all' ? { error_type: errorType } : {}
        return instance.get('/grammar-errors', { params })
    },
    deleteGrammarError: (id) => instance.delete(`/grammar-errors/${id}`),
    batchDeleteGrammarErrors: (ids) => instance.post('/grammar-errors/batch-delete', { ids }),
    clearGrammarErrorsByType: (type) => instance.delete(`/grammar-errors/clear/${type}`),
    updateGrammarErrorType: (id, errorType) =>
        instance.put(`/grammar-errors/${id}/type`, { error_type: errorType }),

    // 管理员接口
    admin: {
        getAllUsers: () => instance.get('/admin/users'),
        deleteUser: (id) => instance.delete(`/admin/users/${id}`),
        updateUserRole: (id, role) => instance.put(`/admin/users/${id}/role`, { role }),
        getStats: () => instance.get('/admin/stats')
    }
}

export default api
