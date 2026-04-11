import './assets/main.css'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import App from './App.vue'
import router from "@/router/index.ts";
import { installAuthFetchInterceptor } from '@/utils/auth'

installAuthFetchInterceptor()

createApp(App)
    .use(ElementPlus, { locale: zhCn })
    .use(router)
    .mount('#app')
