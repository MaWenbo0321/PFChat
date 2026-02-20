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
        usernameLength: '用户名长度应为 2-20 个字符',
        passwordLength: '密码至少 6 个字符',
        loginSuccess: '登录成功',
        registerSuccess: '注册成功',
        registerCanceled: '已取消注册',
        usernamePlaceholder: '用户名 (2-20个字符)',
        passwordPlaceholder: '密码 (至少6个字符)',
        privacyNotice: '注册即表示同意数据用于实验目的'
    },

    // 隐私声明
    privacy: {
        title: '隐私声明',
        agree: '同意',
        cancel: '取消',
        notice: '本软件在聊天中会有 LLM 实时获取聊天记录,相关数据仅用于实验目的,感谢您的支持与配合!',
        dataUsage: '数据使用说明',
        point1: '聊天内容将被发送到本地 LLM 模型进行分析',
        point2: '相关数据仅用于实验目的',
        point3: '数据在本地处理,不会上传到外部服务器',
        agreement: '点击"同意"即表示您已了解并同意以上内容。感谢您的支持与配合!'
    },

    // 聊天界面
    chat: {
        // 🔧 添加发送前检测相关文本
        errorDetectedTitle: '检测到语用失误',
        errorDetected: '您的消息可能存在语用失误',
        errorType: '错误类型',
        originalMessage: '原始消息',
        suggestion: '建议修改',
        explanation: '错误说明',
        editMessage: '编辑消息',
        editPlaceholder: '请修改您的消息...',
        applySuggestion: '应用建议',
        sendOriginal: '发送原文',
        sendEdited: '发送修改',
        checkFailed: '检测失败',
        sendConfirmTitle: '发送确认',
        sendAnyway: '仍要发送',
        sendCancelled: '已取消发送',
        sendSuccess: '发送成功',
        pragmalinguisticError: '语言语用失误',
        sociopragmaticError: '社会语用失误',
        searchUser: '搜索用户',
        connected: '已连接',
        disconnected: '未连接',
        noUsers: '暂无用户',
        selectUser: '请选择一个用户开始聊天',
        noMessages: '暂无消息,开始聊天吧',
        inputPlaceholder: '输入消息... (Ctrl+Enter 发送)',
        send: '发送',
        logout: '退出登录',
        errorRecord: '错误记录',
        logoutSuccess: '已退出登录',
        logoutConfirm: '确定要退出登录吗?',
        clearChat: '清空聊天记录',
        deleteMyMessages: '删除我的消息',
        deleteMessage: '删除消息',
        adminDelete: '管理员删除',
        deleteConfirm: '确定要删除这条消息吗?',
        clearConfirm: '确定要清空与 {username} 的所有聊天记录吗?此操作不可恢复!',
        deleteMyConfirm: '确定要删除我发送给 {username} 的所有消息吗?',
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
        clearType: '清空当前分类',
        search: '搜索错误内容',
        typeFilter: '错误类型',
        allTypes: '全部类型',
        errorType1: '语言语用失误',
        errorType2: '社会语用失误',
        type1Count: '语言语用失误数量',
        type2Count: '社会语用失误数量',
        noErrors: '暂无错误记录',
        originalText: '原始文本',
        suggestion: '建议修改',
        explanation: '错误说明',
        errorTypeLabel: '错误类型',
        copy: '复制',
        delete: '删除',
        changeType: '更改类型',
        deleteConfirm: '确定要删除这条记录吗?',
        clearAllConfirm: '确定要清空全部 {count} 条记录吗?此操作不可恢复!',
        clearTypeConfirm: '确定要清空当前分类的 {count} 条记录吗?此操作不可恢复!',
        deleteSuccess: '删除成功',
        clearSuccess: '已清空所有记录',
        copySuccess: '已复制到剪贴板',
        copyFailed: '复制失败',
        messageDeleted: '原始消息已删除',
        updateTypeSuccess: '类型更新成功',
        updateTypeFailed: '类型更新失败'
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
    },

// 在 zh.chat 中添加:
    zhChatAdditions: {
        // 实时检测相关
        checking: '检测中...',
        errorsFound: '发现 {count} 个问题',
        noErrors: '未发现问题',
        sendWithErrorsConfirm: '你的消息中检测到了语用失误，确定要发送吗？',
        sendAnyway: '仍然发送',

        // AI Chat 相关
        aiAssistant: 'AI 助手',
        aiWelcome: '向我提问跨文化沟通、语用差异相关的问题，或获取消息修改建议。',
        aiInputPlaceholder: '询问文化或语用相关问题...',
        aiThinking: '思考中...',
        aiError: '抱歉，出现了错误，请重试。',
        aiSuggest1: '中英文化交际中常见的语用失误有哪些？',
        aiSuggest2: '不同文化中如何礼貌地拒绝邀请？',
        aiSuggest3: '解释语言语用失误和社会语用失误的区别',
    },


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
        usernameLength: 'Username should be 2-20 characters',
        passwordLength: 'Password should be at least 6 characters',
        loginSuccess: 'Login successful',
        registerSuccess: 'Registration successful',
        registerCanceled: 'Registration canceled',
        usernamePlaceholder: 'Username (2-20 characters)',
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
        errorDetectedTitle: 'Pragmatic Error Detected',

        editMessage: 'Edit Message',
        editPlaceholder: 'Please revise your message...',
        applySuggestion: 'Apply Suggestion',
        sendOriginal: 'Send Original',
        sendEdited: 'Send Revised',

        checkFailed: 'Check Failed',
        errorDetected: 'Pragmatic Error Detected',
        errorType: 'Error Type',
        originalMessage: 'Original Message',
        suggestion: 'Suggested Revision',
        explanation: 'Explanation',
        sendConfirmTitle: 'Send Confirmation',
        sendAnyway: 'Send Anyway',
        sendCancelled: 'Send Cancelled',
        sendSuccess: 'Sent Successfully',
        pragmalinguisticError: 'Pragmalinguistic Failure',
        sociopragmaticError: 'Sociopragmatic Failure',

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
        logoutConfirm: 'Are you sure you want to logout?',
        clearChat: 'Clear Chat History',
        deleteMyMessages: 'Delete My Messages',
        deleteMessage: 'Delete Message',
        adminDelete: 'Admin Delete',
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
        clearType: 'Clear Current Category',
        totalErrors: 'Total Errors',
        weekErrors: 'Week Errors',
        todayErrors: 'Today Errors',
        search: 'Search error content',
        typeFilter: 'Error Type',
        allTypes: 'All Types',
        errorType1: 'Pragmalinguistic Failure',
        errorType2: 'Sociopragmatic Failure',
        type1Count: 'Pragmalinguistic Failure Count',
        type2Count: 'Sociopragmatic Failure Count',
        noErrors: 'No error records',
        originalText: 'Original Text',
        suggestion: 'Suggestion',
        explanation: 'Explanation',
        errorTypeLabel: 'Error Type',
        copy: 'Copy',
        delete: 'Delete',
        changeType: 'Change Type',
        deleteConfirm: 'Are you sure to delete this record?',
        clearAllConfirm: 'Are you sure to clear all {count} records? This cannot be undone!',
        clearTypeConfirm: 'Are you sure to clear {count} records in current category? This cannot be undone!',
        deleteSuccess: 'Deleted successfully',
        clearSuccess: 'All records cleared',
        copySuccess: 'Copied to clipboard',
        copyFailed: 'Copy failed',
        messageDeleted: 'Original message deleted',
        updateTypeSuccess: 'Type updated successfully',
        updateTypeFailed: 'Failed to update type'
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
    },
    // 在 en.chat 中添加:
    enChatAdditions : {
        // 实时检测相关
        checking: 'Checking...',
        errorsFound: '{count} issue(s) found',
        noErrors: 'No issues found',
        sendWithErrorsConfirm: 'Pragmatic errors have been detected in your message. Are you sure you want to send it?',
        sendAnyway: 'Send Anyway',

        // AI Chat 相关
        aiAssistant: 'AI Assistant',
        aiWelcome: 'Ask me about cross-cultural communication, pragmatic differences, or get help with your messages.',
        aiInputPlaceholder: 'Ask about culture or pragmatics...',
        aiThinking: 'Thinking...',
        aiError: 'Sorry, there was an error. Please try again.',
        aiSuggest1: 'What are common pragmatic errors between Chinese and English speakers?',
        aiSuggest2: 'How to politely decline an invitation in different cultures?',
        aiSuggest3: 'Explain the difference between pragmalinguistic and sociopragmatic failure',

        // 保留原有的 key 不变
    },
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