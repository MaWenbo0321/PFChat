import { ElMessage } from 'element-plus'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'

class WebSocketManager {
    constructor() {
        this.ws = null
        this.reconnectTimer = null
        this.heartbeatTimer = null
        this.reconnectAttempts = 0
        this.maxReconnectAttempts = 5
        this.messageHandlers = []
        // 【Fix】标记是否为主动断开（logout 时触发），主动断开不触发重连
        this.manualDisconnect = false
    }

    connect() {
        // 【Fix】如果已有连接且状态正常，不重复连接（防止重复挂载时建立多条连接）
        if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
            console.log('WebSocket already connected or connecting, skip')
            return
        }

        const userStore = useUserStore()
        const token = userStore.token

        if (!token) {
            console.error('No token found, cannot connect WebSocket')
            return
        }

        this.manualDisconnect = false

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//8.148.76.156:8080/ws?token=${token}`

        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
            console.log('WebSocket connected')
            const chatStore = useChatStore()
            chatStore.setWebSocket(this.ws)
            this.reconnectAttempts = 0
            this.startHeartbeat()
        }

        this.ws.onmessage = (event) => {
            try {
                const message = JSON.parse(event.data)
                this.handleMessage(message)
            } catch (error) {
                console.error('Failed to parse WebSocket message:', error)
            }
        }

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error)
        }

        this.ws.onclose = (event) => {
            console.log('WebSocket disconnected, code:', event.code, 'manual:', this.manualDisconnect)
            this.stopHeartbeat()

            // 【Fix】通知 chatStore 连接断开，但不调用 ws.close()（连接已经关了）
            const chatStore = useChatStore()
            chatStore.setConnected(false)

            // 【Fix】只有非主动断开才触发重连（避免 logout / 页面卸载时的无效重连）
            if (!this.manualDisconnect) {
                this.attemptReconnect()
            }
        }
    }

    handleMessage(message) {
        const chatStore = useChatStore()
        const userStore = useUserStore()

        switch (message.type) {
            case 'message': {
                const msg = message.data
                const otherUserId = msg.sender_id === userStore.userInfo.id
                    ? msg.receiver_id
                    : msg.sender_id

                chatStore.addMessage(otherUserId, msg)

                if (chatStore.currentUser?.id !== otherUserId) {
                    ElMessage.info(`${msg.sender.username} 发来新消息`)
                }
                break
            }

            case 'grammar_check': {
                // 通过 handlers 通知 Chat.vue 刷新错误标识
                break
            }

            case 'receiver_error_notify': {
                // 通过 handlers 通知 Chat.vue 刷新错误标识
                break
            }
        }

        // 调用所有注册的消息处理函数
        this.messageHandlers.forEach(handler => {
            try {
                handler(message)
            } catch (error) {
                console.error('Error in message handler:', error)
            }
        })
    }

    onMessage(handler) {
        if (typeof handler === 'function') {
            // 【Fix】防止重复注册同一个 handler 引用
            if (!this.messageHandlers.includes(handler)) {
                this.messageHandlers.push(handler)
            }
        }
    }

    offMessage(handler) {
        const index = this.messageHandlers.indexOf(handler)
        if (index > -1) {
            this.messageHandlers.splice(index, 1)
        }
    }

    sendMessage(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({
                type: 'message',
                data: data,
                timestamp: new Date().toISOString()
            }))
        }
    }

    startHeartbeat() {
        this.stopHeartbeat() // 先清理旧定时器，防止重叠
        this.heartbeatTimer = setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({ type: 'ping' }))
            }
        }, 30000)
    }

    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer)
            this.heartbeatTimer = null
        }
    }

    attemptReconnect() {
        // 【Fix】取消上一个待执行的重连定时器，防止叠加触发
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
            this.reconnectTimer = null
        }

        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++
            const delay = 3000 * this.reconnectAttempts
            console.log(`Reconnecting in ${delay}ms... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

            this.reconnectTimer = setTimeout(() => {
                this.reconnectTimer = null
                const userStore = useUserStore()
                // 双重检查：token 存在且仍是非主动断开状态
                if (userStore.token && !this.manualDisconnect) {
                    this.connect()
                }
            }, delay)
        } else {
            ElMessage.error('WebSocket 连接失败，请刷新页面重试')
        }
    }

    // 【Fix】logout 或页面彻底销毁时调用，完全断开不再重连
    disconnect() {
        this.manualDisconnect = true
        this.stopHeartbeat()
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
            this.reconnectTimer = null
        }
        if (this.ws) {
            this.ws.close()
            this.ws = null
        }
        this.reconnectAttempts = 0
        // 【Fix 关键】不清空 messageHandlers！
        // 原来此处有 this.messageHandlers = []，导致 Chat.vue 重新挂载后 handlers 丢失
        // handlers 由各组件自己通过 offMessage 注销
    }
}

export default new WebSocketManager()