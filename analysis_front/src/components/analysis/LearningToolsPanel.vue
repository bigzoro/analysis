<template>
  <div class="learning-tools-panel">
    <div class="panel-header">
      <h3>📚 AI学习工具</h3>
      <p>通过历史数据理解AI决策逻辑，提升投资决策质量</p>
    </div>

    <!-- 推荐选择器 -->
    <div class="recommendation-selector">
      <div class="selector-controls">
        <div class="form-group">
          <label>选择要分析的推荐：</label>
          <select v-model="selectedRecommendationId" @change="loadRecommendationAnalysis">
            <option value="">请选择推荐...</option>
            <option
              v-for="rec in availableRecommendations"
              :key="rec.id"
              :value="rec.id"
            >
              {{ rec.symbol }} - {{ formatDate(rec.date) }} - {{ rec.action }} ({{ rec.score }}分)
            </option>
          </select>
        </div>

        <div class="form-group">
          <label>分析深度：</label>
          <select v-model="analysisDepth">
            <option value="basic">基础分析</option>
            <option value="detailed">详细分析</option>
            <option value="comprehensive">全面分析</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 决策分析结果 -->
    <div v-if="analysisResult" class="analysis-result">
      <!-- 整体评估 -->
      <div class="overall-assessment">
        <div class="assessment-header">
          <h4>🎯 决策评估</h4>
          <div class="confidence-indicator">
            <div class="confidence-gauge">
              <div class="confidence-fill" :style="{ width: analysisResult.confidence * 100 + '%' }"></div>
            </div>
            <div class="confidence-text">
              <span class="confidence-value">{{ (analysisResult.confidence * 100).toFixed(1) }}%</span>
              <span class="confidence-label">置信度</span>
            </div>
          </div>
        </div>

        <div class="assessment-content">
          <div class="primary-reason">
            <h5>主要决策理由</h5>
            <div class="reason-card primary">
              <div class="reason-icon">🎯</div>
              <div class="reason-content">
                <h6>{{ analysisResult.primaryReason }}</h6>
                <p>{{ analysisResult.primaryExplanation }}</p>
              </div>
            </div>
          </div>

          <div class="decision-factors">
            <h5>决策因素权重</h5>
            <div class="factors-chart">
              <div
                v-for="factor in analysisResult.factors"
                :key="factor.name"
                class="factor-item"
              >
                <div class="factor-header">
                  <span class="factor-name">{{ factor.name }}</span>
                  <span class="factor-weight">{{ (factor.weight * 100).toFixed(0) }}%</span>
                </div>
                <div class="factor-bar">
                  <div
                    class="factor-fill"
                    :style="{
                      width: factor.weight * 100 + '%',
                      backgroundColor: getFactorColor(factor.impact)
                    }"
                  ></div>
                </div>
                <div class="factor-explanation">{{ factor.explanation }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 技术指标分析 -->
      <div class="technical-analysis">
        <h4>📊 技术指标深度分析</h4>
        <div class="indicators-grid">
          <div
            v-for="indicator in analysisResult.technicalAnalysis.indicators"
            :key="indicator.name"
            class="indicator-card"
            :class="indicator.signal"
          >
            <div class="indicator-header">
              <h5>{{ indicator.name }}</h5>
              <span class="signal-badge" :class="indicator.signal">
                {{ getSignalLabel(indicator.signal) }}
              </span>
            </div>

            <div class="indicator-details">
              <div class="indicator-values">
                <div class="value-item">
                  <span class="value-label">当前值</span>
                  <span class="value-number">{{ indicator.currentValue }}</span>
                </div>
                <div class="value-item">
                  <span class="value-label">参考值</span>
                  <span class="value-number">{{ indicator.referenceValue }}</span>
                </div>
              </div>

              <div class="indicator-explanation">
                {{ indicator.explanation }}
              </div>

              <div class="indicator-strength">
                <div class="strength-meter">
                  <div class="strength-fill" :style="{ width: indicator.strength * 100 + '%' }"></div>
                </div>
                <span class="strength-label">{{ (indicator.strength * 100).toFixed(0) }}% 强度</span>
              </div>
            </div>

            <div class="indicator-visualization">
              <div class="mini-chart">
                <!-- 这里可以添加小的指标趋势图 -->
                <div class="chart-placeholder">
                  <span>{{ indicator.name }}趋势</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 市场环境分析 -->
      <div class="market-analysis">
        <h4>🌍 市场环境分析</h4>
        <div class="market-context">
          <div class="market-overview">
            <div class="market-metrics">
              <div class="metric-item">
                <span class="metric-label">市场趋势</span>
                <span class="metric-value" :class="analysisResult.marketCondition.trend">
                  {{ getTrendLabel(analysisResult.marketCondition.trend) }}
                </span>
              </div>
              <div class="metric-item">
                <span class="metric-label">波动率</span>
                <span class="metric-value" :class="getVolatilityLevel(analysisResult.marketCondition.volatility)">
                  {{ (analysisResult.marketCondition.volatility * 100).toFixed(1) }}%
                </span>
              </div>
              <div class="metric-item">
                <span class="metric-label">市场情绪</span>
                <span class="metric-value" :class="analysisResult.marketCondition.sentiment">
                  {{ getSentimentLabel(analysisResult.marketCondition.sentiment) }}
                </span>
              </div>
            </div>
          </div>

          <div class="market-factors">
            <h5>关键市场因素</h5>
            <div class="factors-list">
              <div
                v-for="factor in analysisResult.marketCondition.keyFactors"
                :key="factor.name"
                class="factor-tag"
                :class="factor.impact"
              >
                {{ factor.name }}
                <span class="factor-impact">{{ getImpactLabel(factor.impact) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 风险评估 -->
      <div class="risk-assessment">
        <h4>⚠️ 风险评估</h4>
        <div class="risk-metrics">
          <div class="risk-gauge-item">
            <h5>市场风险</h5>
            <div class="gauge-container">
              <div class="gauge">
                <div class="gauge-fill market-risk" :style="{ width: analysisResult.riskAnalysis.marketRisk * 100 + '%' }"></div>
              </div>
              <span class="gauge-value">{{ (analysisResult.riskAnalysis.marketRisk * 100).toFixed(0) }}%</span>
            </div>
            <p>{{ getMarketRiskDesc(analysisResult.riskAnalysis.marketRisk) }}</p>
          </div>

          <div class="risk-gauge-item">
            <h5>波动风险</h5>
            <div class="gauge-container">
              <div class="gauge">
                <div class="gauge-fill volatility-risk" :style="{ width: analysisResult.riskAnalysis.volatilityRisk * 100 + '%' }"></div>
              </div>
              <span class="gauge-value">{{ (analysisResult.riskAnalysis.volatilityRisk * 100).toFixed(0) }}%</span>
            </div>
            <p>{{ getVolatilityRiskDesc(analysisResult.riskAnalysis.volatilityRisk) }}</p>
          </div>

          <div class="risk-gauge-item">
            <h5>执行风险</h5>
            <div class="gauge-container">
              <div class="gauge">
                <div class="gauge-fill execution-risk" :style="{ width: analysisResult.riskAnalysis.executionRisk * 100 + '%' }"></div>
              </div>
              <span class="gauge-value">{{ (analysisResult.riskAnalysis.executionRisk * 100).toFixed(0) }}%</span>
            </div>
            <p>{{ getExecutionRiskDesc(analysisResult.riskAnalysis.executionRisk) }}</p>
          </div>
        </div>

        <div class="risk-recommendations">
          <h5>风险管理建议</h5>
          <ul class="recommendations-list">
            <li v-for="rec in analysisResult.riskAnalysis.recommendations" :key="rec.id" :class="rec.priority">
              <span class="rec-priority">{{ rec.priority === 'high' ? '🔴' : rec.priority === 'medium' ? '🟡' : '🟢' }}</span>
              {{ rec.text }}
            </li>
          </ul>
        </div>
      </div>

      <!-- 替代方案比较 -->
      <div class="alternative-comparison">
        <h4>🔄 替代方案比较</h4>
        <div class="comparison-table">
          <table>
            <thead>
              <tr>
                <th>方案</th>
                <th>预期收益</th>
                <th>风险水平</th>
                <th>置信度</th>
                <th>优势</th>
                <th>劣势</th>
              </tr>
            </thead>
            <tbody>
              <tr class="current-recommendation">
                <td><strong>当前AI推荐</strong></td>
                <td class="expected-return">{{ (analysisResult.expectedReturn * 100).toFixed(1) }}%</td>
                <td><span class="risk-level" :class="analysisResult.riskLevel">{{ getRiskLabel(analysisResult.riskLevel) }}</span></td>
                <td>{{ (analysisResult.confidence * 100).toFixed(0) }}%</td>
                <td>{{ analysisResult.advantages.join(', ') }}</td>
                <td>{{ analysisResult.disadvantages.join(', ') }}</td>
              </tr>

              <tr v-for="alt in analysisResult.alternatives" :key="alt.id">
                <td>{{ alt.name }}</td>
                <td class="expected-return">{{ (alt.expectedReturn * 100).toFixed(1) }}%</td>
                <td><span class="risk-level" :class="alt.riskLevel">{{ getRiskLabel(alt.riskLevel) }}</span></td>
                <td>{{ (alt.confidence * 100).toFixed(0) }}%</td>
                <td>{{ alt.advantages.join(', ') }}</td>
                <td>{{ alt.disadvantages.join(', ') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- 学习建议 -->
      <div class="learning-suggestions">
        <h4>🎓 学习建议</h4>
        <div class="suggestions-grid">
          <div
            v-for="suggestion in analysisResult.learningSuggestions"
            :key="suggestion.id"
            class="suggestion-card"
          >
            <div class="suggestion-icon">{{ suggestion.icon }}</div>
            <div class="suggestion-content">
              <h5>{{ suggestion.title }}</h5>
              <p>{{ suggestion.description }}</p>
              <div class="suggestion-actions">
                <button @click="exploreConcept(suggestion.concept)">深入学习</button>
                <button @click="practiceScenario(suggestion.scenario)">实践应用</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <div class="loading-spinner"></div>
      <div class="loading-text">正在分析AI决策逻辑...</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LearningToolsPanel',
  props: {
    recommendations: {
      type: Array,
      default: () => []
    }
  },
  emits: ['explanation-requested'],
  data() {
    return {
      selectedRecommendationId: '',
      analysisDepth: 'detailed',
      analysisResult: null,
      loading: false,
      availableRecommendations: []
    }
  },
  watch: {
    recommendations: {
      handler(newRecs) {
        this.availableRecommendations = newRecs.map(rec => ({
          id: rec.id,
          symbol: rec.symbol,
          date: rec.recommendedAt || rec.date,
          action: this.getActionLabel(rec),
          score: rec.totalScore || rec.overall_score || 0
        }))
      },
      immediate: true
    }
  },
  methods: {
    async loadRecommendationAnalysis() {
      if (!this.selectedRecommendationId) {
        this.analysisResult = null
        return
      }

      this.loading = true

      try {
        // 这里应该调用后端API获取详细的决策分析
        // 目前使用模拟数据
        this.analysisResult = await this.generateMockAnalysis(this.selectedRecommendationId)

        this.$emit('explanation-requested', {
          recommendationId: this.selectedRecommendationId,
          analysis: this.analysisResult
        })

      } catch (error) {
        console.error('加载推荐分析失败:', error)
      } finally {
        this.loading = false
      }
    },

    async generateMockAnalysis(recommendationId) {
      // 模拟详细的AI决策分析
      const selectedRec = this.availableRecommendations.find(r => r.id === recommendationId)

      return {
        confidence: 0.78,
        primaryReason: "RSI指标显示超卖信号，结合MACD金叉，市场情绪相对乐观",
        primaryExplanation: "在当前震荡向上的市场环境中，技术指标显示出较强的买入信号。RSI从超卖区回升，MACD形成金叉，这些都是经典的技术买入信号。",

        factors: [
          {
            name: "RSI超卖信号",
            weight: 0.25,
            impact: "positive",
            explanation: "RSI指标从28回升至45，脱离超卖区间，显示下跌动能减弱"
          },
          {
            name: "MACD金叉",
            weight: 0.20,
            impact: "positive",
            explanation: "MACD快线从下方穿越慢线，形成看涨交叉信号"
          },
          {
            name: "布林带支撑",
            weight: 0.18,
            impact: "positive",
            explanation: "价格触及布林带下轨后获得支撑，显示较强反弹动能"
          },
          {
            name: "市场情绪",
            weight: 0.15,
            impact: "positive",
            explanation: "恐慌指数较低，投资者情绪相对乐观"
          },
          {
            name: "成交量放大",
            weight: 0.12,
            impact: "positive",
            explanation: "成交量较前日有所放大，显示资金流入迹象"
          },
          {
            name: "高波动风险",
            weight: 0.10,
            impact: "negative",
            explanation: "近期波动率较高，可能增加执行难度"
          }
        ],

        technicalAnalysis: {
          indicators: [
            {
              name: "RSI",
              currentValue: "45.2",
              referenceValue: "30-70",
              signal: "bullish",
              explanation: "RSI从超卖区回升至中性区间，显示上涨动能增强",
              strength: 0.75
            },
            {
              name: "MACD",
              currentValue: "快线上穿慢线",
              referenceValue: "金叉信号",
              signal: "bullish",
              explanation: "MACD快线从下方穿越慢线，形成经典的金叉买入信号",
              strength: 0.80
            },
            {
              name: "布林带",
              currentValue: "触及下轨支撑",
              referenceValue: "下轨支撑",
              signal: "bullish",
              explanation: "价格触及布林带下轨后获得支撑，显示较强反弹动能",
              strength: 0.70
            },
            {
              name: "移动平均线",
              currentValue: "MA5上穿MA20",
              referenceValue: "金叉形态",
              signal: "bullish",
              explanation: "短期均线上穿中期均线，形成多头排列",
              strength: 0.65
            }
          ]
        },

        marketCondition: {
          trend: "sideways_up",
          volatility: 0.23,
          sentiment: "optimistic",
          keyFactors: [
            { name: "美联储政策", impact: "positive" },
            { name: "比特币减半", impact: "positive" },
            { name: "机构入场", impact: "positive" },
            { name: "地缘政治风险", impact: "negative" }
          ]
        },

        riskAnalysis: {
          marketRisk: 0.35,
          volatilityRisk: 0.42,
          executionRisk: 0.28,
          recommendations: [
            { id: 1, text: "设置5%的止损位以控制风险", priority: "high" },
            { id: 2, text: "分批建仓，建议分3次完成", priority: "medium" },
            { id: 3, text: "密切关注MACD指标变化", priority: "medium" },
            { id: 4, text: "如突破阻力位可考虑加仓", priority: "low" }
          ]
        },

        expectedReturn: 0.15,
        riskLevel: "medium",
        advantages: ["技术指标配合良好", "市场情绪相对乐观", "有较好的支撑位"],
        disadvantages: ["波动率较高", "市场环境不确定性较大"],

        alternatives: [
          {
            id: 1,
            name: "保守观望",
            expectedReturn: 0.02,
            riskLevel: "low",
            confidence: 0.65,
            advantages: ["风险极低", "资金安全"],
            disadvantages: ["可能错过机会", "收益有限"]
          },
          {
            id: 2,
            name: "激进全仓",
            expectedReturn: 0.28,
            riskLevel: "high",
            confidence: 0.45,
            advantages: ["潜在收益高"],
            disadvantages: ["风险极大", "波动剧烈"]
          }
        ],

        learningSuggestions: [
          {
            id: 1,
            icon: "📈",
            title: "技术指标组合应用",
            description: "学习如何综合多个技术指标形成决策，避免单一指标的局限性",
            concept: "technical_analysis",
            scenario: "rsi_macd_combination"
          },
          {
            id: 2,
            icon: "🎯",
            title: "市场时机把握",
            description: "理解在不同市场环境下，如何把握最佳的买入和卖出时机",
            concept: "market_timing",
            scenario: "oversold_reversal"
          },
          {
            id: 3,
            icon: "⚖️",
            title: "风险收益平衡",
            description: "掌握风险控制的重要性，理解高收益必然伴随高风险",
            concept: "risk_reward",
            scenario: "position_sizing"
          }
        ]
      }
    },

    exploreConcept(concept) {
      console.log('探索概念:', concept)
      // 这里可以跳转到学习资料页面
    },

    practiceScenario(scenario) {
      console.log('练习场景:', scenario)
      // 这里可以跳转到模拟练习页面
    },

    getActionLabel(rec) {
      if (rec.strategyType === 'LONG' || rec.action === 'buy') return '买入'
      if (rec.strategyType === 'SHORT' || rec.action === 'sell') return '卖出'
      return '持有'
    },

    // 辅助方法
    getFactorColor(impact) {
      const colors = {
        positive: '#10b981',
        negative: '#ef4444',
        neutral: '#6b7280'
      }
      return colors[impact] || colors.neutral
    },

    getSignalLabel(signal) {
      const labels = {
        bullish: '看涨',
        bearish: '看跌',
        neutral: '中性'
      }
      return labels[signal] || signal
    },

    getTrendLabel(trend) {
      const labels = {
        bullish: '上涨',
        bearish: '下跌',
        sideways: '震荡',
        sideways_up: '震荡向上',
        sideways_down: '震荡向下'
      }
      return labels[trend] || trend
    },

    getVolatilityLevel(volatility) {
      if (volatility < 0.15) return 'low'
      if (volatility < 0.25) return 'medium'
      return 'high'
    },

    getSentimentLabel(sentiment) {
      const labels = {
        optimistic: '乐观',
        pessimistic: '悲观',
        neutral: '中性'
      }
      return labels[sentiment] || sentiment
    },

    getImpactLabel(impact) {
      const labels = {
        high: '高',
        medium: '中',
        low: '低'
      }
      return labels[impact] || impact
    },

    getRiskLabel(level) {
      const labels = {
        low: '低风险',
        medium: '中风险',
        high: '高风险'
      }
      return labels[level] || level
    },

    getMarketRiskDesc(risk) {
      if (risk < 0.3) return '市场风险较低，适合当前仓位'
      if (risk < 0.6) return '市场风险中等，需谨慎操作'
      return '市场风险较高，建议减少仓位'
    },

    getVolatilityRiskDesc(risk) {
      if (risk < 0.3) return '波动风险可控，执行相对容易'
      if (risk < 0.6) return '波动风险中等，可能影响执行'
      return '波动风险较高，执行难度较大'
    },

    getExecutionRiskDesc(risk) {
      if (risk < 0.3) return '执行风险低，信号清晰'
      if (risk < 0.6) return '执行风险中等，需把握时机'
      return '执行风险较高，可能出现滑点'
    },

    formatDate(date) {
      return new Date(date).toLocaleDateString('zh-CN')
    }
  }
}
</script>

<style scoped>
.learning-tools-panel {
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

.recommendation-selector {
  background: #f8fafc;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
}

.selector-controls {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-weight: 600;
  color: #374151;
  font-size: 14px;
}

.form-group select {
  padding: 10px 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  font-size: 14px;
  background: white;
  cursor: pointer;
}

.form-group select:focus {
  outline: none;
  border-color: #3b82f6;
}

.analysis-result {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.overall-assessment {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.assessment-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.assessment-header h4 {
  margin: 0;
  color: #1f2937;
}

.confidence-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
}

.confidence-gauge {
  width: 120px;
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}

.confidence-fill {
  height: 100%;
  background: linear-gradient(90deg, #ef4444 0%, #f59e0b 50%, #10b981 100%);
  transition: width 0.3s ease;
}

.confidence-text {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.confidence-value {
  font-size: 1.2rem;
  font-weight: 700;
  color: #1f2937;
}

.confidence-label {
  font-size: 12px;
  color: #6b7280;
}

.primary-reason {
  margin-bottom: 24px;
}

.primary-reason h5 {
  margin: 0 0 12px 0;
  color: #374151;
}

.reason-card {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  padding: 16px;
  border-radius: 8px;
}

.reason-card.primary {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  border: 1px solid #bfdbfe;
}

.reason-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.reason-content h6 {
  margin: 0 0 8px 0;
  color: #1f2937;
  font-size: 1rem;
}

.reason-content p {
  margin: 0;
  color: #6b7280;
  line-height: 1.5;
}

.decision-factors h5 {
  margin: 0 0 16px 0;
  color: #374151;
}

.factors-chart {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.factor-item {
  padding: 12px;
  background: #f9fafb;
  border-radius: 6px;
}

.factor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.factor-name {
  font-weight: 600;
  color: #374151;
}

.factor-weight {
  font-size: 14px;
  color: #6b7280;
}

.factor-bar {
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.factor-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.factor-explanation {
  font-size: 14px;
  color: #6b7280;
}

.technical-analysis, .market-analysis, .risk-assessment, .alternative-comparison, .learning-suggestions {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.technical-analysis h4, .market-analysis h4, .risk-assessment h4, .alternative-comparison h4, .learning-suggestions h4 {
  margin: 0 0 20px 0;
  color: #1f2937;
}

.indicators-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
}

.indicator-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  transition: all 0.2s ease;
}

.indicator-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.indicator-card.bullish {
  border-left: 4px solid #10b981;
}

.indicator-card.bearish {
  border-left: 4px solid #ef4444;
}

.indicator-card.neutral {
  border-left: 4px solid #6b7280;
}

.indicator-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.indicator-header h5 {
  margin: 0;
  color: #1f2937;
}

.signal-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.signal-badge.bullish {
  background: #dcfce7;
  color: #166534;
}

.signal-badge.bearish {
  background: #fee2e2;
  color: #991b1b;
}

.signal-badge.neutral {
  background: #f3f4f6;
  color: #374151;
}

.indicator-values {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
}

.value-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.value-label {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}

.value-number {
  font-size: 1.1rem;
  font-weight: 600;
  color: #1f2937;
}

.indicator-explanation {
  color: #6b7280;
  font-size: 14px;
  line-height: 1.4;
  margin-bottom: 12px;
}

.indicator-strength {
  display: flex;
  align-items: center;
  gap: 8px;
}

.strength-meter {
  flex: 1;
  height: 6px;
  background: #e5e7eb;
  border-radius: 3px;
  overflow: hidden;
}

.strength-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981 0%, #f59e0b 50%, #ef4444 100%);
}

.strength-label {
  font-size: 12px;
  color: #6b7280;
  white-space: nowrap;
}

.indicator-visualization {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.mini-chart {
  height: 60px;
  background: #f9fafb;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chart-placeholder {
  color: #9ca3af;
  font-size: 12px;
}

.market-context {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.market-metrics {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
}

.metric-item:last-child {
  border-bottom: none;
}

.metric-label {
  font-weight: 500;
  color: #374151;
}

.metric-value {
  font-weight: 600;
}

.metric-value.bullish {
  color: #10b981;
}

.metric-value.bearish {
  color: #ef4444;
}

.metric-value.low {
  color: #10b981;
}

.metric-value.medium {
  color: #f59e0b;
}

.metric-value.high {
  color: #ef4444;
}

.metric-value.optimistic {
  color: #10b981;
}

.metric-value.pessimistic {
  color: #ef4444;
}

.market-factors h5 {
  margin: 0 0 12px 0;
  color: #374151;
}

.factors-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.factor-tag {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid;
}

.factor-tag.high {
  background: #fef2f2;
  border-color: #fecaca;
  color: #dc2626;
}

.factor-tag.medium {
  background: #fefce8;
  border-color: #fde68a;
  color: #d97706;
}

.factor-tag.low {
  background: #f0fdf4;
  border-color: #bbf7d0;
  color: #16a34a;
}

.factor-impact {
  margin-left: 4px;
  opacity: 0.8;
}

.risk-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.risk-gauge-item {
  text-align: center;
}

.risk-gauge-item h5 {
  margin: 0 0 12px 0;
  color: #374151;
}

.gauge-container {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.gauge {
  flex: 1;
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}

.gauge-fill {
  height: 100%;
  transition: width 0.3s ease;
}

.gauge-fill.market-risk {
  background: linear-gradient(90deg, #10b981 0%, #f59e0b 50%, #ef4444 100%);
}

.gauge-fill.volatility-risk {
  background: linear-gradient(90deg, #3b82f6 0%, #8b5cf6 100%);
}

.gauge-fill.execution-risk {
  background: linear-gradient(90deg, #06b6d4 0%, #0891b2 100%);
}

.gauge-value {
  font-weight: 600;
  color: #1f2937;
  min-width: 40px;
  text-align: right;
}

.risk-gauge-item p {
  margin: 0;
  font-size: 14px;
  color: #6b7280;
  line-height: 1.4;
}

.risk-recommendations h5 {
  margin: 0 0 12px 0;
  color: #374151;
}

.recommendations-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.recommendations-list li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  background: #f9fafb;
  border-radius: 6px;
  font-size: 14px;
  color: #374151;
}

.recommendations-list li.high {
  background: #fef2f2;
  border-left: 3px solid #ef4444;
}

.recommendations-list li.medium {
  background: #fefce8;
  border-left: 3px solid #f59e0b;
}

.recommendations-list li.low {
  background: #f0fdf4;
  border-left: 3px solid #10b981;
}

.rec-priority {
  font-size: 14px;
  flex-shrink: 0;
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
  border-bottom: 1px solid #e5e7eb;
}

.comparison-table th {
  background: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.current-recommendation {
  background: #eff6ff;
}

.expected-return {
  font-weight: 600;
}

.risk-level {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.risk-level.low {
  background: #dcfce7;
  color: #166534;
}

.risk-level.medium {
  background: #fef3c7;
  color: #92400e;
}

.risk-level.high {
  background: #fee2e2;
  color: #991b1b;
}

.learning-suggestions .suggestions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
}

.suggestion-card {
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.suggestion-icon {
  font-size: 1.5rem;
  flex-shrink: 0;
}

.suggestion-content {
  flex: 1;
}

.suggestion-content h5 {
  margin: 0 0 8px 0;
  color: #1f2937;
}

.suggestion-content p {
  margin: 0 0 12px 0;
  color: #6b7280;
  font-size: 14px;
  line-height: 1.4;
}

.suggestion-actions {
  display: flex;
  gap: 8px;
}

.suggestion-actions button {
  padding: 6px 12px;
  border: 1px solid #d1d5db;
  background: white;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.suggestion-actions button:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #e5e7eb;
  border-top: 4px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-text {
  color: #6b7280;
  font-size: 16px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .selector-controls {
    grid-template-columns: 1fr;
  }

  .indicators-grid {
    grid-template-columns: 1fr;
  }

  .market-context {
    grid-template-columns: 1fr;
  }

  .risk-metrics {
    grid-template-columns: 1fr;
  }

  .suggestions-grid {
    grid-template-columns: 1fr;
  }

  .assessment-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
}
</style>
