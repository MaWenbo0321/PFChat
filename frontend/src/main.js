import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import en from 'element-plus/dist/locale/en.mjs'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import i18n from './i18n'

const app = createApp(App)
const pinia = createPinia()

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
}

// 根据当前语言设置 Element Plus 的语言
const elementLocale = i18n.global.locale.value === 'zh-CN' ? zhCn : en

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(ElementPlus, {
    locale: elementLocale
})

app.mount('#app')