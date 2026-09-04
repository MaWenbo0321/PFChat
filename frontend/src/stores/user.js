import { defineStore } from 'pinia'
import { ref } from 'vue'

const readStoredUser = () => {
    try {
        return JSON.parse(localStorage.getItem('userInfo') || 'null')
    } catch {
        localStorage.removeItem('userInfo')
        localStorage.removeItem('token')
        return null
    }
}

export const useUserStore = defineStore('user', () => {
    const token = ref(localStorage.getItem('token') || '')
    const userInfo = ref(readStoredUser())

    const setToken = (newToken) => {
        token.value = newToken
        localStorage.setItem('token', newToken)
    }

    const setUserInfo = (info) => {
        userInfo.value = info
        localStorage.setItem('userInfo', JSON.stringify(info))
    }

    const logout = () => {
        token.value = ''
        userInfo.value = null
        localStorage.removeItem('token')
        localStorage.removeItem('userInfo')
    }

    return {
        token,
        userInfo,
        setToken,
        setUserInfo,
        logout
    }
})
