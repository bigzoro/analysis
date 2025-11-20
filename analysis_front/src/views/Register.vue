<template>
  <section class="panel authbox">
    <h2>注册</h2>
    <div class="form">
      <label>用户名</label>
      <input v-model="username" type="text" />
      <label>密码（≥6位）</label>
      <input v-model="password" type="password" />
      <label>确认密码</label>
      <input v-model="confirm" type="password" />
      <button @click="submit" :disabled="loading">注册</button>
      <p class="muted">已有账号？<RouterLink to="/login">去登录</RouterLink></p>
      <p v-if="err" class="err">{{ err }}</p>
    </div>
  </section>
</template>

<script setup>
import { ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/api.js'
import { useAuth } from '../stores/auth.js'

const router = useRouter()
const { setAuth } = useAuth()

const username = ref('')
const password = ref('')
const confirm = ref('')
const loading = ref(false)
const err = ref('')

async function submit() {
  if (!username.value || !password.value) { err.value = '请输入用户名和密码'; return }
  if (password.value.length < 6) { err.value = '密码至少 6 位'; return }
  if (password.value !== confirm.value) { err.value = '两次输入不一致'; return }
  loading.value = true; err.value = ''
  try {
    const r = await api.register({ username: username.value.trim(), password: password.value })
    const token = r?.token
    const user = r?.user
    if (!token) throw new Error('empty token')
    setAuth(token, user?.username || username.value.trim())
    await nextTick()
    router.replace('/dashboard')
  } catch (e) {
    err.value = e?.message || '注册失败'
  } finally { loading.value = false }
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
