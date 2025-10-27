import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useChatStore = defineStore('chat', () => {
    const users = ref([])
    const messages = ref({}) // { userId: [messages] }
    const currentChatUser = ref(null)
    const ws = ref(null)
    const connected = ref(false)

    const currentMessages = computed(() => {
        if (!currentChatUser.value) return []
        return messages.value[currentChatUser.value.id] || []
    })

    const setUsers = (userList) => {
        users.value = userList
    }

    const setCurrentChatUser = (user) => {
        currentChatUser.value = user
    }

    const setMessages = (userId, messageList) => {
        messages.value[userId] = messageList
    }

    const addMessage = (userId, message) => {
        if (!messages.value[userId]) {
            messages.value[userId] = []
        }
        messages.value[userId].push(message)
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
        currentMessages,
        ws,
        connected,
        setUsers,
        setCurrentChatUser,
        setMessages,
        addMessage,
        setWebSocket,
        closeWebSocket
    }
})