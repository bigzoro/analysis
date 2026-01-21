<template>
  <!-- 全局加载组件 -->
  <GlobalLoading
    v-model="globalLoading"
    :text="loadingText"
    :icon="loadingIcon"
    :progress="loadingProgress"
  />

  <!-- 全局消息提示 -->
  <GlobalToast ref="globalToastRef" />

  <!-- 用户引导 -->
  <UserGuide v-model="showUserGuide" @complete="onGuideComplete" />

  <!-- 导航栏 -->
  <TopNav />

  <!-- 页面内容 -->
  <div class="page-container">
    <RouterView v-slot="{ Component }">
      <transition name="page" mode="out-in">
        <KeepAlive :include="cachedViews">
          <component :is="Component" />
        </KeepAlive>
      </transition>
    </RouterView>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, provide } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from './components/TopNav.vue'
import GlobalLoading from './components/GlobalLoading.vue'
import GlobalToast from './components/GlobalToast.vue'
import UserGuide from './components/UserGuide.vue'

const globalToastRef = ref(null)
const route = useRoute()

// 全局状态
const globalLoading = ref(false)
const loadingText = ref('加载中...')
const loadingIcon = ref('⏳')
const loadingProgress = ref(null)

// 用户引导
const showUserGuide = ref(false)

// 需要缓存的视图组件名称（根据路由配置）
const cachedViews = computed(() => {
  // 缓存数据密集型页面，避免重复加载
  const cacheable = [
    'Dashboard',
    'BinanceGainers',
    'RealTimeGainers', // 实时涨幅榜 - 保持WebSocket连接
    'Announcements',
    'TwitterFeed',
    'ChainFlows',
    'Transfers',
    'AIRecommendations',
    'AdvancedRisk',
    'AdvancedBacktest'
  ]
  return cacheable
})

// 全局方法提供
const showLoading = (text = '加载中...', icon = '⏳', progress = null) => {
  loadingText.value = text
  loadingIcon.value = icon
  loadingProgress.value = progress
  globalLoading.value = true
}

const hideLoading = () => {
  globalLoading.value = false
  loadingProgress.value = null
}

const showToast = (type, title, message = '', duration = 4000) => {
  if (globalToastRef.value) {
    globalToastRef.value[type](title, message, { duration })
  }
}

// 提供全局方法给子组件
provide('globalLoading', {
  show: showLoading,
  hide: hideLoading
})

provide('globalToast', {
  show: showToast,
  success: (title, message) => showToast('success', title, message),
  error: (title, message) => showToast('error', title, message),
  warning: (title, message) => showToast('warning', title, message),
  info: (title, message) => showToast('info', title, message)
})

// 用户引导完成
const onGuideComplete = () => {
  showUserGuide.value = false
  if (window.$toast) {
    window.$toast.success('🎉 欢迎开始您的AI量化投资之旅！', '如有疑问，请随时查看帮助文档')
  }
}

onMounted(() => {
  // 初始化全局样式
  initGlobalStyles()
})

// 全局样式初始化
const initGlobalStyles = () => {
  // 检测系统主题偏好
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches

  if (prefersDark) {
    document.documentElement.setAttribute('data-theme', 'dark')
  }

  // 监听主题变化
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    document.documentElement.setAttribute('data-theme', e.matches ? 'dark' : 'light')
  })

  // 添加页面加载完成的类
  document.documentElement.classList.add('page-loaded')
}
</script>

<style>
/* 全局页面样式 */
.page-container {
  min-height: calc(100vh - 60px); /* 减去导航栏高度 */
  background: var(--bg-secondary);
}

/* 页面切换动画 */
.page-enter-active,
.page-leave-active {
  transition: all var(--transition-normal);
}

.page-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-20px);
}

/* 全局滚动条样式 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
}

::-webkit-scrollbar-thumb {
  background: var(--border-medium);
  border-radius: var(--radius-md);
  transition: background var(--transition-fast);
}

::-webkit-scrollbar-thumb:hover {
  background: var(--border-dark);
}

/* 选择文本样式 */
::selection {
  background: var(--primary-100);
  color: var(--primary-900);
}

/* 焦点样式 */
*:focus-visible {
  outline: 2px solid var(--primary-500);
  outline-offset: 2px;
}

/* 页面加载动画 */
html:not(.page-loaded) {
  opacity: 0;
}

html.page-loaded {
  animation: fadeIn 0.5s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* 深色主题变量 */
[data-theme="dark"] {
  --bg-primary: #0f172a;
  --bg-secondary: #1e293b;
  --bg-tertiary: #334155;
  --bg-overlay: rgba(15, 23, 42, 0.95);

  --text-primary: #f8fafc;
  --text-secondary: #cbd5e1;
  --text-muted: #94a3b8;

  --border-light: #334155;
  --border-medium: #475569;
  --border-dark: #64748b;

  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.3);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -1px rgba(0, 0, 0, 0.3);
  --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.5), 0 4px 6px -2px rgba(0, 0, 0, 0.4);
  --shadow-xl: 0 20px 25px -5px rgba(0, 0, 0, 0.6), 0 10px 10px -5px rgba(0, 0, 0, 0.5);
}

/* 减少动画偏好 */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
