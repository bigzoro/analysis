import { createRouter, createWebHistory } from 'vue-router'

import Dashboard from '../views/Dashboard.vue'
import Flows from '../views/Flows.vue'
import Runs from '../views/Runs.vue'
import Transfers from '../views/Transfers.vue'
import ChainFlows from '../views/ChainFlows.vue'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import ScheduledOrders from "../views/ScheduledOrders.vue"
import BinanceGainers from "../views/BinanceGainers.vue"
import Announcements from "../views/Announcements.vue"
import TwitterFeed from "../views/TwitterFeed.vue"
import Recommendations from "../views/Recommendations.vue"

import { useAuth } from '../stores/auth.js'

const routes = [
    { path: '/', redirect: '/market' },
    { path: '/login', component: Login, meta: { requiresAuth: false } },
    { path: '/register', component: Register, meta: { requiresAuth: false } },

    { path: '/dashboard', component: Dashboard, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/flows', component: Flows, meta: { requiresAuth: true } },
    { path: '/runs', component: Runs, meta: { requiresAuth: true } },
    { path: '/transfers', component: Transfers, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/chain-flows', component: ChainFlows, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/scheduled-orders', component: ScheduledOrders, meta: { requiresAuth: true } },
    { path: '/market', component: BinanceGainers, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/announcements', component: Announcements, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/twitter', component: TwitterFeed, meta: { requiresAuth: true, keepAlive: true } },
    { path: '/recommendations', component: Recommendations, meta: { requiresAuth: true, keepAlive: true } },
]

const router = createRouter({
    history: createWebHistory(),
    routes,
})

router.beforeEach((to) => {
    const { isAuthed } = useAuth()
    // 需要登录但未登录 → 去登录页，并带上重定向
    if (to.meta?.requiresAuth && !isAuthed.value) {
        return { path: '/login', query: { redirect: to.fullPath } }
    }
    // 已登录仍访问 login / register → 回到首页
    if (!to.meta?.requiresAuth && isAuthed.value && (to.path === '/login' || to.path === '/register')) {
        return { path: '/market' }
    }
})

export default router
