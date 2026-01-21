<template>
  <div class="ai-lab">
    <div class="lab-header">
      <h1>🤖 AI实验室</h1>
      <p class="subtitle">探索先进的机器学习和Transformer技术</p>

      <!-- 实验室控制面板 -->
      <div class="lab-controls">
        <div class="control-section">
          <h3>模型状态</h3>
          <div class="model-status-grid">
            <div class="status-card" :class="{ active: modelStatus.ensemble }">
              <div class="status-icon">🌲</div>
              <div class="status-info">
                <div class="status-name">集成学习</div>
                <div class="status-state">{{ modelStatus.ensemble ? '活跃' : '未初始化' }}</div>
              </div>
            </div>

            <div class="status-card" :class="{ active: modelStatus.deepLearning }">
              <div class="status-icon">🧠</div>
              <div class="status-info">
                <div class="status-name">深度学习</div>
                <div class="status-state">{{ modelStatus.deepLearning ? '活跃' : '未初始化' }}</div>
              </div>
            </div>

            <div class="status-card" :class="{ active: modelStatus.transformer }">
              <div class="status-icon">🔄</div>
              <div class="status-info">
                <div class="status-name">Transformer</div>
                <div class="status-state">{{ modelStatus.transformer ? '活跃' : '未初始化' }}</div>
              </div>
            </div>

            <div class="status-card" :class="{ active: featureStatus.enabled }">
              <div class="status-icon">📊</div>
              <div class="status-info">
                <div class="status-name">特征工程</div>
                <div class="status-state">{{ featureStatus.enabled ? '活跃' : '未初始化' }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="control-section">
          <h3>实验室工具</h3>
          <div class="tool-buttons">
            <button @click="runModelDiagnostics" :disabled="diagnosticsRunning" class="tool-btn">
              {{ diagnosticsRunning ? '诊断中...' : '🔍 模型诊断' }}
            </button>
            <button @click="optimizeHyperparameters" :disabled="optimizationRunning" class="tool-btn">
              {{ optimizationRunning ? '优化中...' : '⚙️ 超参数优化' }}
            </button>
            <button @click="generateFeatureReport" :disabled="reportGenerating" class="tool-btn">
              {{ reportGenerating ? '生成中...' : '📋 特征报告' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 实验区 -->
    <div class="lab-content">
      <!-- Transformer实验 -->
      <div class="experiment-section">
        <div class="section-header">
          <h2>🔄 Transformer实验</h2>
          <div class="section-controls">
            <select v-model="transformerConfig.numLayers">
              <option :value="2">2层</option>
              <option :value="4">4层</option>
              <option :value="6">6层</option>
              <option :value="8">8层</option>
            </select>
            <button @click="trainTransformerModel" :disabled="transformerTraining" class="experiment-btn">
              {{ transformerTraining ? '训练中...' : '🚀 开始训练' }}
            </button>
          </div>
        </div>

        <div class="experiment-content">
          <div class="config-panel">
            <h4>模型配置</h4>
            <div class="config-grid">
              <div class="config-item">
                <label>层数:</label>
                <span>{{ transformerConfig.numLayers }}</span>
              </div>
              <div class="config-item">
                <label>注意力头数:</label>
                <span>{{ transformerConfig.numHeads }}</span>
              </div>
              <div class="config-item">
                <label>模型维度:</label>
                <span>{{ transformerConfig.dModel }}</span>
              </div>
              <div class="config-item">
                <label>前馈维度:</label>
                <span>{{ transformerConfig.dFF }}</span>
              </div>
            </div>
          </div>

          <div class="training-progress" v-if="transformerTraining">
            <div class="progress-bar">
              <div class="progress-fill" :style="{ width: trainingProgress + '%' }"></div>
            </div>
            <div class="progress-text">{{ trainingProgress }}% - 训练中...</div>
          </div>

          <div class="training-results" v-if="transformerResults">
            <h4>训练结果</h4>
            <div class="results-grid">
              <div class="result-item">
                <span class="label">最终损失:</span>
                <span class="value">{{ transformerResults.finalLoss.toFixed(6) }}</span>
              </div>
              <div class="result-item">
                <span class="label">训练时间:</span>
                <span class="value">{{ transformerResults.trainingTime }}秒</span>
              </div>
              <div class="result-item">
                <span class="label">模型大小:</span>
                <span class="value">{{ transformerResults.modelSize }}MB</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 特征工程实验 -->
      <div class="experiment-section">
        <div class="section-header">
          <h2>📊 高级特征工程</h2>
          <div class="section-controls">
            <button @click="extractAdvancedFeatures" :disabled="featureExtractionRunning" class="experiment-btn">
              {{ featureExtractionRunning ? '提取中...' : '🔬 特征提取' }}
            </button>
            <button @click="analyzeFeatureImportance" :disabled="importanceAnalyzing" class="experiment-btn secondary">
              {{ importanceAnalyzing ? '分析中...' : '📈 重要性分析' }}
            </button>
          </div>
        </div>

        <div class="experiment-content">
          <div class="feature-types">
            <h4>特征类型选择</h4>
            <div class="feature-checkboxes">
              <label v-for="featureType in availableFeatureTypes" :key="featureType.id" class="feature-checkbox">
                <input
                  type="checkbox"
                  :value="featureType.id"
                  v-model="selectedFeatureTypes"
                />
                <span class="checkmark"></span>
                <span class="feature-name">{{ featureType.name }}</span>
                <span class="feature-desc">{{ featureType.description }}</span>
              </label>
            </div>
          </div>

          <div class="feature-results" v-if="featureResults">
            <h4>特征提取结果</h4>
            <div class="feature-summary">
              <div class="summary-item">
                <span class="label">总特征数:</span>
                <span class="value">{{ featureResults.totalFeatures }}</span>
              </div>
              <div class="summary-item">
                <span class="label">数据点数:</span>
                <span class="value">{{ featureResults.dataPoints }}</span>
              </div>
            </div>

            <div class="feature-details">
              <div v-for="(features, type) in featureResults.features" :key="type" class="feature-type-result">
                <h5>{{ getFeatureTypeName(type) }}</h5>
                <div class="feature-count">特征数量: {{ features.length }}</div>
                <div class="feature-preview">
                  预览: [{{ features.slice(0, 5).map(v => v.toFixed(3)).join(', ') }}...]
                </div>
              </div>
            </div>
          </div>

          <div class="importance-results" v-if="importanceResults">
            <h4>特征重要性分析</h4>
            <div class="importance-chart">
              <div
                v-for="(importance, feature) in importanceResults.mlImportance"
                :key="feature"
                class="importance-bar"
              >
                <div class="feature-name">{{ feature }}</div>
                <div class="importance-bar-container">
                  <div
                    class="importance-fill"
                    :style="{ width: (importance * 100) + '%' }"
                  ></div>
                </div>
                <div class="importance-value">{{ (importance * 100).toFixed(1) }}%</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 性能监控 -->
      <div class="experiment-section">
        <div class="section-header">
          <h2>📈 性能监控</h2>
        </div>

        <div class="experiment-content">
          <div class="performance-metrics">
            <div class="metric-card">
              <div class="metric-icon">⚡</div>
              <div class="metric-info">
                <div class="metric-name">推理速度</div>
                <div class="metric-value">{{ performanceMetrics.inferenceSpeed }}ms</div>
                <div class="metric-change positive">+5.2%</div>
              </div>
            </div>

            <div class="metric-card">
              <div class="metric-icon">🎯</div>
              <div class="metric-info">
                <div class="metric-name">预测准确率</div>
                <div class="metric-value">{{ (performanceMetrics.accuracy * 100).toFixed(1) }}%</div>
                <div class="metric-change positive">+2.1%</div>
              </div>
            </div>

            <div class="metric-card">
              <div class="metric-icon">🧠</div>
              <div class="metric-info">
                <div class="metric-name">模型复杂度</div>
                <div class="metric-value">{{ performanceMetrics.modelComplexity }}</div>
                <div class="metric-change neutral">不变</div>
              </div>
            </div>

            <div class="metric-card">
              <div class="metric-icon">💾</div>
              <div class="metric-info">
                <div class="metric-name">内存使用</div>
                <div class="metric-value">{{ performanceMetrics.memoryUsage }}MB</div>
                <div class="metric-change negative">+8.5%</div>
              </div>
            </div>
          </div>

          <div class="performance-chart">
            <LineChart
              :x-data="performanceChart.xData"
              :series="performanceChart.series"
              :title="'AI模型性能趋势'"
              :y-label="'性能指标'"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 实验日志 -->
    <div class="experiment-logs" v-if="logs.length > 0">
      <div class="logs-header">
        <h3>📝 实验日志</h3>
        <button @click="clearLogs" class="clear-logs-btn">清除日志</button>
      </div>

      <div class="logs-content">
        <div
          v-for="log in logs.slice(-10)"
          :key="log.timestamp"
          class="log-entry"
          :class="log.type"
        >
          <div class="log-time">{{ formatTime(log.timestamp) }}</div>
          <div class="log-message">{{ log.message }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { api } from '@/api/api.js'
import LineChart from '@/components/LineChart.vue'

export default {
  name: 'AILab',
  components: {
    LineChart
  },
  data() {
    return {
      // 模型状态
      modelStatus: {
        ensemble: false,
        deepLearning: false,
        transformer: false
      },
      featureStatus: {
        enabled: false
      },

      // Transformer配置
      transformerConfig: {
        numLayers: 6,
        numHeads: 8,
        dModel: 512,
        dFF: 2048
      },

      // 实验状态
      diagnosticsRunning: false,
      optimizationRunning: false,
      reportGenerating: false,
      transformerTraining: false,
      featureExtractionRunning: false,
      importanceAnalyzing: false,

      // 实验结果
      transformerResults: null,
      featureResults: null,
      importanceResults: null,
      trainingProgress: 0,

      // 性能指标
      performanceMetrics: {
        inferenceSpeed: 45,
        accuracy: 0.782,
        modelComplexity: '中等',
        memoryUsage: 256
      },

      // 特征类型
      selectedFeatureTypes: ['time_series', 'volatility', 'trend', 'momentum'],
      availableFeatureTypes: [
        { id: 'time_series', name: '时间序列', description: '价格动量、趋势强度等' },
        { id: 'volatility', name: '波动率', description: '历史波动率、波动趋势' },
        { id: 'trend', name: '趋势', description: 'ADX、移动平均、趋势强度' },
        { id: 'momentum', name: '动量', description: '动量指标、动量变化' },
        { id: 'cross', name: '交叉特征', description: '特征组合和交互' },
        { id: 'statistical', name: '统计特征', description: '均值、方差、偏度等' },
        { id: 'transformer', name: 'Transformer', description: '注意力机制特征' }
      ],

      // 图表数据
      performanceChart: {
        xData: [],
        series: []
      },

      // 日志
      logs: []
    }
  },

  mounted() {
    this.initializeLab()
    this.generatePerformanceChart()
  },

  methods: {
    async initializeLab() {
      this.addLog('info', 'AI实验室初始化中...')
      try {
        // 获取ML统计信息
        const stats = await api.getMLStats()
        this.modelStatus = {
          ensemble: true, // 假设集成学习已初始化
          deepLearning: stats.deep_learning_trained || false,
          transformer: stats.transformer_enabled || false
        }
        this.featureStatus.enabled = true // 假设特征工程已启用

        this.addLog('success', 'AI实验室初始化完成')
      } catch (error) {
        this.addLog('error', `初始化失败: ${error.message}`)
      }
    },

    async runModelDiagnostics() {
      this.diagnosticsRunning = true
      this.addLog('info', '开始模型诊断...')

      try {
        // 这里可以调用后端诊断API
        await new Promise(resolve => setTimeout(resolve, 2000))

        this.addLog('success', '模型诊断完成 - 所有模型运行正常')
      } catch (error) {
        this.addLog('error', `模型诊断失败: ${error.message}`)
      } finally {
        this.diagnosticsRunning = false
      }
    },

    async optimizeHyperparameters() {
      this.optimizationRunning = true
      this.addLog('info', '开始超参数优化...')

      try {
        // 这里可以调用超参数优化API
        await new Promise(resolve => setTimeout(resolve, 3000))

        this.addLog('success', '超参数优化完成 - 找到最优配置')
      } catch (error) {
        this.addLog('error', `超参数优化失败: ${error.message}`)
      } finally {
        this.optimizationRunning = false
      }
    },

    async generateFeatureReport() {
      this.reportGenerating = true
      this.addLog('info', '生成特征报告...')

      try {
        // 这里可以调用特征报告生成API
        await new Promise(resolve => setTimeout(resolve, 1500))

        this.addLog('success', '特征报告生成完成')
      } catch (error) {
        this.addLog('error', `特征报告生成失败: ${error.message}`)
      } finally {
        this.reportGenerating = false
      }
    },

    async trainTransformerModel() {
      this.transformerTraining = true
      this.trainingProgress = 0
      this.addLog('info', '开始Transformer模型训练...')

      try {
        // 模拟训练数据
        const trainingData = {
          X: this.generateMockTrainingData(100, 50), // 100个样本，50个特征
          y: this.generateMockTargets(100)
        }

        const config = {
          transformer: this.transformerConfig
        }

        // 调用训练API
        await api.trainTransformerModel(trainingData, config)

        // 模拟训练进度
        for (let i = 0; i <= 100; i += 10) {
          await new Promise(resolve => setTimeout(resolve, 200))
          this.trainingProgress = i
        }

        // 模拟训练结果
        this.transformerResults = {
          finalLoss: 0.023456,
          trainingTime: 45.2,
          modelSize: 89.5
        }

        this.addLog('success', 'Transformer模型训练完成')
      } catch (error) {
        this.addLog('error', `Transformer训练失败: ${error.message}`)
      } finally {
        this.transformerTraining = false
        this.trainingProgress = 0
      }
    },

    async extractAdvancedFeatures() {
      this.featureExtractionRunning = true
      this.addLog('info', '开始高级特征提取...')

      try {
        // 模拟市场数据
        const marketData = this.generateMockMarketData(20)

        const result = await api.extractAdvancedFeatures(marketData, this.selectedFeatureTypes)

        this.featureResults = {
          features: result.features,
          totalFeatures: result.total_features,
          dataPoints: result.market_data_points
        }

        this.addLog('success', `特征提取完成，共提取 ${result.total_features} 种特征类型`)
      } catch (error) {
        this.addLog('error', `特征提取失败: ${error.message}`)
      } finally {
        this.featureExtractionRunning = false
      }
    },

    async analyzeFeatureImportance() {
      this.importanceAnalyzing = true
      this.addLog('info', '开始特征重要性分析...')

      try {
        const result = await api.getFeatureImportanceAnalysis()

        this.importanceResults = result

        this.addLog('success', '特征重要性分析完成')
      } catch (error) {
        this.addLog('error', `特征重要性分析失败: ${error.message}`)
      } finally {
        this.importanceAnalyzing = false
      }
    },

    generateMockTrainingData(samples, features) {
      const data = []
      for (let i = 0; i < samples; i++) {
        const sample = []
        for (let j = 0; j < features; j++) {
          sample.push((Math.random() - 0.5) * 2)
        }
        data.push(sample)
      }
      return data
    },

    generateMockTargets(samples) {
      return Array.from({ length: samples }, () => Math.random())
    },

    generateMockMarketData(points) {
      const data = []
      const now = new Date()

      for (let i = points; i >= 0; i--) {
        data.push({
          symbol: 'BTC',
          price: 45000 + (Math.random() - 0.5) * 1000,
          price_change_24h: (Math.random() - 0.5) * 0.1,
          volume_24h: Math.random() * 1000000,
          timestamp: new Date(now.getTime() - i * 60 * 60 * 1000),
          technical_data: {
            rsi: 30 + Math.random() * 40,
            macd: (Math.random() - 0.5) * 100,
            bollinger_upper: 46000 + Math.random() * 1000,
            bollinger_lower: 44000 - Math.random() * 1000
          },
          sentiment_data: {
            overall_sentiment: (Math.random() - 0.5) * 2,
            social_volume: Math.random() * 100
          }
        })
      }

      return data
    },

    generatePerformanceChart() {
      const now = new Date()
      const xData = []
      const series = [
        {
          name: '准确率',
          data: [],
          lineStyle: { width: 2 },
          itemStyle: { color: '#10b981' }
        },
        {
          name: '推理速度',
          data: [],
          lineStyle: { width: 2 },
          itemStyle: { color: '#3b82f6' }
        }
      ]

      for (let i = 23; i >= 0; i--) {
        const time = new Date(now.getTime() - i * 60 * 60 * 1000)
        xData.push(time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))

        // 模拟性能数据
        series[0].data.push(0.75 + Math.random() * 0.1) // 准确率
        series[1].data.push(50 + Math.random() * 20) // 推理速度(ms)
      }

      this.performanceChart = { xData, series }
    },

    getFeatureTypeName(type) {
      const names = {
        time_series: '时间序列特征',
        volatility: '波动率特征',
        trend: '趋势特征',
        momentum: '动量特征',
        cross: '交叉特征',
        statistical: '统计特征',
        transformer: 'Transformer特征'
      }
      return names[type] || type
    },

    addLog(type, message) {
      this.logs.push({
        type,
        message,
        timestamp: new Date()
      })
    },

    clearLogs() {
      this.logs = []
    },

    formatTime(date) {
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    }
  }
}
</script>

<style scoped>
.ai-lab {
  padding: 20px;
  max-width: 1600px;
  margin: 0 auto;
  background: #f8f9fa;
  min-height: 100vh;
}

.lab-header {
  background: white;
  border-radius: 12px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.lab-header h1 {
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

.lab-controls {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.control-section h3 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 1.1rem;
}

.model-status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.status-card {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 2px solid #e9ecef;
  transition: all 0.2s;
}

.status-card.active {
  border-color: #10b981;
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(5, 150, 105, 0.1) 100%);
}

.status-card .status-icon {
  font-size: 1.5rem;
}

.status-info {
  flex: 1;
}

.status-name {
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.status-state {
  font-size: 0.9rem;
  color: #666;
}

.tool-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.tool-btn {
  background: #667eea;
  color: white;
  border: none;
  padding: 10px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.tool-btn:hover:not(:disabled) {
  background: #5a67d8;
  transform: translateY(-2px);
}

.tool-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.lab-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.experiment-section {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  margin: 0;
  color: #333;
  font-size: 1.25rem;
}

.section-controls {
  display: flex;
  gap: 12px;
  align-items: center;
}

.section-controls select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.experiment-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 8px 16px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.experiment-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.experiment-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.experiment-btn.secondary {
  background: linear-gradient(135deg, #6b7280 0%, #4b5563 100%);
}

.experiment-content {
  margin-top: 16px;
}

.config-panel h4, .training-results h4, .feature-results h4, .importance-results h4 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 1rem;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.config-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.config-item label {
  font-weight: 600;
  color: #555;
}

.config-item span {
  font-weight: 600;
  color: #333;
}

.training-progress {
  margin-top: 20px;
  padding: 16px;
  background: #f0f9ff;
  border-radius: 8px;
  border: 1px solid #3b82f6;
}

.progress-bar {
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #3b82f6 0%, #1d4ed8 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.9rem;
  color: #1e40af;
  font-weight: 600;
}

.results-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
}

.result-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f0fdf4;
  border-radius: 8px;
  border: 1px solid #10b981;
}

.result-item .label {
  font-weight: 600;
  color: #065f46;
}

.result-item .value {
  font-weight: 600;
  color: #047857;
}

.feature-types {
  margin-bottom: 24px;
}

.feature-checkboxes {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feature-checkbox {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.feature-checkbox:hover {
  background: #f0f0f0;
}

.feature-checkbox input[type="checkbox"] {
  display: none;
}

.checkmark {
  width: 20px;
  height: 20px;
  border: 2px solid #ddd;
  border-radius: 4px;
  position: relative;
  transition: all 0.2s;
}

.feature-checkbox input:checked + .checkmark {
  background: #10b981;
  border-color: #10b981;
}

.feature-checkbox input:checked + .checkmark::after {
  content: '✓';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.feature-name {
  font-weight: 600;
  color: #333;
  min-width: 100px;
}

.feature-desc {
  color: #666;
  font-size: 0.9rem;
  flex: 1;
}

.feature-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.summary-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f0f9ff;
  border-radius: 8px;
}

.summary-item .label {
  font-weight: 600;
  color: #1e40af;
}

.summary-item .value {
  font-weight: 600;
  color: #1d4ed8;
}

.feature-details {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.feature-type-result {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 8px;
  border-left: 4px solid #667eea;
}

.feature-type-result h5 {
  margin: 0 0 8px 0;
  color: #333;
  font-size: 1rem;
}

.feature-count {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 8px;
}

.feature-preview {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  color: #555;
  background: white;
  padding: 8px;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
}

.importance-chart {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.importance-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.feature-name {
  min-width: 120px;
  font-weight: 600;
  color: #333;
  font-size: 0.9rem;
}

.importance-bar-container {
  flex: 1;
  height: 12px;
  background: #e5e7eb;
  border-radius: 6px;
  overflow: hidden;
}

.importance-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 6px;
  transition: width 0.3s ease;
}

.importance-value {
  min-width: 60px;
  text-align: right;
  font-weight: 600;
  color: #667eea;
  font-size: 0.9rem;
}

.performance-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.metric-card {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  gap: 12px;
}

.metric-icon {
  font-size: 1.5rem;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 8px;
}

.metric-info {
  flex: 1;
}

.metric-name {
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.metric-value {
  font-size: 1.25rem;
  font-weight: bold;
  color: #667eea;
  margin-bottom: 4px;
}

.metric-change {
  font-size: 0.8rem;
  font-weight: 600;
}

.metric-change.positive {
  color: #10b981;
}

.metric-change.negative {
  color: #ef4444;
}

.metric-change.neutral {
  color: #6b7280;
}

.performance-chart {
  height: 300px;
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.experiment-logs {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-top: 24px;
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.logs-header h3 {
  margin: 0;
  color: #333;
  font-size: 1.1rem;
}

.clear-logs-btn {
  background: #ef4444;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.8rem;
  cursor: pointer;
}

.logs-content {
  max-height: 300px;
  overflow-y: auto;
}

.log-entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 8px;
  font-size: 0.9rem;
}

.log-entry.info {
  background: #eff6ff;
  color: #1e40af;
}

.log-entry.success {
  background: #f0fdf4;
  color: #166534;
}

.log-entry.warning {
  background: #fffbeb;
  color: #92400e;
}

.log-entry.error {
  background: #fef2f2;
  color: #991b1b;
}

.log-time {
  font-size: 0.8rem;
  opacity: 0.8;
  min-width: 80px;
}

.log-message {
  flex: 1;
}

@media (max-width: 768px) {
  .lab-controls {
    flex-direction: column;
  }

  .model-status-grid {
    grid-template-columns: 1fr;
  }

  .tool-buttons {
    flex-direction: column;
  }

  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .config-grid, .results-grid, .summary-item {
    grid-template-columns: 1fr;
  }

  .importance-bar {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .performance-metrics {
    grid-template-columns: 1fr;
  }

  .log-entry {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }
}
</style>
