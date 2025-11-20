<template>
  <header class="topnav">
    <div class="nav-inner container">
      <div class="brand">交易所监控</div>

      <nav class="tabs" v-if="isAuthed">
        <RouterLink to="/market" class="tab" active-class="active">币安涨幅</RouterLink>
        <RouterLink to="/dashboard" class="tab">仪表盘</RouterLink>
        <!-- <RouterLink to="/flows" class="tab">资金流</RouterLink> -->
        <RouterLink to="/chain-flows" class="tab">资金链</RouterLink>
        <!-- <RouterLink to="/runs" class="tab">运行记录</RouterLink> -->
        <RouterLink to="/transfers" class="tab">转账实时</RouterLink>
        <RouterLink to="/scheduled-orders" class="tab">定时合约单</RouterLink>
        <RouterLink to="/announcements" class="tab">公告</RouterLink>
        <RouterLink to="/twitter" class="tab">Twitter</RouterLink>
        <RouterLink to="/recommendations" class="tab">币种推荐</RouterLink>
      </nav>

      <div class="spacer"></div>

<!--      <div class="env">ENV: {{ envMode }}</div>-->

      <div class="auth" v-if="isAuthed">
        <span class="user">{{ username || '已登录' }}</span>
        <button class="logout" @click="onLogout">退出</button>
      </div>

      <div v-else class="auth">
        <RouterLink class="tab" to="/login">登录</RouterLink>
        <RouterLink class="tab" to="/register">注册</RouterLink>
      </div>
    </div>
  </header>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useAuth } from '../stores/auth.js'

const router = useRouter()
const { isAuthed, username, logout } = useAuth()
const envMode = import.meta.env.MODE

function onLogout() {
  logout()
  router.replace('/login')
}
</script>

<style scoped lang="scss">
.topnav {
  position: sticky; top: 0; z-index: 10;
  backdrop-filter: blur(6px);
  background: rgba(255,255,255,.8);
  border-bottom: 1px solid var(--border);
}
.nav-inner { display: flex; align-items: center; gap: 14px; padding: 10px 18px; }
.brand { font-weight: 700; letter-spacing: .4px; color: var(--primary); white-space: nowrap; }
.tabs { display: flex; gap: 10px; white-space: nowrap; overflow-x: auto; -webkit-overflow-scrolling: touch; }
.tab { text-decoration: none; color: var(--text); padding: 8px 12px; border-radius: 10px; border: 1px solid transparent; font-weight: 500; }
.tab:hover { background: #f3f4f6; }
.tab.router-link-exact-active { background: #e5e7eb; border-color: var(--border); }
.spacer { flex: 1; }
.env { font-size: 12px; color: var(--muted); white-space: nowrap; margin-right: 8px; }
.auth { display: flex; align-items: center; gap: 8px; }
.logout { height: 28px; padding: 0 10px; border-radius: 8px; border: 1px solid var(--border); background: #f3f4f6; color: #111827; }
.user { color: #4f46e5; margin-right: 6px; }
@media (max-width: 900px) { .env { display: none; } }
</style>
