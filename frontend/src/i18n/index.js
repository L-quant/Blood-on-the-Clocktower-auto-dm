// Vue I18n 初始化，浏览器语言检测（非英语默认中文），英文兜底
//
// [IN]  en.json（英文翻译）
// [IN]  zh.json（中文翻译）
// [OUT] main.js（全局 i18n 注入）
// [POS] 国际化配置，为所有组件提供 $t/$te 翻译能力

import Vue from "vue";
import VueI18n from "vue-i18n";
import en from "./en.json";
import zh from "./zh.json";

Vue.use(VueI18n);

const browserLang = (
  navigator.language ||
  navigator.userLanguage ||
  "zh"
).toLowerCase();
const defaultLocale = browserLang.startsWith("zh") ? "zh" : "en";

export default new VueI18n({
  locale: defaultLocale,
  fallbackLocale: "en",
  messages: { en, zh }
});
