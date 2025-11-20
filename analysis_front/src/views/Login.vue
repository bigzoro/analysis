<template>
  <section class="panel authbox">
    <h2>登录</h2>
    <div class="form">
      <label>用户名</label>
      <input v-model="username" type="text" />
      <label>密码</label>
      <input v-model="password" type="password" />
      <button @click="submit" :disabled="loading">登录</button>
      <p class="muted">还没有账号？<RouterLink to="/register">去注册</RouterLink></p>
      <p v-if="err" class="err">{{ err }}</p>
    </div>
  </section>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { api } from '../api/api.js'
import { useAuth } from '../stores/auth.js'

const router = useRouter()
const route = useRoute()
const { setAuth } = useAuth()

const username = ref('')
const password = ref('')
const loading = ref(false)
const err = ref('')

async function submit() {
  if (loading.value) return
  if (!username.value || !password.value) { err.value = '请输入用户名和密码'; return }
  loading.value = true
  err.value = ''
  try {
    const r = await api.login({ username: username.value.trim(), password: password.value })
    const token = r?.token
    const user = r?.user
    if (!token) throw new Error('empty token')
    // 先更新响应式登录状态（含 localStorage 持久化）
    setAuth(token, user?.username || username.value.trim())
    // 等待一帧，确保依赖 isAuthed 的组件（如 TopNav）已就绪
    await nextTick()
    const redirect = route.query.redirect ? String(route.query.redirect) : '/dashboard'
    router.replace(redirect)
  } catch (e) {
    err.value = e?.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.authbox { max-width: 420px; margin: 30px auto; }
.form { display: grid; gap: 10px; }
input {
  height: 36px; border: 1px solid var(--border); border-radius: 8px;
  padding: 0 10px; background: var(--panel); color: #111827;
}
button {
  height: 36px; border-radius: 8px; border: 1px solid var(--border);
  background: #f3f4f6; color: #111827; cursor: pointer;
}
.muted { color: var(--muted); font-size: 12px; }
.err { color: #ef4444; font-size: 12px; }
</style>
