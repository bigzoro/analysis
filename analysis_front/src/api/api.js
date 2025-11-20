// 后端基地址：
// - 生产：用 Nginx 反代，保持同源 => VITE_API_BASE=/api
// - 开发：直连后端 => VITE_API_BASE=http://127.0.0.1:8010
const API_BASE = (import.meta.env.VITE_API_BASE ?? '/api')

// 默认交易所（避免后端 400: missing entity）
const DEFAULT_ENTITY = import.meta.env.VITE_DEFAULT_ENTITY || 'binance'

// 浏览器时区（默认 Asia/Taipei）
const BROWSER_TZ =
    (typeof Intl !== 'undefined' &&
        Intl.DateTimeFormat().resolvedOptions().timeZone) ||
    'Asia/Taipei'

// 读取本地登录令牌
function authToken () {
    try { return localStorage.getItem('auth_token') || '' } catch { return '' }
}

// 把字符串参数优雅地转成对象：'okx' -> { entity: 'okx' }
function asEntityParams (params) {
    if (params == null) return {}
    return typeof params === 'string' ? { entity: params } : params
}

// 构建 URL（自动附带 tz，允许追加查询参数）
function buildURL (path, params = {}) {
    const base = (API_BASE || '/api').replace(/\/$/, '')
    const url = new URL(base + path, window.location.origin)

    // 默认追加 tz（若外部显式传入 tz 则不覆盖）
    if (params.tz === undefined || params.tz === null || params.tz === '') {
        url.searchParams.set('tz', BROWSER_TZ)
    }
    for (const [k, v] of Object.entries(params)) {
        if (v === undefined || v === null || v === '') continue
        url.searchParams.set(k, String(v))
    }
    return url
}

async function requestJSON (method, path, { params, body, headers } = {}) {
    const url = buildURL(path, params)
    const token = authToken()

    const init = {
        method,
        headers: {
            'Accept': 'application/json',
            ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...(headers || {}),
        },
        // 本项目使用 JWT 头部鉴权，不依赖 Cookie
        credentials: 'omit',
    }
    if (body !== undefined) {
        init.body = JSON.stringify(body)
    }

    try {
        const res = await fetch(url.toString(), init)
        const text = await res.text()
        let data = null
        try { data = text ? JSON.parse(text) : null } catch { data = text }

        if (!res.ok) {
            // 统一错误处理：提取错误消息
            const msg =
                (data && (data.error || data.message)) ||
                `HTTP ${res.status} ${res.statusText}`
            
            // 创建错误对象，包含更多上下文
            const error = new Error(msg)
            error.status = res.status
            error.statusText = res.statusText
            error.data = data
            error.url = url.toString()
            
            throw error
        }
        return data
    } catch (error) {
        // 如果是网络错误或其他错误，添加更多上下文
        if (!error.status) {
            error.url = url.toString()
            error.method = method
        }
        throw error
    }
}

const getJSON   = (path, params)        => requestJSON('GET',    path, { params })
const postJSON  = (path, body, params)  => requestJSON('POST',   path, { params, body })
const putJSON   = (path, body, params)  => requestJSON('PUT',    path, { params, body })
const patchJSON = (path, body, params)  => requestJSON('PATCH',  path, { params, body })
const delJSON   = (path, params)        => requestJSON('DELETE', path, { params })

// =================== 统一对外 API ===================
export const api = {
    // ---- 认证 ----
    login (payload) {               // 响应：{ token, user: { id, username } }
        return postJSON('/auth/login', payload)
    },
    register (payload) {
        return postJSON('/auth/register', payload)
    },

    // ---- Dashboard / Portfolio ----
    // GET /portfolio/latest?entity=...
    latestPortfolio (params = {}) {
        const p = asEntityParams(params)
        if (!p.entity || p.entity === 'all') p.entity = DEFAULT_ENTITY // 兜底，避免 400
        return getJSON('/portfolio/latest', p)
    },

    // ---- 基础数据 ----
    // GET /entities
    listEntities ({ entity, chain, coin, limit = 100, cursor } = {}) {
        return getJSON('/entities', { entity, chain, coin, limit, cursor })
    },
    
    // GET /runs?entity=&page=1&page_size=50
    listRuns ({ entity, page = 1, page_size = 50, keyword, start_date, end_date } = {}) {
        const params = { page, page_size }
        if (entity) params.entity = entity
        if (keyword) params.keyword = keyword
        if (start_date) params.start_date = start_date
        if (end_date) params.end_date = end_date
        return getJSON('/runs', params)
    },

    // ---- 最近转账 ----
    // GET /transfers/recent
    recentTransfers ({ entity, chain, coin, page = 1, page_size = 50, tz, keyword, direction, start_time, end_time, min_amount, max_amount } = {}) {
        const params = { entity, chain, coin, page, page_size }
        if (tz) params.tz = tz
        if (keyword) params.keyword = keyword
        if (direction) params.direction = direction
        if (start_time) params.start_time = start_time
        if (end_time) params.end_time = end_time
        if (min_amount !== undefined && min_amount !== null) params.min_amount = min_amount
        if (max_amount !== undefined && max_amount !== null) params.max_amount = max_amount
        return getJSON('/transfers/recent', params)
    },

    // ---- 链上资金流（日粒度，支持 entity=all/具体交易所，chain=all/具体链）----
    // ✅ ChainFlows.vue 调用的方法
    // GET /flows/daily_by_chain
    dailyFlowsByChain (params = {}) {
        return getJSON('/flows/daily_by_chain', asEntityParams(params))
    },

    // （保留旧接口，若有页面仍使用）
    // GET /flows/chain
    chainFlows (params = {}) {
        return getJSON('/flows/chain', asEntityParams(params))
    },

    // 若别处使用到：
    // GET /flows/daily
    dailyFlows (params = {}) {
        return getJSON('/flows/daily', asEntityParams(params))
    },
    // GET /flows/weekly
    weeklyFlows (params = {}) {
        return getJSON('/flows/weekly', asEntityParams(params))
    },

    // ---- 定时合约单 ----
    // POST /orders/schedule
    createScheduledOrder (body) {
        return postJSON('/orders/schedule', body)
    },
    // GET /orders/schedule?page=1&page_size=50
    listScheduledOrders ({ page = 1, page_size = 50 } = {}) {
        return getJSON('/orders/schedule', { page, page_size })
    },
    // POST /orders/schedule/:id/cancel
    cancelScheduledOrder (id) {
        return postJSON(`/orders/schedule/${id}/cancel`)
    },

    // ---- 公告 ----
    // GET /announcements/recent
    // listAnnouncements ({ src = {}, limit = 50 } = {}) {
    //     return getJSON('/announcements/recent', { ...src, limit })
    // },

    listAnnouncements ({ sources, categories, src = {}, cats = {}, q, page = 1, page_size = 10, limit, offset, is_event, verified, sentiment, exchange, start_date, end_date } = {}) {
        // 优先使用 page/page_size，兼容 limit/offset
        const params = {}
        if (page !== undefined) params.page = page
        if (page_size !== undefined) params.page_size = page_size
        // 兼容旧格式
        if (limit !== undefined) params.limit = limit
        if (offset !== undefined) params.offset = offset
        
        if (q) params.q = q

        // sources：优先透传字符串；否则默认使用 coincarp
        if (typeof sources === 'string') {
            params.sources = sources
        } else {
            // 默认只使用 CoinCarp
            params.sources = 'coincarp'
        }

        // categories：优先透传字符串；否则从布尔 map 组装
        if (typeof categories === 'string') {
            params.categories = categories
        } else if (cats && typeof cats === 'object') {
            const c = []
            if (cats.newcoin) c.push('newcoin')
            if (cats.finance) c.push('finance')
            if (cats.event) c.push('event')
            if (cats.other) c.push('other')
            if (c.length) params.categories = c.join(',')
        }

        // 新增筛选参数
        if (is_event !== undefined && is_event !== null) {
            params.is_event = is_event
        }
        if (verified !== undefined && verified !== null) {
            params.verified = verified
        }
        if (sentiment) {
            params.sentiment = sentiment
        }
        if (exchange) {
            params.exchange = exchange
        }
        if (start_date) {
            params.start_date = start_date
        }
        if (end_date) {
            params.end_date = end_date
        }

        return getJSON('/announcements/recent', params)
    },

    // ---- Twitter ----
    // GET /twitter/fetch?username=...&limit=...&store=1&pagination_token=...
    twitterFetch ({ username, limit = 10, store = 1, pagination_token } = {}) {
        const params = { username, limit, store }
        if (pagination_token) params.pagination_token = pagination_token
        return getJSON('/twitter/fetch', params)
    },
    // GET /twitter/posts?username=...&page=1&page_size=5
    twitterPosts ({ username, page = 1, page_size = 5, keyword, start_date, end_date } = {}) {
        const params = { page, page_size }
        if (username) params.username = username
        if (keyword) params.keyword = keyword
        if (start_date) params.start_date = start_date
        if (end_date) params.end_date = end_date
        return getJSON('/twitter/posts', params)
    },

    // ---- 涨幅榜（示例：2 小时分段）----
    // GET /market/segments?kind=spot|futures
    marketSegments ({ kind = 'spot', tz } = {}) {
        return getJSON('/market/segments', { kind, tz })
    },

    // 币安涨幅榜（按 2 小时一段聚合）
    // GET /market/binance/top?kind=spot|futures&interval=120&date=YYYY-MM-DD&tz=Asia/Taipei
    binanceTop ({ kind = 'spot', interval = 120, date, tz } = {}) {
        const params = { kind, interval }
        if (date) params.date = date
        // 默认用浏览器时区，后端会按这个时区把"哪一天的第几段"换算回 UTC 查询
        params.tz = tz || BROWSER_TZ
        return getJSON('/market/binance/top', params)
    },

    // ---- 币安黑名单管理 ----
    // GET /market/binance/blacklist?kind=spot|futures
    listBinanceBlacklist ({ kind } = {}) {
        const params = {}
        if (kind) params.kind = kind
        return getJSON('/market/binance/blacklist', params)
    },
    // POST /market/binance/blacklist
    addBinanceBlacklist ({ kind, symbol }) {
        return postJSON('/market/binance/blacklist', { kind, symbol })
    },
    // DELETE /market/binance/blacklist/:kind/:symbol
    deleteBinanceBlacklist (kind, symbol) {
        return delJSON(`/market/binance/blacklist/${encodeURIComponent(kind)}/${encodeURIComponent(symbol)}`)
    },

    // ---- 币种推荐 ----
    // GET /recommendations/coins?kind=spot&limit=5&refresh=false
    getCoinRecommendations ({ kind = 'spot', limit = 5, refresh = false } = {}) {
        const params = { kind, limit }
        if (refresh) params.refresh = 'true'
        return getJSON('/recommendations/coins', params)
    },

    // ---- 回测功能 ----
    // GET /recommendations/backtest?limit=50
    getBacktestRecords ({ limit = 50 } = {}) {
        return getJSON('/recommendations/backtest', { limit })
    },
    // GET /recommendations/backtest/stats
    getBacktestStats () {
        return getJSON('/recommendations/backtest/stats')
    },
    // POST /recommendations/backtest
    createBacktestFromRecommendation (body) {
        return postJSON('/recommendations/backtest', body)
    },
    // POST /recommendations/backtest/:id/update
    updateBacktestRecord (id, body) {
        return postJSON(`/recommendations/backtest/${id}/update`, body)
    },

    // ---- 模拟交易 ----
    // POST /recommendations/simulation/trade
    createSimulatedTrade (body) {
        return postJSON('/recommendations/simulation/trade', body)
    },
    // GET /recommendations/simulation/trades?is_open=true
    getSimulatedTrades ({ is_open } = {}) {
        const params = {}
        if (is_open !== undefined && is_open !== null) {
            params.is_open = is_open
        }
        return getJSON('/recommendations/simulation/trades', params)
    },
    // POST /recommendations/simulation/trades/:id/close
    closeSimulatedTrade (id, body) {
        return postJSON(`/recommendations/simulation/trades/${id}/close`, body)
    },
    // POST /recommendations/simulation/trades/:id/update-price
    updateSimulatedTradePrice (id, body) {
        return postJSON(`/recommendations/simulation/trades/${id}/update-price`, body)
    },

    // ---- WebSocket ----
    wsTransfersURL ({ entity } = {}) {
                const token = authToken()

                    // 绝对地址：如 http://127.0.0.1:8010
                        if (/^https?:\/\//i.test(API_BASE || '')) {
                        const abs = (API_BASE || '').replace(/^http/i, 'ws').replace(/\/$/, '')
                            const url = new URL('/ws/transfers', abs)
                            if (entity) url.searchParams.set('entity', entity)
                        if (token)  url.searchParams.set('token', token)
                        return url.toString()
                        }

                    // 相对地址：走同源 + '/ws/transfers'（不要带 '/api'）
                        const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
                    const url = new URL(`${proto}://${window.location.host}/ws/transfers`)
                    if (entity) url.searchParams.set('entity', entity)
                if (token)  url.searchParams.set('token', token)
                return url.toString()
                },
}

// 若你有少量地方直接用到基础封装，也保留导出
export { getJSON, postJSON, putJSON, patchJSON, delJSON }
