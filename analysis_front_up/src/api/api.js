// src/api/api.js

const API_BASE = import.meta.env.VITE_API_BASE || 'http://127.0.0.1:8010'

// ✅ 自动取浏览器时区，取不到就用 Asia/Taipei
const BROWSER_TZ =
    (typeof Intl !== 'undefined' &&
        Intl.DateTimeFormat().resolvedOptions().timeZone) ||
    'Asia/Taipei'

function authToken() {
    return localStorage.getItem('auth_token') || ''
}

function buildURL(path, params) {
    const url = new URL(API_BASE + path)
    if (params) {
        Object.entries(params).forEach(([k, v]) => {
            if (v !== undefined && v !== null && v !== '') {
                url.searchParams.set(k, String(v))
            }
        })
    }
    return url
}

async function getJSON(path, params) {
    const url = buildURL(path, params)
    const r = await fetch(url.toString(), {
        headers: {
            'Authorization': authToken() ? `Bearer ${authToken()}` : ''
        }
    })
    if (!r.ok) throw new Error(`${r.status} ${r.statusText}`)
    return r.json()
}

async function postJSON(path, body) {
    const r = await fetch(API_BASE + path, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': authToken() ? `Bearer ${authToken()}` : ''
        },
        body: JSON.stringify(body || {})
    })
    if (!r.ok) throw new Error(`${r.status} ${r.statusText}`)
    return r.json()
}

export const api = {
    // auth
    register: ({ username, password }) => postJSON('/auth/register', { username, password }),
    login: ({ username, password }) => postJSON('/auth/login', { username, password }),
    me: () => getJSON('/me'),

    // business
    listEntities: () => getJSON('/entities'),
    listRuns: (entity) => getJSON('/runs', { entity }),
    latestPortfolio: (entity) => getJSON('/portfolio/latest', { entity }),
    dailyFlows: ({ entity, coin, latest = true, start, end }) =>
        getJSON('/flows/daily', { entity, coin, latest, start, end }),
    weeklyFlows: ({ entity, coin, latest = true }) =>
        getJSON('/flows/weekly', { entity, coin, latest }),
    dailyFlowsByChain: ({ entity, chain, start, end, coin }) =>
        getJSON('/flows/daily_by_chain', { entity, chain, start, end, coin }),
    recentTransfers: ({ entity, chain, coin, limit = 50, before_ts, before_id }) =>
        getJSON('/transfers/recent', { entity, chain, coin, limit, before_ts, before_id }),

    wsTransfersURL: ({ entity }) => {
        const url = buildURL('/ws/transfers', { entity, token: authToken() })
        return url.toString().replace(/^http/, 'ws')
    },

    scheduleOrder: (payload) => postJSON('/orders/schedule', payload),
    listScheduledOrders: () => getJSON('/orders/schedule'),
    cancelScheduledOrder: (id) => postJSON(`/orders/schedule/${id}/cancel`),

    // ✅ 币安涨幅榜
    // 现在会自动带 tz 给后端，后端就能按你的时区对齐 2 小时
    binanceTop: ({ kind = 'spot', start, end, interval = 120, date, slot, tz } = {}) => {
        const params = { kind, interval }

        // 优先用调用方传的 tz，其次用浏览器的
        params.tz = tz || BROWSER_TZ

        if (slot !== undefined && slot !== null && slot !== '') {
            params.slot = slot
        }
        if (date) {
            params.date = date
        } else {
            if (start) params.start = start
            if (end) params.end = end
        }
        return getJSON('/market/binance/top', params)
    },
}

export { getJSON, postJSON }
