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
