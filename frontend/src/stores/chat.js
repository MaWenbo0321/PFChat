import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useChatStore = defineStore('chat', () => {
    const users = ref([])
    const messages = ref({}) // { userId: [messages] }
    const currentChatUser = ref(null)
    const ws = ref(null)
    const connected = ref(false)

    // 计算属性 - 当前聊天用户
    const currentUser = computed(() => currentChatUser.value)

    // 计算属性 - 当前消息列表
    const currentMessages = computed(() => {
        if (!currentChatUser.value) return []
        return messages.value[currentChatUser.value.id] || []
    })
    const draftMessages = ref({}) // { userId: draftText }
    // 新增3个方法:
    const saveDraft = (userId, text) => {
        if (userId) {
            draftMessages.value[userId] = text
        }
    }

    const getDraft = (userId) => {
        if (!userId) return ''
        return draftMessages.value[userId] || ''
    }

    const clearDraft = (userId) => {
        if (userId) {
            delete draftMessages.value[userId]
        }
    }



    // 计算属性 - 连接状态
    const isConnected = computed(() => connected.value)

    const setUsers = (userList) => {
        users.value = userList
    }

    const setCurrentChatUser = (user) => {
        currentChatUser.value = user
    }

    // 别名方法
    const selectUser = (user) => {
        currentChatUser.value = user
    }

    const setCurrentMessages = (messageList) => {
        if (currentChatUser.value) {
            messages.value[currentChatUser.value.id] = messageList
        }
    }

    const setMessages = (userId, messageList) => {
        messages.value[userId] = messageList
    }

    const addMessage = (userId, message) => {
        if (!messages.value[userId]) {
            messages.value[userId] = []
        }
        // 避免重复
        const exists = messages.value[userId].some(m => m.id === message.id)
        if (!exists) {
            messages.value[userId].push(message)
        }
    }

    const removeMessage = (messageId) => {
        if (currentChatUser.value) {
            const userId = currentChatUser.value.id
            if (messages.value[userId]) {
                messages.value[userId] = messages.value[userId].filter(
                    msg => msg.id !== messageId
                )
            }
        }
    }

    const setWebSocket = (websocket) => {
        ws.value = websocket
        connected.value = true
    }

    const closeWebSocket = () => {
        if (ws.value) {
            ws.value.close()
            ws.value = null
        }
        connected.value = false
    }

    return {
        users,
        messages,
        currentChatUser,
        currentUser,
        currentMessages,
        ws,
        connected,
        isConnected,
        setUsers,
        setCurrentChatUser,
        selectUser,
        setCurrentMessages,
        setMessages,
        addMessage,
        removeMessage,
        setWebSocket,
        closeWebSocket,

        draftMessages,
        saveDraft,
        getDraft,
        clearDraft
    }
})