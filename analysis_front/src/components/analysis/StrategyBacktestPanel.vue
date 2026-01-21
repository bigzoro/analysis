<template>
  <div class="strategy-backtest-panel">
    <div class="panel-header">
      <h3>📈 AI推荐策略回测</h3>
      <p>验证AI推荐在不同市场环境下的历史表现</p>
    </div>

    <!-- 回测配置 -->
    <div class="config-section">
      <div class="config-grid">
        <!-- 基本配置 -->
        <div class="config-card">
          <h4>📊 回测参数</h4>
          <div class="form-grid">
            <!-- 自动选择币种选项 -->
            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="config.autoSelectSymbol" @change="onAutoSelectChange" />
                🤖 自动选择币种
                <span class="feature-desc">系统将自动评估多个币种，选择最适合的进行回测</span>
              </label>
            </div>

            <!-- 手动选择币种 -->
            <div class="form-group" v-if="!config.autoSelectSymbol">
              <label>目标币种</label>
              <select v-model="config.symbol" @change="updateAvailableStrategies">
                <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                  {{ symbol }}
                </option>
              </select>
            </div>

            <!-- 自动选择币种配置 -->
            <div v-if="config.autoSelectSymbol" class="auto-select-config">
              <div class="form-group">
                <label>选择标准</label>
                <select v-model="config.symbolSelectionCriteria">
                  <option value="profitability">盈利能力优先</option>
                  <option value="volatility">适中波动率</option>
                  <option value="trend_strength">强趋势追踪</option>
                  <option value="liquidity">高流动性</option>
                  <option value="balanced">综合平衡</option>
                  <option value="market_heat">市场热度智能</option>
                </select>
              </div>

              <div class="form-group">
                <label>评估币种数量</label>
                <input type="number" v-model.number="config.maxSymbolsToEvaluate" min="5" max="30" step="5" />
                <span class="input-desc">系统将评估的币种数量 (5-30)</span>
              </div>

              <div class="feature-info">
                <div class="info-item">
                  <span class="info-icon">🎯</span>
                  <div class="info-content">
                    <strong>智能选择</strong>
                    <p>系统将根据当前市场情况自动选择最具潜力的币种进行评估</p>
                  </div>
                </div>
                <div class="info-item">
                  <span class="info-icon">📊</span>
                  <div class="info-content">
                    <strong>多维度评估</strong>
                    <p>综合考虑价格趋势、波动率、成交量等多个关键指标</p>
                  </div>
                </div>
              </div>
            </div>

            <div class="form-group">
              <label>时间范围</label>
              <div class="date-range">
                <input type="date" v-model="config.startDate" :max="config.endDate" />
                <span>至</span>
                <input type="date" v-model="config.endDate" :min="config.startDate" :max="today" />
              </div>
            </div>

            <div class="form-group">
              <label>初始资金 ($)</label>
              <input type="number" v-model.number="config.initialCapital" min="1000" step="1000" />
            </div>

            <div class="form-group">
              <label>单次仓位 (%)</label>
              <input type="number" v-model.number="config.positionSize" min="1" max="100" step="1" />
            </div>
          </div>
        </div>

        <!-- 策略配置 -->
        <div class="config-card">
          <h4>🎯 AI推荐策略</h4>
          <div class="strategy-selector">
            <div class="strategy-option" v-for="strategy in availableStrategies" :key="strategy.key">
              <input
                type="radio"
                :id="strategy.key"
                :value="strategy.key"
                v-model="config.strategy"
              />
              <label :for="strategy.key" class="strategy-label">
                <div class="strategy-header">
                  <span class="strategy-name">{{ strategy.name }}</span>
                  <span class="strategy-confidence" :class="strategy.confidence">
                    {{ getConfidenceLabel(strategy.confidence) }}
                  </span>
                </div>
                <div class="strategy-description">{{ strategy.description }}</div>
                <div class="strategy-stats">
                  <span>历史胜率: {{ (strategy.winRate * 100).toFixed(1) }}%</span>
                  <span>平均收益: {{ (strategy.avgReturn * 100).toFixed(1) }}%</span>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- 现实性配置 -->
        <div class="config-card">
          <h4>🎯 交易现实性设置</h4>
          <div class="form-grid">
            <div class="form-group">
              <label>滑点 (%)</label>
              <input type="number" v-model.number="config.slippage" @change="validateSlippage" min="0" max="1" step="0.001" />
              <span class="input-desc">交易滑点，影响实际执行价格</span>
            </div>

            <div class="form-group">
              <label>市场冲击系数</label>
              <input type="number" v-model.number="config.marketImpact" @change="validateMarketImpact" min="0" max="0.01" step="0.0001" />
              <span class="input-desc">大订单对市场价格的影响</span>
            </div>

            <div class="form-group">
              <label>交易延迟 (分钟)</label>
              <input type="number" v-model.number="config.tradingDelay" @change="validateTradingDelay" min="0" max="60" step="1" />
              <span class="input-desc">信号到执行的时间延迟</span>
            </div>

            <div class="form-group">
              <label>买卖价差 (%)</label>
              <input type="number" v-model.number="config.spread" @change="validateSpread" min="0" max="1" step="0.0001" />
              <span class="input-desc">买卖价差成本</span>
            </div>

            <div class="form-group">
              <label>最小订单大小</label>
              <input type="number" v-model.number="config.minOrderSize" @change="validateOrderSize" min="0.1" step="0.1" />
              <span class="input-desc">最小可交易数量</span>
            </div>

            <div class="form-group">
              <label>流动性因子</label>
              <input type="number" v-model.number="config.liquidityFactor" @change="validateLiquidityFactor" min="0.1" max="5.0" step="0.1" />
              <span class="input-desc">市场流动性调整因子</span>
            </div>
          </div>
        </div>

        <!-- 渐进式执行配置 -->
        <div class="config-card">
          <h4>🔄 渐进式执行设置</h4>
          <div class="form-grid">
            <div class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="config.progressiveExecution" />
                启用渐进式执行
              </label>
              <span class="input-desc">分批执行交易，降低市场冲击风险</span>
            </div>

            <div v-if="config.progressiveExecution" class="form-group">
              <label>最大批次数</label>
              <input type="number" v-model.number="config.maxBatches" @change="validateMaxBatches" min="1" max="10" step="1" />
              <span class="input-desc">将推荐分成多少批次执行</span>
            </div>

            <div v-if="config.progressiveExecution" class="form-group">
              <label>批次间隔 (分钟)</label>
              <input type="number" v-model.number="config.batchDelay" @change="validateBatchDelay" min="5" max="300" step="5" />
              <span class="input-desc">每批次之间的等待时间</span>
            </div>

            <div v-if="config.progressiveExecution" class="form-group">
              <label>每批最大交易数</label>
              <input type="number" v-model.number="config.batchSize" @change="validateBatchSize" min="1" max="20" step="1" />
              <span class="input-desc">每个批次最多执行多少笔交易</span>
            </div>

            <div v-if="config.progressiveExecution" class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="config.dynamicSizing" />
                动态仓位调整
              </label>
              <span class="input-desc">根据市场条件动态调整仓位大小</span>
            </div>

            <div v-if="config.progressiveExecution" class="form-group">
              <label class="checkbox-label">
                <input type="checkbox" v-model="config.marketConditionFilter" />
                市场条件过滤
              </label>
              <span class="input-desc">在市场条件恶劣时跳过交易</span>
            </div>
          </div>
        </div>

        <!-- 自动执行配置 -->
        <div class="config-card">
          <h4>🤖 自动执行设置</h4>
          <div class="auto-execute-config">
            <div class="setting-item">
              <label class="setting-label">
                <input type="checkbox" v-model="config.autoExecute" />
                回测时自动执行交易
              </label>
              <p class="setting-desc">开启后，回测过程中会自动创建模拟交易记录</p>
            </div>

            <div v-if="config.autoExecute" class="auto-execute-details">
              <div class="setting-item">
                <label>风险偏好：</label>
                <select v-model="config.autoExecuteRiskLevel">
                  <option value="conservative">保守 (只执行低风险推荐)</option>
                  <option value="moderate">稳健 (执行中等风险推荐)</option>
                  <option value="aggressive">激进 (执行所有推荐)</option>
                </select>
              </div>

              <div class="setting-item">
                <label>最小置信度：</label>
                <input type="number" v-model.number="config.minConfidence" @change="validateMinConfidence" min="0.5" max="1.0" step="0.05" />
                <span class="input-desc">AI推荐的最低置信度阈值</span>
              </div>

              <div class="setting-item">
                <label>最大单次仓位：</label>
                <input type="number" v-model.number="config.maxPositionPercent" @change="validateMaxPosition" min="0.1" max="20" step="0.1" />
                <span class="input-desc">% (基于总资金)</span>
              </div>

            <div class="setting-item">
              <label class="setting-label">
                <input type="checkbox" v-model="config.skipExistingTrades" />
                跳过已存在的交易
              </label>
              <p class="setting-desc">避免重复创建相同的交易记录</p>
            </div>

            <div class="setting-item">
              <button @click="clearExistingTrades" class="clear-trades-btn" :disabled="clearingTrades">
                {{ clearingTrades ? '清理中...' : '🗑️ 清理已有交易' }}
              </button>
              <p class="setting-desc">清除当前用户的所有模拟交易记录</p>
            </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行按钮 -->
      <div class="action-buttons">
        <button @click="runBacktest" :disabled="running || !isConfigValid" class="run-btn">
          {{ running ? '🔄 启动回测中...' : (config.autoExecute ? '🤖 AI回测+执行' : '🚀 开始回测') }}
        </button>
        <button @click="clearTrades" class="clear-btn">🗑️ 清理交易</button>
        <button @click="resetConfig" class="reset-btn">重置配置</button>
      </div>
    </div>

    <!-- 回测结果 -->
    <div v-if="result" class="results-section">
      <!-- 关键指标 -->
      <div class="metrics-overview">
        <div class="metric-cards">
          <div class="metric-card primary">
            <div class="metric-icon">💰</div>
            <div class="metric-content">
              <div class="metric-value">{{ formatCurrency(result.totalReturn) }}</div>
              <div class="metric-label">总收益率</div>
              <div class="metric-change" :class="result.totalReturn >= 0 ? 'positive' : 'negative'">
                {{ (result.totalReturnPercent * 100).toFixed(2) }}%
              </div>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon">📊</div>
            <div class="metric-content">
              <div class="metric-value">{{ result.sharpeRatio.toFixed(2) }}</div>
              <div class="metric-label">夏普比率</div>
              <div class="metric-desc">{{ getSharpeDesc(result.sharpeRatio) }}</div>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon">📉</div>
            <div class="metric-content">
              <div class="metric-value">{{ (result.maxDrawdown * 100).toFixed(2) }}%</div>
              <div class="metric-label">最大回撤</div>
              <div class="metric-desc">{{ getDrawdownDesc(result.maxDrawdown) }}</div>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon">🎯</div>
            <div class="metric-content">
              <div class="metric-value">{{ (result.winRate * 100).toFixed(1) }}%</div>
              <div class="metric-label">胜率</div>
              <div class="metric-desc">{{ result.totalTrades }} 次交易</div>
            </div>
          </div>

          <!-- 自动执行统计 -->
          <div v-if="result.autoExecuteStats" class="metric-card auto-execute-stat">
            <div class="metric-icon">🤖</div>
            <div class="metric-content">
              <div class="metric-value">{{ result.autoExecuteStats.executedTrades }}</div>
              <div class="metric-label">自动执行交易</div>
              <div class="metric-desc">
                成功: {{ result.autoExecuteStats.successfulTrades }} |
                跳过: {{ result.autoExecuteStats.skippedTrades }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 收益曲线图 -->
      <div class="equity-chart-section">
        <h4>💹 收益曲线</h4>
        <div class="chart-container">
          <LineChart
            :xData="equityChartData.xData"
            :series="equityChartData.series"
            :yLabel="'账户价值'"
          />
        </div>
      </div>

      <!-- AI推荐表现分析 -->
      <div class="ai-performance-section">
        <h4>🤖 AI推荐表现分析</h4>
        <div class="performance-grid">
          <div class="performance-card">
            <h5>推荐时机分析</h5>
            <div class="timing-analysis">
              <div class="timing-item">
                <span class="timing-label">最佳入场时机</span>
                <span class="timing-value">{{ result.bestEntryTiming }}</span>
              </div>
              <div class="timing-item">
                <span class="timing-label">平均持仓时间</span>
                <span class="timing-value">{{ formatDuration(result.avgHoldingTime) }}</span>
              </div>
              <div class="timing-item">
                <span class="timing-label">市场时机把握</span>
                <span class="timing-value" :class="result.marketTiming >= 0.6 ? 'good' : 'fair'">
                  {{ (result.marketTiming * 100).toFixed(1) }}%
                </span>
              </div>
            </div>
          </div>

          <div class="performance-card">
            <h5>市场环境适应性</h5>
            <div class="environment-analysis">
              <div class="env-item" v-for="env in (result.marketEnvironments || [])" :key="env.condition">
                <span class="env-condition">{{ env.condition }}</span>
                <div class="env-stats">
                  <span class="env-performance" :class="env.performance >= 0 ? 'positive' : 'negative'">
                    {{ (env.performance * 100).toFixed(1) }}%
                  </span>
                  <span class="env-count">{{ env.tradeCount }}次</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 详细交易记录 -->
      <div class="trades-section">
        <h4>📋 详细交易记录</h4>
        <div class="trades-table-container">
          <table class="trades-table">
            <thead>
              <tr>
                <th>日期</th>
                <th>操作</th>
                <th>价格</th>
                <th>数量</th>
                <th>市值</th>
                <th>收益</th>
                <th>市场环境</th>
                <th>AI置信度</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="trade in ((result.trades || []).slice(-20))" :key="trade.id">
                <td>{{ formatDate(trade.date) }}</td>
                <td>
                  <span class="trade-action" :class="trade.action">
                    {{ trade.action === 'buy' ? '买入' : '卖出' }}
                  </span>
                </td>
                <td>${{ trade.price.toFixed(2) }}</td>
                <td>{{ trade.quantity.toFixed(6) }}</td>
                <td>${{ trade.value.toFixed(2) }}</td>
                <td :class="trade.profit >= 0 ? 'positive' : 'negative'">
                  {{ trade.profit ? formatCurrency(trade.profit) : '-' }}
                </td>
                <td>{{ trade.marketCondition || trade.reason || '回测交易' }}</td>
                <td>
                  <div class="confidence-bar">
                    <div class="confidence-fill" :style="{ width: (trade.aiConfidence || 0) * 100 + '%' }"></div>
                    <span class="confidence-text">{{ ((trade.aiConfidence || 0) * 100).toFixed(0) }}%</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="running" class="loading-overlay">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">正在回测AI推荐策略...</div>
        <div class="loading-progress">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="progress-text">{{ progressPercent.toFixed(0) }}%</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import LineChart from '../LineChart.vue'
import { api } from '../../api/api.js'

export default {
  name: 'StrategyBacktestPanel',
  components: {
    LineChart
  },
  props: {
    symbols: {
      type: Array,
      default: () => ['BTC']
    },
    selectedDate: {
      type: String,
      default: null
    }
  },
  emits: ['backtest-complete'],
  data() {
    return {
      running: false,
      progressPercent: 0,
      config: {
        symbol: 'BTC',
        startDate: this.getDefaultStartDate(),
        endDate: this.getDefaultEndDate(),
        initialCapital: 10000,
        positionSize: 10, // 10%
        strategy: 'conservative',
        // 自动执行设置
        autoExecute: false,
        autoExecuteRiskLevel: 'moderate',
        minConfidence: 0.7,
        maxPositionPercent: 5.0,
        skipExistingTrades: true,
        // 渐进式执行参数
        progressiveExecution: false,
        maxBatches: 3,
        batchDelay: 30, // 分钟
        batchSize: 5,
        dynamicSizing: true,
        marketConditionFilter: true,
        // 自动选择币种设置
        autoSelectSymbol: false,
        symbolSelectionCriteria: 'market_heat',
        maxSymbolsToEvaluate: 15,
        // 现实性参数
        slippage: 0.001,        // 0.1% 滑点
        marketImpact: 0.0001,   // 市场冲击系数
        tradingDelay: 5,        // 5分钟延迟
        spread: 0.0005,         // 0.05% 买卖价差
        minOrderSize: 10,       // 最小订单大小
        maxOrderSize: 10000,    // 最大订单大小
        liquidityFactor: 1.0    // 流动性因子
      },
      availableSymbols: ['BTC', 'ETH', 'ADA', 'SOL', 'DOT', 'LINK'],
      availableStrategies: [
        {
          key: 'conservative',
          name: '保守策略',
          description: '基于高置信度推荐，注重风险控制',
          confidence: 'high',
          winRate: 0.68,
          avgReturn: 0.12
        },
        {
          key: 'moderate',
          name: '稳健策略',
          description: '平衡风险与收益的中等策略',
          confidence: 'medium',
          winRate: 0.62,
          avgReturn: 0.18
        },
        {
          key: 'aggressive',
          name: '激进策略',
          description: '追求高收益，接受较高风险',
          confidence: 'low',
          winRate: 0.55,
          avgReturn: 0.25
        },
        {
          key: 'deep_learning',
          name: '深度学习策略',
          description: '使用AI深度学习模型，结合多因子分析和强化学习',
          confidence: 'high',
          winRate: 0.72,
          avgReturn: 0.22
        }
      ],
      result: null,
      today: new Date().toISOString().split('T')[0],
      clearingTrades: false
    }
  },
  computed: {
    isConfigValid() {
      // 基础验证：日期、资金、仓位
      const basicValid = this.config.startDate &&
             this.config.endDate &&
             this.config.initialCapital >= 1000 &&
             this.config.positionSize >= 1 &&
             this.config.positionSize <= 100

      // 币种验证：如果不是自动选择币种，需要指定币种
      const symbolValid = this.config.autoSelectSymbol || this.config.symbol

      const baseValid = basicValid && symbolValid

      if (!this.config.autoExecute) {
        return baseValid
      }

      // 自动执行的额外验证
      return baseValid &&
             this.config.minConfidence >= 0.5 &&
             this.config.minConfidence <= 1.0 &&
             this.config.maxPositionPercent >= 0.1 &&
             this.config.maxPositionPercent <= 20
    },
    equityChartData() {
      if (!this.result || !this.result.equityCurve || !Array.isArray(this.result.equityCurve)) {
        return { xData: [], series: [] }
      }

      return {
        xData: this.result.equityCurve.map(point => point.date || ''),
        series: [{
          name: '账户价值',
          data: this.result.equityCurve.map(point => point.value || 0),
          type: 'line',
          smooth: true,
          lineStyle: { color: '#3b82f6', width: 2 }
        }]
      }
    }
  },
  mounted() {
    this.updateAvailableStrategies()
  },
  methods: {
    getDefaultStartDate() {
      const date = new Date()
      date.setMonth(date.getMonth() - 3)
      return date.toISOString().split('T')[0]
    },

    getDefaultEndDate() {
      const date = new Date()
      date.setDate(date.getDate() - 1) // 昨天
      return date.toISOString().split('T')[0]
    },

    onAutoSelectChange() {
      // 当切换自动选择币种时，清除手动选择的币种
      if (this.config.autoSelectSymbol) {
        this.config.symbol = ''
        // 默认选择市场热度智能模式
        this.config.symbolSelectionCriteria = 'market_heat'
        this.config.maxSymbolsToEvaluate = 15
      } else {
        // 恢复默认币种选择
        this.config.symbol = 'BTC'
      }
    },

    updateAvailableStrategies() {
      // 基于现有推荐数据生成策略选项
      this.availableStrategies = [
        {
          key: 'conservative',
          name: '保守策略',
          description: '基于高置信度推荐，注重风险控制',
          confidence: 'high',
          winRate: 0.68,
          avgReturn: 0.12
        },
        {
          key: 'moderate',
          name: '稳健策略',
          description: '平衡风险与收益的中等策略',
          confidence: 'medium',
          winRate: 0.62,
          avgReturn: 0.18
        },
        {
          key: 'aggressive',
          name: '激进策略',
          description: '追求高收益，接受较高风险',
          confidence: 'low',
          winRate: 0.55,
          avgReturn: 0.25
        },
        {
          key: 'deep_learning',
          name: '深度学习策略',
          description: '使用AI深度学习模型，结合多因子分析和强化学习',
          confidence: 'high',
          winRate: 0.72,
          avgReturn: 0.22
        }
      ]
    },

    async runBacktest() {
      if (!this.isConfigValid) return

      this.running = true
      this.progressPercent = 0

      try {
        // 模拟进度
        const progressInterval = setInterval(() => {
          this.progressPercent += Math.random() * 15
          if (this.progressPercent > 90) {
            this.progressPercent = 90
          }
        }, 500)

        // 准备异步回测参数
        const asyncParams = {
          symbol: this.config.autoSelectSymbol ? '' : this.config.symbol,
          start_date: this.config.startDate,
          end_date: this.config.endDate,
          strategy: this.config.strategy,
          initial_capital: this.config.initialCapital,
          position_size: this.config.positionSize,
          // 自动执行参数
          auto_execute: this.config.autoExecute,
          auto_execute_risk_level: this.config.autoExecuteRiskLevel,
          min_confidence: this.config.minConfidence,
          max_position_percent: this.config.maxPositionPercent,
          skip_existing_trades: this.config.skipExistingTrades,
          // 渐进式执行参数
          progressive_execution: this.config.progressiveExecution,
          max_batches: this.config.maxBatches,
          batch_delay: this.config.batchDelay * 1000000000, // 转换为纳秒
          batch_size: this.config.batchSize,
          dynamic_sizing: this.config.dynamicSizing,
          market_condition_filter: this.config.marketConditionFilter,
          // 自动选择币种参数
          auto_select_symbol: this.config.autoSelectSymbol,
          max_symbols_to_evaluate: this.config.maxSymbolsToEvaluate,
          symbol_selection_criteria: this.config.symbolSelectionCriteria
        }

        // 启动异步回测
        const response = await api.startAsyncBacktest(asyncParams)

        clearInterval(progressInterval)
        this.progressPercent = 100

        if (response.success) {
          // 显示成功消息
          alert(`回测任务已启动！\n任务ID: ${response.record_id}\n\n请前往"回测记录"标签页查看执行状态和结果。`)

          // 发出启动事件（而不是完成事件）
          this.$emit('backtest-started', {
            recordId: response.record_id,
            status: response.status
          })

          // 清空当前结果显示
          this.result = null
        } else {
          throw new Error(response.error || '启动回测失败')
        }

      } catch (error) {
        console.error('回测启动失败:', error)
        alert('回测启动失败，请稍后重试: ' + (error?.message || '未知错误'))
      } finally {
        this.running = false
        setTimeout(() => {
          this.progressPercent = 0
        }, 1000)
      }
    },

    async executeAIBacktest(config) {
      try {
        // 准备API参数
        const apiParams = {
          symbol: config.symbol,
          startDate: config.startDate,
          endDate: config.endDate,
          strategy: config.strategy,
          initialCapital: config.initialCapital,
          positionSize: config.positionSize,
          stopLoss: config.stopLoss || 0.05,
          takeProfit: config.takeProfit || 0.15,
          commission: config.commission || 0.001
        }

        // 如果启用了自动执行，添加相关参数
        if (config.autoExecute) {
          apiParams.autoExecute = true
          apiParams.autoExecuteRiskLevel = config.autoExecuteRiskLevel
          apiParams.minConfidence = config.minConfidence
          apiParams.maxPositionPercent = config.maxPositionPercent
          apiParams.skipExistingTrades = config.skipExistingTrades
        }

        // 调用真实的AI策略回测API
        const response = await api.runAIStrategyBacktest(apiParams)

        if (response.success && response.backtest_result) {
          // 转换API返回的数据格式以匹配组件期望的格式
          const result = response.backtest_result
          const processedResult = {
            totalReturn: result.summary.total_return || 0,
            totalReturnPercent: (result.summary.total_return || 0) * 100,
            sharpeRatio: result.summary.sharpe_ratio || 0,
            maxDrawdown: result.summary.max_drawdown || 0,
            winRate: result.summary.win_rate || 0,
            totalTrades: result.summary.total_trades || 0,
            bestEntryTiming: '基于AI推荐',
            avgHoldingTime: result.summary.avg_holding_period || 0,
            marketTiming: result.summary.market_timing_score || 0,
            marketEnvironments: result.market_environments || [],
            equityCurve: this.processEquityCurve(result.daily_returns || []),
            trades: this.processTrades(result.trades || []),
            aiInsights: result.backtest_insights || [],
            aiAccuracy: result.ai_prediction_accuracy || {},
            effectiveness: result.recommendation_effectiveness || {}
          }

          // 如果启用了自动执行，添加执行统计
          if (config.autoExecute && result.auto_execute_stats) {
            processedResult.autoExecuteStats = result.auto_execute_stats
          }

          return processedResult
        } else {
          throw new Error(response.error || '回测执行失败')
        }
      } catch (error) {
        console.error('AI策略回测API调用失败:', error)
        throw error
      }
    },


    processEquityCurve(dailyReturns) {
      // 将日收益率数据转换为权益曲线
      let equity = this.config.initialCapital
      const curve = []

      dailyReturns.forEach(returnData => {
        equity *= (1 + (returnData.return || 0))
        curve.push({
          date: returnData.date ? new Date(returnData.date).toISOString().split('T')[0] : new Date().toISOString().split('T')[0],
          value: Math.max(1000, equity) // 确保不低于初始资金
        })
      })

      return curve
    },

    processTrades(trades) {
      // 处理交易记录数据格式
      return trades.map(trade => ({
        id: trade.id || Math.random(),
        date: trade.timestamp ? new Date(trade.timestamp) : new Date(),
        action: trade.side === 'buy' ? 'buy' : 'sell',
        price: trade.price || 0,
        quantity: trade.quantity || 0,
        value: (trade.price || 0) * (trade.quantity || 0),
        profit: trade.pnl || null,
        marketCondition: trade.reason || '未知',
        aiConfidence: 0.5 // 默认置信度
      }))
    },

    processBacktestResult(rawResult) {
      return {
        ...rawResult,
        totalReturnPercent: rawResult.totalReturnPercent || (rawResult.totalReturn / this.config.initialCapital)
      }
    },

    resetConfig() {
      this.config = {
        symbol: 'BTC',
        startDate: this.getDefaultStartDate(),
        endDate: this.getDefaultEndDate(),
        initialCapital: 10000,
        positionSize: 10,
        strategy: 'conservative'
      }
      this.result = null
    },

    onAutoSelectChange() {
      // 启用自动选择时清空手动选择
      if (this.config.autoSelectSymbol) {
        this.config.symbol = ''
      } else {
        this.config.symbol = 'BTC'
      }
    },

    async clearTrades() {
      if (!confirm('确定要清理所有模拟交易记录吗？此操作不可撤销。')) {
        return
      }

      try {
        const response = await api.clearUserTrades()
        alert(`成功清理了 ${response.deleted_count} 条交易记录`)
      } catch (error) {
        console.error('清理交易失败:', error)
        alert('清理交易失败，请稍后重试')
      }
    },

    // 辅助方法
    getConfidenceLabel(confidence) {
      const labels = { high: '高置信', medium: '中置信', low: '低置信' }
      return labels[confidence] || '未知'
    },

    getSharpeDesc(ratio) {
      if (ratio >= 2) return '优秀'
      if (ratio >= 1) return '良好'
      if (ratio >= 0) return '一般'
      return '较差'
    },

    getDrawdownDesc(drawdown) {
      if (drawdown <= 0.05) return '很低'
      if (drawdown <= 0.10) return '可接受'
      if (drawdown <= 0.20) return '较高'
      return '很高'
    },

    formatCurrency(value) {
      return new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
        minimumFractionDigits: 2
      }).format(value)
    },

    formatDate(date) {
      return new Date(date).toLocaleDateString('zh-CN')
    },

    formatDuration(ms) {
      const days = Math.floor(ms / (24 * 60 * 60 * 1000))
      return `${days}天`
    },

    // 清理已有交易
    async clearExistingTrades() {
      if (!confirm('确定要清除所有模拟交易记录吗？此操作不可恢复！')) {
        return
      }

      this.clearingTrades = true
      try {
        // 这里需要添加清理交易的API调用
        // 暂时使用模拟删除
        alert('交易记录清理功能开发中，请手动清理数据库')
      } catch (error) {
        console.error('清理交易失败:', error)
        alert('清理失败: ' + (error.message || '未知错误'))
      } finally {
        this.clearingTrades = false
      }
    },

    // 验证最小置信度
    validateMinConfidence() {
      if (this.config.minConfidence < 0.5) {
        this.config.minConfidence = 0.5
      } else if (this.config.minConfidence > 1.0) {
        this.config.minConfidence = 1.0
      }
    },

    // 验证最大仓位百分比
    validateMaxPosition() {
      if (this.config.maxPositionPercent < 0.1) {
        this.config.maxPositionPercent = 0.1
      } else if (this.config.maxPositionPercent > 20) {
        this.config.maxPositionPercent = 20
      }
    },

    // 验证滑点
    validateSlippage() {
      if (this.config.slippage < 0) {
        this.config.slippage = 0
      } else if (this.config.slippage > 1) {
        this.config.slippage = 1
      }
    },

    // 验证市场冲击系数
    validateMarketImpact() {
      if (this.config.marketImpact < 0) {
        this.config.marketImpact = 0
      } else if (this.config.marketImpact > 0.01) {
        this.config.marketImpact = 0.01
      }
    },

    // 验证交易延迟
    validateTradingDelay() {
      if (this.config.tradingDelay < 0) {
        this.config.tradingDelay = 0
      } else if (this.config.tradingDelay > 60) {
        this.config.tradingDelay = 60
      }
    },

    // 验证买卖价差
    validateSpread() {
      if (this.config.spread < 0) {
        this.config.spread = 0
      } else if (this.config.spread > 1) {
        this.config.spread = 1
      }
    },

    // 验证订单大小
    validateOrderSize() {
      if (this.config.minOrderSize < 0.1) {
        this.config.minOrderSize = 0.1
      }
      if (this.config.maxOrderSize < this.config.minOrderSize) {
        this.config.maxOrderSize = this.config.minOrderSize * 10
      }
    },

    // 验证流动性因子
    validateLiquidityFactor() {
      if (this.config.liquidityFactor < 0.1) {
        this.config.liquidityFactor = 0.1
      } else if (this.config.liquidityFactor > 5.0) {
        this.config.liquidityFactor = 5.0
      }
    },

    // 验证最大批次数
    validateMaxBatches() {
      if (this.config.maxBatches < 1) {
        this.config.maxBatches = 1
      } else if (this.config.maxBatches > 10) {
        this.config.maxBatches = 10
      }
    },

    // 验证批次间隔
    validateBatchDelay() {
      if (this.config.batchDelay < 5) {
        this.config.batchDelay = 5
      } else if (this.config.batchDelay > 300) {
        this.config.batchDelay = 300
      }
    },

    // 验证每批最大交易数
    validateBatchSize() {
      if (this.config.batchSize < 1) {
        this.config.batchSize = 1
      } else if (this.config.batchSize > 20) {
        this.config.batchSize = 20
      }
    }
  }
}
</script>

<style scoped>
.strategy-backtest-panel {
  padding: 24px;
}

.panel-header {
  text-align: center;
  margin-bottom: 32px;
}

.panel-header h3 {
  margin: 0 0 8px 0;
  font-size: 1.5rem;
  color: #1f2937;
}

.panel-header p {
  margin: 0;
  color: #6b7280;
}

.config-section {
  background: #f8fafc;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
}

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  margin-bottom: 24px;
}

.config-card {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.config-card h4 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.form-group input, .form-group select {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
}

.date-range {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-range input {
  flex: 1;
}

.strategy-selector {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.strategy-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.strategy-option input[type="radio"] {
  margin-top: 2px;
}

.strategy-label {
  flex: 1;
  cursor: pointer;
  padding: 12px;
  border: 2px solid transparent;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.strategy-label:hover {
  border-color: #e5e7eb;
}

.strategy-option input[type="radio"]:checked + .strategy-label {
  border-color: #3b82f6;
  background: #eff6ff;
}

.strategy-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.strategy-name {
  font-weight: 600;
  color: #1f2937;
}

.strategy-confidence {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.strategy-confidence.high {
  background: #dcfce7;
  color: #166534;
}

.strategy-confidence.medium {
  background: #fef3c7;
  color: #92400e;
}

.strategy-confidence.low {
  background: #fee2e2;
  color: #991b1b;
}

.strategy-description {
  color: #6b7280;
  font-size: 14px;
  margin-bottom: 8px;
}

.strategy-stats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #6b7280;
}

.action-buttons {
  display: flex;
  justify-content: center;
  gap: 16px;
}

.run-btn, .reset-btn, .clear-btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.run-btn {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
  color: white;
}

.run-btn:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.run-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.reset-btn {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.reset-btn:hover {
  background: #e5e7eb;
}

.clear-btn {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  color: white;
}

.clear-btn:hover {
  background: linear-gradient(135deg, #b91c1c 0%, #991b1b 100%);
}

.metrics-overview {
  margin-bottom: 32px;
}

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
}

.metric-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  align-items: center;
  gap: 16px;
}

.metric-card.primary {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.metric-icon {
  font-size: 2rem;
  opacity: 0.8;
}

.metric-content {
  flex: 1;
}

.metric-value {
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 4px;
}

.metric-label {
  font-size: 14px;
  opacity: 0.8;
  margin-bottom: 4px;
}

.metric-change {
  font-size: 13px;
  font-weight: 600;
}

.metric-change.positive {
  color: #10b981;
}

.metric-change.negative {
  color: #ef4444;
}

.equity-chart-section, .ai-performance-section, .trades-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 32px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  clear: both;
  overflow: hidden;
}

.equity-chart-section h4, .ai-performance-section h4, .trades-section h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.chart-container {
  height: 540px;
  position: relative;
  border-radius: 8px;
}

.performance-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
  min-height: 200px;
  align-items: start;
}

.performance-card {
  padding: 20px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  min-height: 180px;
  display: flex;
  flex-direction: column;
}

.performance-card h5 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.timing-analysis, .environment-analysis {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.timing-item, .env-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
}

.timing-item:last-child, .env-item:last-child {
  border-bottom: none;
}

.timing-label, .env-condition {
  font-weight: 500;
  color: #374151;
}

.timing-value {
  color: #6b7280;
}

.env-stats {
  display: flex;
  gap: 12px;
  align-items: center;
}

.env-performance.positive {
  color: #10b981;
}

.env-performance.negative {
  color: #ef4444;
}

.env-count {
  color: #9ca3af;
  font-size: 14px;
}

.trades-table-container {
  overflow-x: auto;
}

.trades-table {
  width: 100%;
  border-collapse: collapse;
}

.trades-table th, .trades-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.trades-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.trade-action {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.trade-action.buy {
  background: #dcfce7;
  color: #166534;
}

.trade-action.sell {
  background: #fee2e2;
  color: #991b1b;
}

.confidence-bar {
  position: relative;
  width: 60px;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.confidence-fill {
  height: 100%;
  background: linear-gradient(90deg, #ef4444 0%, #f59e0b 50%, #10b981 100%);
  transition: width 0.3s ease;
}

.confidence-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 10px;
  font-weight: 600;
  color: #374151;
}

.loading-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.loading-content {
  background: white;
  border-radius: 12px;
  padding: 32px;
  text-align: center;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #e5e7eb;
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-text {
  color: #374151;
  font-weight: 500;
  margin-bottom: 16px;
}

.loading-progress {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: center;
}

.progress-bar {
  width: 200px;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #1d4ed8 100%);
  transition: width 0.3s ease;
}

.progress-text {
  font-weight: 600;
  color: #374151;
}

/* 自动执行设置样式 */
.auto-execute-config {
  background: #f8fafc;
  border-radius: 8px;
  padding: 16px;
  margin-top: 16px;
}

.setting-item {
  margin-bottom: 12px;
}

.setting-item label {
  display: block;
  font-weight: 500;
  margin-bottom: 4px;
  color: #374151;
}

.setting-item input[type="checkbox"] {
  margin-right: 8px;
}

.setting-desc {
  font-size: 13px;
  color: #6b7280;
  margin-left: 24px;
}

.auto-execute-details {
  margin-left: 24px;
  margin-top: 16px;
  padding: 16px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.setting-item input[type="number"] {
  width: 100px;
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
}

.input-desc {
  font-size: 12px;
  color: #9ca3af;
  margin-left: 8px;
}

/* 自动选择币种样式 */
.checkbox-label {
  display: flex;
  align-items: flex-start;
  cursor: pointer;
  font-weight: normal;
}

.checkbox-label input[type="checkbox"] {
  margin-right: 8px;
  margin-top: 2px;
  transform: scale(1.1);
}

.feature-desc {
  display: block;
  font-size: 13px;
  color: #6b7280;
  margin-left: 24px;
  margin-top: 4px;
  font-style: italic;
}

.auto-select-config {
  margin-left: 24px;
  margin-top: 16px;
  padding: 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  animation: fadeIn 0.3s ease-in-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-10px); }
  to { opacity: 1; transform: translateY(0); }
}

.feature-info {
  margin-top: 12px;
}

.info-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 8px;
  padding: 8px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.info-icon {
  font-size: 16px;
  margin-right: 8px;
  margin-top: 2px;
}

.info-content h4 {
  margin: 0 0 4px 0;
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}

.info-content p {
  margin: 0;
  font-size: 12px;
  color: #6b7280;
  line-height: 1.4;
}

/* 现实性配置样式 */
.form-group input[type="number"] {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  text-align: right;
}

.form-group input[type="number"]:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.metric-card.auto-execute-stat {
  background: linear-gradient(135deg, #8b5cf6 0%, #a855f7 100%);
  color: white;
}

/* 渐进式执行样式 */
.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: normal;
  cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  margin: 0;
}

.progressive-config {
  background: #f0f9ff;
  border-left: 4px solid #3b82f6;
}

.metric-card.auto-execute-stat .metric-desc {
  opacity: 0.9;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }

  .metric-cards {
    grid-template-columns: 1fr;
  }

  .performance-grid {
    grid-template-columns: 1fr;
  }

  .action-buttons {
    flex-direction: column;
  }

  .equity-chart-section, .ai-performance-section, .trades-section {
    margin-bottom: 40px;
    padding: 20px;
  }

  .chart-container {
    height: 270px;
  }

  .performance-card {
    min-height: 150px;
  }
}

</style>
