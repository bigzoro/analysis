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
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/api.js'

const route = useRoute()
const router = useRouter()
const username = ref('')
const password = ref('')
const loading = ref(false)
const err = ref('')

async function submit() {
  if (!username.value || !password.value) { err.value = '请输入用户名和密码'; return }
  loading.value = true; err.value = ''
  try {
    const r = await api.login({ username: username.value, password: password.value })
    localStorage.setItem('auth_token', r.token)
    localStorage.setItem('auth_username', r.user?.username || username.value)
    router.replace(route.query.redirect || '/dashboard')
  } catch (e) {
    err.value = '登录失败'
  } finally { loading.value = false }
}
</script>

<style scoped>
.authbox { max-width: 420px; margin: 30px auto; }
.form { display: grid; gap: 10px; }
input { height: 36px; border: 1px solid var(--border); border-radius: 8px; padding: 0 10px; background: #0b1320; color: var(--text); }
button { height: 36px; border-radius: 8px; border: 1px solid var(--border); background: #1f2937; color: #e5e7eb; }
.muted { color: var(--muted); font-size: 12px; }
.err { color: #ef4444; font-size: 12px; }
</style>
