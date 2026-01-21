<template>
  <div class="backtest-advanced">
    <!-- 页面头部 -->
    <section class="panel">
      <div class="row">
        <h2>高级回测分析</h2>
        <div class="spacer"></div>
        <RouterLink to="/backtest-history" class="secondary history-link-top" title="查看所有回测历史记录">
          📊 历史记录
        </RouterLink>
        <button @click="loadTemplates" class="secondary">
          加载模板
        </button>
        <button @click="resetForm" class="secondary">
          重置
        </button>
      </div>

      <!-- 策略信息显示 -->
      <div v-if="strategyInfo" class="strategy-info-banner">
        <div class="strategy-info-header">
          <h3>📊 策略回测: {{ strategyInfo.name }}</h3>
          <span class="strategy-id">ID: {{ strategyInfo.id }}</span>
        </div>
        <div class="strategy-conditions">
          <div v-if="strategyInfo.conditions.spot_contract" class="condition-tag">
            需要现货+合约
          </div>
          <div v-if="strategyInfo.conditions.short_on_gainers" class="condition-tag">
            涨幅前{{ strategyInfo.conditions.gainers_rank_limit }}做空
          </div>
          <div v-if="strategyInfo.conditions.long_on_small_gainers" class="condition-tag">
            市值<{{ strategyInfo.conditions.market_cap_limit_long }}万做多
          </div>
          <div v-if="strategyInfo.conditions.enable_stop_loss" class="condition-tag">
            止损: {{ strategyInfo.conditions.stop_loss_percent }}%
          </div>
          <div v-if="strategyInfo.conditions.enable_take_profit" class="condition-tag">
            止盈: {{ strategyInfo.conditions.take_profit_percent }}%
          </div>
          <div v-if="strategyInfo.conditions.enable_margin_loss_stop_loss" class="condition-tag">
            保证金止损: {{ strategyInfo.conditions.margin_loss_stop_loss_percent }}%
          </div>
          <div v-if="strategyInfo.conditions.enable_margin_profit_take_profit" class="condition-tag">
            保证金止盈: {{ strategyInfo.conditions.margin_profit_take_profit_percent }}%
          </div>
        </div>
      </div>
    </section>

    <!-- 配置面板 -->
    <section class="panel config-panel">
      <h3>回测配置</h3>
      <div class="config-grid">
        <!-- 基本配置 -->
        <div class="config-section">
          <h4>基本设置</h4>
          <div class="form-row">
            <label>交易对</label>
            <select v-model="config.symbol">
              <option value="BTC">BTC/USDT</option>
              <option value="ETH">ETH/USDT</option>
              <option value="BNB">BNB/USDT</option>
              <option value="ADA">ADA/USDT</option>
            </select>
          </div>
          <div class="form-row">
            <label>策略</label>
            <select v-model="config.strategy">
              <option v-for="strategy in availableStrategies" :key="strategy.name" :value="strategy.name">
                {{ strategy.display_name }}
              </option>
            </select>
          </div>
          <div class="form-row">
            <label>时间范围</label>
            <div class="date-range">
              <input type="date" v-model="config.startDate" />
              <span>至</span>
              <input type="date" v-model="config.endDate" />
            </div>
          </div>
        </div>

        <!-- 交易参数 -->
        <div class="config-section" v-if="!isTradingStrategySelected">
          <h4>交易参数</h4>
          <div class="form-row">
            <label>初始资金</label>
            <input type="number" v-model.number="config.initialCash" step="1000" min="1000" />
          </div>
          <div class="form-row">
            <label>最大仓位比例</label>
            <input type="number" v-model.number="config.maxPosition" step="0.1" min="0.1" max="1" />
          </div>
          <div class="form-row">
            <label>手续费率</label>
            <input type="number" v-model.number="config.commission" step="0.001" min="0" max="0.01" />
          </div>
        </div>

        <!-- 风险控制 -->
        <div class="config-section" v-if="!isTradingStrategySelected">
          <h4>风险控制</h4>
          <div class="form-row">
            <label>止损比例</label>
            <input type="number" v-model.number="config.stopLoss" step="0.01" min="0" max="0.5" />
          </div>
          <div class="form-row">
            <label>止盈比例</label>
            <input type="number" v-model.number="config.takeProfit" step="0.01" min="0" max="1" />
          </div>
        </div>

        <!-- 交易策略信息 -->
        <div class="config-section" v-if="isTradingStrategySelected">
          <h4>策略配置</h4>
          <div class="strategy-info-box">
            <div class="strategy-desc">{{ selectedStrategyInfo.description }}</div>
            <div class="strategy-note">此策略将根据其配置条件自动选择交易时机，无需手动设置风险参数。</div>
          </div>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="action-buttons">
        <button @click="runBacktest" :disabled="running" class="primary">
          {{ running ? '运行中...' : '开始回测' }}
        </button>
        <button @click="saveConfig" class="secondary">
          保存配置
        </button>
        <RouterLink to="/backtest-history" class="secondary">
          📊 查看历史记录
        </RouterLink>
      </div>
    </section>

    <!-- 结果展示 -->
    <div v-if="result" class="results-section">
      <!-- 汇总统计 -->
      <section class="panel">
        <h3>回测结果汇总</h3>
        <div class="summary-grid">
          <div class="summary-card">
            <div class="summary-label">总收益率</div>
            <div class="summary-value" :class="getReturnClass(result.summary.total_return)">
              {{ formatPercent(result.summary.total_return) }}
            </div>
          </div>
          <div class="summary-card">
            <div class="summary-label">年化收益率</div>
            <div class="summary-value" :class="getReturnClass(result.summary.annual_return)">
              {{ formatPercent(result.summary.annual_return) }}
            </div>
          </div>
          <div class="summary-card">
            <div class="summary-label">胜率</div>
            <div class="summary-value positive">
              {{ formatPercent(result.summary.win_rate) }}
            </div>
          </div>
          <div class="summary-card">
            <div class="summary-label">最大回撤</div>
            <div class="summary-value negative">
              {{ formatPercent(result.summary.max_drawdown) }}
            </div>
          </div>
          <div class="summary-card">
            <div class="summary-label">夏普比率</div>
            <div class="summary-value" :class="getSharpeClass(result.summary.sharpe_ratio)">
              {{ result.summary.sharpe_ratio.toFixed(2) }}
            </div>
          </div>
          <div class="summary-card">
            <div class="summary-label">总交易次数</div>
            <div class="summary-value">
              {{ result.summary.total_trades }}
            </div>
          </div>
        </div>
      </section>

      <!-- 图表展示 -->
      <section class="panel">
        <h3>收益曲线</h3>
        <div class="chart-container">
          <canvas ref="returnsChart"></canvas>
        </div>
      </section>

      <!-- 交易记录 -->
      <section class="panel">
        <div class="section-header">
        <h3>交易记录</h3>
          <div class="section-actions">
            <RouterLink to="/backtest-history" class="quick-history-link">
              📊 查看完整历史记录
            </RouterLink>
          </div>
        </div>
        <div class="table-container">
          <table class="trades-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>方向</th>
                <th>数量</th>
                <th>价格</th>
                <th>手续费</th>
                <th>盈亏</th>
                <th>原因</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="trade in result.trades.slice(-20)" :key="trade.timestamp">
                <td>{{ formatDate(trade.timestamp) }}</td>
                <td :class="trade.side === 'buy' ? 'positive' : 'negative'">
                  {{ trade.side === 'buy' ? '买入' : '卖出' }}
                </td>
                <td>{{ trade.quantity.toFixed(6) }}</td>
                <td>${{ trade.price.toFixed(4) }}</td>
                <td>${{ trade.commission.toFixed(2) }}</td>
                <td :class="getPnLClass(trade.pnl)">
                  {{ trade.pnl ? formatCurrency(trade.pnl) : '-' }}
                </td>
                <td>{{ trade.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- 风险指标 -->
      <section class="panel">
        <h3>风险分析</h3>
        <div class="risk-grid">
          <div class="risk-item">
            <div class="risk-label">VaR(95%)</div>
            <div class="risk-value">
              {{ formatPercent(result.risk_metrics.value_at_risk_95) }}
            </div>
          </div>
          <div class="risk-item">
            <div class="risk-label">VaR(99%)</div>
            <div class="risk-value">
              {{ formatPercent(result.risk_metrics.value_at_risk_99) }}
            </div>
          </div>
          <div class="risk-item">
            <div class="risk-label">期望亏空</div>
            <div class="risk-value">
              {{ formatPercent(result.risk_metrics.expected_shortfall) }}
            </div>
          </div>
          <div class="risk-item">
            <div class="risk-label">Calmar比率</div>
            <div class="risk-value" :class="getCalmarClass(result.performance.calmar_ratio)">
              {{ result.performance.calmar_ratio.toFixed(2) }}
            </div>
          </div>
          <div class="risk-item">
            <div class="risk-label">Sortino比率</div>
            <div class="risk-value" :class="getSortinoClass(result.performance.sortino_ratio)">
              {{ result.performance.sortino_ratio.toFixed(2) }}
            </div>
          </div>
          <div class="risk-item">
            <div class="risk-label">Omega比率</div>
            <div class="risk-value" :class="getOmegaClass(result.performance.omega_ratio)">
              {{ result.performance.omega_ratio.toFixed(2) }}
            </div>
          </div>
        </div>
      </section>

      <!-- 保存结果和历史记录 -->
      <section class="panel">
        <div class="result-actions">
        <div class="save-result">
          <input v-model="resultName" placeholder="输入结果名称" class="result-name-input" />
          <button @click="saveResult" class="primary">保存结果</button>
          </div>
          <div class="history-link">
            <RouterLink to="/backtest-history" class="history-btn">
              📊 查看所有历史记录
            </RouterLink>
            <span class="history-tip">查看和管理所有回测历史记录</span>
          </div>
        </div>
      </section>
    </div>

    <!-- 策略对比 -->
    <section class="panel" v-if="comparisonResult">
      <h3>策略对比结果</h3>
      <div class="comparison-table">
        <table>
          <thead>
            <tr>
              <th>策略</th>
              <th>总收益率</th>
              <th>年化收益率</th>
              <th>胜率</th>
              <th>最大回撤</th>
              <th>夏普比率</th>
              <th>排名</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="strategy in comparisonResult.strategies" :key="strategy.name"
                :class="{ 'best-strategy': strategy.name === comparisonResult.bestStrategy }">
              <td>{{ strategy.name }}</td>
              <td :class="getReturnClass(strategy.result.summary.total_return)">
                {{ formatPercent(strategy.result.summary.total_return) }}
              </td>
              <td :class="getReturnClass(strategy.result.summary.annual_return)">
                {{ formatPercent(strategy.result.summary.annual_return) }}
              </td>
              <td class="positive">
                {{ formatPercent(strategy.result.summary.win_rate) }}
              </td>
              <td class="negative">
                {{ formatPercent(strategy.result.summary.max_drawdown) }}
              </td>
              <td :class="getSharpeClass(strategy.result.summary.sharpe_ratio)">
                {{ strategy.result.summary.sharpe_ratio.toFixed(2) }}
              </td>
              <td class="ranking">{{ strategy.ranking }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script>
import Chart from 'chart.js/auto'
import { api } from '@/api/api.js'

export default {
  name: 'BacktestAdvanced',
  data() {
    return {
      config: {
        symbol: 'BTC',
        strategy: 'buy_and_hold',
        startDate: this.getDefaultStartDate(),
        endDate: this.getDefaultEndDate(),
        initialCash: 10000,
        maxPosition: 1.0,
        stopLoss: 0.1,
        takeProfit: 0.2,
        commission: 0.001,
        timeframe: '1d'
      },
      availableStrategies: [],
      templates: [],
      strategyInfo: null, // 当前策略信息
      result: null,
      comparisonResult: null,
      running: false,
      resultName: '',
      returnsChart: null
    }
  },
  mounted() {
    this.loadAvailableStrategies()
    this.loadTemplates()

    // 检查URL参数，如果有策略ID，则自动加载策略配置
    const urlParams = new URLSearchParams(window.location.search)
    const strategyId = urlParams.get('strategy_id')
    const strategyName = urlParams.get('strategy_name')

    if (strategyId && strategyName) {
      this.loadStrategyForBacktest(strategyId, strategyName)
    }
  },
  beforeUnmount() {
    if (this.returnsChart) {
      this.returnsChart.destroy()
    }
  },
  computed: {
    isTradingStrategySelected() {
      const selectedStrategy = this.availableStrategies.find(s => s.name === this.config.strategy)
      return selectedStrategy && selectedStrategy.type === 'trading_strategy'
    },
    selectedStrategyInfo() {
      return this.availableStrategies.find(s => s.name === this.config.strategy) || {}
    }
  },
  methods: {
    async loadStrategyForBacktest(strategyId, strategyName) {
      try {
        // 显示页面标题
        this.$nextTick(() => {
          document.title = `策略回测 - ${strategyName} - 专业量化分析`
        })

        // 从后端获取策略详情
        const response = await api.getTradingStrategy(strategyId)
        const strategy = response.data

        // 保存策略信息
        this.strategyInfo = {
          id: strategy.id,
          name: strategy.name,
          conditions: strategy.conditions
        }

        // 映射策略条件到回测配置
        this.mapStrategyToBacktestConfig(strategy)

      } catch (error) {
        console.error('加载策略回测配置失败:', error)
        alert('加载策略配置失败: ' + error.message)
      }
    },

    mapStrategyToBacktestConfig(strategy) {
      // 保留用户设置的时间范围，只更新策略相关的参数
      const currentStartDate = this.config.startDate;
      const currentEndDate = this.config.endDate;

      this.config = {
        ...this.config, // 保留所有现有配置
        symbol: 'BTC', // 默认使用BTC
        strategy: 'momentum', // 默认策略
        // 保留用户设置的时间范围，不覆盖
        startDate: currentStartDate,
        endDate: currentEndDate,
        stopLoss: strategy.conditions.stop_loss_percent || this.config.stopLoss || 0.05,
        takeProfit: strategy.conditions.take_profit_percent || this.config.takeProfit || 0.1,
        timeframe: '1d'
      }

      // 根据策略条件智能选择币种
      if (strategy.conditions.market_cap_limit_short) {
        // 如果有限制市值，则选择市值合适的币种
        this.config.symbol = 'ETH' // 示例：选择ETH
      } else if (strategy.conditions.gainers_rank_limit) {
        // 如果关注涨幅排名，选择热门币种
        this.config.symbol = 'BTC'
      }

      // 根据策略类型选择合适的回测策略
      if (strategy.conditions.futures_spot_arb_enabled) {
        this.config.strategy = 'arbitrage'
      } else if (strategy.conditions.short_on_gainers) {
        this.config.strategy = 'mean_reversion'
      } else if (strategy.conditions.long_on_small_gainers) {
        this.config.strategy = 'momentum'
      }
    },

    async loadAvailableStrategies() {
      try {
        const response = await api.getAvailableStrategies()
        this.availableStrategies = response.data
      } catch (error) {
        console.error('加载可用策略失败:', error)
      }
    },

    async loadTemplates() {
      try {
        const response = await api.getBacktestTemplates()
        this.templates = response.data
      } catch (error) {
        console.error('加载回测模板失败:', error)
      }
    },

    async runBacktest() {
      this.running = true
      try {
        // 检查是否是从策略跳转过来的
        const urlParams = new URLSearchParams(window.location.search)
        const strategyId = urlParams.get('strategy_id')

        let response

        // 检查是否选择了交易策略
        const selectedStrategy = this.availableStrategies.find(s => s.name === this.config.strategy)
        const isTradingStrategy = selectedStrategy && selectedStrategy.type === 'trading_strategy'

        if (strategyId) {
          // 使用策略回测API（从策略列表跳转过来）
          response = await api.runStrategyBacktest(
            parseInt(strategyId),
            this.config.symbol,
            new Date(this.config.startDate).toISOString(),
            new Date(this.config.endDate).toISOString()
          )
        } else if (isTradingStrategy) {
          // 使用策略回测API（在回测页面选择了交易策略）
          response = await api.runStrategyBacktest(
            selectedStrategy.strategy_id,
            this.config.symbol,
            new Date(this.config.startDate).toISOString(),
            new Date(this.config.endDate).toISOString()
          )
        } else {
          // 使用普通回测API
        const backtestConfig = {
          symbol: this.config.symbol,
          strategy: this.config.strategy,
          start_date: new Date(this.config.startDate).toISOString(),
          end_date: new Date(this.config.endDate).toISOString(),
          initial_cash: this.config.initialCash,
          max_position: this.config.maxPosition,
          stop_loss: this.config.stopLoss,
          take_profit: this.config.takeProfit,
          commission: this.config.commission,
          timeframe: this.config.timeframe
        }

          response = await api.runBacktest(backtestConfig)
        }

        // 根据实际API响应结构，直接使用response.data
        this.result = response.data

        this.$nextTick(() => {
          this.renderReturnsChart()
        })
      } catch (error) {
        console.error('运行回测失败:', error)
        console.error('Error details:', error.response?.data || error.message)
        alert('回测运行失败: ' + (error.response?.data?.error || error.message))
      } finally {
        console.log('Finally block: setting running to false')
        this.running = false
      }
    },

    async saveResult() {
      if (!this.resultName.trim()) {
        alert('请输入结果名称')
        return
      }

      try {
        await api.saveBacktestResult({
          name: this.resultName,
          description: `回测结果 - ${this.config.symbol} - ${this.config.strategy}`,
          config: this.config,
          result: this.result
        })

        alert('结果保存成功')
        this.resultName = ''
      } catch (error) {
        console.error('保存结果失败:', error)
        alert('保存失败: ' + error.message)
      }
    },

    renderReturnsChart() {
      if (!this.result || !this.result.daily_returns) return

      const ctx = this.$refs.returnsChart
      if (!ctx) return

      const data = this.result.daily_returns
      const labels = data.map(d => this.formatDate(d.date))
      const values = data.map(d => d.value)

      if (this.returnsChart) {
        this.returnsChart.destroy()
      }

      this.returnsChart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: labels,
          datasets: [{
            label: '资产净值',
            data: values,
            borderColor: 'rgb(75, 192, 192)',
            backgroundColor: 'rgba(75, 192, 192, 0.1)',
            tension: 0.1
          }]
        },
        options: {
          responsive: true,
          plugins: {
            title: {
              display: true,
              text: '回测收益曲线'
            }
          },
          scales: {
            y: {
              beginAtZero: false,
              ticks: {
                callback: (value) => '$' + value.toLocaleString()
              }
            }
          }
        }
      })
    },

    resetForm() {
      this.config = {
        symbol: 'BTC',
        strategy: 'buy_and_hold',
        startDate: this.getDefaultStartDate(),
        endDate: this.getDefaultEndDate(),
        initialCash: 10000,
        maxPosition: 1.0,
        stopLoss: 0.1,
        takeProfit: 0.2,
        commission: 0.001,
        timeframe: '1d'
      }
      this.result = null
      this.comparisonResult = null
      if (this.returnsChart) {
        this.returnsChart.destroy()
        this.returnsChart = null
      }
    },

    saveConfig() {
      // 保存当前配置到本地存储
      const configKey = 'backtest_config_' + Date.now()
      localStorage.setItem(configKey, JSON.stringify(this.config))
      alert('配置已保存')
    },

    // 工具函数
    getDefaultStartDate() {
      const date = new Date()
      date.setMonth(date.getMonth() - 6)
      return date.toISOString().split('T')[0]
    },

    getDefaultEndDate() {
      return new Date().toISOString().split('T')[0]
    },

    formatPercent(value) {
      return (value * 100).toFixed(2) + '%'
    },

    formatCurrency(value) {
      return '$' + Math.abs(value).toFixed(2)
    },

    formatDate(dateString) {
      return new Date(dateString).toLocaleDateString()
    },

    getReturnClass(value) {
      return value >= 0 ? 'positive' : 'negative'
    },

    getPnLClass(value) {
      if (!value) return ''
      return value >= 0 ? 'positive' : 'negative'
    },

    getSharpeClass(value) {
      if (value >= 2) return 'positive'
      if (value >= 1) return 'neutral'
      return 'negative'
    },

    getCalmarClass(value) {
      if (value >= 1) return 'positive'
      if (value >= 0.5) return 'neutral'
      return 'negative'
    },

    getSortinoClass(value) {
      if (value >= 2) return 'positive'
      if (value >= 1) return 'neutral'
      return 'negative'
    },

    getOmegaClass(value) {
      if (value >= 1.5) return 'positive'
      if (value >= 1.2) return 'neutral'
      return 'negative'
    }
  }
}
</script>

<style scoped>
.backtest-advanced {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.strategy-info-banner {
  margin-top: 16px;
  padding: 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
  border: 1px solid #cbd5e1;
  border-radius: 8px;
}

.strategy-info-box {
  padding: 16px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
}

.strategy-desc {
  font-size: 14px;
  color: #374151;
  margin-bottom: 8px;
  line-height: 1.5;
}

.strategy-note {
  font-size: 12px;
  color: #6b7280;
  font-style: italic;
}

.strategy-info-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.strategy-info-header h3 {
  margin: 0;
  color: #334155;
  font-size: 18px;
}

.strategy-id {
  color: #64748b;
  font-size: 14px;
  font-weight: 500;
}

.strategy-conditions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.condition-tag {
  padding: 4px 8px;
  background: #e0f2fe;
  color: #0369a1;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.config-panel {
  margin-bottom: 24px;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.config-section {
  background: var(--bg-secondary);
  padding: 20px;
  border-radius: 8px;
}

.config-section h4 {
  margin: 0 0 16px 0;
  color: var(--text-primary);
}

.form-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.form-row label {
  width: 120px;
  margin-right: 12px;
  font-weight: 500;
}

.form-row input, .form-row select {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg-primary);
  color: var(--text-primary);
}

.date-range {
  display: flex;
  align-items: center;
  gap: 8px;
}

.date-range input {
  flex: 1;
}

.action-buttons {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.summary-card {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: 8px;
  text-align: center;
}

.summary-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.summary-value {
  font-size: 24px;
  font-weight: bold;
}

.positive {
  color: #10b981;
}

.negative {
  color: #ef4444;
}

.neutral {
  color: #f59e0b;
}

.chart-container {
  height: 400px;
  position: relative;
}

.table-container {
  max-height: 400px;
  overflow-y: auto;
}

.trades-table {
  width: 100%;
  border-collapse: collapse;
}

.trades-table th, .trades-table td {
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.trades-table th {
  background: var(--bg-secondary);
  font-weight: 600;
  position: sticky;
  top: 0;
}

.risk-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.risk-item {
  background: var(--bg-secondary);
  padding: 16px;
  border-radius: 8px;
  text-align: center;
}

.risk-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.risk-value {
  font-size: 18px;
  font-weight: bold;
}

.result-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 20px;
}

.save-result {
  display: flex;
  gap: 12px;
  align-items: center;
  flex: 1;
}

.result-name-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg-primary);
  color: var(--text-primary);
}

.history-link {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.history-btn {
  padding: 8px 16px;
  background: var(--primary);
  color: white;
  text-decoration: none;
  border-radius: 6px;
  font-weight: 500;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.history-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0,0,0,0.2);
}

.history-tip {
  font-size: 12px;
  color: var(--text-muted);
  text-align: right;
}

.history-link-top {
  margin-right: 8px;
  font-size: 14px;
  padding: 6px 12px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  margin: 0;
}

.section-actions {
  display: flex;
  gap: 12px;
}

.quick-history-link {
  padding: 6px 12px;
  background: var(--bg-secondary);
  color: var(--text-secondary);
  text-decoration: none;
  border-radius: 4px;
  font-size: 14px;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.quick-history-link:hover {
  background: var(--primary);
  color: white;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .result-actions {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }

  .history-link {
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
  }

  .history-tip {
    text-align: left;
    font-size: 11px;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .section-actions {
    width: 100%;
    justify-content: flex-end;
  }
}

.comparison-table {
  overflow-x: auto;
}

.comparison-table table {
  width: 100%;
  border-collapse: collapse;
}

.comparison-table th, .comparison-table td {
  padding: 12px;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.comparison-table th {
  background: var(--bg-secondary);
  font-weight: 600;
}

.best-strategy {
  background: rgba(16, 185, 129, 0.1);
  border-left: 4px solid #10b981;
}

.ranking {
  font-weight: bold;
  text-align: center;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .backtest-advanced {
    padding: 12px;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }

  .summary-grid, .risk-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .form-row {
    flex-direction: column;
    align-items: stretch;
  }

  .form-row label {
    width: auto;
    margin-bottom: 4px;
  }
}
</style>
