<template>
  <TopNav />
  <div class="container">
    <KeepAlive :include="cachedViews">
      <RouterView />
    </KeepAlive>
  </div>
  <Toast ref="toastRef" />
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import TopNav from './components/TopNav.vue'
import Toast from './components/Toast.vue'
import { initToast } from './utils/errorHandler.js'

const toastRef = ref(null)
const route = useRoute()

// 需要缓存的视图组件名称（根据路由配置）
const cachedViews = computed(() => {
  // 缓存数据密集型页面，避免重复加载
  const cacheable = [
    'Dashboard',
    'BinanceGainers',
    'Announcements',
    'TwitterFeed',
    'ChainFlows',
    'Transfers',
  ]
  return cacheable
})

onMounted(() => {
  if (toastRef.value) {
    initToast(toastRef.value)
  }
})
</script>
