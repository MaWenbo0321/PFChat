// 国家代码到语言的映射
const countryToLocale = {
    'CN': 'zh-CN',     // 中国 → 中文
    'TW': 'zh-CN',     // 台湾 → 中文
    'HK': 'zh-CN',     // 香港 → 中文
    'SG': 'zh-CN',     // 新加坡 → 中文
    'US': 'en-US',     // 美国 → 英文
    'GB': 'en-US',     // 英国 → 英文
    'CA': 'en-US',     // 加拿大 → 英文
    'AU': 'en-US',     // 澳大利亚 → 英文
    'NZ': 'en-US',     // 新西兰 → 英文
    'JP': 'en-US',     // 日本 → 英文（可以添加日语包）
    'KR': 'en-US',     // 韩国 → 英文（可以添加韩语包）
    'FR': 'en-US',     // 法国 → 英文（可以添加法语包）
    'DE': 'en-US',     // 德国 → 英文（可以添加德语包）
    'OTHER': 'en-US'   // 其他 → 英文
}

/**
 * 根据国家代码获取对应的语言
 * @param {string} countryCode - 国家代码（如 'CN', 'US'）
 * @returns {string} 语言代码（如 'zh-CN', 'en-US'）
 */
export function getLocaleByCountry(countryCode) {
    return countryToLocale[countryCode] || 'en-US'
}

/**
 * 切换应用语言
 * @param {string} newLocale - 语言代码
 * @param {{ value: string }} localeRef - vue-i18n 的响应式 locale
 */
export function switchLocale(newLocale, localeRef) {
    localeRef.value = newLocale
    localStorage.setItem('locale', newLocale)
    document.documentElement.lang = newLocale
}
