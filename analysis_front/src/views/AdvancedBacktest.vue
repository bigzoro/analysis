<template>
  <div class="advanced-backtest">
    <div class="backtest-header">
      <h1>🔬 高级回测分析</h1>
      <p class="subtitle">专业的量化策略验证与优化</p>

      <!-- 回测概览面板 -->
      <div class="backtest-overview">
        <div class="overview-card">
          <div class="card-icon">📊</div>
          <div class="card-content">
            <div class="card-title">最新回测</div>
            <div class="card-value">{{ latestBacktest?.summary?.totalReturn ? (latestBacktest.summary.totalReturn * 100).toFixed(2) + '%' : '无' }}</div>
            <div class="card-subtitle">总收益率</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">🎯</div>
          <div class="card-content">
            <div class="card-title">胜率</div>
            <div class="card-value success">{{ latestBacktest?.summary?.winRate ? (latestBacktest.summary.winRate * 100).toFixed(1) + '%' : '无' }}</div>
            <div class="card-subtitle">交易胜率</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">📈</div>
          <div class="card-content">
            <div class="card-title">夏普比率</div>
            <div class="card-value">{{ latestBacktest?.summary?.sharpeRatio ? latestBacktest.summary.sharpeRatio.toFixed(2) : '无' }}</div>
            <div class="card-subtitle">风险调整收益</div>
          </div>
        </div>

        <div class="overview-card">
          <div class="card-icon">⚠️</div>
          <div class="card-content">
            <div class="card-title">最大回撤</div>
            <div class="card-value warning">{{ latestBacktest?.summary?.maxDrawdown ? (latestBacktest.summary.maxDrawdown * 100).toFixed(2) + '%' : '无' }}</div>
            <div class="card-subtitle">最大亏损幅度</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 主要功能区域 -->
    <div class="backtest-content">
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

      <!-- 基础回测 -->
      <div v-if="activeTab === 'basic'" class="tab-content">
        <div class="content-header">
          <h2>基础回测分析</h2>
          <div class="controls">
            <select v-model="basicBacktest.symbol">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <select v-model="basicBacktest.strategy">
              <option value="buy_and_hold">买入持有</option>
              <option value="ml_prediction">机器学习预测</option>
              <option value="ensemble">集成策略</option>
            </select>
            <button @click="runBasicBacktest" :disabled="!basicBacktest.symbol || runningBasic" class="analyze-btn">
              {{ runningBasic ? '回测中...' : '🚀 开始回测' }}
            </button>
          </div>
        </div>

        <!-- 回测参数设置 -->
        <div class="backtest-params">
          <div class="param-row">
            <div class="param-group">
              <label>开始日期:</label>
              <input v-model="basicBacktest.startDate" type="date" class="param-input" />
            </div>
            <div class="param-group">
              <label>结束日期:</label>
              <input v-model="basicBacktest.endDate" type="date" class="param-input" />
            </div>
            <div class="param-group">
              <label>初始资金:</label>
              <input v-model.number="basicBacktest.initialCash" type="number" min="1000" class="param-input" />
            </div>
          </div>

          <div class="param-row">
            <div class="param-group">
              <label>最大仓位:</label>
              <input v-model.number="basicBacktest.maxPosition" type="number" min="0" max="1" step="0.1" class="param-input" />
            </div>
            <div class="param-group">
              <label>止损比例:</label>
              <input v-model.number="basicBacktest.stopLoss" type="number" min="0" max="0.5" step="0.01" class="param-input" />
            </div>
            <div class="param-group">
              <label>止盈比例:</label>
              <input v-model.number="basicBacktest.takeProfit" type="number" min="0" max="1" step="0.05" class="param-input" />
            </div>
          </div>
        </div>

        <!-- 回测结果展示 -->
        <div v-if="basicResult" class="backtest-result">
          <div class="result-header">
            <h3>回测结果</h3>
            <div class="result-meta">
              <span>策略: {{ basicResult.config.strategy }}</span>
              <span>期间: {{ formatDate(basicResult.config.startDate) }} - {{ formatDate(basicResult.config.endDate) }}</span>
            </div>
          </div>

          <div class="result-metrics">
            <div class="metric-grid">
              <div class="metric-item">
                <div class="metric-name">总收益率</div>
                <div class="metric-value" :class="{ positive: basicResult.summary.totalReturn > 0, negative: basicResult.summary.totalReturn < 0 }">
                  {{ (basicResult.summary.totalReturn * 100).toFixed(2) }}%
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">年化收益率</div>
                <div class="metric-value" :class="{ positive: basicResult.summary.annualReturn > 0 }">
                  {{ (basicResult.summary.annualReturn * 100).toFixed(2) }}%
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">胜率</div>
                <div class="metric-value success">
                  {{ (basicResult.summary.winRate * 100).toFixed(1) }}%
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">总交易次数</div>
                <div class="metric-value">
                  {{ basicResult.summary.totalTrades }}
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">夏普比率</div>
                <div class="metric-value">
                  {{ basicResult.summary.sharpeRatio.toFixed(2) }}
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">最大回撤</div>
                <div class="metric-value warning">
                  {{ (basicResult.summary.maxDrawdown * 100).toFixed(2) }}%
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">波动率</div>
                <div class="metric-value">
                  {{ (basicResult.summary.volatility * 100).toFixed(2) }}%
                </div>
              </div>

              <div class="metric-item">
                <div class="metric-name">总手续费</div>
                <div class="metric-value">
                  {{ basicResult.summary.totalCommission.toFixed(2) }}
                </div>
              </div>
            </div>
          </div>

          <!-- 收益曲线图表 -->
          <div class="returns-chart">
            <div class="chart-placeholder">
              <div class="placeholder-icon">📈</div>
              <p>收益曲线可视化图表</p>
              <small>展示策略的收益变化过程</small>
            </div>
          </div>
        </div>
      </div>

      <!-- 走步前进分析 -->
      <div v-if="activeTab === 'walk-forward'" class="tab-content">
        <div class="content-header">
          <h2>走步前进分析</h2>
          <div class="controls">
            <select v-model="walkForward.symbol">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <button @click="runWalkForwardAnalysis" :disabled="!walkForward.symbol || runningWalkForward" class="analyze-btn">
              {{ runningWalkForward ? '分析中...' : '🔍 开始分析' }}
            </button>
          </div>
        </div>

        <!-- 走步前进参数 -->
        <div class="analysis-params">
          <div class="param-row">
            <div class="param-group">
              <label>样本内周期(月):</label>
              <input v-model.number="walkForward.inSamplePeriod" type="number" min="3" max="24" class="param-input" />
            </div>
            <div class="param-group">
              <label>样本外周期(月):</label>
              <input v-model.number="walkForward.outOfSamplePeriod" type="number" min="1" max="12" class="param-input" />
            </div>
            <div class="param-group">
              <label>步长(月):</label>
              <input v-model.number="walkForward.stepSize" type="number" min="1" max="6" class="param-input" />
            </div>
          </div>
        </div>

        <!-- 走步前进结果 -->
        <div v-if="walkForwardResult" class="analysis-result">
          <div class="result-summary">
            <div class="summary-stat">
              <span class="stat-label">总窗口数:</span>
              <span class="stat-value">{{ walkForwardResult.summary.totalWindows }}</span>
            </div>
            <div class="summary-stat">
              <span class="stat-label">平均稳健性:</span>
              <span class="stat-value">{{ walkForwardResult.summary.averageRobustness.toFixed(3) }}</span>
            </div>
            <div class="summary-stat">
              <span class="stat-label">样本外收益率:</span>
              <span class="stat-value" :class="{ positive: walkForwardResult.summary.outOfSampleReturn > 0 }">
                {{ (walkForwardResult.summary.outOfSampleReturn * 100).toFixed(2) }}%
              </span>
            </div>
            <div class="summary-stat">
              <span class="stat-label">一致性得分:</span>
              <span class="stat-value">{{ walkForwardResult.summary.consistencyScore.toFixed(3) }}</span>
            </div>
          </div>

          <!-- 窗口详情 -->
          <div class="windows-detail">
            <h4>窗口分析详情</h4>
            <div class="windows-table">
              <div class="table-header">
                <div>窗口</div>
                <div>样本内期间</div>
                <div>样本外期间</div>
                <div>样本内收益</div>
                <div>样本外收益</div>
                <div>稳健性</div>
              </div>
              <div
                v-for="window in walkForwardResult.windows"
                :key="window.windowId"
                class="table-row"
              >
                <div>{{ window.windowId }}</div>
                <div>{{ formatDate(window.inSampleStart) }} - {{ formatDate(window.inSampleEnd) }}</div>
                <div>{{ formatDate(window.outOfSampleStart) }} - {{ formatDate(window.outOfSampleEnd) }}</div>
                <div :class="{ positive: window.inSampleResult?.summary?.totalReturn > 0 }">
                  {{ window.inSampleResult ? (window.inSampleResult.summary.totalReturn * 100).toFixed(1) + '%' : 'N/A' }}
                </div>
                <div :class="{ positive: window.outOfSampleResult?.summary?.totalReturn > 0 }">
                  {{ window.outOfSampleResult ? (window.outOfSampleResult.summary.totalReturn * 100).toFixed(1) + '%' : 'N/A' }}
                </div>
                <div :class="{ good: window.robustnessScore > 0.7, poor: window.robustnessScore < 0.3 }">
                  {{ window.robustnessScore.toFixed(3) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 蒙特卡洛分析 -->
      <div v-if="activeTab === 'monte-carlo'" class="tab-content">
        <div class="content-header">
          <h2>蒙特卡洛分析</h2>
          <div class="controls">
            <select v-model="monteCarlo.symbol">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <button @click="runMonteCarloAnalysis" :disabled="!monteCarlo.symbol || runningMonteCarlo" class="analyze-btn">
              {{ runningMonteCarlo ? '模拟中...' : '🎲 开始模拟' }}
            </button>
          </div>
        </div>

        <!-- 蒙特卡洛参数 -->
        <div class="analysis-params">
          <div class="param-row">
            <div class="param-group">
              <label>模拟次数:</label>
              <input v-model.number="monteCarlo.simulations" type="number" min="100" max="10000" step="100" class="param-input" />
            </div>
            <div class="param-group">
              <label>自举样本大小:</label>
              <input v-model.number="monteCarlo.bootstrapSize" type="number" min="30" max="1000" class="param-input" />
            </div>
          </div>
        </div>

        <!-- 蒙特卡洛结果 -->
        <div v-if="monteCarloResult" class="analysis-result">
          <div class="scenario-summary">
            <div class="scenario-card best-case">
              <h4>最佳情景</h4>
              <div class="scenario-metrics">
                <div class="metric">收益: {{ (monteCarloResult.bestCase.return * 100).toFixed(2) }}%</div>
                <div class="metric">风险: {{ (monteCarloResult.bestCase.risk * 100).toFixed(2) }}%</div>
                <div class="metric">胜率: {{ (monteCarloResult.bestCase.winRate * 100).toFixed(1) }}%</div>
              </div>
            </div>

            <div class="scenario-card expected-case">
              <h4>期望情景</h4>
              <div class="scenario-metrics">
                <div class="metric">收益: {{ (monteCarloResult.returnDistribution.mean * 100).toFixed(2) }}%</div>
                <div class="metric">风险: {{ (monteCarloResult.riskDistribution.mean * 100).toFixed(2) }}%</div>
                <div class="metric">夏普: {{ (monteCarloResult.returnDistribution.mean / monteCarloResult.riskDistribution.mean).toFixed(2) }}</div>
              </div>
            </div>

            <div class="scenario-card worst-case">
              <h4>最差情景</h4>
              <div class="scenario-metrics">
                <div class="metric">收益: {{ (monteCarloResult.worstCase.return * 100).toFixed(2) }}%</div>
                <div class="metric">风险: {{ (monteCarloResult.worstCase.risk * 100).toFixed(2) }}%</div>
                <div class="metric">最大回撤: {{ (monteCarloResult.worstCase.maxDrawdown * 100).toFixed(2) }}%</div>
              </div>
            </div>
          </div>

          <!-- 分布统计 -->
          <div class="distribution-stats">
            <h4>收益分布统计</h4>
            <div class="stats-grid">
              <div class="stat-item">
                <span class="stat-label">均值:</span>
                <span class="stat-value">{{ (monteCarloResult.returnDistribution.mean * 100).toFixed(2) }}%</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">标准差:</span>
                <span class="stat-value">{{ (monteCarloResult.returnDistribution.stdDev * 100).toFixed(2) }}%</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">偏度:</span>
                <span class="stat-value">{{ monteCarloResult.returnDistribution.skewness.toFixed(3) }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">峰度:</span>
                <span class="stat-value">{{ monteCarloResult.returnDistribution.kurtosis.toFixed(3) }}</span>
              </div>
            </div>
          </div>

          <!-- 置信区间 -->
          <div class="confidence-intervals">
            <h4>置信区间</h4>
            <div class="intervals-table">
              <div class="interval-row">
                <div v-for="(interval, level) in monteCarloResult.confidenceIntervals" :key="level" class="interval-item">
                  <div class="interval-level">{{ level }}</div>
                  <div class="interval-range">
                    {{ (interval.lower * 100).toFixed(2) }}% - {{ (interval.upper * 100).toFixed(2) }}%
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 策略优化 -->
      <div v-if="activeTab === 'optimization'" class="tab-content">
        <div class="content-header">
          <h2>策略参数优化</h2>
          <div class="controls">
            <select v-model="optimization.symbol">
              <option value="">选择交易对</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <select v-model="optimization.method">
              <option value="grid">网格搜索</option>
              <option value="random">随机搜索</option>
              <option value="genetic">遗传算法</option>
            </select>
            <button @click="runStrategyOptimization" :disabled="!optimization.symbol || runningOptimization" class="analyze-btn">
              {{ runningOptimization ? '优化中...' : '⚡ 开始优化' }}
            </button>
          </div>
        </div>

        <!-- 优化参数配置 -->
        <div class="optimization-params">
          <div class="param-section">
            <h4>优化目标</h4>
            <select v-model="optimization.objective" class="param-select">
              <option value="sharpe">最大化夏普比率</option>
              <option value="return">最大化总收益</option>
              <option value="win_rate">最大化胜率</option>
              <option value="drawdown">最小化最大回撤</option>
            </select>
          </div>

          <div class="param-section">
            <h4>参数范围</h4>
            <div class="parameter-list">
              <div
                v-for="(param, index) in optimization.parameters"
                :key="param.name"
                class="parameter-item"
              >
                <div class="param-name">{{ param.name }}</div>
                <div class="param-controls">
                  <input v-model.number="param.minValue" type="number" step="0.01" class="param-input-small" placeholder="最小值" />
                  <span>-</span>
                  <input v-model.number="param.maxValue" type="number" step="0.01" class="param-input-small" placeholder="最大值" />
                  <input v-model.number="param.stepSize" type="number" step="0.01" class="param-input-small" placeholder="步长" />
                  <button @click="removeOptimizationParameter(index)" class="remove-param-btn">✕</button>
                </div>
              </div>
              <button @click="addOptimizationParameter" class="add-param-btn">➕ 添加参数</button>
            </div>
          </div>
        </div>

        <!-- 优化结果 -->
        <div v-if="optimizationResult" class="optimization-result">
          <div class="best-params">
            <h4>最优参数组合</h4>
            <div class="params-display">
              <div
                v-for="(value, name) in optimizationResult.bestParams"
                :key="name"
                class="param-display"
              >
                <span class="param-label">{{ name }}:</span>
                <span class="param-value">{{ typeof value === 'number' ? value.toFixed(3) : value }}</span>
              </div>
            </div>
          </div>

          <div class="best-result">
            <h4>最优结果表现</h4>
            <div class="result-metrics">
              <div class="metric-item">
                <span class="metric-label">总收益:</span>
                <span class="metric-value" :class="{ positive: optimizationResult.bestResult.summary.totalReturn > 0 }">
                  {{ (optimizationResult.bestResult.summary.totalReturn * 100).toFixed(2) }}%
                </span>
              </div>
              <div class="metric-item">
                <span class="metric-label">夏普比率:</span>
                <span class="metric-value">{{ optimizationResult.bestResult.summary.sharpeRatio.toFixed(2) }}</span>
              </div>
              <div class="metric-item">
                <span class="metric-label">胜率:</span>
                <span class="metric-value success">{{ (optimizationResult.bestResult.summary.winRate * 100).toFixed(1) }}%</span>
              </div>
              <div class="metric-item">
                <span class="metric-label">最大回撤:</span>
                <span class="metric-value warning">{{ (optimizationResult.bestResult.summary.maxDrawdown * 100).toFixed(2) }}%</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 归因分析 -->
      <div v-if="activeTab === 'attribution'" class="tab-content">
        <div class="content-header">
          <h2>策略归因分析</h2>
          <div class="controls">
            <select v-model="attribution.symbol">
              <option value="">选择策略</option>
              <option v-for="symbol in availableSymbols" :key="symbol" :value="symbol">
                {{ symbol }}
              </option>
            </select>
            <select v-model="attribution.benchmarkSymbol">
              <option value="">选择基准</option>
              <option value="BTC">BTC</option>
              <option value="SPY">S&P 500</option>
              <option value="QQQ">纳斯达克</option>
            </select>
            <button @click="runAttributionAnalysis" :disabled="!attribution.symbol || !attribution.benchmarkSymbol || runningAttribution" class="analyze-btn">
              {{ runningAttribution ? '分析中...' : '📊 开始分析' }}
            </button>
          </div>
        </div>

        <!-- 归因分析结果 -->
        <div v-if="attributionResult" class="analysis-result">
          <div class="attribution-summary">
            <div class="summary-card">
              <h4>总归因分解</h4>
              <div class="attribution-breakdown">
                <div class="breakdown-item">
                  <span class="breakdown-label">策略总收益:</span>
                  <span class="breakdown-value">{{ (attributionResult.totalAttribution.totalReturn * 100).toFixed(2) }}%</span>
                </div>
                <div class="breakdown-item">
                  <span class="breakdown-label">基准收益:</span>
                  <span class="breakdown-value">{{ (attributionResult.totalAttribution.benchmarkReturn * 100).toFixed(2) }}%</span>
                </div>
                <div class="breakdown-item">
                  <span class="breakdown-label">超额收益:</span>
                  <span class="breakdown-value" :class="{ positive: attributionResult.totalAttribution.excessReturn > 0, negative: attributionResult.totalAttribution.excessReturn < 0 }">
                    {{ (attributionResult.totalAttribution.excessReturn * 100).toFixed(2) }}%
                  </span>
                </div>
                <div class="breakdown-item">
                  <span class="breakdown-label">资产配置贡献:</span>
                  <span class="breakdown-value">{{ (attributionResult.totalAttribution.assetAllocation * 100).toFixed(2) }}%</span>
                </div>
                <div class="breakdown-item">
                  <span class="breakdown-label">证券选择贡献:</span>
                  <span class="breakdown-value">{{ (attributionResult.totalAttribution.securitySelection * 100).toFixed(2) }}%</span>
                </div>
              </div>
            </div>

            <div class="summary-card">
              <h4>风险归因</h4>
              <div class="risk-attribution">
                <div class="risk-item">
                  <span class="risk-label">总波动率:</span>
                  <span class="risk-value">{{ (attributionResult.riskAttribution.totalVolatility * 100).toFixed(2) }}%</span>
                </div>
                <div class="risk-item">
                  <span class="risk-label">主动风险:</span>
                  <span class="risk-value">{{ (attributionResult.riskAttribution.activeRisk * 100).toFixed(2) }}%</span>
                </div>
                <div class="risk-item">
                  <span class="risk-label">资产配置风险:</span>
                  <span class="risk-value">{{ (attributionResult.riskAttribution.assetAllocationRisk * 100).toFixed(2) }}%</span>
                </div>
                <div class="risk-item">
                  <span class="risk-label">证券选择风险:</span>
                  <span class="risk-value">{{ (attributionResult.riskAttribution.securitySelectionRisk * 100).toFixed(2) }}%</span>
                </div>
              </div>
            </div>
          </div>

          <!-- 周期归因 -->
          <div class="periodic-attribution">
            <h4>周期归因分析</h4>
            <div class="periods-table">
              <div class="table-header">
                <div>期间</div>
                <div>策略收益</div>
                <div>基准收益</div>
                <div>超额收益</div>
                <div>资产配置</div>
                <div>证券选择</div>
              </div>
              <div
                v-for="period in attributionResult.periodicAttribution"
                :key="period.period"
                class="table-row"
              >
                <div>{{ period.period }}</div>
                <div :class="{ positive: period.return > 0 }">{{ (period.return * 100).toFixed(1) }}%</div>
                <div :class="{ positive: period.benchmarkReturn > 0 }">{{ (period.benchmarkReturn * 100).toFixed(1) }}%</div>
                <div :class="{ positive: period.excessReturn > 0, negative: period.excessReturn < 0 }">{{ (period.excessReturn * 100).toFixed(1) }}%</div>
                <div>{{ (period.attribution.assetAllocation * 100).toFixed(1) }}%</div>
                <div>{{ (period.attribution.securitySelection * 100).toFixed(1) }}%</div>
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
  name: 'AdvancedBacktest',
  data() {
    return {
      activeTab: 'basic',
      runningBasic: false,
      runningWalkForward: false,
      runningMonteCarlo: false,
      runningOptimization: false,
      runningAttribution: false,

      // 标签页
      tabs: [
        { id: 'basic', title: '基础回测', icon: '📊' },
        { id: 'walk-forward', title: '走步前进', icon: '🚶' },
        { id: 'monte-carlo', title: '蒙特卡洛', icon: '🎲' },
        { id: 'optimization', title: '策略优化', icon: '⚡' },
        { id: 'attribution', title: '归因分析', icon: '📈' }
      ],

      // 可用交易对
      availableSymbols: ['BTC', 'ETH', 'ADA', 'SOL', 'DOT', 'LINK', 'UNI', 'AAVE'],

      // 基础回测
      basicBacktest: {
        symbol: '',
        strategy: 'ml_prediction',
        startDate: this.getDefaultStartDate(),
        endDate: this.getDefaultEndDate(),
        initialCash: 10000,
        maxPosition: 1.0,
        stopLoss: 0.05,
        takeProfit: 0.10
      },
      basicResult: null,
      latestBacktest: null,

      // 走步前进分析
      walkForward: {
        symbol: '',
        inSamplePeriod: 12,
        outOfSamplePeriod: 3,
        stepSize: 3
      },
      walkForwardResult: null,

      // 蒙特卡洛分析
      monteCarlo: {
        symbol: '',
        simulations: 1000,
        bootstrapSize: 252
      },
      monteCarloResult: null,

      // 策略优化
      optimization: {
        symbol: '',
        method: 'grid',
        objective: 'sharpe',
        parameters: [
          {
            name: 'stop_loss',
            minValue: 0.01,
            maxValue: 0.10,
            stepSize: 0.01
          },
          {
            name: 'take_profit',
            minValue: 0.05,
            maxValue: 0.20,
            stepSize: 0.02
          }
        ]
      },
      optimizationResult: null,

      // 归因分析
      attribution: {
        symbol: '',
        benchmarkSymbol: '',
        timeHorizon: 'monthly'
      },
      attributionResult: null
    }
  },

  methods: {
    getDefaultStartDate() {
      const date = new Date()
      date.setMonth(date.getMonth() - 12)
      return date.toISOString().split('T')[0]
    },

    getDefaultEndDate() {
      return new Date().toISOString().split('T')[0]
    },

    formatDate(dateString) {
      if (!dateString) return ''
      const date = new Date(dateString)
      return date.toLocaleDateString('zh-CN')
    },

    async runBasicBacktest() {
      if (!this.basicBacktest.symbol) return

      this.runningBasic = true
      try {
        // 这里应该调用后端的回测API
        // 暂时使用模拟数据
        await new Promise(resolve => setTimeout(resolve, 2000)) // 模拟API调用

        this.basicResult = {
          config: this.basicBacktest,
          summary: {
            totalReturn: 0.156,
            annualReturn: 0.142,
            winRate: 0.583,
            totalTrades: 47,
            sharpeRatio: 1.87,
            maxDrawdown: 0.089,
            volatility: 0.234,
            totalCommission: 23.50
          },
          trades: [],
          dailyReturns: []
        }

        this.latestBacktest = this.basicResult

        this.$toast?.success('基础回测完成')
      } catch (error) {
        this.$toast?.error(`回测失败: ${error.message}`)
        console.error('基础回测失败:', error)
      } finally {
        this.runningBasic = false
      }
    },

    async runWalkForwardAnalysis() {
      if (!this.walkForward.symbol) return

      this.runningWalkForward = true
      try {
        const result = await api.runWalkForwardAnalysis(
          this.walkForward.symbol,
          new Date(this.basicBacktest.startDate),
          new Date(this.basicBacktest.endDate),
          'ml_prediction',
          this.walkForward.inSamplePeriod,
          this.walkForward.outOfSamplePeriod,
          this.walkForward.stepSize
        )

        this.walkForwardResult = result.walk_forward_analysis
        this.$toast?.success('走步前进分析完成')
      } catch (error) {
        this.$toast?.error(`走步前进分析失败: ${error.message}`)
        console.error('走步前进分析失败:', error)
      } finally {
        this.runningWalkForward = false
      }
    },

    async runMonteCarloAnalysis() {
      if (!this.monteCarlo.symbol) return

      this.runningMonteCarlo = true
      try {
        const result = await api.runMonteCarloAnalysis(
          this.monteCarlo.symbol,
          new Date(this.basicBacktest.startDate),
          new Date(this.basicBacktest.endDate),
          'ml_prediction',
          this.monteCarlo.simulations,
          this.monteCarlo.bootstrapSize
        )

        this.monteCarloResult = result.monte_carlo_analysis
        this.$toast?.success('蒙特卡洛分析完成')
      } catch (error) {
        this.$toast?.error(`蒙特卡洛分析失败: ${error.message}`)
        console.error('蒙特卡洛分析失败:', error)
      } finally {
        this.runningMonteCarlo = false
      }
    },

    async runStrategyOptimization() {
      if (!this.optimization.symbol) return

      this.runningOptimization = true
      try {
        const result = await api.runStrategyOptimization(
          this.optimization.symbol,
          new Date(this.basicBacktest.startDate),
          new Date(this.basicBacktest.endDate),
          'ml_prediction',
          this.optimization.parameters,
          this.optimization.method,
          100,
          this.optimization.objective
        )

        this.optimizationResult = result.strategy_optimization
        this.$toast?.success('策略优化完成')
      } catch (error) {
        this.$toast?.error(`策略优化失败: ${error.message}`)
        console.error('策略优化失败:', error)
      } finally {
        this.runningOptimization = false
      }
    },

    async runAttributionAnalysis() {
      if (!this.attribution.symbol || !this.attribution.benchmarkSymbol) return

      this.runningAttribution = true
      try {
        const result = await api.runAttributionAnalysis(
          this.attribution.symbol,
          this.attribution.benchmarkSymbol,
          new Date(this.basicBacktest.startDate),
          new Date(this.basicBacktest.endDate),
          'ml_prediction',
          this.attribution.timeHorizon
        )

        this.attributionResult = result.attribution_analysis
        this.$toast?.success('归因分析完成')
      } catch (error) {
        this.$toast?.error(`归因分析失败: ${error.message}`)
        console.error('归因分析失败:', error)
      } finally {
        this.runningAttribution = false
      }
    },

    addOptimizationParameter() {
      this.optimization.parameters.push({
        name: '',
        minValue: 0,
        maxValue: 1,
        stepSize: 0.1
      })
    },

    removeOptimizationParameter(index) {
      this.optimization.parameters.splice(index, 1)
    }
  }
}
</script>

<style scoped>
.advanced-backtest {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
  background: #f8f9fa;
  min-height: 100vh;
}

.backtest-header {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.backtest-header h1 {
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

.backtest-overview {
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

.card-value.success {
  color: #10b981;
}

.card-value.warning {
  color: #f59e0b;
}

.card-subtitle {
  font-size: 0.8rem;
  color: #888;
}

.backtest-content {
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
  overflow-x: auto;
}

.tab-button {
  flex: 1;
  min-width: 120px;
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
  white-space: nowrap;
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

.analyze-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.analyze-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.analyze-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* 基础回测样式 */
.backtest-params {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 24px;
}

.param-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.param-row:last-child {
  margin-bottom: 0;
}

.param-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.param-group label {
  font-weight: 600;
  color: #333;
  font-size: 0.9rem;
}

.param-input {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.param-input:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.1);
}

/* 回测结果样式 */
.backtest-result {
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

.result-meta {
  display: flex;
  gap: 16px;
  font-size: 0.9rem;
  color: #3730a3;
}

.result-metrics {
  margin-bottom: 20px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.metric-item {
  background: white;
  border-radius: 6px;
  padding: 12px;
  text-align: center;
  border: 1px solid #e5e7eb;
}

.metric-name {
  font-size: 0.8rem;
  color: #6b7280;
  margin-bottom: 4px;
}

.metric-value {
  font-size: 1.1rem;
  font-weight: bold;
  color: #1f2937;
}

.metric-value.positive {
  color: #059669;
}

.metric-value.negative {
  color: #dc2626;
}

.metric-value.success {
  color: #059669;
}

.metric-value.warning {
  color: #d97706;
}

/* 走步前进样式 */
.analysis-params {
  background: #fef3c7;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 24px;
  border: 1px solid #f59e0b;
}

.analysis-result {
  background: white;
  border-radius: 8px;
  padding: 20px;
  border: 1px solid #e5e7eb;
}

.result-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.summary-stat {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 6px;
}

.stat-label {
  font-weight: 600;
  color: #374151;
}

.stat-value {
  font-weight: bold;
  color: #1f2937;
}

.stat-value.positive {
  color: #059669;
}

.windows-detail h4 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.windows-table {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 60px 1fr 1fr 100px 100px 100px;
  gap: 12px;
  padding: 12px;
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
  font-size: 0.8rem;
}

.table-row {
  display: grid;
  grid-template-columns: 60px 1fr 1fr 100px 100px 100px;
  gap: 12px;
  padding: 12px;
  border-top: 1px solid #e5e7eb;
  font-size: 0.8rem;
}

.table-row:hover {
  background: #f9fafb;
}

.table-row .good {
  color: #059669;
  font-weight: 600;
}

.table-row .poor {
  color: #dc2626;
  font-weight: 600;
}

/* 蒙特卡洛样式 */
.scenario-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.scenario-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  border: 2px solid #e5e7eb;
}

.scenario-card.best-case {
  border-color: #059669;
  background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
}

.scenario-card.expected-case {
  border-color: #3b82f6;
  background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
}

.scenario-card.worst-case {
  border-color: #dc2626;
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
}

.scenario-card h4 {
  margin: 0 0 12px 0;
  color: #1f2937;
}

.scenario-metrics {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.scenario-metrics .metric {
  font-size: 0.9rem;
  color: #374151;
}

.distribution-stats h4 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.stat-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
}

.confidence-intervals h4 {
  margin: 20px 0 16px 0;
  color: #1f2937;
}

.intervals-table {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 16px;
}

.interval-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.interval-item {
  text-align: center;
}

.interval-level {
  font-weight: 600;
  color: #374151;
  margin-bottom: 4px;
}

.interval-range {
  font-size: 0.9rem;
  color: #6b7280;
}

/* 策略优化样式 */
.optimization-params {
  background: #f0f9ff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 24px;
  border: 1px solid #3b82f6;
}

.param-section {
  margin-bottom: 20px;
}

.param-section:last-child {
  margin-bottom: 0;
}

.param-section h4 {
  margin: 0 0 12px 0;
  color: #1e40af;
}

.param-select {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.parameter-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.parameter-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e5e7eb;
}

.param-name {
  min-width: 100px;
  font-weight: 600;
  color: #374151;
}

.param-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.param-input-small {
  width: 80px;
  padding: 6px 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  text-align: center;
  font-size: 0.8rem;
}

.remove-param-btn {
  background: #ef4444;
  color: white;
  border: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  cursor: pointer;
  font-size: 0.8rem;
}

.add-param-btn {
  background: #10b981;
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
}

.optimization-result {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border-radius: 8px;
  padding: 20px;
  border: 1px solid #f59e0b;
}

.best-params h4,
.best-result h4 {
  margin: 0 0 12px 0;
  color: #92400e;
}

.params-display {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}

.param-display {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 4px;
}

.result-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 4px;
}

/* 归因分析样式 */
.attribution-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
  margin-bottom: 24px;
}

.summary-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e5e7eb;
}

.summary-card h4 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.attribution-breakdown,
.risk-attribution {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.breakdown-item,
.risk-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 0;
  border-bottom: 1px solid #f3f4f6;
}

.breakdown-item:last-child,
.risk-item:last-child {
  border-bottom: none;
}

.breakdown-label,
.risk-label {
  font-weight: 500;
  color: #374151;
}

.breakdown-value,
.risk-value {
  font-weight: 600;
  color: #1f2937;
}

.breakdown-value.positive {
  color: #059669;
}

.breakdown-value.negative {
  color: #dc2626;
}

.periodic-attribution h4 {
  margin: 0 0 16px 0;
  color: #1f2937;
}

.periods-table {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  overflow: hidden;
}

.periods-table .table-header {
  grid-template-columns: 100px 80px 80px 80px 80px 80px;
}

.periods-table .table-row {
  grid-template-columns: 100px 80px 80px 80px 80px 80px;
}

/* 占位符样式 */
.returns-chart {
  height: 300px;
  background: #f8f9fa;
  border-radius: 8px;
  border: 2px dashed #ddd;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 20px;
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

/* 响应式设计 */
@media (max-width: 768px) {
  .backtest-header h1 {
    font-size: 2rem;
  }

  .backtest-overview {
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

  .param-row {
    grid-template-columns: 1fr;
  }

  .metric-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .result-summary {
    grid-template-columns: 1fr;
  }

  .scenario-summary {
    grid-template-columns: 1fr;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .interval-row {
    grid-template-columns: 1fr;
  }

  .params-display {
    grid-template-columns: 1fr;
  }

  .result-metrics {
    grid-template-columns: 1fr;
  }

  .attribution-summary {
    grid-template-columns: 1fr;
  }

  .table-header,
  .table-row {
    grid-template-columns: 1fr !important;
    gap: 4px !important;
  }

  .table-row {
    padding: 8px 4px;
    font-size: 0.7rem;
  }
}
</style>
