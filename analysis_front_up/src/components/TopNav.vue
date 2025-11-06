<template>
  <header class="topnav">
    <div class="nav-inner container">
      <div class="brand">交易所监控</div>

      <nav class="tabs" v-if="isAuthed">
        <RouterLink to="/dashboard" class="tab">仪表盘</RouterLink>
<!--        <RouterLink to="/flows" class="tab">资金流</RouterLink>-->
        <RouterLink to="/chain-flows" class="tab">资金链</RouterLink>
<!--        <RouterLink to="/runs" class="tab">运行记录</RouterLink>-->
        <RouterLink to="/transfers" class="tab">转账实时</RouterLink>
        <RouterLink to="/market" class="tab" active-class="active">币安涨幅</RouterLink>
        <RouterLink to="/scheduled-orders" class="tab">定时合约单</RouterLink>
      </nav>

      <div class="spacer"></div>
<!--      <div class="env">接口：{{ apiBase }}</div>-->

      <div class="auth">
        <template v-if="!isAuthed">
          <RouterLink to="/login" class="tab">登录</RouterLink>
          <RouterLink to="/register" class="tab">注册</RouterLink>
        </template>
        <template v-else>
          <span class="user">{{ username }}</span>
          <button class="logout" @click="logout">退出</button>
        </template>
      </div>
    </div>
  </header>
</template>

<script setup>
import { computed } from 'vue'
const apiBase = import.meta.env.VITE_API_BASE || 'http://127.0.0.1:8010'
const isAuthed = computed(() => !!localStorage.getItem('auth_token'))
const username = computed(() => localStorage.getItem('auth_username') || '用户')
function logout() {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_username')
  window.location.href = '/login'
}
</script>

<style scoped lang="scss">
.topnav { position: sticky; top: 0; z-index: 10; background: rgba(10,14,20,.88); backdrop-filter: blur(6px); border-bottom: 1px solid var(--border); }
.nav-inner { display: flex; align-items: center; gap: 14px; padding: 10px 18px; }
.brand { font-weight: 700; letter-spacing: .4px; color: var(--primary); white-space: nowrap; }
.tabs { display: flex; gap: 10px; white-space: nowrap; overflow-x: auto; -webkit-overflow-scrolling: touch; }
.tab { text-decoration: none; color: var(--text); padding: 8px 12px; border-radius: 10px; border: 1px solid transparent; font-weight: 500; }
.tab:hover { background: #161c28; }
.tab.router-link-exact-active { background: #1a2233; border-color: var(--border); }
.spacer { flex: 1; }
.env { font-size: 12px; color: var(--muted); white-space: nowrap; margin-right: 8px; }
.auth { display: flex; align-items: center; gap: 8px; }
.logout { height: 28px; padding: 0 10px; border-radius: 8px; border: 1px solid var(--border); background: #1f2937; color: #e5e7eb; }
.user { color: #a5b4fc; margin-right: 6px; }
@media (max-width: 900px) { .env { display: none; } }
</style>
