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
import RealTimeGainers from "../views/RealTimeGainers.vue"
import Announcements from "../views/Announcements.vue"
import TwitterFeed from "../views/TwitterFeed.vue"
import Recommendations from "../views/Recommendations.vue"
import WhaleWatch from "../views/WhaleWatch.vue"
import DataQuality from "../views/DataQuality.vue"
import BacktestAdvanced from "../views/BacktestAdvanced.vue"
import AIRecommendations from "../views/AIRecommendations.vue"
import AIRecommendationDetail from "../views/AIRecommendationDetail.vue"
import RiskMonitoring from "../views/RiskMonitoring.vue"
import AdvancedRisk from "../views/AdvancedRisk.vue"
import AdvancedBacktest from "../views/AdvancedBacktest.vue"
import AIAnalysisDashboard from "../views/AIAnalysisDashboard.vue"
import AILab from "../views/AILab.vue"
import AIDashboard from "../views/AIDashboard.vue"
import HistoricalRecommendations from "../views/HistoricalRecommendations.vue"
import BacktestDetail from "../views/BacktestDetail.vue"
import BacktestHistory from "../views/BacktestHistory.vue"
import RealTimeQuotes from "../views/RealTimeQuotes.vue"
import ScheduledOrderDetail from "../views/ScheduledOrderDetail.vue"
import StrategyStats from "../views/StrategyStats.vue"
import DataSyncMonitor from "../views/DataSyncMonitor.vue"
import CreateStrategy from "../views/CreateStrategy.vue"

import { useAuth } from '../stores/auth.js'

const routes = [
    { path: '/', redirect: '/market' },
    { path: '/login', component: Login, meta: { requiresAuth: false, title: '登录 - 区块链数据分析平台' } },
    // { path: '/register', component: Register, meta: { requiresAuth: false, title: '注册 - 区块链数据分析平台' } },

    // 除了定时合约单，其他页面都不需要登录
    { path: '/dashboard', component: Dashboard, meta: { requiresAuth: false, keepAlive: true, title: '仪表盘 - 投资组合分析' } },
    { path: '/flows', component: Flows, meta: { requiresAuth: false, title: '资金流分析' } },
    { path: '/runs', component: Runs, meta: { requiresAuth: false, title: '运行记录' } },
    { path: '/transfers', component: Transfers, meta: { requiresAuth: false, keepAlive: true, title: '转账记录 - 链上数据分析' } },
    { path: '/chain-flows', component: ChainFlows, meta: { requiresAuth: false, keepAlive: true, title: '链上资金流 - 实时监控' } },
    { path: '/whales', component: WhaleWatch, meta: { requiresAuth: false, keepAlive: true, title: '大户/机构监控' } },
    { path: '/scheduled-orders', component: ScheduledOrders, meta: { requiresAuth: true, title: '交易中心 - 定时下单与策略配置' } }, // 交易中心需要登录
    { path: '/create-strategy', component: CreateStrategy, meta: { requiresAuth: true, title: '新建策略 - 量化交易策略配置' } },
    { path: '/strategy-stats/:id', component: StrategyStats, meta: { requiresAuth: true, title: '策略运行统计 - 详细分析' } },
    { path: '/orders/schedule/:id', component: ScheduledOrderDetail, meta: { requiresAuth: true, title: '订单详情 - 合约交易详情' } },
    { path: '/market', component: BinanceGainers, meta: { requiresAuth: false, keepAlive: true, title: '币安涨幅榜 - 市场行情分析' } },
    { path: '/realtime-gainers', component: RealTimeGainers, meta: { requiresAuth: false, keepAlive: true, title: '实时涨幅榜 - 币安实时行情' } },
    { path: '/realtime-quotes', component: RealTimeQuotes, meta: { requiresAuth: false, keepAlive: true, title: '实时行情 - 币种实时价格监控' } },
    { path: '/announcements', component: Announcements, meta: { requiresAuth: false, keepAlive: true, title: '项目公告 - 最新动态' } },
    { path: '/twitter', component: TwitterFeed, meta: { requiresAuth: false, keepAlive: true, title: 'Twitter动态 - 社交媒体监控' } },
    { path: '/recommendations', component: Recommendations, meta: { requiresAuth: false, keepAlive: true, title: '智能投研 - AI投资策略分析' } },
    { path: '/ai-dashboard', component: AIDashboard, meta: { requiresAuth: false, keepAlive: true, title: 'AI投资仪表盘 - 智能投资决策平台' } },
    { path: '/ai-lab', component: AILab, meta: { requiresAuth: false, keepAlive: true, title: 'AI实验室 - 高级机器学习实验平台' } },
    { path: '/ai-recommendations', component: AIRecommendations, meta: { requiresAuth: false, keepAlive: true, title: 'AI智能推荐 - 机器学习币种推荐' } },
    { path: '/ai-recommendation/:symbol', component: AIRecommendationDetail, meta: { requiresAuth: false, title: 'AI推荐详情 - 详细分析报告' } },
    { path: '/historical-recommendations', component: HistoricalRecommendations, meta: { requiresAuth: false, keepAlive: true, title: '历史推荐查询 - 回顾过去投资机会' } },
    { path: '/advanced-risk', component: AdvancedRisk, meta: { requiresAuth: false, keepAlive: true, title: '高级风险管理 - 专业量化风险分析' } },
    { path: '/advanced-backtest', component: AdvancedBacktest, meta: { requiresAuth: false, keepAlive: true, title: '高级回测分析 - 专业策略验证与优化' } },
    { path: '/ai-analysis-dashboard', component: AIAnalysisDashboard, meta: { requiresAuth: false, keepAlive: true, title: 'AI推荐分析仪表板 - 深度策略分析平台' } },
    { path: '/backtest/:id', component: BacktestDetail, meta: { requiresAuth: false, title: '回测详情 - 详细分析报告' } },
    { path: '/risk-monitoring', component: RiskMonitoring, meta: { requiresAuth: false, keepAlive: true, title: '风险监控中心 - 实时风险评估与告警' } },
    { path: '/data-quality', component: DataQuality, meta: { requiresAuth: false, keepAlive: true, title: '数据质量监控 - 多源数据集成' } },
    { path: '/data-sync-monitor', component: DataSyncMonitor, meta: { requiresAuth: false, keepAlive: true, title: '数据同步监控 - 系统状态实时监控' } },
    { path: '/backtest', component: BacktestAdvanced, meta: { requiresAuth: false, title: '策略回测 - 专业量化分析' } },
    { path: '/backtest-history', component: BacktestHistory, meta: { requiresAuth: true, keepAlive: true, title: '回测记录历史 - 查看历史回测结果' } },
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

// 路由变化后更新页面标题（SEO优化）
router.afterEach((to) => {
    const title = to.meta?.title || '区块链数据分析平台'
    document.title = title
})

export default router
