import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSessionStore = defineStore('session', () => {
    const currentSession = ref(null)
    const botUser = ref(null)
    const messages = ref([])

    const setSession = (session) => {
        currentSession.value = session
    }

    const setBotUser = (user) => {
        botUser.value = user
    }

    const setMessages = (msgs) => {
        messages.value = msgs
    }

    const addMessage = (msg) => {
        messages.value.push(msg)
    }

    const clearSession = () => {
        currentSession.value = null
        botUser.value = null
        messages.value = []
    }

    const incrementRound = () => {
        if (currentSession.value) {
            currentSession.value.round_count = (currentSession.value.round_count || 0) + 1
        }
    }

    const setRoundCount = (roundCount) => {
        if (currentSession.value && Number.isInteger(roundCount) && roundCount >= 0) {
            currentSession.value.round_count = roundCount
        }
    }

    return {
        currentSession,
        botUser,
        messages,
        setSession,
        setBotUser,
        setMessages,
        addMessage,
        clearSession,
        incrementRound,
        setRoundCount
    }
})
