import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router/router.js'
import './styles/style.scss'
import Toast from './components/Toast.vue'

// 创建应用实例
const app = createApp(App)

// 注册全局Toast组件
const toastInstance = createApp(Toast).mount(document.createElement('div'))
document.body.appendChild(toastInstance.$el)

// 添加全局toast方法
app.config.globalProperties.$toast = {
  success: (message, duration) => toastInstance.add(message, 'success', duration),
  error: (message, duration) => toastInstance.add(message, 'error', duration),
  warning: (message, duration) => toastInstance.add(message, 'warning', duration),
  info: (message, duration) => toastInstance.add(message, 'info', duration)
}

app.use(router).mount('#app')
