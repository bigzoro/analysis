// src/stores/auth.js
// 轻量全局响应式登录状态（无需 Pinia）
import { ref, computed } from 'vue'

const token = ref(localStorage.getItem('auth_token') || '')
const username = ref(localStorage.getItem('auth_username') || '')

function setAuth(t = '', u = '') {
    token.value = t || ''
    username.value = u || ''
    if (t) localStorage.setItem('auth_token', t)
    else localStorage.removeItem('auth_token')
    if (u) localStorage.setItem('auth_username', u)
    else localStorage.removeItem('auth_username')
}

function logout() {
    setAuth('', '')
}

const isAuthed = computed(() => !!token.value)

export function useAuth() {
    return { token, username, isAuthed, setAuth, logout }
}
