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
const getElementLocale = () => {
    return i18n.global.locale.value === 'zh-CN' ? zhCn : en
}

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(ElementPlus, {
    locale: getElementLocale()
})

// 监听语言变化,动态更新 Element Plus 语言
import { watch } from 'vue'
watch(() => i18n.global.locale.value, () => {
    // 重新配置 Element Plus 语言
    app.config.globalProperties.$ELEMENT = {
        locale: getElementLocale()
    }
})

app.mount('#app')