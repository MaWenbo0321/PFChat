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
 * @param {string} locale - 语言代码
 */
export function switchLocale(locale, i18n) {
    i18n.global.locale.value = locale
    localStorage.setItem('locale', locale)

    // 同时切换 Element Plus 的语言
    // 注意：需要在 main.js 中配置 Element Plus 的国际化
}

/**
 * 根据国家代码自动切换语言
 * @param {string} countryCode - 国家代码
 * @param {object} i18n - i18n 实例
 */
export function autoSwitchLocaleByCountry(countryCode, i18n) {
    const locale = getLocaleByCountry(countryCode)
    switchLocale(locale, i18n)
    return locale
}

/**
 * 获取当前语言
 * @param {object} i18n - i18n 实例
 * @returns {string} 当前语言代码
 */
export function getCurrentLocale(i18n) {
    return i18n.global.locale.value
}

/**
 * 判断当前是否为中文
 * @param {object} i18n - i18n 实例
 * @returns {boolean}
 */
export function isChinese(i18n) {
    return i18n.global.locale.value === 'zh-CN'
}

/**
 * 判断当前是否为英文
 * @param {object} i18n - i18n 实例
 * @returns {boolean}
 */
export function isEnglish(i18n) {
    return i18n.global.locale.value === 'en-US'
}