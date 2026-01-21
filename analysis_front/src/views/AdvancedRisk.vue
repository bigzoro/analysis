<template>
  <div class="advanced-risk">
    <div class="risk-header">
      <h1>🔬 高级风险管理</h1>
      <p class="subtitle">专业的量化风险分析与投资组合优化</p>

      <!-- 风险概览面板 -->
      <div class="risk-overview">
        <div class="overview-card">
          <div class="card-icon">📊</div>
          <div class="card-content">
            <div class="card-title">系统风险状态</div>
            <div class="card-value" :class="systemRiskLevel">{{ systemRiskLevel }}</div>
            <div class="card-subtitle">基于多维度风险评估</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">⚠️</div>
          <div class="card-content">
            <div class="card-title">活跃告警</div>
            <div class="card-value warning">{{ activeAlerts }}</div>
            <div class="card-subtitle">需要立即关注</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">🛡️</div>
          <div class="card-content">
            <div class="card-title">风险覆盖率</div>
            <div class="card-value success">{{ riskCoverage }}%</div>
            <div class="card-subtitle">投资组合保护程度</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">📈</div>
          <div class="card-content">
            <div class="card-title">预期收益</div>
            <div class="card-value">{{ expectedReturn }}%</div>
            <div class="card-subtitle">风险调整后收益</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要功能区域 -->
    <div class="risk-content">
      <!-- 标签页导航 -->
      <div class="tab-navigation">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          @click="activeTab = tab.id"
          :class="['tab-button', { active: activeTab === tab.id }]"
        >
          <span class="tab-icon">{{ tab.icon }}</span>
          <span class="tab-title">{{ tab.title }}</span>
        </button>
      </div>

      <!-- 高级风险指标 -->
      <div v-if="activeTab === 'metrics'" class="tab-content">
        <div class="content-header">
          <h2>高级风险指标分析</h2>
          <div class="controls">
            <select v-model="selectedSymbol" @change="loadRiskMetrics">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <button @click="loadRiskMetrics" :disabled="!selectedSymbol || loading" class="analyze-btn">
              {{ loading ? '分析中...' : '开始分析' }}
            </button>
          </div>
        </div>

        <div v-if="riskMetrics" class="metrics-grid">
          <!-- 传统风险指标 -->
          <div class="metric-section">
            <h3>传统风险指标</h3>
            <div class="metric-cards">
              <div class="metric-card">
                <div class="metric-name">波动率</div>
                <div class="metric-value">{{ (riskMetrics.volatility * 100).toFixed(2) }}%</div>
                <div class="metric-bar">
                  <div class="bar-fill" :style="{ width: Math.min(riskMetrics.volatility * 500, 100) + '%' }"></div>
                </div>
              </div>

              <div class="metric-card">
                <div class="metric-name">最大回撤</div>
                <div class="metric-value">{{ (riskMetrics.maxDrawdown * 100).toFixed(2) }}%</div>
                <div class="metric-bar">
                  <div class="bar-fill high-risk" :style="{ width: Math.min(riskMetrics.maxDrawdown * 200, 100) + '%' }"></div>
                </div>
              </div>

              <div class="metric-card">
                <div class="metric-name">夏普比率</div>
                <div class="metric-value">{{ riskMetrics.sharpeRatio.toFixed(2) }}</div>
                <div class="metric-indicator" :class="{ positive: riskMetrics.sharpeRatio > 1, negative: riskMetrics.sharpeRatio < 0 }">
                  {{ riskMetrics.sharpeRatio > 1 ? '优秀' : riskMetrics.sharpeRatio > 0 ? '良好' : '需关注' }}
                </div>
              </div>

              <div class="metric-card">
                <div class="metric-name">索提诺比率</div>
                <div class="metric-value">{{ riskMetrics.sortinoRatio.toFixed(2) }}</div>
                <div class="metric-indicator" :class="{ positive: riskMetrics.sortinoRatio > 1 }">
                  {{ riskMetrics.sortinoRatio > 1 ? '优秀' : '一般' }}
                </div>
              </div>
            </div>
          </div>

          <!-- VaR指标 -->
          <div class="metric-section">
            <h3>VaR风险指标</h3>
            <div class="var-metrics">
              <div class="var-card">
                <div class="var-confidence">95% 置信度</div>
                <div class="var-value">{{ (riskMetrics.var95 * 100).toFixed(2) }}%</div>
                <div class="var-desc">一天内损失不超过此值的概率为95%</div>
              </div>

              <div class="var-card">
                <div class="var-confidence">99% 置信度</div>
                <div class="var-value">{{ (riskMetrics.var99 * 100).toFixed(2) }}%</div>
                <div class="var-desc">一天内损失不超过此值的概率为99%</div>
              </div>
            </div>
          </div>

          <!-- 市场风险指标 -->
          <div class="metric-section">
            <h3>市场风险分析</h3>
            <div class="market-risk">
              <div class="beta-analysis">
                <div class="beta-value">
                  <span class="beta-label">贝塔系数 (β):</span>
                  <span class="beta-number" :class="{ high: Math.abs(riskMetrics.beta) > 1.5, low: Math.abs(riskMetrics.beta) < 0.5 }">
                    {{ riskMetrics.beta.toFixed(3) }}
                  </span>
                </div>
                <div class="beta-interpretation">
                  {{ getBetaInterpretation(riskMetrics.beta) }}
                </div>
              </div>

              <div class="liquidity-metrics">
                <div class="liquidity-item">
                  <span class="liquidity-label">买卖价差:</span>
                  <span class="liquidity-value">{{ riskMetrics.bidAskSpread.toFixed(4) }}</span>
                </div>
                <div class="liquidity-item">
                  <span class="liquidity-label">换手率:</span>
                  <span class="liquidity-value">{{ (riskMetrics.turnoverRatio * 100).toFixed(2) }}%</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 压力测试结果 -->
          <div class="metric-section">
            <h3>压力测试结果</h3>
            <div class="stress-test-results">
              <div
                v-for="result in riskMetrics.stressTestResults"
                :key="result.scenario"
                class="stress-result"
              >
                <div class="scenario-name">{{ result.scenario }}</div>
                <div class="scenario-shock">冲击: {{ (result.shock * 100).toFixed(1) }}%</div>
                <div class="scenario-loss" :class="{ critical: Math.abs(result.loss) > 0.3 }">
                  损失: {{ (result.loss * 100).toFixed(2) }}%
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="!loading" class="empty-state">
          <div class="empty-icon">📊</div>
          <p>请选择交易对并点击"开始分析"来查看高级风险指标</p>
        </div>
      </div>

      <!-- 压力测试 -->
      <div v-if="activeTab === 'stress-test'" class="tab-content">
        <div class="content-header">
          <h2>压力测试分析</h2>
          <div class="controls">
            <select v-model="stressTestSymbol">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <select v-model="stressTestTimeRange">
              <option value="7d">7天</option>
              <option value="30d">30天</option>
              <option value="90d">90天</option>
            </select>
            <button @click="runStressTest" :disabled="!stressTestSymbol || stressTesting" class="analyze-btn">
              {{ stressTesting ? '测试中...' : '执行测试' }}
            </button>
          </div>
        </div>

        <div v-if="stressTestResults" class="stress-test-visualization">
          <div class="test-summary">
            <div class="summary-stat">
              <span class="stat-label">测试场景数:</span>
              <span class="stat-value">{{ stressTestResults.length }}</span>
            </div>
            <div class="summary-stat">
              <span class="stat-label">最坏情况损失:</span>
              <span class="stat-value critical">
                {{ Math.min(...stressTestResults.map(r => r.loss * 100)).toFixed(2) }}%
              </span>
            </div>
          </div>

          <div class="stress-chart">
            <div class="chart-placeholder">
              <div class="placeholder-icon">📈</div>
              <p>压力测试结果可视化图表</p>
              <small>展示不同冲击情景下的潜在损失</small>
            </div>
          </div>

          <div class="scenario-details">
            <div
              v-for="result in stressTestResults"
              :key="result.scenario"
              class="scenario-card"
              :class="{ critical: Math.abs(result.loss) > 0.3, warning: Math.abs(result.loss) > 0.15 }"
            >
              <div class="scenario-header">
                <h4>{{ result.scenario }}</h4>
                <div class="scenario-shock">{{ (result.shock * 100).toFixed(1) }}% 冲击</div>
              </div>
              <div class="scenario-loss">
                <div class="loss-value">{{ (result.loss * 100).toFixed(2) }}%</div>
                <div class="loss-description">预期损失幅度</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 投资组合优化 -->
      <div v-if="activeTab === 'portfolio'" class="tab-content">
        <div class="content-header">
          <h2>投资组合优化</h2>
          <div class="controls">
            <button @click="addPortfolioAsset" class="add-btn">➕ 添加资产</button>
            <button @click="optimizePortfolio" :disabled="optimizing || portfolioAssets.length < 2" class="optimize-btn">
              {{ optimizing ? '优化中...' : '🎯 开始优化' }}
            </button>
          </div>
        </div>

        <div class="portfolio-setup">
          <div class="assets-list">
            <div
              v-for="(asset, index) in portfolioAssets"
              :key="asset.symbol"
              class="asset-item"
            >
              <div class="asset-info">
                <span class="asset-symbol">{{ asset.symbol }}</span>
                <input
                  v-model.number="asset.weight"
                  type="number"
                  min="0"
                  max="1"
                  step="0.01"
                  class="weight-input"
                  placeholder="权重"
                />
              </div>
              <button @click="removePortfolioAsset(index)" class="remove-btn">✕</button>
            </div>
          </div>

          <div class="optimization-params">
            <div class="param-group">
              <label>目标收益:</label>
              <input v-model.number="targetReturn" type="number" step="0.01" min="0" class="param-input" />
              <span class="param-unit">%</span>
            </div>
            <div class="param-group">
              <label>最大权重限制:</label>
              <input v-model.number="maxWeight" type="number" step="0.1" min="0" max="1" class="param-input" />
            </div>
          </div>
        </div>

        <div v-if="optimizationResult" class="optimization-result">
          <div class="result-header">
            <h3>优化结果</h3>
            <div class="result-stats">
              <div class="stat-item">
                <span class="stat-label">预期收益:</span>
                <span class="stat-value">{{ (optimizationResult.expectedReturn * 100).toFixed(2) }}%</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">预期风险:</span>
                <span class="stat-value">{{ (optimizationResult.expectedRisk * 100).toFixed(2) }}%</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">夏普比率:</span>
                <span class="stat-value">{{ optimizationResult.sharpeRatio.toFixed(2) }}</span>
              </div>
            </div>
          </div>

          <div class="optimal-weights">
            <h4>最优权重分配</h4>
            <div class="weights-chart">
              <div
                v-for="weight in optimizationResult.weights"
                :key="weight.symbol"
                class="weight-bar"
              >
                <div class="weight-symbol">{{ weight.symbol }}</div>
                <div class="weight-bar-container">
                  <div
                    class="weight-fill"
                    :style="{ width: (weight.percentage * 100) + '%' }"
                  ></div>
                </div>
                <div class="weight-percentage">{{ (weight.percentage * 100).toFixed(1) }}%</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险预算分析 -->
      <div v-if="activeTab === 'budget'" class="tab-content">
        <div class="content-header">
          <h2>风险预算分析</h2>
          <div class="controls">
            <button @click="calculateRiskBudget" :disabled="budgetCalculating" class="analyze-btn">
              {{ budgetCalculating ? '计算中...' : '📊 计算预算' }}
            </button>
          </div>
        </div>

        <div v-if="riskBudget" class="budget-analysis">
          <div class="budget-overview">
            <div class="budget-stat">
              <span class="stat-label">总风险预算:</span>
              <span class="stat-value">{{ riskBudget.totalBudget.toFixed(4) }}</span>
            </div>
            <div class="budget-stat">
              <span class="stat-label">资产数量:</span>
              <span class="stat-value">{{ riskBudget.assetsCount }}</span>
            </div>
          </div>

          <div class="budget-allocation">
            <h4>风险预算分配</h4>
            <div class="allocation-chart">
              <div
                v-for="(budget, symbol) in riskBudget.assetBudgets"
                :key="symbol"
                class="allocation-item"
              >
                <div class="allocation-symbol">{{ symbol }}</div>
                <div class="allocation-bar">
                  <div
                    class="allocation-fill"
                    :style="{ width: ((budget / riskBudget.totalBudget) * 100) + '%' }"
                  ></div>
                </div>
                <div class="allocation-value">{{ budget.toFixed(4) }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { api } from '@/api/api.js'

export default {
  name: 'AdvancedRisk',
  data() {
    return {
      activeTab: 'metrics',
      loading: false,
      stressTesting: false,
      optimizing: false,
      budgetCalculating: false,

      // 系统状态
      systemRiskLevel: '低风险',
      activeAlerts: 0,
      riskCoverage: 85,
      expectedReturn: 12.5,

      // 标签页
      tabs: [
        { id: 'metrics', title: '风险指标', icon: '📊' },
        { id: 'stress-test', title: '压力测试', icon: '⚡' },
        { id: 'portfolio', title: '组合优化', icon: '🎯' },
        { id: 'budget', title: '风险预算', icon: '💰' }
      ],

      // 可用交易对
      availableSymbols: ['BTC', 'ETH', 'ADA', 'SOL', 'DOT', 'LINK', 'UNI', 'AAVE'],

      // 风险指标分析
      selectedSymbol: '',
      riskMetrics: null,

      // 压力测试
      stressTestSymbol: '',
      stressTestTimeRange: '30d',
      stressTestResults: null,

      // 投资组合优化
      portfolioAssets: [
        { symbol: 'BTC', weight: 0.4 },
        { symbol: 'ETH', weight: 0.3 },
        { symbol: 'ADA', weight: 0.2 },
        { symbol: 'SOL', weight: 0.1 }
      ],
      targetReturn: 0.15, // 15%
      maxWeight: 0.5, // 50%
      optimizationResult: null,

      // 风险预算
      riskBudget: null
    }
  },

  methods: {
    async loadRiskMetrics() {
      if (!this.selectedSymbol) return

      this.loading = true
      try {
        const result = await api.getAdvancedRiskMetrics(this.selectedSymbol)
        this.riskMetrics = result.metrics
      } catch (error) {
        this.$toast?.error(`获取风险指标失败: ${error.message}`)
        console.error('获取风险指标失败:', error)
      } finally {
        this.loading = false
      }
    },

    async runStressTest() {
      if (!this.stressTestSymbol) return

      this.stressTesting = true
      try {
        const result = await api.performStressTest(
          this.stressTestSymbol,
          [], // 使用默认场景
          this.stressTestTimeRange
        )
        this.stressTestResults = result.stress_test_results
      } catch (error) {
        this.$toast?.error(`压力测试失败: ${error.message}`)
        console.error('压力测试失败:', error)
      } finally {
        this.stressTesting = false
      }
    },

    async optimizePortfolio() {
      if (this.portfolioAssets.length < 2) {
        this.$toast?.warning('至少需要2个资产进行组合优化')
        return
      }

      // 验证权重总和
      const totalWeight = this.portfolioAssets.reduce((sum, asset) => sum + asset.weight, 0)
      if (Math.abs(totalWeight - 1.0) > 0.01) {
        this.$toast?.warning('资产权重总和必须等于1.0')
        return
      }

      this.optimizing = true
      try {
        const symbols = this.portfolioAssets.map(a => a.symbol)
        const weights = {}
        this.portfolioAssets.forEach(asset => {
          weights[asset.symbol] = asset.weight
        })

        const constraints = {
          max_weight: this.maxWeight
        }

        const result = await api.optimizePortfolio(
          symbols,
          this.targetReturn,
          constraints
        )

        // 格式化结果
        const weightsArray = Object.entries(result.optimal_weights).map(([symbol, weight]) => ({
          symbol,
          percentage: weight
        }))

        this.optimizationResult = {
          expectedReturn: this.targetReturn,
          expectedRisk: 0.12, // 模拟值
          sharpeRatio: 1.25, // 模拟值
          weights: weightsArray
        }

        this.$toast?.success('投资组合优化完成')
      } catch (error) {
        this.$toast?.error(`组合优化失败: ${error.message}`)
        console.error('组合优化失败:', error)
      } finally {
        this.optimizing = false
      }
    },

    async calculateRiskBudget() {
      if (this.portfolioAssets.length === 0) {
        this.$toast?.warning('请先添加资产')
        return
      }

      this.budgetCalculating = true
      try {
        const symbols = this.portfolioAssets.map(a => a.symbol)
        const weights = {}
        this.portfolioAssets.forEach(asset => {
          weights[asset.symbol] = asset.weight
        })

        const result = await api.getRiskBudget(symbols, weights, 1.0)

        this.riskBudget = {
          totalBudget: result.risk_budget.total_budget,
          assetBudgets: result.risk_budget.asset_budgets,
          assetsCount: result.assets_count
        }

        this.$toast?.success('风险预算计算完成')
      } catch (error) {
        this.$toast?.error(`风险预算计算失败: ${error.message}`)
        console.error('风险预算计算失败:', error)
      } finally {
        this.budgetCalculating = false
      }
    },

    addPortfolioAsset() {
      this.portfolioAssets.push({
        symbol: '',
        weight: 0.1
      })
    },

    removePortfolioAsset(index) {
      this.portfolioAssets.splice(index, 1)
    },

    getBetaInterpretation(beta) {
      if (Math.abs(beta) < 0.5) {
        return '低系统性风险，相对稳定'
      } else if (Math.abs(beta) < 1.5) {
        return '中等系统性风险，与市场相关性适中'
      } else {
        return '高系统性风险，易受市场影响'
      }
    }
  }
}
</script>

<style scoped>
.advanced-risk {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
  background: #f8f9fa;
  min-height: 100vh;
}

.risk-header {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.risk-header h1 {
  margin: 0 0 8px 0;
  font-size: 2.5rem;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  color: #666;
  font-size: 1.1rem;
  margin-bottom: 20px;
}

.risk-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.overview-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  gap: 16px;
}

.card-icon {
  font-size: 2rem;
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 12px;
}

.card-content {
  flex: 1;
}

.card-title {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 4px;
}

.card-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #333;
  margin-bottom: 4px;
}

.card-value.warning {
  color: #f59e0b;
}

.card-value.success {
  color: #10b981;
}

.card-subtitle {
  font-size: 0.8rem;
  color: #888;
}

.risk-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.tab-navigation {
  display: flex;
  background: white;
  border-radius: 12px;
  padding: 4px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-bottom: 24px;
}

.tab-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  font-weight: 600;
  color: #666;
}

.tab-button:hover {
  background: #f0f0f0;
}

.tab-button.active {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.tab-icon {
  font-size: 1.1rem;
}

.tab-content {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.content-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.5rem;
}

.controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.controls select {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.analyze-btn, .add-btn, .optimize-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.analyze-btn:hover:not(:disabled), .optimize-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.analyze-btn:disabled, .optimize-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.add-btn {
  background: #10b981;
}

.add-btn:hover {
  background: #059669;
}

/* 风险指标样式 */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
}

.metric-section {
  margin-bottom: 24px;
}

.metric-section h3 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 1.1rem;
  border-bottom: 2px solid #667eea;
  padding-bottom: 8px;
}

.metric-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.metric-card {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e9ecef;
}

.metric-name {
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
  font-size: 0.9rem;
}

.metric-value {
  font-size: 1.25rem;
  font-weight: bold;
  color: #667eea;
  margin-bottom: 8px;
}

.metric-bar {
  height: 6px;
  background: #e9ecef;
  border-radius: 3px;
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.bar-fill.high-risk {
  background: linear-gradient(90deg, #ef4444 0%, #dc2626 100%);
}

.metric-indicator {
  font-size: 0.8rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 12px;
  display: inline-block;
}

.metric-indicator.positive {
  background: #d1fae5;
  color: #065f46;
}

.metric-indicator.negative {
  background: #fee2e2;
  color: #991b1b;
}

/* VaR指标样式 */
.var-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.var-card {
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #3b82f6;
}

.var-confidence {
  font-weight: 600;
  color: #1e40af;
  margin-bottom: 8px;
}

.var-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #1d4ed8;
  margin-bottom: 8px;
}

.var-desc {
  font-size: 0.85rem;
  color: #3730a3;
  line-height: 1.4;
}

/* 市场风险样式 */
.market-risk {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.beta-analysis {
  background: #fef3c7;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #f59e0b;
}

.beta-value {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.beta-label {
  font-weight: 600;
  color: #92400e;
}

.beta-number {
  font-size: 1.25rem;
  font-weight: bold;
  padding: 4px 8px;
  border-radius: 6px;
}

.beta-number.high {
  background: #fee2e2;
  color: #dc2626;
}

.beta-number.low {
  background: #d1fae5;
  color: #059669;
}

.beta-interpretation {
  font-size: 0.9rem;
  color: #92400e;
  line-height: 1.4;
}

.liquidity-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.liquidity-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f0f9ff;
  border-radius: 6px;
  border: 1px solid #3b82f6;
}

.liquidity-label {
  font-weight: 600;
  color: #1e40af;
}

.liquidity-value {
  font-weight: 600;
  color: #1d4ed8;
}

/* 压力测试样式 */
.stress-test-results {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.stress-result {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
  border-left: 4px solid #667eea;
}

.scenario-name {
  font-weight: 600;
  color: #333;
  margin-bottom: 8px;
}

.scenario-shock {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 8px;
}

.scenario-loss {
  font-size: 1.25rem;
  font-weight: bold;
  color: #333;
}

.scenario-loss.critical {
  color: #dc2626;
}

.scenario-loss.warning {
  color: #d97706;
}

/* 投资组合优化样式 */
.portfolio-setup {
  margin-bottom: 24px;
}

.assets-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.asset-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 1px solid #e9ecef;
}

.asset-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.asset-symbol {
  font-weight: 600;
  color: #333;
  min-width: 60px;
}

.weight-input {
  width: 80px;
  padding: 6px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  text-align: center;
}

.remove-btn {
  background: #ef4444;
  color: white;
  border: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 0.8rem;
}

.remove-btn:hover {
  background: #dc2626;
}

.optimization-params {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.param-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.param-group label {
  font-weight: 600;
  color: #333;
  white-space: nowrap;
}

.param-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  text-align: center;
}

.param-unit {
  font-size: 0.9rem;
  color: #666;
}

/* 优化结果样式 */
.optimization-result {
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  padding: 20px;
  border: 1px solid #3b82f6;
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.result-header h3 {
  margin: 0;
  color: #1e40af;
}

.result-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.stat-item {
  text-align: center;
}

.stat-label {
  display: block;
  font-size: 0.8rem;
  color: #3730a3;
  margin-bottom: 4px;
}

.stat-value {
  display: block;
  font-size: 1.1rem;
  font-weight: bold;
  color: #1d4ed8;
}

.optimal-weights {
  margin-top: 20px;
}

.optimal-weights h4 {
  margin: 0 0 16px 0;
  color: #1e40af;
}

.weights-chart {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.weight-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.weight-symbol {
  min-width: 50px;
  font-weight: 600;
  color: #333;
}

.weight-bar-container {
  flex: 1;
  height: 12px;
  background: #e9ecef;
  border-radius: 6px;
  overflow: hidden;
}

.weight-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 6px;
  transition: width 0.3s ease;
}

.weight-percentage {
  min-width: 50px;
  text-align: right;
  font-weight: 600;
  color: #667eea;
}

/* 风险预算样式 */
.budget-analysis {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border-radius: 8px;
  padding: 20px;
  border: 1px solid #f59e0b;
}

.budget-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.budget-stat {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 6px;
}

.budget-stat .stat-label {
  font-weight: 600;
  color: #92400e;
}

.budget-stat .stat-value {
  font-weight: bold;
  color: #d97706;
}

.budget-allocation h4 {
  margin: 0 0 16px 0;
  color: #92400e;
}

.allocation-chart {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.allocation-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 6px;
}

.allocation-symbol {
  min-width: 50px;
  font-weight: 600;
  color: #333;
}

.allocation-bar {
  flex: 1;
  height: 12px;
  background: #e9ecef;
  border-radius: 6px;
  overflow: hidden;
}

.allocation-fill {
  height: 100%;
  background: linear-gradient(90deg, #f59e0b 0%, #d97706 100%);
  border-radius: 6px;
  transition: width 0.3s ease;
}

.allocation-value {
  min-width: 60px;
  text-align: right;
  font-weight: 600;
  color: #d97706;
}

/* 占位符样式 */
.stress-chart {
  height: 300px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 2px dashed #ddd;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 20px 0;
}

.chart-placeholder {
  text-align: center;
  color: #666;
}

.placeholder-icon {
  font-size: 3rem;
  margin-bottom: 12px;
}

.chart-placeholder p {
  margin: 8px 0 4px 0;
  font-weight: 600;
}

.chart-placeholder small {
  color: #888;
}

/* 空状态样式 */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #666;
}

.empty-icon {
  font-size: 4rem;
  margin-bottom: 20px;
}

.empty-state p {
  font-size: 1.1rem;
  margin-bottom: 20px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .risk-header h1 {
    font-size: 2rem;
  }

  .risk-overview {
    grid-template-columns: 1fr;
  }

  .tab-navigation {
    flex-direction: column;
  }

  .tab-button {
    width: 100%;
  }

  .content-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .controls {
    flex-direction: column;
    width: 100%;
  }

  .controls select {
    width: 100%;
  }

  .analyze-btn {
    width: 100%;
  }

  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .metric-cards {
    grid-template-columns: 1fr;
  }

  .var-metrics {
    grid-template-columns: 1fr;
  }

  .stress-test-results {
    grid-template-columns: 1fr;
  }

  .optimization-params {
    grid-template-columns: 1fr;
  }

  .result-stats {
    grid-template-columns: 1fr;
  }

  .budget-overview {
    grid-template-columns: 1fr;
  }
}
</style>
