import { createRouter, createWebHistory } from 'vue-router'

import Dashboard from '../views/Dashboard.vue'
import Flows from '../views/Flows.vue'
import Runs from '../views/Runs.vue'
import Transfers from '../views/Transfers.vue'
import ChainFlows from '../views/ChainFlows.vue'
import Login from '../views/Login.vue'
import Register from '../views/Register.vue'
import ScheduledOrders from "../views/ScheduledOrders.vue";
import BinanceGainers from "../views/BinanceGainers.vue";

const routes = [
    { path: '/', redirect: '/dashboard' },
    { path: '/login', component: Login },
    { path: '/register', component: Register },
    { path: '/dashboard', component: Dashboard, meta: { requiresAuth: true } },
    { path: '/flows', component: Flows, meta: { requiresAuth: true } },
    { path: '/chain-flows', component: ChainFlows, meta: { requiresAuth: true } },
    { path: '/runs', component: Runs, meta: { requiresAuth: true } },
    { path: '/transfers', component: Transfers, meta: { requiresAuth: true } },
    { path: '/scheduled-orders', component: ScheduledOrders, meta: { requiresAuth: true } },
    {
        path: '/market',
        component: BinanceGainers,
        meta: { requiresAuth: true }
    },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('auth_token')
    if (to.meta?.requiresAuth && !token) {
        next({ path: '/login', query: { redirect: to.fullPath } })
    } else {
        next()
    }
})

export default router
