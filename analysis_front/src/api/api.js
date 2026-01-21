// 导入智能缓存工具
import smartCache, { CACHE_LEVELS } from '@/utils/dataCache.js'

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

// ===== 数据同步监控 API =====

// 获取数据同步服务状态
export async function getDataSyncStatus() {
    return requestJSON('GET', '/api/data-sync/status')
}

// 触发手动同步
export async function triggerManualSync(syncerType) {
    return requestJSON('POST', '/api/data-sync/trigger', {
        body: { syncer_type: syncerType }
    })
}

// 获取数据一致性检查状态
export async function getDataConsistencyStatus() {
    return requestJSON('GET', '/api/data-sync/consistency')
}

// 获取告警信息
export async function getAlerts() {
    return requestJSON('GET', '/api/data-sync/alerts')
}

// 获取监控统计信息
export async function getMonitoringStats() {
    return requestJSON('GET', '/api/data-sync/monitoring/stats')
}

// 解决告警
export async function resolveAlert(alertId) {
    return requestJSON('POST', '/api/data-sync/alerts/resolve', {
        body: { alert_id: alertId }
    })
}

// 重新连接WebSocket
export async function reconnectWebSocket() {
    return requestJSON('POST', '/api/data-sync/websocket/reconnect')
}

// 执行一致性检查
export async function triggerConsistencyCheck() {
    return requestJSON('POST', '/api/data-sync/consistency/check')
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

async function requestJSON (method, path, { params, body, headers, timeout = 300000000 } = {}) {
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
        // 创建带超时的 AbortController
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), timeout)
        init.signal = controller.signal

        const res = await fetch(url.toString(), init)
        clearTimeout(timeoutId)
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
        // 处理超时错误
        if (error.name === 'AbortError') {
            const timeoutError = new Error(`请求超时 (${timeout}ms)`)
            timeoutError.status = 408
            timeoutError.statusText = 'Request Timeout'
            timeoutError.url = url.toString()
            timeoutError.method = method
            timeoutError.timeout = timeout
            throw timeoutError
        }

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

// 用户行为追踪工具
import behaviorTracker from '@/utils/behaviorTracker.js'

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

    // ---- 大户监控 ----
    // GET /whales/watchlist
    listWhaleWatches () {
        return getJSON('/whales/watchlist')
    },
    // POST /whales/watchlist
    createWhaleWatch (watch) {
        return postJSON('/whales/watchlist', watch)
    },
    // DELETE /whales/watchlist/:address
    deleteWhaleWatch (address) {
        return delJSON(`/whales/watchlist/${address}`)
    },

    // GET /whales/arkham
    listArkhamWatches () {
        return getJSON('/whales/arkham')
    },
    // POST /whales/arkham
    createArkhamWatch (watch) {
        return postJSON('/whales/arkham', watch)
    },
    // DELETE /whales/arkham/:address
    deleteArkhamWatch (address) {
        return delJSON(`/whales/arkham/${address}`)
    },
    // POST /whales/arkham/query
    queryArkhamAddress (queryData) {
        return postJSON('/whales/arkham/query', queryData)
    },
    // POST /whales/arkham/sync
    syncArkhamData () {
        return postJSON('/whales/arkham/sync')
    },

    // GET /whales/nansen
    listNansenWatches () {
        return getJSON('/whales/nansen')
    },
    // POST /whales/nansen
    createNansenWatch (watch) {
        return postJSON('/whales/nansen', watch)
    },
    // DELETE /whales/nansen/:address
    deleteNansenWatch (address) {
        return delJSON(`/whales/nansen/${address}`)
    },
    // POST /whales/nansen/query
    queryNansenAddress (queryData) {
        return postJSON('/whales/nansen/query', queryData)
    },
    // POST /whales/nansen/sync
    syncNansenData () {
        return postJSON('/whales/nansen/sync')
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
    // POST /orders/schedule/batch
    createBatchScheduledOrders (body) {
        return postJSON('/orders/schedule/batch', body)
    },
    // GET /orders/schedule?page=1&page_size=50
    listScheduledOrders ({ page = 1, page_size = 50 } = {}) {
        return getJSON('/orders/schedule', { page, page_size })
    },
    // POST /orders/schedule/:id/cancel
    cancelScheduledOrder (id) {
        return postJSON(`/orders/schedule/${id}/cancel`)
    },
    // POST /orders/schedule/:id/close-position
    closePosition (id) {
        return postJSON(`/orders/schedule/${id}/close-position`)
    },
    // GET /orders/schedule/:id
    getScheduledOrderDetail (id) {
        return getJSON(`/orders/schedule/${id}`)
    },
    // DELETE /orders/schedule/:id
    deleteScheduledOrder (id, options = {}) {
        const params = new URLSearchParams()
        if (options.cascade !== undefined) {
            params.append('cascade', options.cascade.toString())
        }
        if (options.closeOrderIds && Array.isArray(options.closeOrderIds)) {
            params.append('closeOrderIds', options.closeOrderIds.join(','))
        }
        const queryString = params.toString()
        const url = queryString ? `/orders/schedule/${id}?${queryString}` : `/orders/schedule/${id}`
        return delJSON(url)
    },

    // ---- 交易策略管理 ----
    // POST /strategies
    createTradingStrategy (data) {
        return postJSON('/strategies', data)
    },
    // GET /strategies
    listTradingStrategies () {
        return getJSON('/strategies')
    },
    // GET /strategies/:id
    getTradingStrategy (id) {
        return getJSON(`/strategies/${id}`)
    },
    // PUT /strategies/:id
    updateTradingStrategy (id, data) {
        return putJSON(`/strategies/${id}`, data)
    },
    // DELETE /strategies/:id
    deleteTradingStrategy (id) {
        return delJSON(`/strategies/${id}`)
    },

    // DELETE /strategies/executions/:execution_id
    deleteStrategyExecution (executionId) {
        return delJSON(`/strategies/executions/${executionId}`)
    },

    // ---- 策略运行 ----
    // POST /strategies/:id/start
    startStrategyExecution (data) {
        return postJSON(`/strategies/${data.strategy_id}/start`, data)
    },
    // POST /strategies/:id/stop
    stopStrategyExecution (strategyId) {
        return postJSON(`/strategies/${strategyId}/stop`)
    },
    // GET /strategies/executions
    listStrategyExecutions (params = {}) {
        return getJSON('/strategies/executions', params)
    },
    // GET /strategies/executions/:execution_id
    getStrategyExecution (executionId) {
        return getJSON(`/strategies/executions/${executionId}`)
    },
    // GET /strategies/:id/stats
    getStrategyExecutionStats (strategyId, params = {}) {
        const queryParams = new URLSearchParams()
        if (params.page) queryParams.append('page', params.page)
        if (params.page_size) queryParams.append('page_size', params.page_size)

        const queryString = queryParams.toString()
        const url = `/strategies/${strategyId}/stats${queryString ? '?' + queryString : ''}`
        return getJSON(url)
    },

    // 获取策略执行步骤详情
    getStrategyExecutionSteps (executionId) {
        return getJSON(`/strategies/executions/${executionId}/steps`)
    },

    // 获取策略相关的订单记录
    getStrategyOrders (strategyId, params = {}) {
        const queryParams = new URLSearchParams()
        if (params.page) queryParams.append('page', params.page)
        if (params.page_size) queryParams.append('page_size', params.page_size)

        const queryString = queryParams.toString()
        const url = `/strategies/${strategyId}/orders${queryString ? '?' + queryString : ''}`
        return getJSON(url)
    },

    // 获取策略健康状态
    getStrategyHealth (strategyId) {
        return getJSON(`/strategies/${strategyId}/health`)
    },

    // ---- 策略执行 ----
    // POST /strategies/execute
    executeStrategy (data) {
        return postJSON('/strategies/execute', data)
    },
    // POST /strategies/batch-execute
    batchExecuteStrategies (data) {
        return postJSON('/strategies/batch-execute', data)
    },
    // POST /strategies/scan-eligible
    scanEligibleSymbols (strategyId) {
        return postJSON('/strategies/scan-eligible', { strategy_id: strategyId })
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

    // ---- 涨幅榜（示例：1 小时分段）----
    // GET /market/segments?kind=spot|futures
    marketSegments ({ kind = 'spot', tz } = {}) {
        return getJSON('/market/segments', { kind, tz })
    },

    // 币安涨幅榜（按 1 小时一段聚合）
    // GET /market/binance/top?kind=spot|futures&interval=60&date=YYYY-MM-DD&tz=Asia/Taipei
    binanceTop ({ kind = 'spot', interval = 60, date, tz, category = 'all' } = {}) {
        const params = { kind, interval, category }
        if (date) params.date = date
        // 默认用浏览器时区，后端会按这个时区把"哪一天的第几段"换算回 UTC 查询
        params.tz = tz || BROWSER_TZ
        return getJSON('/market/binance/top', params)
    },

    // 实时涨幅榜
    // GET /market/binance/realtime-gainers?kind=spot&limit=15&sort_by=change&sort_order=desc&filter_positive_only=false&filter_large_cap=false&min_volume=0
    realtimeGainers ({ kind = 'spot', limit = 15, sort_by = 'change', sort_order = 'desc', filter_positive_only = false, filter_large_cap = false, min_volume = 0 } = {}) {
        return getJSON('/market/binance/realtime-gainers', {
            kind,
            limit,
            sort_by,
            sort_order,
            filter_positive_only: filter_positive_only.toString(),
            filter_large_cap: filter_large_cap.toString(),
            min_volume: min_volume.toString()
        })
    },

    // 涨幅榜历史数据
    // GET /market/binance/realtime-gainers/history?kind=spot&start_time=2024-01-01T00:00:00Z&end_time=2024-01-02T00:00:00Z&symbol=BTC&limit=10
    realtimeGainersHistory ({ kind = 'spot', start_time, end_time, symbol, limit = 20 } = {}) {
        const params = { kind, limit }
        if (start_time) params.start_time = start_time
        if (end_time) params.end_time = end_time
        if (symbol) params.symbol = symbol
        return getJSON('/market/binance/realtime-gainers/history', params)
    },

    // 涨幅榜数据统计
    // GET /market/binance/realtime-gainers/stats
    realtimeGainersStats () {
        return getJSON('/market/binance/realtime-gainers/stats')
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
        return getJSON('/recommendations/coins', params, { timeout: 60000 }) // 60秒超时
    },

    // 获取历史推荐
    getHistoricalRecommendations ({ kind = 'spot', date, includePerformance = true, includeRealtimeData = false, page = 1, page_size = 10 } = {}) {
        const params = { kind, include_performance: includePerformance, include_realtime_data: includeRealtimeData, page, page_size }
        if (date) params.date = date
        return getJSON('/recommendations/historical', params, { timeout: 45000 }) // 45秒超时
    },

    // 获取推荐时间列表
    getRecommendationTimeList ({ kind = 'spot', limit = 30 } = {}) {
        return getJSON('/recommendations/times', { kind, limit })
    },

    // 批量更新回测记录
    batchUpdateBacktestRecords ({ ids = [] } = {}) {
        return postJSON('/recommendations/backtest/batch-update', { ids })
    },

    // 为指定日期生成推荐
    generateRecommendationsForDate ({ kind = 'spot', date, limit = 10 } = {}) {
        const params = { kind, limit }
        if (date) params.date = date
        return postJSON('/recommendations/generate', {}, params)
    },

    // ---- 回测功能 ----
    // GET /recommendations/backtest?page=1&limit=20&status=completed&symbol=BTC&start_date=2024-01-01&end_date=2024-12-31&sort_by=recommended_at&sort_order=desc
    getBacktestRecords ({ page = 1, limit = 20, status, symbol, start_date, end_date, sort_by, sort_order } = {}) {
        const params = { page, limit }
        if (status) params.status = status
        if (symbol) params.symbol = symbol
        if (start_date) params.start_date = start_date
        if (end_date) params.end_date = end_date
        if (sort_by) params.sort_by = sort_by
        if (sort_order) params.sort_order = sort_order
        return getJSON('/recommendations/backtest', params)
    },

    // 策略回测功能
    // POST /recommendations/backtest/strategy
    executeStrategyBacktest ({ performance_id } = {}) {
        return postJSON('/recommendations/backtest/strategy', { performance_id })
    },

    // POST /recommendations/backtest/strategy/test
    testStrategyBacktest ({ performance_id } = {}) {
        return postJSON('/recommendations/backtest/strategy/test', { performance_id })
    },

    // POST /recommendations/backtest/strategy/batch
    batchExecuteStrategyBacktest ({ performance_ids, limit } = {}) {
        return postJSON('/recommendations/backtest/strategy/batch', { performance_ids, limit })
    },
    // GET /recommendations/backtest/stats
    getBacktestStats () {
        return getJSON('/recommendations/backtest/stats')
    },
    // GET /recommendations/performance?recommendation_id=123&symbol=BTCUSDT&limit=10
    getRecommendationPerformance ({ recommendation_id, symbol, limit = 10 } = {}) {
        const params = { limit }
        if (recommendation_id) params.recommendation_id = recommendation_id
        if (symbol) params.symbol = symbol
        return getJSON('/recommendations/performance', params)
    },
    // GET /recommendations/performance/stats?days=30
    getPerformanceStats ({ days = 30 } = {}) {
        return getJSON('/recommendations/performance/stats', { days }, { timeout: 30000 }) // 30秒超时
    },
    // GET /recommendations/performance/factor-stats?days=30
    getFactorPerformanceStats ({ days = 30 } = {}) {
        return getJSON('/recommendations/performance/factor-stats', { days })
    },
    // GET /recommendations/performance/trend?days=30&interval=daily
    getPerformanceTrend ({ days = 30, interval = 'daily' } = {}) {
        return getJSON('/recommendations/performance/trend', { days, interval })
    },

    // ---- 数据质量监控 ----
    getDataSources () {
        return getJSON('/data/sources')
    },

       getDataQualityReport () {
           return getJSON('/data/quality-report')
       },

       // 回测API
       runBacktest (config) {
           return postJSON('/backtest/run', config)
       },
       runStrategyBacktest (strategyId, symbol, startDate, endDate) {
           return postJSON('/backtest/strategy', {
               strategy_id: strategyId,
               symbol: symbol,
               start_date: startDate,
               end_date: endDate
           })
       },
       compareStrategies (configs) {
           return postJSON('/backtest/compare', { configs })
       },
       batchBacktest (configs) {
           return postJSON('/backtest/batch', { configs })
       },
       optimizeStrategy (baseConfig, paramRanges) {
           return postJSON('/backtest/optimize', { base_config: baseConfig, param_ranges: paramRanges })
       },
       getBacktestTemplates () {
           return getJSON('/backtest/templates')
       },
       getAvailableStrategies () {
           return getJSON('/backtest/strategies')
       },
       saveBacktestResult (resultData) {
           return postJSON('/backtest/save', resultData)
       },
       getSavedBacktests () {
           return getJSON('/backtest/saved')
       },

       // 过滤器修正统计
       getFilterCorrectionStats () {
           return getJSON('/backtest/filter-corrections/stats')
       },
       getFilterCorrectionsBySymbol (symbol) {
           return getJSON(`/backtest/filter-corrections/${symbol}`)
       },
       cleanupOldFilterCorrections (days = 30) {
           return postJSON('/backtest/filter-corrections/cleanup', { days })
       },

       // 异步回测API
       startAsyncBacktest (config) {
           return postJSON('/api/backtest/async/start', config)
       },
       getBacktestRecords ({ page = 1, limit = 10, status, symbol, sort_by, sort_order } = {}) {
           return getJSON('/api/backtest/async/records', { page, limit, status, symbol, sort_by, sort_order })
       },
       getBacktestRecord (recordId) {
           return getJSON(`/api/backtest/async/records/${recordId}`)
       },
       getBacktestTrades (recordId, { page = 1, limit = 20, sort_by, sort_order } = {}) {
           return getJSON(`/api/backtest/async/trades/${recordId}`, { page, limit, sort_by, sort_order })
       },
       deleteBacktestRecord (recordId) {
           return deleteJSON(`/api/backtest/async/records/${recordId}`)
       },

    getMultiSourceData (symbols = 'BTC,ETH') {
        return getJSON('/data/multi-source', { symbols })
    },

    refreshDataSources (symbols = 'BTC,ETH') {
        return postJSON('/data/refresh', { symbols })
    },
    // GET /recommendations/performance/batch?recommendation_ids=1,2,3,4,5
    getBatchRecommendationPerformance ({ recommendation_ids } = {}) {
        if (Array.isArray(recommendation_ids)) {
            recommendation_ids = recommendation_ids.join(',')
        }
        return getJSON('/recommendations/performance/batch', { recommendation_ids })
    },
    // POST /recommendations/backtest
    createBacktestFromRecommendation (body) {
        return postJSON('/recommendations/backtest', body)
    },
    // POST /recommendations/backtest/:id/update
    updateBacktestRecord (id) {
        return postJSON(`/recommendations/backtest/${id}/update`, {})
    },
    // POST /recommendations/backtest/batch-update
    batchUpdateBacktestRecords ({ ids } = {}) {
        return postJSON('/recommendations/backtest/batch-update', { ids })
    },

    // ---- 模拟交易 ----
    // POST /recommendations/simulation/trade
    createSimulatedTrade (body) {
        return postJSON('/recommendations/simulation/trade', body)
    },

    // ---- 自动执行设置 ----
    // GET /user/auto-execute/settings
    getAutoExecuteSettings () {
        return getJSON('/user/auto-execute/settings')
    },
    // PUT /user/auto-execute/settings
    updateAutoExecuteSettings (settings) {
        return putJSON('/user/auto-execute/settings', settings)
    },
    // POST /recommendations/auto-execute
    executeRecommendations ({ date, symbols = [], riskLevel = 'medium' } = {}) {
        return postJSON('/recommendations/auto-execute', { date, symbols, risk_level: riskLevel })
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
    // DELETE /recommendations/simulations/trades - 清理用户的模拟交易记录
    clearUserTrades () {
        return delJSON('/recommendations/simulations/trades')
    },
    // GET /market/price-history?symbol=BTC&days=30&interval=daily
    getMarketPriceHistory ({ symbol, days = 30, interval } = {}) {
        const params = { symbol, days }
        if (interval) params.interval = interval
        return getJSON('/market/price-history', params)
    },

    // ---- 用户行为追踪 ----
    // POST /user/behavior/track
    trackUserBehavior (events) {
        return postJSON('/user/behavior/track', { events })
    },

    // ---- 推荐反馈 ----
    // POST /user/feedback
    submitRecommendationFeedback (feedback) {
        return postJSON('/user/feedback', feedback)
    },
    // GET /user/feedback/history?page=1&limit=20
    getUserFeedbackHistory ({ page = 1, limit = 20 } = {}) {
        return getJSON('/user/feedback/history', { page, limit })
    },
    // GET /recommendations/stats?recommendation_id=123
    getRecommendationStats ({ recommendation_id } = {}) {
        return getJSON('/recommendations/stats', { recommendation_id })
    },
    // GET /analytics/feedback?days=30
    getFeedbackAnalytics ({ days = 30 } = {}) {
        return getJSON('/analytics/feedback', { days })
    },

    // ---- A/B测试 ----
    // POST /ab-test
    createABTest (testConfig) {
        return postJSON('/ab-test', testConfig)
    },
    // GET /ab-test/:test_name/results
    getABTestResults (testName) {
        return getJSON(`/ab-test/${testName}/results`)
    },
    // GET /ab-test/active
    listActiveABTests () {
        return getJSON('/ab-test/active')
    },
    // GET /ab-test/group?test_name=xxx
    getUserTestGroup (testName) {
        return getJSON('/ab-test/group', { test_name: testName })
    },

    // ---- 算法优化 ----
    // POST /optimization/trigger
    triggerAlgorithmOptimization () {
        return postJSON('/optimization/trigger')
    },
    // GET /optimization/status
    getOptimizationStatus () {
        return getJSON('/optimization/status')
    },
    // GET /optimization/latest-result
    getLatestOptimizationResult () {
        return getJSON('/optimization/latest-result')
    },

    // ---- AI推荐系统 (新增) ----
    // POST /api/v1/recommend - 获取AI币种推荐
    getAIRecommendations ({ symbols = ['BTC', 'ETH', 'ADA', 'SOL', 'DOT'], limit = 5, risk_level = 'medium', date } = {}) {
        const data = { symbols, limit, risk_level }
        if (date) data.date = date
        return postJSON('/api/v1/recommend', data)
    },

    // GET /api/v1/recommend/detail/:symbol - 获取推荐详情
    getRecommendationDetail (symbol) {
        return getJSON(`/api/v1/recommend/detail/${symbol}`)
    },

    // GET /api/v1/market/price/:symbol - 获取当前价格
    getCurrentPrice (symbol) {
        return getJSON(`/api/v1/market/price/${symbol}`)
    },

    // POST /api/v1/market/batch-prices - 批量获取当前价格
    getBatchCurrentPrices (symbols, kind = 'spot') {
        return postJSON('/api/v1/market/batch-prices', { symbols, kind })
    },

    // GET /api/v1/market/symbols-with-marketcap - 获取带市值信息的币种列表
    getSymbolsWithMarketCap ({ kind = 'spot', limit = 50 } = {}) {
        return getJSON('/api/v1/market/symbols-with-marketcap', { kind, limit })
    },

    // GET /api/v1/market/symbols - 获取可用的交易对列表
    getAvailableSymbols ({ kind = 'spot', limit = 50 } = {}) {
        return getJSON('/api/v1/market/symbols', { kind, limit })
    },

    // GET /api/v1/market/symbol-analysis/:symbol - 分析币种用于网格交易
    analyzeSymbolForGridTrading (symbol) {
        return getJSON(`/api/v1/market/symbol-analysis/${symbol}`)
    },

    // GET /api/v1/market/grid-symbols - 获取适合网格交易的币种列表
    getGridTradingSymbols ({ kind = 'spot', limit = 50, page = 1 } = {}) {
        return getJSON('/api/v1/market/grid-symbols', { kind, limit, page })
    },

    // GET /api/v1/market/klines/:symbol - 获取K线数据
    getKlines (symbol, interval = '1h', limit = 100) {
        return getJSON(`/api/v1/market/klines/${symbol}`, { interval, limit })
    },

    // GET /api/v1/recommend/performance/:symbol - 获取历史表现
    getRecommendationPerformance (symbol, period = '30d') {
        return getJSON(`/api/v1/recommend/performance/${symbol}`, { period })
    },

    // GET /api/v1/sentiment/:symbol - 获取情绪分析
    getSentimentAnalysis (symbol) {
        return getJSON(`/api/v1/sentiment/${symbol}`)
    },

    // POST /api/v1/recommend/advanced - 高级组合推荐
    getAdvancedRecommendations (config) {
        return postJSON('/api/v1/recommend/advanced', config)
    },

    // WebSocket /ws/recommend - 实时推荐流
    getRealtimeRecommendWS () {
        // 绝对地址：如 http://127.0.0.1:8010
        if (/^https?:\/\//i.test(API_BASE || '')) {
            const abs = (API_BASE || '').replace(/^http/i, 'ws').replace(/\/$/, '')
            return `${abs}/ws/recommend`
        }
        // 相对地址：走同源
        const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
        return `${proto}://${window.location.host}/ws/recommend`
    },

    // GET /api/v1/recommend/history - 推荐历史
    getRecommendationHistory ({ symbol, days = 7 } = {}) {
        return getJSON('/api/v1/recommend/history', { symbol, days })
    },

    // ---- 风险控制系统 (新增) ----
    // GET /api/v1/risk/report - 风险报告
    getRiskReport ({ format = 'summary' } = {}) {
        return getJSON('/api/v1/risk/report', { format })
    },

    // POST /api/v1/risk/assess - 风险评估
    assessRisk ({ symbol, include_history = true, time_range = '30d' } = {}) {
        return postJSON('/api/v1/risk/assess', { symbol, include_history, time_range })
    },

    // GET /api/v1/risk/alerts - 风险告警
    getRiskAlerts ({ status = 'active', severity, limit = 20 } = {}) {
        const params = { status, limit }
        if (severity) params.severity = severity
        return getJSON('/api/v1/risk/alerts', params)
    },

    // POST /api/v1/risk/alerts/{alert_id}/acknowledge - 确认告警
    acknowledgeAlert (alertId, comment = '') {
        return postJSON(`/api/v1/risk/alerts/${alertId}/acknowledge`, {
            action: 'acknowledge',
            comment
        })
    },

    // POST /api/v1/risk/portfolio/analyze - 投资组合风险分析
    analyzePortfolio (positions, config = {}) {
        return postJSON('/api/v1/risk/portfolio/analyze', {
            positions,
            total_value: config.totalValue || 100000,
            risk_tolerance: config.riskTolerance || 'medium'
        })
    },

    // ---- 高级风险管理 (新增) ----
    // GET /api/v1/risk/advanced-metrics - 高级风险指标
    getAdvancedRiskMetrics (symbol) {
        return getJSON('/api/v1/risk/advanced-metrics', { symbol })
    },

    // POST /api/v1/risk/stress-test - 压力测试
    performStressTest (symbol, scenarios = [], timeRange = '30d') {
        return postJSON('/api/v1/risk/stress-test', {
            symbol,
            scenarios,
            time_range: timeRange
        })
    },

    // POST /api/v1/risk/portfolio/optimize - 投资组合优化
    optimizePortfolio (symbols, targetReturn = 0.1, constraints = {}, timeRange = '30d') {
        return postJSON('/api/v1/risk/portfolio/optimize', {
            symbols,
            target_return: targetReturn,
            constraints,
            time_range: timeRange
        })
    },

    // POST /api/v1/risk/budget - 风险预算分析
    getRiskBudget (symbols, weights, totalBudget = 1.0, timeRange = '30d') {
        return postJSON('/api/v1/risk/budget', {
            symbols,
            weights,
            total_budget: totalBudget,
            time_range: timeRange
        })
    },

    // ---- 高级回测分析 (新增) ----
    // POST /api/v1/backtest/walk-forward - 走步前进分析
    runWalkForwardAnalysis (symbol, startDate, endDate, strategy = 'ml_prediction', inSamplePeriod = 12, outOfSamplePeriod = 3, stepSize = 3) {
        return postJSON('/api/v1/backtest/walk-forward', {
            symbol,
            start_date: startDate,
            end_date: endDate,
            strategy,
            in_sample_period: inSamplePeriod,
            out_of_sample_period: outOfSamplePeriod,
            step_size: stepSize
        })
    },

    // POST /api/v1/backtest/monte-carlo - 蒙特卡洛分析
    runMonteCarloAnalysis (symbol, startDate, endDate, strategy = 'ml_prediction', simulations = 1000, bootstrapSize = 252) {
        return postJSON('/api/v1/backtest/monte-carlo', {
            symbol,
            start_date: startDate,
            end_date: endDate,
            strategy,
            simulations,
            bootstrap_size: bootstrapSize
        })
    },

    // POST /api/v1/backtest/optimize - 策略优化
    runStrategyOptimization (symbol, startDate, endDate, strategy = 'ml_prediction', parameters = [], method = 'grid', maxIterations = 100, objective = 'sharpe') {
        return postJSON('/api/v1/backtest/optimize', {
            symbol,
            start_date: startDate,
            end_date: endDate,
            strategy,
            parameters,
            method,
            max_iterations: maxIterations,
            objective
        })
    },

    // POST /api/v1/backtest/attribution - 归因分析
    runAttributionAnalysis (symbol, benchmarkSymbol, startDate, endDate, strategy = 'ml_prediction', timeHorizon = 'monthly') {
        return postJSON('/api/v1/backtest/attribution', {
            symbol,
            benchmark_symbol: benchmarkSymbol,
            start_date: startDate,
            end_date: endDate,
            strategy,
            time_horizon: timeHorizon
        })
    },

    // POST /api/ai-recommendation/backtest - AI推荐策略回测
    runAIStrategyBacktest (config) {
        const defaultConfig = {
            symbol: 'BTC',
            startDate: null,
            endDate: null,
            strategy: 'ml_prediction',
            initialCapital: 10000,
            positionSize: 1.0,
            stopLoss: 0.05,
            takeProfit: 0.15,
            commission: 0.001,
            timeframe: '1d',
            autoExecute: false,
            autoExecuteRiskLevel: 'moderate',
            minConfidence: 0.7,
            maxPositionPercent: 5.0,
            skipExistingTrades: true,

            // 渐进式执行参数
            progressiveExecution: false,
            maxBatches: 3,
            batchDelay: 30 * 60 * 1000, // 30分钟（毫秒）
            batchSize: 5,
            dynamicSizing: true,
            marketConditionFilter: true,

            // 新增现实性参数
            slippage: 0.001,        // 滑点 (0.1%)
            marketImpact: 0.0001,   // 市场冲击系数
            tradingDelay: 5,        // 交易延迟(分钟)
            spread: 0.0005,         // 买卖价差
            minOrderSize: 10,       // 最小订单大小
            maxOrderSize: 10000,    // 最大订单大小
            liquidityFactor: 1.0    // 流动性因子
        }

        // 合并配置
        const finalConfig = { ...defaultConfig, ...config }

        const requestBody = {
            symbol: finalConfig.symbol,
            start_date: finalConfig.startDate,
            end_date: finalConfig.endDate,
            strategy: finalConfig.strategy,
            initial_cash: finalConfig.initialCapital,
            max_position: finalConfig.positionSize / 100, // 将百分比转换为小数
            stop_loss: finalConfig.stopLoss,
            take_profit: finalConfig.takeProfit,
            commission: finalConfig.commission,
            timeframe: finalConfig.timeframe,

            // 添加现实性参数
            slippage: finalConfig.slippage,
            market_impact: finalConfig.marketImpact,
            trading_delay: finalConfig.tradingDelay,
            spread: finalConfig.spread,
            min_order_size: finalConfig.minOrderSize,
            max_order_size: finalConfig.maxOrderSize,
            liquidity_factor: finalConfig.liquidityFactor
        }

        // 如果启用了自动执行，添加相关参数
        if (finalConfig.autoExecute) {
            requestBody.auto_execute = true
            requestBody.auto_execute_risk_level = finalConfig.autoExecuteRiskLevel
            requestBody.min_confidence = finalConfig.minConfidence
            requestBody.max_position_percent = finalConfig.maxPositionPercent
            requestBody.skip_existing_trades = finalConfig.skipExistingTrades
        }

        return postJSON('/api/ai-recommendation/backtest', requestBody)
    },

    // ---- 特征工程系统 (新增) ----
    // POST /api/v1/features/extract - 提取单币种特征
    extractFeatures ({ symbol, include_deep = true, feature_types = ['time_series', 'volatility', 'trend'] } = {}) {
        return postJSON('/api/v1/features/extract', { symbol, include_deep, feature_types })
    },

    // POST /api/v1/features/batch-extract - 批量提取特征
    batchExtractFeatures ({ symbols, feature_types = ['time_series', 'volatility'], batch_size = 10 } = {}) {
        return postJSON('/api/v1/features/batch-extract', { symbols, feature_types, batch_size })
    },

    // GET /api/v1/features/importance - 特征重要性分析
    getFeatureImportance ({ limit = 20, model = 'random_forest' } = {}) {
        return getJSON('/api/v1/features/importance', { limit, model })
    },

    // GET /api/v1/features/quality - 特征质量报告
    getFeatureQuality ({ symbol, time_range = '7d' } = {}) {
        return getJSON('/api/v1/features/quality', { symbol, time_range })
    },

    // ---- 机器学习系统 (新增) ----
    // GET /api/v1/ml/models/performance - 模型性能
    getModelPerformance () {
        return getJSON('/api/v1/ml/models/performance')
    },

    // POST /api/v1/ml/models/train - 训练新模型
    trainModel (config) {
        return postJSON('/api/v1/ml/models/train', config)
    },

    // POST /api/v1/ml/predict - 进行预测
    predictWithModel ({ symbol, model_name = 'random_forest', include_features = true, include_confidence = true } = {}) {
        return postJSON('/api/v1/ml/predict', { symbol, model_name, include_features, include_confidence })
    },

    // ---- 数据质量系统 (新增) ----
    // GET /api/v1/data-quality/report - 数据质量报告
    getDataQualityReport () {
        return getJSON('/api/v1/data-quality/report')
    },

    // GET /api/v1/data-sources/status - 数据源状态
    getDataSourcesStatus () {
        return getJSON('/api/v1/data-sources/status')
    },

    // GET /api/v1/data-fusion/stats - 数据融合统计
    getDataFusionStats () {
        return getJSON('/api/v1/data-fusion/stats')
    },

    // ---- 系统监控 (新增) ----
    // GET /api/v1/status - 系统状态
    getSystemStatus () {
        return getJSON('/api/v1/status')
    },

    // GET /api/v1/stats - 系统统计
    getSystemStats () {
        return getJSON('/api/v1/stats')
    },

    // ---- Cache Management ----
    // GET /cache/stats
    getCacheStats () {
        return getJSON('/cache/stats')
    },

    // POST /cache/warmup
    warmupCache () {
        return postJSON('/cache/warmup')
    },

    // POST /cache/clear
    clearCache () {
        return postJSON('/cache/clear')
    },

    // POST /cache/invalidate/user/:userId
    invalidateUserCache (userId) {
        return postJSON(`/cache/invalidate/user/${userId}`)
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

    // ---- 机器学习 (AI增强) ----
    // GET /api/v1/ml/models/performance
    getMLModelPerformance () {
        return getJSON('/api/v1/ml/models/performance')
    },
    // POST /api/v1/ml/models/train
    trainMLModel (config) {
        return postJSON('/api/v1/ml/models/train', config)
    },
    // POST /api/v1/ml/predict
    predictWithML (features) {
        return postJSON('/api/v1/ml/predict', { features })
    },
    // GET /api/v1/ml/stats
    getMLStats () {
        return getJSON('/api/v1/ml/stats')
    },

    // ---- Transformer模型 ----
    // POST /api/v1/ml/transformer/train
    trainTransformerModel (trainingData) {
        return postJSON('/api/v1/ml/transformer/train', trainingData)
    },
    // POST /api/v1/ml/transformer/predict
    predictWithTransformer (features) {
        return postJSON('/api/v1/ml/transformer/predict', { features })
    },
    // POST /api/v1/ml/transformer/features
    extractTransformerFeatures (timeSeriesData) {
        return postJSON('/api/v1/ml/transformer/features', { time_series_data: timeSeriesData })
    },

    // ---- 高级特征工程 ----
    // POST /api/v1/features/advanced-extract
    extractAdvancedFeatures (marketData) {
        return postJSON('/api/v1/features/advanced-extract', { market_data: marketData })
    },
    // GET /api/v1/features/importance-analysis
    getFeatureImportanceAnalysis () {
        return getJSON('/api/v1/features/importance-analysis')
    },

    // ---- 市场分析 API ----
    // 获取市场环境分析
    getMarketAnalysis() {
        return requestJSON('GET', '/market-analysis/environment')
    },

    // 获取技术指标数据
    getTechnicalIndicators() {
        return requestJSON('GET', '/market-analysis/technical-indicators')
    },

    // 获取策略推荐
    getStrategyRecommendations() {
        return requestJSON('GET', '/market-analysis/strategy-recommendations')
    },

    // 获取综合市场分析数据（推荐使用）
    getComprehensiveMarketAnalysis() {
        return requestJSON('GET', '/market-analysis/comprehensive')
    },
}

// ===== 市场分析 API =====

// 获取市场环境分析
export async function getMarketAnalysis() {
    return requestJSON('GET', '/market-analysis/environment')
}

// 获取技术指标数据
export async function getTechnicalIndicators() {
    return requestJSON('GET', '/market-analysis/technical-indicators')
}

// 获取策略推荐
export async function getStrategyRecommendations() {
    return requestJSON('GET', '/market-analysis/strategy-recommendations')
}

// 获取综合市场分析数据（推荐使用）
export async function getComprehensiveMarketAnalysis() {
    return requestJSON('GET', '/market-analysis/comprehensive')
}

// 获取市场波动率分析
export async function getMarketVolatilityAnalysis() {
    return requestJSON('GET', '/market-analysis/volatility')
}

// 若你有少量地方直接用到基础封装，也保留导出
export { getJSON, postJSON, putJSON, patchJSON, delJSON }
