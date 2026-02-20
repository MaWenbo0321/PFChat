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
    }

    connect() {
        const userStore = useUserStore()
        const token = userStore.token

        if (!token) {
            console.error('No token found, cannot connect WebSocket')
            return
        }

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

        this.ws.onclose = () => {
            console.log('WebSocket disconnected')
            const chatStore = useChatStore()
            chatStore.closeWebSocket()
            this.stopHeartbeat()
            this.attemptReconnect()
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

                // 如果不是当前聊天用户发的消息,显示通知
                if (chatStore.currentUser?.id !== otherUserId) {
                    ElMessage.info(`${msg.sender.username} 发来新消息`)
                }
                break
            }

            case 'grammar_check': {
                // 🆕 不再使用弹窗, 只触发消息刷新让 Chat.vue 中的 loadMessageErrors 重新加载
                // 通过 handlers 通知 Chat.vue 刷新
                break
            }

            case 'receiver_error_notify': {
                // 🆕 接收方错误通知: 不弹窗, 只通知刷新
                // 消息旁边会显示小标识, 无需弹窗
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
            this.messageHandlers.push(handler)
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
        if (this.reconnectAttempts < this.maxReconnectAttempts) {
            this.reconnectAttempts++
            console.log(`Attempting to reconnect... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)

            this.reconnectTimer = setTimeout(() => {
                const userStore = useUserStore()
                if (userStore.token) {
                    this.connect()
                }
            }, 3000 * this.reconnectAttempts)
        } else {
            ElMessage.error('WebSocket 连接失败,请刷新页面重试')
        }
    }

    disconnect() {
        this.stopHeartbeat()
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer)
        }
        if (this.ws) {
            this.ws.close()
            this.ws = null
        }
        this.messageHandlers = []
    }
}

export default new WebSocketManager()