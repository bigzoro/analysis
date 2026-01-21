<template>
  <div class="ai-analysis-dashboard">
    <!-- 页面头部 -->
    <section class="panel header-panel">
      <div class="row">
        <div class="header-content">
          <h1>🤖 AI推荐分析仪表板</h1>
          <p class="subtitle">深度分析AI推荐的策略表现、历史趋势、决策逻辑和风险特征</p>
        </div>
        <div class="header-actions">
          <button @click="exportReport" class="export-btn">
            📊 导出报告
          </button>
          <button @click="refreshData" :disabled="loading" class="refresh-btn">
            {{ loading ? '刷新中...' : '🔄 刷新数据' }}
          </button>
        </div>
      </div>
    </section>

    <!-- 功能导航 -->
    <section class="panel nav-panel">
      <div class="nav-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          @click="activeTab = tab.key"
          :class="{ active: activeTab === tab.key }"
          class="nav-tab"
        >
          <span class="tab-icon">{{ tab.icon }}</span>
          <span class="tab-label">{{ tab.label }}</span>
        </button>
      </div>
    </section>

    <!-- 策略回测面板 -->
    <div v-if="activeTab === 'backtest'" class="analysis-panel">
      <StrategyBacktestPanel
        :symbols="symbols"
        :selectedDate="selectedDate"
        @backtest-complete="handleBacktestComplete"
      />
    </div>

    <!-- 回测记录面板 -->
    <div v-if="activeTab === 'backtest-records'" class="analysis-panel">
      <BacktestRecordsPanel
        :symbols="symbols"
        @record-selected="handleRecordSelected"
      />
    </div>

    <!-- 历史分析面板 -->
    <div v-if="activeTab === 'historical'" class="analysis-panel">
      <HistoricalAnalysisPanel
        :symbols="symbols"
        :selectedDate="selectedDate"
        @period-selected="handlePeriodSelected"
      />
    </div>

    <!-- 学习工具面板 -->
    <div v-if="activeTab === 'learning'" class="analysis-panel">
      <LearningToolsPanel
        :recommendations="recommendations"
        @explanation-requested="handleExplanationRequest"
      />
    </div>

    <!-- 风险评估面板 -->
    <div v-if="activeTab === 'risk'" class="analysis-panel">
      <RiskAssessmentPanel
        :symbols="symbols"
        :selectedDate="selectedDate"
        @risk-alert="handleRiskAlert"
      />
    </div>
  </div>
</template>

<script>
import StrategyBacktestPanel from '../components/analysis/StrategyBacktestPanel.vue'
import BacktestRecordsPanel from '../components/analysis/BacktestRecordsPanel.vue'
import HistoricalAnalysisPanel from '../components/analysis/HistoricalAnalysisPanel.vue'
import LearningToolsPanel from '../components/analysis/LearningToolsPanel.vue'
import RiskAssessmentPanel from '../components/analysis/RiskAssessmentPanel.vue'
import { useRoute } from 'vue-router'

export default {
  name: 'AIAnalysisDashboard',
  components: {
    StrategyBacktestPanel,
    BacktestRecordsPanel,
    HistoricalAnalysisPanel,
    LearningToolsPanel,
    RiskAssessmentPanel
  },
  data() {
    return {
      activeTab: 'backtest',
      loading: false,
      symbols: [],
      selectedDate: null,
      recommendations: [],
      tabs: [
        {
          key: 'backtest',
          icon: '📈',
          label: '策略回测'
        },
        {
          key: 'backtest-records',
          icon: '📋',
          label: '回测记录'
        },
        {
          key: 'historical',
          icon: '🎯',
          label: '历史分析'
        },
        {
          key: 'learning',
          icon: '📚',
          label: '学习工具'
        },
        {
          key: 'risk',
          icon: '🔍',
          label: '风险评估'
        }
      ]
    }
  },
  mounted() {
    this.parseRouteParams()
    this.loadInitialData()
  },
  watch: {
    activeTab() {
      this.handleTabChange()
    }
  },
  methods: {
    parseRouteParams() {
      const route = useRoute()
      this.symbols = route.query.symbols ? route.query.symbols.split(',') : ['BTC']
      this.selectedDate = route.query.date || new Date().toISOString().split('T')[0]
    },

    async loadInitialData() {
      this.loading = true
      try {
        // 加载推荐数据用于分析
        const recs = await this.loadRecommendations()
        this.recommendations = recs
      } catch (error) {
        console.error('加载初始数据失败:', error)
      } finally {
        this.loading = false
      }
    },

    async loadRecommendations() {
      // 这里可以调用现有的推荐API
      // 暂时返回示例数据
      return []
    },

    exportReport() {
      // 导出分析报告
      console.log('导出分析报告')
    },

    async refreshData() {
      this.loading = true
      await this.loadInitialData()
      this.loading = false
    },

    handleBacktestComplete(result) {
      console.log('回测完成:', result)
    },

    handleRecordSelected(record) {
      console.log('选择回测记录:', record)
      // 可以在这里跳转到详情页面或打开模态框
    },

    handleTabChange() {
      // 当切换标签页时，可以在这里处理一些清理逻辑
      console.log('切换到标签页:', this.activeTab)
    },

    handlePeriodSelected(period) {
      console.log('选择时间段:', period)
    },

    handleExplanationRequest(recommendation) {
      console.log('请求解释:', recommendation)
    },

    handleRiskAlert(alert) {
      console.log('风险警报:', alert)
    }
  }
}
</script>

<style scoped>
.ai-analysis-dashboard {
  max-width: 1400px;
  margin: 0 auto;
  padding: 20px;
  background: #f8fafc;
  min-height: 100vh;
}

.header-panel {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  margin-bottom: 20px;
}

.header-content h1 {
  margin: 0 0 8px 0;
  font-size: 2rem;
  font-weight: 700;
}

.subtitle {
  margin: 0;
  opacity: 0.9;
  font-size: 1rem;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.export-btn, .refresh-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.export-btn {
  background: rgba(255, 255, 255, 0.2);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.export-btn:hover {
  background: rgba(255, 255, 255, 0.3);
}

.refresh-btn {
  background: rgba(255, 255, 255, 0.1);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.refresh-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.2);
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.nav-panel {
  margin-bottom: 20px;
  padding: 0;
}

.nav-tabs {
  display: flex;
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.nav-tab {
  flex: 1;
  padding: 16px 20px;
  border: none;
  background: transparent;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-weight: 500;
  color: #64748b;
}

.nav-tab:hover {
  background: #f1f5f9;
}

.nav-tab.active {
  background: #3b82f6;
  color: white;
}

.tab-icon {
  font-size: 1.2rem;
}

.analysis-panel {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .ai-analysis-dashboard {
    padding: 10px;
  }

  .header-content h1 {
    font-size: 1.5rem;
  }

  .header-actions {
    flex-direction: column;
    width: 100%;
  }

  .nav-tabs {
    flex-direction: column;
  }

  .nav-tab {
    padding: 12px 16px;
  }
}
</style>
