import { createI18n } from 'vue-i18n'

// 中文语言包
const zhCN = {
    // 登录注册
    login: {
        title: '即时通讯系统',
        loginTab: '登录',
        registerTab: '注册',
        username: '用户名',
        password: '密码',
        country: '国家',
        selectCountry: '请选择国家',
        loginBtn: '登录',
        registerBtn: '注册',
        usernameRequired: '请输入用户名',
        passwordRequired: '请输入密码',
        countryRequired: '请选择国家',
        usernameLength: '用户名长度应为 3-20 个字符',
        passwordLength: '密码至少 6 个字符',
        loginSuccess: '登录成功',
        registerSuccess: '注册成功',
        registerCanceled: '已取消注册',
        usernamePlaceholder: '用户名 (3-20个字符)',
        passwordPlaceholder: '密码 (至少6个字符)',
        privacyNotice: '注册即表示同意数据用于实验目的'
    },

    // 隐私声明
    privacy: {
        title: '隐私声明',
        agree: '同意',
        cancel: '取消',
        notice: '本软件在聊天中会有 LLM 实时获取聊天记录，相关数据仅用于实验目的，感谢您的支持与配合！',
        dataUsage: '数据使用说明',
        point1: '聊天内容将被发送到本地 LLM 模型进行分析',
        point2: '相关数据仅用于实验目的',
        point3: '数据在本地处理，不会上传到外部服务器',
        agreement: '点击"同意"即表示您已了解并同意以上内容。感谢您的支持与配合！'
    },

    // 聊天界面
    chat: {
        searchUser: '搜索用户',
        connected: '已连接',
        disconnected: '未连接',
        noUsers: '暂无用户',
        selectUser: '请选择一个用户开始聊天',
        noMessages: '暂无消息，开始聊天吧',
        inputPlaceholder: '输入消息... (Ctrl+Enter 发送)',
        send: '发送',
        logout: '退出登录',
        errorRecord: '错误记录',
        logoutSuccess: '已退出登录',
        clearChat: '清空聊天记录',
        deleteMyMessages: '删除我的消息',
        deleteMessage: '删除消息',
        deleteConfirm: '确定要删除这条消息吗？',
        clearConfirm: '确定要清空与 {username} 的所有聊天记录吗？此操作不可恢复！',
        deleteMyConfirm: '确定要删除我发送给 {username} 的所有消息吗？',
        warning: '警告',
        hint: '提示',
        confirm: '确定',
        confirmDelete: '确定删除',
        confirmClear: '确定清空',
        messageDeleted: '消息已删除',
        chatCleared: '聊天记录已清空',
        messagesDeleted: '已删除 {count} 条消息',
        sendFailed: '发送失败',
        justNow: '刚刚',
        minutesAgo: '{n}分钟前'
    },

    // 语法错误
    grammar: {
        title: '语法错误记录',
        back: '返回',
        clearAll: '清空全部',
        totalErrors: '总错误数',
        weekErrors: '本周错误',
        todayErrors: '今日错误',
        search: '搜索错误内容',
        timeFilter: '时间筛选',
        all: '全部',
        today: '今天',
        week: '本周',
        month: '本月',
        noErrors: '暂无错误记录',
        originalText: '原始文本',
        suggestion: '建议修改',
        explanation: '错误说明',
        copy: '复制',
        delete: '删除',
        deleteConfirm: '确定要删除这条记录吗？',
        clearAllConfirm: '确定要清空全部 {count} 条记录吗？此操作不可恢复！',
        deleteSuccess: '删除成功',
        clearSuccess: '已清空所有记录',
        copySuccess: '已复制到剪贴板',
        copyFailed: '复制失败',
        messageDeleted: '原始消息已删除'
    },

    // 语法建议弹窗
    grammarSuggestion: {
        title: '语法建议',
        suggestedChange: '建议修改为:',
        explanation: '说明:',
        copy: '复制建议',
        cancel: '取消'
    },

    // 国家列表
    countries: {
        CN: '中国 (China)',
        US: '美国 (USA)',
        GB: '英国 (UK)',
        JP: '日本 (Japan)',
        KR: '韩国 (Korea)',
        FR: '法国 (France)',
        DE: '德国 (Germany)',
        CA: '加拿大 (Canada)',
        AU: '澳大利亚 (Australia)',
        OTHER: '其他 (Other)'
    },

    // 通用
    common: {
        confirm: '确定',
        cancel: '取消',
        delete: '删除',
        edit: '编辑',
        save: '保存',
        loading: '加载中...',
        error: '错误',
        success: '成功',
        warning: '警告',
        info: '提示'
    }
}

// 英文语言包
const enUS = {
    login: {
        title: 'Instant Messaging System',
        loginTab: 'Login',
        registerTab: 'Register',
        username: 'Username',
        password: 'Password',
        country: 'Country',
        selectCountry: 'Select Country',
        loginBtn: 'Login',
        registerBtn: 'Register',
        usernameRequired: 'Please enter username',
        passwordRequired: 'Please enter password',
        countryRequired: 'Please select country',
        usernameLength: 'Username should be 3-20 characters',
        passwordLength: 'Password should be at least 6 characters',
        loginSuccess: 'Login successful',
        registerSuccess: 'Registration successful',
        registerCanceled: 'Registration canceled',
        usernamePlaceholder: 'Username (3-20 characters)',
        passwordPlaceholder: 'Password (at least 6 characters)',
        privacyNotice: 'Registration means you agree to use data for experimental purposes'
    },

    privacy: {
        title: 'Privacy Notice',
        agree: 'Agree',
        cancel: 'Cancel',
        notice: 'This software uses LLM to access chat records in real-time. Data is used for experimental purposes only. Thank you for your support!',
        dataUsage: 'Data Usage Description',
        point1: 'Chat content will be sent to local LLM model for analysis',
        point2: 'Data is used for experimental purposes only',
        point3: 'Data is processed locally and not uploaded to external servers',
        agreement: 'By clicking "Agree", you acknowledge and agree to the above. Thank you for your support!'
    },

    chat: {
        searchUser: 'Search User',
        connected: 'Connected',
        disconnected: 'Disconnected',
        noUsers: 'No users',
        selectUser: 'Select a user to start chatting',
        noMessages: 'No messages, start chatting',
        inputPlaceholder: 'Type a message... (Ctrl+Enter to send)',
        send: 'Send',
        logout: 'Logout',
        errorRecord: 'Error Record',
        logoutSuccess: 'Logged out',
        clearChat: 'Clear Chat History',
        deleteMyMessages: 'Delete My Messages',
        deleteMessage: 'Delete Message',
        deleteConfirm: 'Are you sure to delete this message?',
        clearConfirm: 'Are you sure to clear all chat history with {username}? This cannot be undone!',
        deleteMyConfirm: 'Are you sure to delete all messages I sent to {username}?',
        warning: 'Warning',
        hint: 'Hint',
        confirm: 'Confirm',
        confirmDelete: 'Confirm Delete',
        confirmClear: 'Confirm Clear',
        messageDeleted: 'Message deleted',
        chatCleared: 'Chat history cleared',
        messagesDeleted: 'Deleted {count} messages',
        sendFailed: 'Send failed',
        justNow: 'Just now',
        minutesAgo: '{n} minutes ago'
    },

    grammar: {
        title: 'Grammar Error Records',
        back: 'Back',
        clearAll: 'Clear All',
        totalErrors: 'Total Errors',
        weekErrors: 'Week Errors',
        todayErrors: 'Today Errors',
        search: 'Search error content',
        timeFilter: 'Time Filter',
        all: 'All',
        today: 'Today',
        week: 'This Week',
        month: 'This Month',
        noErrors: 'No error records',
        originalText: 'Original Text',
        suggestion: 'Suggestion',
        explanation: 'Explanation',
        copy: 'Copy',
        delete: 'Delete',
        deleteConfirm: 'Are you sure to delete this record?',
        clearAllConfirm: 'Are you sure to clear all {count} records? This cannot be undone!',
        deleteSuccess: 'Deleted successfully',
        clearSuccess: 'All records cleared',
        copySuccess: 'Copied to clipboard',
        copyFailed: 'Copy failed',
        messageDeleted: 'Original message deleted'
    },

    grammarSuggestion: {
        title: 'Grammar Suggestion',
        suggestedChange: 'Suggested change:',
        explanation: 'Explanation:',
        copy: 'Copy Suggestion',
        cancel: 'Cancel'
    },

    countries: {
        CN: 'China',
        US: 'United States',
        GB: 'United Kingdom',
        JP: 'Japan',
        KR: 'Korea',
        FR: 'France',
        DE: 'Germany',
        CA: 'Canada',
        AU: 'Australia',
        OTHER: 'Other'
    },

    common: {
        confirm: 'Confirm',
        cancel: 'Cancel',
        delete: 'Delete',
        edit: 'Edit',
        save: 'Save',
        loading: 'Loading...',
        error: 'Error',
        success: 'Success',
        warning: 'Warning',
        info: 'Info'
    }
}

// 创建 i18n 实例
const i18n = createI18n({
    legacy: false, // 使用 Composition API 模式
    locale: localStorage.getItem('locale') || 'zh-CN', // 默认语言
    fallbackLocale: 'zh-CN',
    messages: {
        'zh-CN': zhCN,
        'en-US': enUS
    }
})

export default i18n