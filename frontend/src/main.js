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
