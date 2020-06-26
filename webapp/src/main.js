import Vue from "vue";
import App from "./App.vue";

import Buefy from "buefy";
import "buefy/dist/buefy.css";

import VueResource from "vue-resource";

Vue.use(Buefy, {
  defaultIconPack: "fas",
});
Vue.config.productionTip = false;

Vue.use(VueResource);

new Vue({
  render: (h) => h(App),
}).$mount("#app");
