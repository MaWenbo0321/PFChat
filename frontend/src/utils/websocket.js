import { ElMessage, ElMessageBox } from 'element-plus'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'

class WebSocketManager {
    constructor() {
        this.ws = null
        this.reconnectTimer = null
        this.heartbeatTimer = null
        this.reconnectAttempts = 0
        this.maxReconnectAttempts = 5
        this.grammarCheckCallback = null
    }

    connect(token) {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const wsUrl = `${protocol}//${window.location.hostname}:8080/ws?token=${token}`

        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
            console.log('WebSocket connected')
            const chatStore = useChatStore()
            chatStore.setWebSocket(this.ws)
            this.reconnectAttempts = 0
            this.startHeartbeat()
        }

        this.ws.onmessage = (event) => {
            this.handleMessage(JSON.parse(event.data))
        }

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error)
        }

        this.ws.onclose = () => {
            console.log('WebSocket disconnected')
            const chatStore = useChatStore()
            chatStore.connected = false
            this.stopHeartbeat()
            this.attemptReconnect()
        }
    }

    handleMessage(message) {
        const chatStore = useChatStore()
        const userStore = useUserStore()

        switch (message.type) {
            case 'message':
                // 收到新消息
                const msg = message.data
                const otherUserId = msg.sender_id === userStore.userInfo.id
                    ? msg.receiver_id
                    : msg.sender_id

                chatStore.addMessage(otherUserId, msg)

                // 如果不是当前聊天用户发的消息，显示通知
                if (chatStore.currentChatUser?.id !== otherUserId) {
                    ElMessage.info(`${msg.sender.username} 发来新消息`)
                }
                break

            case 'grammar_check':
                // 语法检查结果
                this.handleGrammarCheck(message.data)
                break
        }
    }

    handleGrammarCheck(data) {
        if (data.has_error) {
            ElMessageBox({
                title: '语法建议',
                message: `
          <div style="line-height: 1.6;">
            <p style="margin-bottom: 10px;"><strong>建议修改为:</strong></p>
            <p style="background: #f5f7fa; padding: 10px; border-radius: 4px; margin-bottom: 10px;">${data.suggestion}</p>
            <p style="margin-bottom: 5px;"><strong>说明:</strong></p>
            <p style="color: #606266;">${data.explanation}</p>
          </div>
        `,
                dangerouslyUseHTMLString: true,
                confirmButtonText: '复制建议',
                cancelButtonText: '取消',
                showCancelButton: true,
                type: 'warning'
            }).then(() => {
                // 复制到剪贴板
                navigator.clipboard.writeText(data.suggestion).then(() => {
                    ElMessage.success('已复制到剪贴板')
                })
            }).catch(() => {})
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
        }, 30000) // 30秒心跳
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
                    this.connect(userStore.token)
                }
            }, 3000 * this.reconnectAttempts) // 递增延迟
        } else {
            ElMessage.error('WebSocket 连接失败，请刷新页面重试')
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
    }
}

export default new WebSocketManager()