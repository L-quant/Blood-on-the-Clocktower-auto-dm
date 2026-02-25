// 应用入口，初始化 Vue 实例、FontAwesome 图标、Vuex Store 与 I18n
//
// [IN]  store（Vuex 状态管理）
// [IN]  i18n（国际化配置）
// [IN]  App.vue（根组件）
// [OUT] 无（顶层入口）
// [POS] 前端应用启动点，组装所有全局依赖

import Vue from "vue";
import App from "./App";
import store from "./store";
import i18n from "./i18n";
import { library } from "@fortawesome/fontawesome-svg-core";
import { fas } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";

const faIcons = [
  "BookOpen",
  "Clipboard",
  "Cog",
  "Copy",
  "Dice",
  "ExclamationTriangle",
  "Globe",
  "Link",
  "MinusCircle",
  "PlusCircle",
  "Question",
  "Skull",
  "Times",
  "TimesCircle",
  "User",
  "UserEdit",
  "Users",
  "VolumeUp",
  "VolumeMute",
  "VoteYea"
];
library.add(...faIcons.map(i => fas["fa" + i]));
Vue.component("font-awesome-icon", FontAwesomeIcon);
Vue.config.productionTip = false;

new Vue({
  render: h => h(App),
  store,
  i18n
}).$mount("#app");
