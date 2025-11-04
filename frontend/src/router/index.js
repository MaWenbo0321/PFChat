import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const routes = [
    {
        path: '/login',
        name: 'Login',
        component: () => import('@/views/Login.vue'),
        meta: { requiresAuth: false }
    },
    {
        path: '/chat',
        name: 'Chat',
        component: () => import('@/views/Chat.vue'),
        meta: { requiresAuth: true }
    },
    {
        path: '/grammar-errors',
        name: 'GrammarErrors',
        component: () => import('@/views/GrammarErrors.vue'),
        meta: { requiresAuth: true }
    },
    {
        path: '/admin',
        name: 'AdminPanel',
        component: () => import('@/views/AdminPanel.vue'),
        meta: {
            requiresAuth: true,
            requiresAdmin: true
        }
    }
]

const router = createRouter({
    history: createWebHistory(),
    routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
    const userStore = useUserStore()

    // 检查是否需要认证
    if (to.meta.requiresAuth && !userStore.token) {
        next('/login')
        return
    }

    // 检查是否需要管理员权限
    if (to.meta.requiresAdmin) {
        if (!userStore.userInfo || userStore.userInfo.role !== 'admin') {
            // 如果不是管理员，重定向到主页
            next('/')
            return
        }
    }

    // 如果已登录用户访问登录页，重定向到主页
    if (to.path === '/login' && userStore.token) {
        next('/')
        return
    }

    next()
})

export default router