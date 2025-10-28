import axios from 'axios'
import { ElMessage } from 'element-plus'
import { useUserStore } from '@/stores/user'

const api = axios.create({
    baseURL: '/api',
    timeout: 10000
})

// 请求拦截器
api.interceptors.request.use(
    config => {
        const userStore = useUserStore()
        if (userStore.token) {
            config.headers.Authorization = `Bearer ${userStore.token}`
        }
        return config
    },
    error => {
        return Promise.reject(error)
    }
)

// 响应拦截器
api.interceptors.response.use(
    response => {
        return response.data
    },
    error => {
        if (error.response) {
            const { status, data } = error.response
            if (status === 401) {
                ElMessage.error('登录已过期，请重新登录')
                const userStore = useUserStore()
                userStore.logout()
                window.location.href = '/login'
            } else if (status === 403) {
                ElMessage.error('权限不足，拒绝访问')
            } else {
                ElMessage.error(data.error || '请求失败')
            }
        } else {
            ElMessage.error('网络错误，请检查连接')
        }
        return Promise.reject(error)
    }
)

// API 方法
export default {
    // 用户认证
    register(data) {
        return api.post('/register', data)
    },
    login(data) {
        return api.post('/login', data)
    },

    // 用户相关
    getUsers() {
        return api.get('/users')
    },
    getUserInfo(id) {
        return api.get(`/users/${id}`)
    },

    // 消息相关
    getMessages(userId) {
        return api.get(`/messages/${userId}`)
    },
    sendMessage(data) {
        return api.post('/messages', data)
    },
    deleteMessage(messageId) {
        return api.delete(`/messages/${messageId}`)
    },
    clearChatHistory(userId) {
        return api.delete(`/messages/clear/${userId}`)
    },
    deleteMyMessages(userId) {
        return api.delete(`/messages/mine/${userId}`)
    },

    // 语法错误
    getGrammarErrors() {
        return api.get('/grammar-errors')
    },
    deleteGrammarError(id) {
        return api.delete(`/grammar-errors/${id}`)
    },

    // 管理员相关 API

    // 获取所有用户（管理员）
    getAllUsersForAdmin(params = {}) {
        return api.get('/admin/users', { params })
    },

    // 删除用户（管理员）
    deleteUser(userId) {
        return api.delete(`/admin/users/${userId}`)
    },

    // 更新用户角色（管理员）
    updateUserRole(userId, data) {
        return api.put(`/admin/users/${userId}/role`, data)
    },

    // 获取系统统计信息（管理员）
    getUserStats() {
        return api.get('/admin/stats')
    },

    // 辅助方法：检查用户是否为管理员
    isAdmin() {
        const userStore = useUserStore()
        return userStore.userInfo?.role === 'admin'
    },

    // 辅助方法：获取当前用户角色
    getCurrentUserRole() {
        const userStore = useUserStore()
        return userStore.userInfo?.role || 'user'
    }
}