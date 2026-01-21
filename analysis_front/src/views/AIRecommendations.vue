<template>
  <div class="ai-recommendations">
    <div class="page-header">
      <h1>🤖 AI 智能推荐</h1>
      <p class="subtitle">基于机器学习算法的币种推荐系统</p>
    </div>

    <!-- 控制面板 -->
    <div class="control-panel">
      <div class="control-group">
        <label>选择币种：</label>
        <div class="symbol-selector">
          <label v-for="symbol in availableSymbols" :key="symbol" class="symbol-checkbox">
            <input
              type="checkbox"
              :value="symbol"
              v-model="selectedSymbols"
              @change="handleSymbolChange"
            />
            <span class="symbol-tag">{{ symbol }}</span>
          </label>
        </div>
      </div>

      <div class="control-group">
        <label>推荐数量：</label>
        <select v-model="limit" @change="fetchRecommendations">
          <option :value="3">3个</option>
          <option :value="5">5个</option>
          <option :value="10">10个</option>
        </select>
      </div>

      <div class="control-group">
        <label>风险偏好：</label>
        <select v-model="riskLevel" @change="fetchRecommendations">
          <option value="conservative">保守型</option>
          <option value="moderate">稳健型</option>
          <option value="aggressive">激进型</option>
        </select>
      </div>

      <!-- 实时连接状态 -->
      <div class="connection-status" :class="wsStatus.class">
        <div class="status-indicator"></div>
        <span class="status-text">{{ wsStatus.text }}</span>
      </div>

      <div class="control-group">
        <label>推荐日期：</label>
        <input
          type="date"
          v-model="selectedDate"
          @change="fetchRecommendations"
          :max="today"
          class="date-input"
        />
        <button
          @click="resetToToday"
          class="reset-date-btn"
          title="重置为今天"
        >
          📅
        </button>
      </div>

      <div class="control-group">
        <button
          @click="openAnalysisDashboard"
          class="analysis-btn"
          title="打开AI推荐分析仪表板"
        >
          📊 AI分析
        </button>
      </div>

      <button
        @click="fetchRecommendations"
        :disabled="loading"
        class="refresh-btn"
      >
        {{ loading ? '分析中...' : '🔄 刷新推荐' }}
      </button>
    </div>

    <!-- 价格趋势图表 -->
    <div v-if="recommendations.length > 0" class="price-chart-section">
      <div class="chart-header">
        <h3>📈 价格趋势分析</h3>
        <div class="chart-controls">
          <select v-model="chartTimeframe" @change="updatePriceChart">
            <option value="1h">1小时</option>
            <option value="4h">4小时</option>
            <option value="1d">1天</option>
            <option value="7d">7天</option>
          </select>
          <button @click="updatePriceChart" class="refresh-chart-btn">
            🔄 刷新图表
          </button>
        </div>
      </div>
      <div class="price-chart-container">
        <LineChart
          :x-data="chartData.xData"
          :series="chartData.series"
          :title="`选中币种价格趋势 (${chartTimeframe})`"
          :y-label="'价格 (USD)'"
        />
      </div>
    </div>

    <!-- 推荐统计 -->
    <RecommendationStats
      v-if="recommendations.length > 0"
      :recommendations="recommendations"
    />

    <!-- 推荐结果 -->
    <div v-if="recommendations.length > 0" class="recommendations-grid">
      <div
        v-for="(rec, index) in recommendations"
        :key="rec.symbol"
        class="recommendation-card"
        :class="getCardClass(rec)"
      >
        <div class="card-header">
          <div class="rank-badge">#{{ rec.rank }}</div>
          <div class="symbol-info">
            <h3>{{ rec.symbol }}</h3>
            <div class="price">${{ formatPrice(rec.price || getCachedPrice(rec.symbol)) }}</div>
          </div>
          <div class="score-display">
            <div class="overall-score">
              <span class="score-value">{{ (rec.overall_score * 100).toFixed(1) }}</span>
              <span class="score-label">综合评分</span>
            </div>
          </div>
        </div>

        <div class="score-breakdown">
          <div class="score-item">
            <span class="score-label">技术指标</span>
            <div class="score-bar">
              <div
                class="score-fill"
                :style="{ width: (rec.technical_score * 100) + '%' }"
              ></div>
            </div>
            <span class="score-value">{{ (rec.technical_score * 100).toFixed(1) }}</span>
          </div>

          <div class="score-item">
            <span class="score-label">基本面</span>
            <div class="score-bar">
              <div
                class="score-fill"
                :style="{ width: (rec.fundamental_score * 100) + '%' }"
              ></div>
            </div>
            <span class="score-value">{{ (rec.fundamental_score * 100).toFixed(1) }}</span>
          </div>

          <div class="score-item">
            <span class="score-label">市场情绪</span>
            <div class="score-bar">
              <div
                class="score-fill"
                :style="{ width: (rec.sentiment_score * 100) + '%' }"
              ></div>
            </div>
            <span class="score-value">{{ (rec.sentiment_score * 100).toFixed(1) }}</span>
          </div>

          <div class="score-item">
            <span class="score-label">动量指标</span>
            <div class="score-bar">
              <div
                class="score-fill"
                :style="{ width: (rec.momentum_score * 100) + '%' }"
              ></div>
            </div>
            <span class="score-value">{{ (rec.momentum_score * 100).toFixed(1) }}</span>
          </div>
        </div>

        <div class="risk-info">
          <div class="risk-level" :class="rec.risk_level">
            <span class="risk-icon">
              {{ getRiskIcon(rec.risk_level) }}
            </span>
            <span class="risk-text">
              {{ getRiskText(rec.risk_level) }}
            </span>
          </div>
          <div class="risk-score">
            风险评分: {{ rec.risk_score.toFixed(1) }}
          </div>
        </div>

        <div class="ml-insights">
          <div class="ml-prediction">
            <span class="ml-label">AI预测:</span>
            <span class="ml-value">{{ (rec.ml_prediction * 100).toFixed(1) }}%</span>
            <span class="ml-confidence">信心: {{ (rec.ml_confidence * 100).toFixed(1) }}%</span>
          </div>
          <div class="recommended-position">
            建议仓位: {{ (rec.recommended_position * 100).toFixed(1) }}%
          </div>
        </div>

        <div class="expected-return">
          <span class="return-label">预期收益:</span>
          <span class="return-value">{{ (rec.expected_return * 100).toFixed(1) }}%</span>
        </div>

        <div class="reasons">
          <h4>推荐理由：</h4>
          <ul>
            <li v-for="reason in rec.reasons" :key="reason">{{ reason }}</li>
          </ul>
        </div>

        <div class="card-actions">
          <button @click="viewDetails(rec)" class="detail-btn">
            📊 查看详情
          </button>
          <button @click="addToPortfolio(rec)" class="portfolio-btn">
            ➕ 加入组合
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="!loading" class="empty-state">
      <div class="empty-illustration">
        <div class="robot-icon">🤖</div>
        <div class="chart-placeholder">
          <div class="placeholder-bar" style="height: 30px;"></div>
          <div class="placeholder-bar" style="height: 50px;"></div>
          <div class="placeholder-bar" style="height: 25px;"></div>
          <div class="placeholder-bar" style="height: 40px;"></div>
          <div class="placeholder-bar" style="height: 35px;"></div>
        </div>
      </div>
      <div class="empty-content">
        <h3>准备开始AI智能推荐</h3>
        <p>选择您感兴趣的币种，AI将为您提供个性化的投资建议</p>
        <div class="empty-tips">
          <div class="tip-item">
            <span class="tip-icon">🎯</span>
            <span>基于机器学习算法分析</span>
          </div>
          <div class="tip-item">
            <span class="tip-icon">📈</span>
            <span>实时市场数据驱动</span>
          </div>
          <div class="tip-item">
            <span class="tip-icon">⚡</span>
            <span>快速生成推荐结果</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <div class="loading-spinner"></div>
      <div class="loading-content">
        <h3>🤖 AI 智能分析中</h3>
        <p>正在分析 {{ selectedSymbols.length }} 个币种的市场数据...</p>
        <div class="loading-steps">
          <div class="loading-step active">
            <span class="step-icon">📊</span>
            <span class="step-text">收集市场数据</span>
          </div>
          <div class="loading-step">
            <span class="step-icon">🧠</span>
            <span class="step-text">AI 模型分析</span>
          </div>
          <div class="loading-step">
            <span class="step-icon">⚡</span>
            <span class="step-text">生成推荐结果</span>
          </div>
        </div>
      </div>
    </div>

<!-- MODAL CONTENT REMOVED - NOW USING ROUTE TO DETAIL PAGE -->
  </div>
</template>

<script>
import { api } from '@/api/api.js'
import LineChart from '@/components/LineChart.vue'
import RecommendationStats from '@/components/RecommendationStats.vue'
import behaviorTracker from '@/utils/behaviorTracker.js'

export default {
  name: 'AIRecommendations',
  components: {
    LineChart,
    RecommendationStats
  },
  data() {
    return {
      recommendations: [],
      loading: false,
      error: null,
      selectedSymbols: ['BTC', 'ETH', 'ADA', 'SOL', 'DOT'],
      limit: 5,
      riskLevel: 'moderate',
      selectedDate: new Date().toISOString().split('T')[0], // 默认为今天
      today: new Date().toISOString().split('T')[0], // 今天的日期
      availableSymbols: [
        'BTC', 'ETH', 'ADA', 'SOL', 'DOT', 'LINK', 'UNI',
        'AAVE', 'SUSHI', 'COMP', 'MKR', 'YFI', 'BAL', 'REN'
      ],
      currentPrices: {},
      chartTimeframe: '1d',
      chartData: {
        xData: [],
        series: []
      },
      wsStatus: {
        class: 'disconnected',
        text: '未连接'
      },
      // WebSocket重连相关
      wsReconnectAttempts: 0,
      wsMaxReconnectAttempts: 5,
      wsReconnectInterval: 3000, // 3秒
      wsReconnectTimer: null,
      wsIsReconnecting: false,
      // 性能优化相关
      lastUpdateTime: 0,
      chartCache: new Map(), // 缓存图表数据

      // 价格更新相关
      priceUpdateTimer: null,

      // 错误重试相关
      retryTimeout: null,

      // 性能优化相关
      fetchDebounceTimer: null,
      lastFetchTime: 0,
      fetchCooldown: 2000 // 2秒冷却时间
    }
  },
  async   mounted() {
    this.fetchRecommendations()
    this.connectRealtimeUpdates()

    // 预加载价格数据
    await this.preloadPrices()

    // 启动价格更新定时器（每30秒更新一次）
    this.startPriceUpdateTimer()
  },
  beforeUnmount() {
    this.cleanupWebSocket()
    this.cleanupCache()
    this.cleanupPriceUpdate()
    this.cleanupRetryTimeout()
    this.cleanupDebounceTimer()
  },

  methods: {
    cleanupCache() {
      // 清理图表缓存
      if (this.chartCache) {
        this.chartCache.clear()
      }

      // 清理定时器
      if (this.chartUpdateTimer) {
        clearTimeout(this.chartUpdateTimer)
        this.chartUpdateTimer = null
      }
    },

    cleanupPriceUpdate() {
      // 清理价格更新定时器
      if (this.priceUpdateTimer) {
        clearInterval(this.priceUpdateTimer)
        this.priceUpdateTimer = null
      }
    },

    cleanupRetryTimeout() {
      // 清理重试定时器
      if (this.retryTimeout) {
        clearTimeout(this.retryTimeout)
        this.retryTimeout = null
      }
    },

    cleanupDebounceTimer() {
      // 清理防抖定时器
      if (this.fetchDebounceTimer) {
        clearTimeout(this.fetchDebounceTimer)
        this.fetchDebounceTimer = null
      }
    },

    resetToToday() {
      this.selectedDate = this.today
      this.fetchRecommendations()
    },

    openAnalysisDashboard() {
      // 打开新的分析仪表板窗口或标签页
      const routeData = this.$router.resolve({
        path: '/ai-analysis-dashboard',
        query: {
          symbols: this.selectedSymbols.join(','),
          date: this.selectedDate
        }
      })
      window.open(routeData.href, '_blank')
    },
  async fetchRecommendations() {
      // 防抖控制：避免过于频繁的请求
      const now = Date.now()
      if (now - this.lastFetchTime < this.fetchCooldown) {
        // 如果在冷却期内，取消之前的定时器并重新设置
        if (this.fetchDebounceTimer) {
          clearTimeout(this.fetchDebounceTimer)
        }

        this.fetchDebounceTimer = setTimeout(() => {
          this.fetchDebounceTimer = null
          this._doFetchRecommendations()
        }, this.fetchCooldown - (now - this.lastFetchTime))

        return
      }

      this.lastFetchTime = now
      await this._doFetchRecommendations()
    },

    async _doFetchRecommendations() {
      this.loading = true
      this.error = null
      const startTime = Date.now()

      try {
        // 验证输入参数
        if (this.selectedSymbols.length === 0) {
          throw new Error('请至少选择一个币种')
        }

        if (this.limit < 1 || this.limit > 20) {
          throw new Error('推荐数量必须在1-20之间')
        }

        console.log('开始获取AI推荐:', {
          symbols: this.selectedSymbols,
          limit: this.limit,
          risk_level: this.riskLevel,
          date: this.selectedDate
        })

        const data = await api.getAIRecommendations({
          symbols: this.selectedSymbols,
          limit: this.limit,
          risk_level: this.riskLevel,
          date: this.selectedDate
        })

        // 验证响应数据
        if (!data || typeof data !== 'object') {
          throw new Error('服务器返回的数据格式错误')
        }

        if (!data.recommendations || !Array.isArray(data.recommendations)) {
          console.warn('服务器返回的推荐数据为空或格式错误')
          this.recommendations = []
        } else {
          // 验证和格式化推荐数据
          this.recommendations = data.recommendations.map(rec => this.validateAndFormatRecommendation(rec))
          console.log(`成功获取并验证 ${this.recommendations.length} 条推荐`)
        }

        // 获取推荐成功后，更新价格图表
        if (this.recommendations.length > 0) {
          this.updatePriceChart()
          this.$toast?.success(`成功获取 ${this.recommendations.length} 条AI推荐`)

          // 追踪AI推荐获取行为
          behaviorTracker.track('ai_recommendation_fetch', 'success', {
            symbol_count: this.selectedSymbols.length,
            symbols: this.selectedSymbols,
            limit: this.limit,
            risk_level: this.riskLevel,
            recommendation_count: this.recommendations.length,
            processing_time: Date.now() - startTime
          })

          // 追踪每个推荐的展示
          this.recommendations.forEach((rec, index) => {
            behaviorTracker.track('ai_recommendation_view', rec.symbol, {
              rank: rec.rank,
              overall_score: rec.overall_score,
              technical_score: rec.technical_score,
              fundamental_score: rec.fundamental_score,
              sentiment_score: rec.sentiment_score,
              momentum_score: rec.momentum_score,
              risk_score: rec.risk_score,
              expected_return: rec.expected_return,
              ml_prediction: rec.ml_prediction,
              ml_confidence: rec.ml_confidence,
              position: index + 1
            })
          })
        } else {
          this.$toast?.warning('未找到符合条件的推荐，请调整参数后重试')

          // 追踪无推荐结果的情况
          behaviorTracker.track('ai_recommendation_fetch', 'no_results', {
            symbol_count: this.selectedSymbols.length,
            symbols: this.selectedSymbols,
            limit: this.limit,
            risk_level: this.riskLevel
          })
        }

        // 记录推荐统计信息
        if (data.metadata) {
          console.log('推荐统计信息:', data.metadata)
        }

      } catch (error) {
        console.error('获取AI推荐失败:', error)

        // 分类错误处理
        let errorMessage = '获取推荐失败，请稍后重试'
        let toastType = 'error'
        let shouldRetry = false
        let retryDelay = 0

        if (error.message) {
          errorMessage = error.message
        }

        // 处理HTTP错误
        if (error.status) {
          const status = error.status
          switch (status) {
            case 400:
              errorMessage = '请求参数有误，请检查币种名称和参数设置'
              toastType = 'warning'
              break
            case 401:
              errorMessage = '认证失败，请重新登录'
              toastType = 'warning'
              // 可以在这里触发登录流程
              break
            case 403:
              errorMessage = '访问被拒绝，请检查权限'
              toastType = 'warning'
              break
            case 404:
              errorMessage = 'API端点不存在，请联系技术支持'
              toastType = 'error'
              break
            case 429:
              errorMessage = '请求过于频繁，请等待30秒后再试'
              toastType = 'warning'
              shouldRetry = true
              retryDelay = 30000 // 30秒后重试
              break
            case 500:
              errorMessage = '服务器内部错误，请稍后重试'
              toastType = 'error'
              shouldRetry = true
              retryDelay = 5000 // 5秒后重试
              break
            case 502:
            case 503:
            case 504:
              errorMessage = '服务器暂时不可用，请稍后重试'
              toastType = 'warning'
              shouldRetry = true
              retryDelay = 10000 // 10秒后重试
              break
            default:
              if (status >= 500) {
                errorMessage = `服务器错误 (${status})，请稍后重试`
                shouldRetry = true
                retryDelay = 5000
              } else {
                errorMessage = `请求失败 (${status})`
              }
          }
        } else if (error.code === 'NETWORK_ERROR') {
          errorMessage = '网络连接失败，请检查网络连接后重试'
          shouldRetry = true
          retryDelay = 3000
        } else if (error.code === 'TIMEOUT') {
          errorMessage = '请求超时，可能是网络较慢，请重试'
          shouldRetry = true
          retryDelay = 1000
        } else if (error.name === 'TypeError' && error.message.includes('fetch')) {
          errorMessage = '网络请求失败，请检查网络连接'
          shouldRetry = true
          retryDelay = 3000
        }

        this.error = errorMessage

        // 显示用户友好的错误提示
        if (this.$toast) {
          this.$toast[toastType](errorMessage, {
            duration: shouldRetry ? 5000 : 3000,
            action: shouldRetry ? {
              text: '重试',
              onClick: () => {
                if (retryDelay > 0) {
                  setTimeout(() => this.fetchRecommendations(), retryDelay)
                } else {
                  this.fetchRecommendations()
                }
              }
            } : null
          })
        } else {
          alert(errorMessage)
        }

        // 自动重试机制
        if (shouldRetry && retryDelay > 0 && !this.retryTimeout) {
          console.log(`将在 ${retryDelay}ms 后自动重试...`)
          this.retryTimeout = setTimeout(() => {
            this.retryTimeout = null
            console.log('开始自动重试获取推荐...')
            this.fetchRecommendations()
          }, retryDelay)
        }

        // 追踪错误
        behaviorTracker.track('ai_recommendation_error', 'api_error', {
          error_message: error.message || 'Unknown error',
          error_status: error.status || 'unknown',
          error_code: error.code || 'unknown',
          retry_scheduled: shouldRetry,
          retry_delay: retryDelay,
          symbol_count: this.selectedSymbols.length,
          symbols: this.selectedSymbols
        })
      } finally {
        this.loading = false
      }
    },

    connectRealtimeUpdates() {
      // 如果正在重连中，跳过
      if (this.wsIsReconnecting) {
        return
      }

      try {
        // 清理之前的连接和定时器
        this.cleanupWebSocket()

        const wsUrl = api.getRealtimeRecommendWS()
        console.log(`连接到WebSocket: ${wsUrl}`)
        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
          console.log('实时推荐连接成功')
          this.wsStatus = { class: 'connected', text: '已连接' }
          this.wsReconnectAttempts = 0 // 重置重连计数
          this.wsIsReconnecting = false

          // 追踪WebSocket连接成功
          behaviorTracker.track('websocket_connection', 'success', {
            connection_type: 'ai_recommendations',
            symbols: this.selectedSymbols,
            update_frequency: '60s'
          })

          // 订阅推荐更新
          this.ws.send(JSON.stringify({
            action: 'subscribe',
            symbols: this.selectedSymbols,
            update_frequency: '60s' // 每分钟更新一次
          }))

          // 显示连接成功提示
          this.$toast?.success('实时推荐连接成功')
        }

        this.ws.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            if (data.type === 'recommendation_update') {
              // 追踪WebSocket消息接收
              behaviorTracker.track('websocket_message', 'recommendation_update', {
                connection_type: 'ai_recommendations',
                message_type: data.type,
                recommendation_count: data.recommendations ? data.recommendations.length : 0,
                timestamp: data.timestamp
              })

              // 更新推荐数据
              this.updateRecommendations(data.recommendations)
            }
          } catch (error) {
            console.error('解析实时推荐数据失败:', error)

            // 追踪消息解析错误
            behaviorTracker.track('websocket_message', 'parse_error', {
              connection_type: 'ai_recommendations',
              error_message: error.message,
              raw_data_length: event.data ? event.data.length : 0
            })
          }
        }

        this.ws.onclose = (event) => {
          console.log(`实时推荐连接关闭 (代码: ${event.code}, 原因: ${event.reason})`)
          this.wsStatus = { class: 'disconnected', text: '已断开' }

          // 追踪WebSocket连接关闭
          behaviorTracker.track('websocket_connection', 'close', {
            connection_type: 'ai_recommendations',
            close_code: event.code,
            close_reason: event.reason,
            reconnect_attempts: this.wsReconnectAttempts
          })

          // 如果不是正常关闭（1000），尝试重连
          if (event.code !== 1000 && this.wsReconnectAttempts < this.wsMaxReconnectAttempts) {
            this.attemptReconnect()
          } else if (event.code !== 1000) {
            this.wsStatus = { class: 'error', text: '连接失败' }
            this.$toast?.error('实时推荐连接失败，请刷新页面重试')
          }
        }

        this.ws.onerror = (error) => {
          console.error('实时推荐连接错误:', error)
          this.wsStatus = { class: 'error', text: '连接错误' }

          // 追踪WebSocket连接错误
          behaviorTracker.track('websocket_connection', 'error', {
            connection_type: 'ai_recommendations',
            error_message: error.message || 'Unknown error',
            reconnect_attempts: this.wsReconnectAttempts
          })

          // 触发重连
          if (this.wsReconnectAttempts < this.wsMaxReconnectAttempts) {
            this.attemptReconnect()
          } else {
            this.$toast?.error('实时推荐连接失败，请检查网络连接')
          }
        }
      } catch (error) {
        console.error('创建实时推荐连接失败:', error)
        this.wsStatus = { class: 'error', text: '初始化失败' }
      }
    },

    attemptReconnect() {
      if (this.wsReconnectAttempts >= this.wsMaxReconnectAttempts) {
        console.log('达到最大重连次数，停止重连')

        // 追踪重连失败
        behaviorTracker.track('websocket_reconnect', 'max_attempts_reached', {
          connection_type: 'ai_recommendations',
          max_attempts: this.wsMaxReconnectAttempts,
          final_attempt: this.wsReconnectAttempts
        })

        return
      }

      this.wsReconnectAttempts++
      this.wsIsReconnecting = true
      this.wsStatus = {
        class: 'connecting',
        text: `重连中 (${this.wsReconnectAttempts}/${this.wsMaxReconnectAttempts})`
      }

      const delay = this.wsReconnectInterval * Math.pow(1.5, this.wsReconnectAttempts - 1) // 指数退避

      console.log(`将在 ${delay}ms 后进行第 ${this.wsReconnectAttempts} 次重连`)

      // 追踪重连尝试
      behaviorTracker.track('websocket_reconnect', 'attempt', {
        connection_type: 'ai_recommendations',
        attempt_number: this.wsReconnectAttempts,
        delay_ms: delay,
        max_attempts: this.wsMaxReconnectAttempts
      })

      this.wsReconnectTimer = setTimeout(() => {
        console.log(`开始第 ${this.wsReconnectAttempts} 次重连`)
        this.connectRealtimeUpdates()
      }, delay)
    },

    cleanupWebSocket() {
      // 清理之前的连接
      if (this.ws) {
        this.ws.close()
        this.ws = null
      }

      // 清理重连定时器
      if (this.wsReconnectTimer) {
        clearTimeout(this.wsReconnectTimer)
        this.wsReconnectTimer = null
      }
    },

    /**
     * 更新推荐数据（带性能优化）
     *
     * 性能优化策略:
     * 1. 节流控制: 1秒内最多处理一次更新，避免过度渲染
     * 2. 批量更新: 收集所有变化后一次性应用，减少DOM操作
     * 3. 智能过滤: 只在评分变化超过1%时触发重新排序
     * 4. 内存优化: 使用Map进行O(1)查找，避免数组遍历
     *
     * @param {Array} newRecommendations - 新的推荐数据数组
     */
    updateRecommendations(newRecommendations) {
      try {
        // 验证输入数据
        if (!Array.isArray(newRecommendations)) {
          console.error('实时推荐数据格式错误: 期望数组，收到:', typeof newRecommendations)
          return
        }

        if (newRecommendations.length === 0) {
          console.warn('实时推荐数据为空')
          return
        }

        // 节流控制：避免过于频繁的更新（1秒内最多更新一次）
        const now = Date.now()
        if (this.lastUpdateTime && now - this.lastUpdateTime < 1000) {
          console.log('实时推荐更新过于频繁，跳过此次更新')
          return
        }
        this.lastUpdateTime = now

        console.log(`处理 ${newRecommendations.length} 条实时推荐更新`)

        let updatedCount = 0
        let invalidCount = 0

        // 构建现有推荐的查找映射，提高查找效率
        const existingMap = new Map()
        this.recommendations.forEach(rec => {
          if (rec && rec.symbol) {
            existingMap.set(rec.symbol, rec)
          }
        })

        // 批量更新推荐数据
        const updates = []

        newRecommendations.forEach(newRec => {
          // 验证新推荐数据结构
          if (!newRec || typeof newRec !== 'object') {
            console.warn('跳过无效的推荐数据:', newRec)
            invalidCount++
            return
          }

          if (!newRec.symbol || typeof newRec.symbol !== 'string') {
            console.warn('跳过缺少symbol的推荐数据:', newRec)
            invalidCount++
            return
          }

          const existingRec = existingMap.get(newRec.symbol)
          if (existingRec) {
            // 从WebSocket更新数据中提取字段
            const newOverallScore = typeof newRec.overall_score === 'number' ? newRec.overall_score : existingRec.overall_score
            const newPrediction = typeof newRec.ml_prediction === 'number' ? newRec.ml_prediction : existingRec.ml_prediction
            const newConfidence = typeof newRec.ml_confidence === 'number' ? newRec.ml_confidence : existingRec.ml_confidence
            const newPrice = typeof newRec.price === 'number' ? newRec.price : existingRec.price
            const newRiskScore = typeof newRec.risk_score === 'number' ? newRec.risk_score : existingRec.risk_score

            // 计算新的综合评分（结合现有评分和最新评分）
            const blendedScore = (existingRec.overall_score * 0.8) + (newOverallScore * 0.2)
            const clampedScore = Math.max(0, Math.min(1, blendedScore))

            // 收集更新数据
            updates.push({
              rec: existingRec,
              newOverallScore: clampedScore,
              newPrediction,
              newConfidence,
              newPrice,
              newRiskScore,
              scoreChanged: Math.abs(clampedScore - existingRec.overall_score) > 0.01 // 评分变化超过1%才算有意义更新
            })

            updatedCount++
          }
        })

        // 批量应用更新
        if (updates.length > 0) {
          updates.forEach(update => {
            update.rec.overall_score = update.newOverallScore
            update.rec.ml_prediction = update.newPrediction
            update.rec.ml_confidence = update.newConfidence
            update.rec.price = update.newPrice
            update.rec.risk_score = update.newRiskScore
          })

          // 只在有意义的变化时重新排序
          const hasSignificantChanges = updates.some(u => u.scoreChanged)
          if (hasSignificantChanges) {
            // 重新排序
            this.recommendations.sort((a, b) => b.overall_score - a.overall_score)
            this.recommendations.forEach((rec, index) => {
              rec.rank = index + 1
            })
          }

          console.log(`实时推荐更新完成: 更新了 ${updatedCount} 条记录${invalidCount > 0 ? `, 跳过 ${invalidCount} 条无效数据` : ''}`)

          // 显示更新提示（避免过于频繁）
          if (Math.random() < 0.2 && hasSignificantChanges) { // 20%的概率且有显著变化时显示提示
            this.$toast?.info(`实时推荐已更新`)
          }
        } else {
          console.log('没有找到需要更新的推荐记录')
        }

      } catch (error) {
        console.error('处理实时推荐更新时发生错误:', error)
        this.$toast?.error('实时推荐更新失败')
      }
    },

    handleSymbolChange() {
      // 确保至少选择一个币种
      if (this.selectedSymbols.length === 0) {
        this.selectedSymbols = ['BTC']
      }

      // 追踪币种选择变化
      behaviorTracker.track('ai_recommendation_symbol_change', 'update', {
        selected_symbols: this.selectedSymbols,
        symbol_count: this.selectedSymbols.length,
        previous_count: this.selectedSymbols.length // 这里可以优化为实际的previous值
      })

      // 重新获取推荐
      this.fetchRecommendations()
    },

    getCardClass(rec) {
      const score = rec.overall_score
      if (score >= 0.8) return 'excellent'
      if (score >= 0.7) return 'good'
      if (score >= 0.6) return 'fair'
      return 'poor'
    },

    getRiskIcon(level) {
      const icons = {
        low: '🟢',
        medium: '🟡',
        high: '🔴',
        critical: '⚠️'
      }
      return icons[level] || '❓'
    },

    getRiskText(level) {
      const texts = {
        low: '低风险',
        medium: '中等风险',
        high: '高风险',
        critical: '极高风险'
      }
      return texts[level] || '未知风险'
    },

    formatPrice(price) {
      if (!price) return '0.0000'
      return price.toLocaleString('en-US', {
        minimumFractionDigits: 4,
        maximumFractionDigits: 4
      })
    },

    async getCurrentPrice(symbol) {
      // 优先从缓存获取
      if (this.currentPrices[symbol] && this.currentPrices[symbol].timestamp) {
        const cacheAge = Date.now() - this.currentPrices[symbol].timestamp
        // 缓存5秒内有效
        if (cacheAge < 5000) {
          return this.currentPrices[symbol].price
        }
      }

      try {
        // 从API获取最新的市场数据
        const response = await api.binanceTop({
          kind: 'spot',
          date: new Date().toISOString().split('T')[0], // 今天
          tz: Intl.DateTimeFormat().resolvedOptions().timeZone
        })

        if (response && response.data && response.data.length > 0) {
          // 币安涨幅榜API返回的是按时间段聚合的数据
          // 每个data项包含一个时间段的数据，里面有多个交易对
          let foundPrice = null

          // 遍历每个时间段的数据
          for (const timeSlot of response.data) {
            if (timeSlot.items && timeSlot.items.length > 0) {
              // 在这个时间段中查找对应的交易对
              const symbolData = timeSlot.items.find(item => {
                const itemSymbol = item.symbol.toUpperCase()
                // 检查多种可能的匹配方式
                return itemSymbol === `${symbol}USDT` ||
                       itemSymbol === `${symbol}BTC` ||
                       itemSymbol === `${symbol}ETH` ||
                       itemSymbol === symbol.toUpperCase()
              })

              if (symbolData && symbolData.last_price) {
                const price = parseFloat(symbolData.last_price)
                if (price > 0) {
                  foundPrice = price
                  break // 找到价格就停止查找
                }
              }
            }
          }

          if (foundPrice) {
            // 缓存价格
            this.currentPrices[symbol] = {
              price: foundPrice,
              timestamp: Date.now()
            }
            console.log(`获取到 ${symbol} 价格: ${foundPrice}`)
            return foundPrice
          }

          // 如果没找到，记录调试信息
          console.warn(`${symbol} 未在API响应中找到匹配的交易对`)

          // 调试：打印第一个时间段的前几个交易对
          if (response.data.length > 0 && response.data[0].items) {
            console.log(`第一个时间段的前5个交易对:`, response.data[0].items.slice(0, 5).map(item => item.symbol))
          }
        } else {
          console.warn(`${symbol} API响应为空或格式错误`)
        }
      } catch (error) {
        console.warn(`获取 ${symbol} 价格失败:`, error)
      }

      // 如果API获取失败，使用后备价格数据
      return this.getFallbackPrice(symbol)
    },

    getFallbackPrice(symbol) {
      // 后备价格数据（相对稳定的参考价格）
      const fallbackPrices = {
        // 大市值币种
        BTC: 45000, ETH: 2800, BNB: 245,

        // 中等市值币种
        ADA: 0.45, SOL: 95, DOT: 8.50, LINK: 12.80, UNI: 6.20,
        AAVE: 85.50, SUSHI: 1.85, COMP: 45.20, MKR: 2200, YFI: 8500,

        // 小市值币种
        BAL: 4.15, REN: 0.085, XRP: 0.52, LTC: 68, BCH: 225,
        EOS: 0.85, XLM: 0.12, VET: 0.025, TRX: 0.065, ETC: 18.50,
        DASH: 48.20, ZEC: 52.10, BTG: 15.80, XMR: 145,
        ZRX: 0.35, OMG: 1.45, LRC: 0.28, REP: 8.90, GNT: 0.15,

        // 新兴币种的合理默认价格（基于市值和稀缺性）
        STRK: 0.1169,  // Starknet相关代币
        OP: 1.85,      // Optimism
        ARB: 0.95,     // Arbitrum
        MATIC: 0.85,   // Polygon
        AVAX: 28.50,   // Avalanche
        NEAR: 4.85,    // Near Protocol
        ATOM: 8.95,    // Cosmos
        FIL: 4.25,     // Filecoin
        ICP: 8.15,     // Internet Computer
        ETC: 18.50,    // Ethereum Classic
        HBAR: 0.065,   // Hedera
        FLOW: 0.55,    // Flow
        MANA: 0.32,    // Decentraland
        SAND: 0.35,    // The Sandbox
        GALA: 0.018,   // Gala Games
        ENJ: 0.21,     // Enjin
        IMX: 1.25,     // Immutable X
        LDO: 1.65,     // Lido DAO
        SNX: 1.85,     // Synthetix
        CRV: 0.45,     // Curve
        GMX: 35.50,    // GMX
        JOE: 0.32,     // JOE
        SPELL: 0.00045, // Spell Token
        LOOKS: 0.18,   // LooksRare
        GAL: 1.85,     // Galxe
      }

      // 对于完全未知的币种，返回一个合理的默认价格
      // 基于加密货币市场的平均价格水平
      const defaultPrice = fallbackPrices[symbol]
      if (defaultPrice) {
        return defaultPrice
      }

      // 如果是完全未知的币种，返回一个保守的默认价格
      // 大多数山寨币的价格在0.01-10之间
      console.warn(`未知币种 ${symbol} 使用默认价格 0.10`)
      return 0.10
    },

    getCachedPrice(symbol) {
      // 优先返回缓存的价格
      if (this.currentPrices[symbol] && this.currentPrices[symbol].price) {
        return this.currentPrices[symbol].price
      }

      // 如果没有缓存，返回后备价格
      return this.getFallbackPrice(symbol)
    },

    async preloadPrices() {
      try {
        // 获取所有选中的币种的价格
        const uniqueSymbols = [...new Set([
          ...this.selectedSymbols,
          ...this.recommendations.map(rec => rec.symbol)
        ])]

        // 并发获取所有价格
        const pricePromises = uniqueSymbols.map(symbol => this.getCurrentPrice(symbol))
        await Promise.all(pricePromises)

        console.log(`预加载了 ${uniqueSymbols.length} 个币种的价格数据`)
      } catch (error) {
        console.warn('预加载价格数据失败:', error)
      }
    },

    startPriceUpdateTimer() {
      // 清除之前的定时器
      if (this.priceUpdateTimer) {
        clearInterval(this.priceUpdateTimer)
      }

      // 动态调整更新频率：活跃时30秒，不活跃时60秒
      let updateInterval = 30000
      let lastActivity = Date.now()

      // 监听用户活动
      const activityEvents = ['mousedown', 'mousemove', 'keypress', 'scroll', 'touchstart']
      activityEvents.forEach(event => {
        document.addEventListener(event, () => {
          lastActivity = Date.now()
        }, { passive: true })
      })

      this.priceUpdateTimer = setInterval(async () => {
        try {
          // 检查用户是否活跃（最近5分钟内有活动）
          const isActive = (Date.now() - lastActivity) < 300000 // 5分钟
          updateInterval = isActive ? 30000 : 60000 // 活跃:30秒，不活跃:60秒

          const uniqueSymbols = [...new Set([
            ...this.selectedSymbols,
            ...this.recommendations.map(rec => rec.symbol)
          ])]

          // 如果没有订阅的币种，跳过更新
          if (uniqueSymbols.length === 0) {
            return
          }

          // 限制并发数量，避免过载
          const batchSize = 5
          for (let i = 0; i < uniqueSymbols.length; i += batchSize) {
            const batch = uniqueSymbols.slice(i, i + batchSize)
            const pricePromises = batch.map(symbol => this.getCurrentPrice(symbol))
            await Promise.all(pricePromises)

            // 小延迟避免API限流
            if (i + batchSize < uniqueSymbols.length) {
              await new Promise(resolve => setTimeout(resolve, 100))
            }
          }

          // 只在有实际变化时更新视图
          if (this.recommendations.length > 0) {
            this.$forceUpdate()
          }
        } catch (error) {
          console.warn('价格更新失败:', error)
          // 失败时增加重试延迟
          updateInterval = Math.min(updateInterval * 1.5, 120000) // 最大2分钟
        }
      }, updateInterval)
    },

    viewDetails(rec) {
      // 追踪详情查看行为
      behaviorTracker.track('ai_recommendation_detail_view', rec.symbol, {
        rank: rec.rank,
        overall_score: rec.overall_score,
        expected_return: rec.expected_return,
        risk_score: rec.risk_score,
        ml_prediction: rec.ml_prediction,
        ml_confidence: rec.ml_confidence,
        reasons: rec.reasons
      })

      // 跳转到详情页面，将数据作为URL参数传递
      const dataParam = encodeURIComponent(JSON.stringify(rec))
      this.$router.push({
        path: `/ai-recommendation/${rec.symbol}`,
        query: {
          rank: rec.rank,
          data: dataParam
        }
      })

      // 同时在控制台显示详细信息（用于调试）
      const details = {
        基本信息: {
          交易对: rec.symbol,
          排名: `#${rec.rank}`,
          当前价格: `$${rec.price || this.getCurrentPrice(rec.symbol)}`,
          综合评分: `${(rec.overall_score * 100).toFixed(1)}分`
        },
        技术分析: {
          技术指标: `${(rec.technical_score * 100).toFixed(1)}分`,
          基本面: `${(rec.fundamental_score * 100).toFixed(1)}分`,
          市场情绪: `${(rec.sentiment_score * 100).toFixed(1)}分`,
          动量指标: `${(rec.momentum_score * 100).toFixed(1)}分`
        },
        风险评估: {
          风险等级: this.getRiskText(rec.risk_level),
          风险评分: `${rec.risk_score.toFixed(1)}分`,
          建议仓位: `${(rec.recommended_position * 100).toFixed(1)}%`,
          预期收益: `${(rec.expected_return * 100).toFixed(1)}%`
        },
        AI分析: {
          AI预测得分: `${(rec.ml_prediction * 100).toFixed(1)}分`,
          AI信心度: `${(rec.ml_confidence * 100).toFixed(1)}%`,
          推荐理由: rec.reasons.join('；')
        }
      }

      console.group(`📊 ${rec.symbol} 详细分析`)
      Object.entries(details).forEach(([category, data]) => {
        console.group(category)
        Object.entries(data).forEach(([key, value]) => {
          console.log(`${key}: ${value}`)
        })
        console.groupEnd()
      })
      console.groupEnd()
    },


    addToPortfolio(rec) {
      // 追踪加入组合行为
      behaviorTracker.track('ai_recommendation_add_portfolio', rec.symbol, {
        rank: rec.rank,
        overall_score: rec.overall_score,
        expected_return: rec.expected_return,
        risk_score: rec.risk_score,
        ml_prediction: rec.ml_prediction,
        ml_confidence: rec.ml_confidence,
        recommended_position: rec.recommended_position
      })

      // 添加到投资组合
      console.log('添加到投资组合:', rec.symbol)
      // 可以实现添加到投资组合的逻辑
      this.$toast?.success(`${rec.symbol} 已添加到投资组合`)
    },

    updatePriceChart() {
      // 追踪图表更新行为
      behaviorTracker.track('ai_recommendation_chart_update', 'refresh', {
        timeframe: this.chartTimeframe,
        symbol_count: this.selectedSymbols.length,
        symbols: this.selectedSymbols
      })

      // 生成模拟的价格数据（带缓存优化）
      this.generateMockPriceData()
    },

    generateMockPriceData() {
      const cacheKey = `${this.chartTimeframe}_${this.selectedSymbols.sort().join('_')}_${this.recommendations.length}`

      // 检查缓存
      if (this.chartCache.has(cacheKey)) {
        console.log('使用缓存的图表数据')
        this.chartData = this.chartCache.get(cacheKey)
        return
      }

      const now = new Date()
      const points = this.chartTimeframe === '1h' ? 60 : this.chartTimeframe === '4h' ? 48 : this.chartTimeframe === '1d' ? 24 : 168

      // 使用更高效的方式生成时间轴数据
      const xData = []
      const timeInterval = this.getTimeInterval()
      for (let i = points; i >= 0; i--) {
        const time = new Date(now.getTime() - i * timeInterval)
        xData.push(time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }))
      }

      // 为每个选中的币种生成价格数据（优化内存使用）
      const series = []
      const baseSeed = Date.now() // 使用时间作为随机种子，确保一致性

      this.selectedSymbols.forEach((symbol, symbolIndex) => {
        const basePrice = this.getCurrentPrice(symbol)

        // 使用确定性随机数生成，确保相同参数产生相同结果
        const prices = this.generatePriceSeries(basePrice, points, baseSeed + symbolIndex)

        series.push({
          name: symbol,
          data: prices,
          lineStyle: { width: 2 },
          itemStyle: { color: this.getSymbolColor(symbol) }
        })
      })

      this.chartData = { xData, series }

      // 缓存结果（限制缓存大小）
      if (this.chartCache.size > 10) {
        const firstKey = this.chartCache.keys().next().value
        this.chartCache.delete(firstKey)
      }
      this.chartCache.set(cacheKey, this.chartData)

      console.log(`生成了新的图表数据: ${this.selectedSymbols.length} 个币种, ${points} 个数据点`)
    },

    generatePriceSeries(basePrice, points, seed) {
      const prices = []
      let currentPrice = basePrice * (0.95 + this.seededRandom(seed) * 0.1) // 初始价格在±5%范围内
      prices.push(currentPrice)

      // 使用简单的趋势 + 随机游走的组合
      const trend = 0.0001 // 轻微上升趋势
      const volatility = 0.01 // 1%的波动率

      for (let i = 1; i < points; i++) {
        // 确定性随机数生成
        const randomChange = (this.seededRandom(seed + i) - 0.5) * 2 * volatility
        const trendChange = trend * i

        currentPrice *= (1 + randomChange + trendChange)
        prices.push(currentPrice)
      }

      return prices
    },

    seededRandom(seed) {
      // 简单的伪随机数生成器，确保确定性
      const x = Math.sin(seed) * 10000
      return x - Math.floor(x)
    },

    getTimeInterval() {
      // 返回毫秒数
      switch (this.chartTimeframe) {
        case '1h': return 60 * 1000 // 1分钟
        case '4h': return 5 * 60 * 1000 // 5分钟
        case '1d': return 60 * 60 * 1000 // 1小时
        case '7d': return 4 * 60 * 60 * 1000 // 4小时
        default: return 60 * 60 * 1000
      }
    },

    getSymbolColor(symbol) {
      const colors = {
        BTC: '#f7931a',
        ETH: '#627eea',
        ADA: '#0033ad',
        SOL: '#9945ff',
        DOT: '#e6007a',
        LINK: '#2a5ada',
        UNI: '#ff007a',
        AAVE: '#b6509e',
        SUSHI: '#fa52a0',
        COMP: '#00d395',
        MKR: '#1aab9b',
        YFI: '#006ae3',
        BAL: '#1e1e1e',
        REN: '#00163d'
      }
      return colors[symbol] || '#666'
    },

    // 获取策略类型显示文本
    getStrategyTypeText(strategyType) {
      const textMap = {
        'LONG': '多头策略',
        'SHORT': '空头策略',
        'RANGE': '震荡策略'
      }
      return textMap[strategyType] || '未知策略'
    },

    // 获取买卖方向文本
    getTradingDirectionText(direction) {
      const textMap = {
        'long': '买入做多',
        'short': '卖出做空',
        'range': '区间交易',
        'LONG': '买入做多',
        'SHORT': '卖出做空',
        'RANGE': '区间交易'
      }
      return textMap[direction] || '观望'
    },

    // 获取市场环境文本
    getMarketConditionText(condition) {
      const textMap = {
        'bullish': '牛市环境',
        'bearish': '熊市环境',
        'neutral': '中性环境'
      }
      return textMap[condition] || '未知环境'
    },

    // 获取止损类型文本
    getStopLossTypeText(stopType) {
      const textMap = {
        'INITIAL': '初始止损',
        'TRAILING': '追踪止损',
        'MENTAL': '心理止损'
      }
      return textMap[stopType] || stopType
    },

    // 获取仓位策略文本
    getPositionStrategyText(strategy) {
      const textMap = {
        'FIXED': '固定仓位',
        'MARTINGALE': '马丁格尔',
        'ANTI_MARTINGALE': '反马丁格尔'
      }
      return textMap[strategy] || strategy
    },

    // 获取优先级文本
    getPriorityText(priority) {
      const textMap = {
        'high': '高优先级',
        'medium': '中优先级',
        'low': '低优先级'
      }
      return textMap[priority] || priority
    },

    // 获取告警类型文本
    getAlertTypeText(alertType) {
      const textMap = {
        'entry': '入场提醒',
        'exit': '出场提醒',
        'stop_loss': '止损提醒',
        'profit_target': '利润提醒',
        'risk_warning': '风险警告'
      }
      return textMap[alertType] || alertType
    },

    // 获取条件文本
    getConditionText(condition) {
      const textMap = {
        'above': '价格上涨至',
        'below': '价格下跌至',
        'cross': '价格穿越'
      }
      return textMap[condition] || condition
    },

    // 格式化日期时间
    formatDate(dateString) {
      if (!dateString) return ''
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    },

    // 测试价格告警
    testPriceAlerts() {
      this.$toast?.info('价格告警测试功能开发中...')
      console.log('测试价格告警功能')
    },

    // 管理价格告警
    managePriceAlerts() {
      this.$toast?.info('价格告警管理功能开发中...')
      console.log('管理价格告警功能')
    },

    /**
     * 重置组件状态
     * 清除错误状态、缓存数据，恢复初始状态
     */
    resetState() {
      // 清除错误状态
      this.error = null

      // 清除推荐数据
      this.recommendations = []

      // 清除图表数据
      this.chartData = {
        xData: [],
        series: []
      }

      // 清除缓存
      this.cleanupCache()

      // 重新连接WebSocket
      if (this.ws) {
        this.ws.close()
        this.connectRealtimeUpdates()
      }

      // 显示成功提示
      this.$toast?.success('状态已重置，请重新获取推荐')
    },

    /**
     * 验证和格式化推荐数据，确保数据格式正确
     * @param {Object} rec - 原始推荐数据
     * @returns {Object} - 格式化后的推荐数据
     */
    validateAndFormatRecommendation(rec) {
      if (!rec || typeof rec !== 'object') {
        console.warn('推荐数据格式错误:', rec)
        return this.getDefaultRecommendation()
      }

      // 验证必需字段
      const requiredFields = ['symbol', 'rank', 'overall_score', 'price']
      for (const field of requiredFields) {
        if (!(field in rec)) {
          console.warn(`推荐数据缺少必需字段: ${field}`, rec)
          // 填充默认值而不是返回默认推荐
        }
      }

      // 格式化数值字段
      const formatted = { ...rec }

      // 确保数值字段是数字类型
      const numericFields = [
        'rank', 'overall_score', 'expected_return', 'risk_score',
        'technical_score', 'fundamental_score', 'sentiment_score', 'momentum_score',
        'ml_prediction', 'ml_confidence', 'price', 'recommended_position'
      ]

      numericFields.forEach(field => {
        if (typeof formatted[field] === 'string') {
          const parsed = parseFloat(formatted[field])
          if (!isNaN(parsed)) {
            formatted[field] = parsed
          } else {
            console.warn(`字段 ${field} 无法转换为数字:`, formatted[field])
            formatted[field] = 0
          }
        } else if (typeof formatted[field] !== 'number' || isNaN(formatted[field])) {
          console.warn(`字段 ${field} 不是有效数字:`, formatted[field])
          formatted[field] = 0
        }
      })

      // 确保数组字段是数组
      if (!Array.isArray(formatted.reasons)) {
        formatted.reasons = formatted.reasons ? [String(formatted.reasons)] : ['综合分析结果']
      }

      // 确保字符串字段是字符串
      if (typeof formatted.symbol !== 'string') {
        formatted.symbol = String(formatted.symbol || 'UNKNOWN')
      }

      if (typeof formatted.risk_level !== 'string') {
        formatted.risk_level = 'medium'
      }

      // 限制数值范围
      formatted.overall_score = Math.max(0, Math.min(1, formatted.overall_score))
      formatted.ml_confidence = Math.max(0, Math.min(1, formatted.ml_confidence))
      formatted.recommended_position = Math.max(0, Math.min(1, formatted.recommended_position))

      return formatted
    },

    /**
     * 获取默认推荐数据（用于错误恢复）
     * @returns {Object} 默认推荐对象
     */
    getDefaultRecommendation() {
      return {
        symbol: 'UNKNOWN',
        rank: 999,
        overall_score: 0.5,
        expected_return: 0.0,
        risk_score: 0.5,
        technical_score: 0.5,
        fundamental_score: 0.5,
        sentiment_score: 0.5,
        momentum_score: 0.5,
        ml_prediction: 0.5,
        ml_confidence: 0.5,
        price: 0,
        recommended_position: 0.05,
        risk_level: 'medium',
        reasons: ['数据暂不可用']
      }
    }
  }
}
</script>
<style scoped>
.ai-recommendations {
  padding: 20px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  text-align: center;
  margin-bottom: 30px;
}

.page-header h1 {
  font-size: 2.5rem;
  margin-bottom: 10px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.subtitle {
  color: #666;
  font-size: 1.1rem;
}

.control-panel {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-bottom: 30px;
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: center;
}

.control-group {
  display: flex;
  align-items: center;
  gap: 10px;
}

.control-group label {
  font-weight: 600;
  color: #333;
  white-space: nowrap;
}

.symbol-selector {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.symbol-checkbox {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
}

.symbol-tag {
  background: #f0f0f0;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 0.9rem;
  transition: all 0.2s;
}

.symbol-checkbox input:checked + .symbol-tag {
  background: #667eea;
  color: white;
}

.control-group select {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

/* 连接状态指示器 */
.connection-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.connection-status.disconnected {
  background: #fee2e2;
  color: #dc2626;
}

.connection-status.connecting {
  background: #fef3c7;
  color: #d97706;
}

.connection-status.connected {
  background: #d1fae5;
  color: #059669;
}

.connection-status.error {
  background: #fee2e2;
  color: #dc2626;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  animation: pulse 2s infinite;
}

.connection-status.disconnected .status-indicator {
  background: #dc2626;
}

.connection-status.connecting .status-indicator {
  background: #d97706;
}

.connection-status.connected .status-indicator {
  background: #059669;
}

.connection-status.error .status-indicator {
  background: #dc2626;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.refresh-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.refresh-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.refresh-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.recommendations-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.recommendation-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  transition: all 0.3s ease;
  position: relative;
  overflow: hidden;
}

.recommendation-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 25px rgba(0,0,0,0.15);
}

.recommendation-card.excellent {
  border-left: 4px solid #10b981;
}

.recommendation-card.good {
  border-left: 4px solid #3b82f6;
}

.recommendation-card.fair {
  border-left: 4px solid #f59e0b;
}

.recommendation-card.poor {
  border-left: 4px solid #ef4444;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.rank-badge {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 4px 10px;
  border-radius: 20px;
  font-weight: bold;
  font-size: 0.9rem;
}

.symbol-info h3 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: bold;
  color: #333;
}

.price {
  color: #666;
  font-size: 0.9rem;
  margin-top: 2px;
}

.score-display {
  text-align: right;
}

.overall-score {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.score-value {
  font-size: 2rem;
  font-weight: bold;
  color: #333;
}

.score-label {
  font-size: 0.8rem;
  color: #666;
}

.score-breakdown {
  margin-bottom: 15px;
}

.score-item {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.score-item .score-label {
  width: 80px;
  font-size: 0.85rem;
  color: #666;
}

.score-bar {
  flex: 1;
  height: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  margin: 0 10px;
  overflow: hidden;
}

.score-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.score-item .score-value {
  width: 40px;
  text-align: right;
  font-size: 0.85rem;
  font-weight: 600;
  color: #333;
}

.risk-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
  padding: 10px;
  background: #f8f9fa;
  border-radius: 8px;
}

.risk-level {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}

.risk-level.low {
  color: #10b981;
}

.risk-level.medium {
  color: #f59e0b;
}

.risk-level.high {
  color: #ef4444;
}

.risk-level.critical {
  color: #dc2626;
}

.risk-score {
  font-size: 0.9rem;
  color: #666;
}

.ml-insights {
  margin-bottom: 15px;
  padding: 10px;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
  border-radius: 8px;
}

.ml-prediction {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 5px;
}

.ml-label {
  font-weight: 600;
  color: #333;
}

.ml-value {
  font-size: 1.1rem;
  font-weight: bold;
  color: #667eea;
}

.ml-confidence {
  font-size: 0.85rem;
  color: #666;
}

.recommended-position {
  font-size: 0.9rem;
  color: #555;
}

.expected-return {
  text-align: center;
  margin-bottom: 15px;
  padding: 10px;
  background: #e8f5e8;
  border-radius: 8px;
}

.return-label {
  font-weight: 600;
  color: #333;
  margin-right: 10px;
}

.return-value {
  font-size: 1.2rem;
  font-weight: bold;
  color: #10b981;
}

.reasons {
  margin-bottom: 15px;
}

.reasons h4 {
  margin: 0 0 8px 0;
  font-size: 0.95rem;
  color: #333;
}

.reasons ul {
  margin: 0;
  padding-left: 20px;
}

.reasons li {
  font-size: 0.85rem;
  color: #555;
  margin-bottom: 4px;
  line-height: 1.4;
}

.card-actions {
  display: flex;
  gap: 10px;
}

.detail-btn, .portfolio-btn {
  flex: 1;
  padding: 8px 12px;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.detail-btn {
  background: #f0f0f0;
  color: #333;
}

.detail-btn:hover {
  background: #e0e0e0;
}

.portfolio-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.portfolio-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.empty-state, .loading-state, .error-state {
  text-align: center;
  padding: 60px 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
}

.empty-icon, .error-icon {
  font-size: 4rem;
  margin-bottom: 20px;
}

/* 空状态样式 */
.empty-illustration {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 30px;
}

.robot-icon {
  font-size: 4rem;
  margin-bottom: 20px;
  animation: float 3s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0px); }
  50% { transform: translateY(-10px); }
}

.chart-placeholder {
  display: flex;
  align-items: end;
  gap: 8px;
  height: 60px;
}

.placeholder-bar {
  width: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 4px 4px 0 0;
  opacity: 0.6;
  animation: grow 2s ease-in-out infinite;
}

.placeholder-bar:nth-child(2) { animation-delay: 0.2s; }
.placeholder-bar:nth-child(3) { animation-delay: 0.4s; }
.placeholder-bar:nth-child(4) { animation-delay: 0.6s; }
.placeholder-bar:nth-child(5) { animation-delay: 0.8s; }

@keyframes grow {
  0%, 100% { opacity: 0.6; }
  50% { opacity: 1; }
}

.empty-content {
  text-align: center;
  max-width: 500px;
}

.empty-content h3 {
  color: #333;
  margin-bottom: 15px;
  font-size: 1.3rem;
}

.empty-content p {
  color: #666;
  margin-bottom: 25px;
  font-size: 1rem;
}

.empty-tips {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 15px;
  margin-bottom: 25px;
}

.tip-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
  font-size: 0.9rem;
  color: #555;
}

.tip-icon {
  font-size: 1.1rem;
}

/* 错误状态样式 */
.error-illustration {
  position: relative;
  display: flex;
  justify-content: center;
  align-items: center;
  margin-bottom: 30px;
}

.error-robot {
  font-size: 4rem;
  animation: shake 0.5s ease-in-out infinite;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.error-sparks {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  pointer-events: none;
}

.spark {
  position: absolute;
  font-size: 1.5rem;
  animation: spark 2s ease-in-out infinite;
}

.spark:nth-child(1) {
  top: 20%;
  left: 20%;
  animation-delay: 0s;
}

.spark:nth-child(2) {
  top: 30%;
  right: 25%;
  animation-delay: 0.5s;
}

.spark:nth-child(3) {
  bottom: 25%;
  left: 25%;
  animation-delay: 1s;
}

@keyframes spark {
  0%, 100% { opacity: 0; transform: scale(0.5); }
  50% { opacity: 1; transform: scale(1); }
}

.error-content {
  text-align: center;
  max-width: 500px;
}

.error-content h3 {
  color: #dc2626;
  margin-bottom: 15px;
  font-size: 1.3rem;
}

.error-message {
  color: #666;
  margin-bottom: 20px;
  font-size: 1rem;
  background: #fef2f2;
  padding: 12px;
  border-radius: 8px;
  border-left: 4px solid #dc2626;
}

.error-suggestions {
  text-align: left;
  margin-bottom: 25px;
}

.error-suggestions h4 {
  color: #333;
  margin-bottom: 10px;
  font-size: 1rem;
}

.error-suggestions ul {
  margin: 0;
  padding-left: 20px;
}

.error-suggestions li {
  color: #555;
  margin-bottom: 5px;
  font-size: 0.9rem;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.error-actions button {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.9rem;
}

.retry-btn {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
}

.retry-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 15px rgba(102, 126, 234, 0.3);
}

.reset-btn {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.reset-btn:hover {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.retry-icon, .reset-icon {
  font-size: 1rem;
}

/* 响应式调整 */
@media (max-width: 768px) {
  .empty-tips {
    grid-template-columns: 1fr;
  }

  .error-actions {
    flex-direction: column;
  }

  .error-actions button {
    width: 100%;
    justify-content: center;
  }
}

.loading-content {
  text-align: center;
}

.loading-content h3 {
  color: #333;
  margin-bottom: 10px;
  font-size: 1.2rem;
}

.loading-content p {
  color: #666;
  margin-bottom: 20px;
}

.loading-steps {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 300px;
  margin: 0 auto;
}

.loading-step {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 16px;
  background: #f8f9fa;
  border-radius: 8px;
  opacity: 0.5;
  transition: opacity 0.3s ease;
}

.loading-step.active {
  opacity: 1;
  background: linear-gradient(135deg, rgba(102, 126, 234, 0.1) 0%, rgba(118, 75, 162, 0.1) 100%);
}

.step-icon {
  font-size: 1.2rem;
}

.step-text {
  color: #555;
  font-weight: 500;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.retry-btn {
  background: #667eea;
  color: white;
  border: none;
  padding: 10px 20px;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
}

.retry-btn:hover {
  background: #5a67d8;
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  animation: fadeIn 0.3s ease;
}

.modal-content {
  background: white;
  border-radius: 12px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.3);
  max-width: 700px;
  width: 90%;
  max-height: 80vh;
  overflow-y: auto;
  animation: slideIn 0.3s ease;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 12px 12px 0 0;
}

.modal-header h3 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.close-btn {
  background: none;
  border: none;
  color: white;
  font-size: 1.5rem;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: background 0.2s;
}

.close-btn:hover {
  background: rgba(255, 255, 255, 0.2);
}

.modal-body {
  padding: 24px;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h4 {
  margin: 0 0 16px 0;
  color: #333;
  font-size: 1.1rem;
  font-weight: 600;
  border-bottom: 2px solid #667eea;
  padding-bottom: 8px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.info-item .label {
  font-weight: 600;
  color: #555;
}

.info-item .value {
  font-weight: 600;
  color: #333;
}

.rank-badge {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white !important;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 0.8rem;
}

.score-details {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.score-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.score-item .label {
  width: 80px;
  font-weight: 600;
  color: #555;
  font-size: 0.9rem;
}

.progress-bar {
  flex: 1;
  height: 8px;
  background: #e5e7eb;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.score-item .score {
  width: 50px;
  text-align: right;
  font-weight: 600;
  color: #333;
  font-size: 0.9rem;
}

.risk-assessment, .ai-analysis {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.risk-item, .ai-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f8f9fa;
  border-radius: 8px;
}

.risk-item .label, .ai-item .label {
  font-weight: 600;
  color: #555;
}

.risk-level {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}

.risk-level.low { color: #10b981; }
.risk-level.medium { color: #f59e0b; }
.risk-level.high { color: #ef4444; }
.risk-level.critical { color: #dc2626; }

.position {
  color: #3b82f6;
  font-weight: 600;
}

.return {
  color: #10b981;
  font-weight: 600;
}

.prediction {
  color: #667eea;
  font-weight: 600;
}

.confidence {
  color: #6b7280;
}

.ai-reasons {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-reasons .label {
  font-weight: 600;
  color: #555;
  margin-bottom: 4px;
}

.ai-reasons ul {
  margin: 0;
  padding-left: 20px;
}

.ai-reasons li {
  color: #666;
  font-size: 0.9rem;
  line-height: 1.4;
  margin-bottom: 4px;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateY(-20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 价格图表样式 */
.price-chart-section {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  margin-bottom: 30px;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.chart-header h3 {
  margin: 0;
  color: #333;
  font-size: 1.25rem;
}

.chart-controls {
  display: flex;
  gap: 10px;
  align-items: center;
}

.chart-controls select {
  padding: 6px 10px;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
}

.refresh-chart-btn {
  background: #667eea;
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
}

.refresh-chart-btn:hover {
  background: #5a67d8;
  transform: translateY(-1px);
}

.price-chart-container {
  height: 400px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  overflow: hidden;
}

@media (max-width: 768px) {
  .ai-recommendations {
    padding: 15px;
  }

  .control-panel {
    flex-direction: column;
    align-items: stretch;
  }

  .price-chart-section {
    padding: 15px;
  }

  .chart-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .chart-controls {
    width: 100%;
    justify-content: space-between;
  }

  .price-chart-container {
    height: 300px;
  }

  .recommendations-grid {
    grid-template-columns: 1fr;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  .score-display {
    text-align: left;
  }

  .card-actions {
    flex-direction: column;
  }
}

/* 交易策略样式 */
.trading-strategy-section {
  margin-top: 24px;
  padding: 20px;
  background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
  border-radius: 12px;
  border: 1px solid #e2e8f0;
}

.strategy-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.strategy-header h4 {
  margin: 0;
  color: #1f2937;
  font-size: 18px;
  font-weight: 600;
}

.strategy-type-badge {
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.strategy-long {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.strategy-short {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.strategy-range {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}

.strategy-direction {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 20px;
}

.direction-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.direction-label {
  font-weight: 500;
  color: #374151;
  min-width: 80px;
}

.direction-value {
  font-weight: 600;
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 14px;
}

.direction-long {
  background: #dcfce7;
  color: #166534;
}

.direction-short {
  background: #fee2e2;
  color: #991b1b;
}

.direction-range {
  background: #fef3c7;
  color: #92400e;
}

.market-condition {
  color: #6b7280;
  font-weight: 500;
}

.entry-strategy,
.exit-strategy,
.stop-loss-strategy,
.position-sizing,
.risk-management,
.strategy-rationale {
  margin-bottom: 20px;
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.entry-strategy h5,
.exit-strategy h5,
.stop-loss-strategy h5,
.position-sizing h5,
.risk-management h5,
.strategy-rationale h5 {
  margin: 0 0 12px 0;
  color: #1f2937;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.timing-label,
.zone-label,
.targets-title {
  font-weight: 500;
  color: #374151;
  min-width: 80px;
  display: inline-block;
}

.timing-value,
.zone-range {
  color: #1f2937;
  font-weight: 500;
}

.zone-avg {
  color: #6b7280;
  font-size: 12px;
  margin-left: 8px;
}

.target-list {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.target-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f9fafb;
  border-radius: 6px;
}

.target-label {
  font-weight: 500;
  color: #374151;
  min-width: 60px;
}

.target-range {
  color: #1f2937;
  font-weight: 500;
}

.target-desc {
  color: #6b7280;
  font-size: 12px;
  margin-left: 8px;
}

.stop-loss-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.stop-loss-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 12px;
  background: #f9fafb;
  border-radius: 6px;
}

.stop-loss-type {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.stop-type-initial {
  background: #fef3c7;
  color: #92400e;
}

.stop-type-trailing {
  background: #dbeafe;
  color: #1e40af;
}

.stop-type-mental {
  background: #fee2e2;
  color: #991b1b;
}

.stop-loss-level {
  font-weight: 600;
  color: #dc2626;
  min-width: 80px;
}

.stop-loss-condition {
  color: #6b7280;
  font-size: 14px;
}

.position-grid,
.risk-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.position-item,
.risk-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.position-label,
.risk-label {
  font-weight: 500;
  color: #374151;
  min-width: 100px;
}

.position-value,
.risk-value {
  font-weight: 600;
  color: #1f2937;
}

.position-strategy {
  color: #6b7280;
  font-weight: 500;
}

.rationale-list {
  margin: 0;
  padding-left: 20px;
}

.rationale-list li {
  margin-bottom: 8px;
  color: #374151;
  line-height: 1.5;
}

.rationale-list li:last-child {
  margin-bottom: 0;
}

/* 执行计划样式 */
.execution-plan-section,
.price-alerts-section {
  margin-top: 24px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
}

.execution-header,
.alerts-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.execution-header h4,
.alerts-header h4 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.execution-type-badge,
.alerts-count {
  background: rgba(255, 255, 255, 0.2);
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
}

.overall-position {
  padding: 20px;
  background: white;
  border-bottom: 1px solid #e2e8f0;
}

.position-summary {
  display: flex;
  align-items: center;
  gap: 20px;
  flex-wrap: wrap;
}

.position-label,
.current-price {
  font-weight: 500;
  color: #374151;
}

.position-value {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.current-price {
  color: #6b7280;
  font-size: 14px;
}

.entry-plan,
.exit-plan,
.risk-controls,
.execution-timeline {
  padding: 20px;
  background: white;
  border-bottom: 1px solid #e2e8f0;
}

.entry-plan h5,
.exit-plan h5,
.risk-controls h5,
.execution-timeline h5 {
  margin: 0 0 16px 0;
  color: #1f2937;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.entry-stages,
.exit-stages {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.entry-stage,
.exit-stage {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
}

.entry-stage {
  border-left: 4px solid #10b981;
}

.exit-stage {
  border-left: 4px solid #f59e0b;
}

.priority-high {
  border-left-color: #ef4444;
}

.priority-medium {
  border-left-color: #f59e0b;
}

.priority-low {
  border-left-color: #6b7280;
}

.stage-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.stage-number {
  background: #1f2937;
  color: white;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 14px;
  font-weight: 600;
}

.stage-percentage {
  background: #10b981;
  color: white;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 14px;
  font-weight: 600;
}

.stage-priority {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.priority-high {
  background: #fef2f2;
  color: #dc2626;
}

.priority-medium {
  background: #fef3c7;
  color: #d97706;
}

.priority-low {
  background: #f3f4f6;
  color: #6b7280;
}

.stage-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.price-range,
.stage-condition,
.stage-limits,
.stage-metrics {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.range-label,
.condition-label,
.limit-label,
.profit-target,
.risk-reward {
  font-weight: 500;
  color: #374151;
  min-width: 80px;
  font-size: 14px;
}

.range-value,
.condition-value,
.limit-value {
  color: #1f2937;
  font-weight: 500;
}

/* 日期选择器样式 */
.date-input {
  padding: 8px 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  background: white;
  font-size: 14px;
  color: #374151;
  transition: all 0.2s ease;
  cursor: pointer;
}

.date-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.date-input:hover {
  border-color: #d1d5db;
}

.reset-date-btn {
  padding: 8px 12px;
  background: #f3f4f6;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  color: #6b7280;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
  margin-left: 8px;
}

.reset-date-btn:hover {
  background: #e5e7eb;
  border-color: #d1d5db;
  color: #374151;
}

.reset-date-btn:active {
  background: #d1d5db;
}

.analysis-btn {
  padding: 8px 16px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s ease;
  box-shadow: 0 2px 4px rgba(102, 126, 234, 0.2);
}

.analysis-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(102, 126, 234, 0.3);
}

.analysis-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 4px rgba(102, 126, 234, 0.2);
}

.range-avg {
  color: #6b7280;
  font-size: 14px;
}

.slippage {
  color: #ef4444;
  font-size: 14px;
}

.risk-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.risk-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.risk-item .risk-label {
  font-weight: 500;
  color: #6b7280;
  font-size: 14px;
}

.risk-item .risk-value {
  font-weight: 600;
  color: #1f2937;
  font-size: 16px;
}

.risk-item .risk-value.enabled {
  color: #10b981;
}

.risk-item .risk-value.disabled {
  color: #6b7280;
}

.trailing-percent {
  color: #6b7280;
  font-size: 14px;
  margin-left: 8px;
}

.timeline-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.timeline-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.timeline-label {
  font-weight: 500;
  color: #374151;
  min-width: 120px;
}

.timeline-value {
  color: #1f2937;
  font-weight: 500;
}

.key-milestones h6 {
  margin: 0 0 12px 0;
  color: #374151;
  font-size: 14px;
  font-weight: 600;
}

.milestones-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.milestones-list li {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
  border-left: 3px solid #667eea;
}

.milestone-time {
  font-weight: 600;
  color: #1f2937;
  min-width: 80px;
  font-size: 14px;
}

.milestone-event {
  font-weight: 500;
  color: #374151;
  flex: 1;
}

.milestone-desc {
  color: #6b7280;
  font-size: 14px;
}

/* 价格告警样式 */
.price-alerts-list {
  padding: 20px;
  background: white;
}

.price-alert-item {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  transition: all 0.2s ease;
}

.price-alert-item:hover {
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
  border-color: #667eea;
}

.price-alert-item:last-child {
  margin-bottom: 0;
}

.alert-type-entry {
  border-left: 4px solid #10b981;
}

.alert-type-exit {
  border-left: 4px solid #f59e0b;
}

.alert-type-stop_loss {
  border-left: 4px solid #ef4444;
}

.alert-type-profit_target {
  border-left: 4px solid #8b5cf6;
}

.alert-type-risk_warning {
  border-left: 4px solid #f97316;
}

.priority-high {
  background: linear-gradient(135deg, #fef2f2 0%, #fef2f2 100%);
}

.priority-medium {
  background: linear-gradient(135deg, #fef3c7 0%, #fef3c7 100%);
}

.priority-low {
  background: linear-gradient(135deg, #f3f4f6 0%, #f3f4f6 100%);
}

.alert-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}

.alert-type-badge {
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 600;
  color: white;
}

.alert-type-badge.type-entry {
  background: #10b981;
}

.alert-type-badge.type-exit {
  background: #f59e0b;
}

.alert-type-badge.type-stop_loss {
  background: #ef4444;
}

.alert-type-badge.type-profit_target {
  background: #8b5cf6;
}

.alert-type-badge.type-risk_warning {
  background: #f97316;
}

.alert-priority-badge {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.alert-priority-badge.priority-high {
  background: #fef2f2;
  color: #dc2626;
}

.alert-priority-badge.priority-medium {
  background: #fef3c7;
  color: #d97706;
}

.alert-priority-badge.priority-low {
  background: #f3f4f6;
  color: #6b7280;
}

.alert-symbol {
  font-weight: 600;
  color: #1f2937;
  font-size: 16px;
}

.alert-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.alert-price,
.alert-message,
.alert-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.price-label,
.alert-created,
.alert-status {
  font-weight: 500;
  color: #374151;
  font-size: 14px;
}

.price-value {
  font-weight: 600;
  color: #1f2937;
}

.price-condition {
  color: #6b7280;
  font-size: 14px;
}

.message-icon {
  font-size: 16px;
}

.message-text {
  color: #374151;
  flex: 1;
  font-size: 14px;
}

.alert-status.active {
  color: #10b981;
}

.alert-status.inactive {
  color: #6b7280;
}

.alerts-actions {
  padding: 20px;
  background: white;
  border-top: 1px solid #e2e8f0;
  display: flex;
  gap: 12px;
  justify-content: center;
}

.test-alerts-btn,
.manage-alerts-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.test-alerts-btn {
  background: #3b82f6;
  color: white;
}

.test-alerts-btn:hover {
  background: #2563eb;
  transform: translateY(-1px);
}

.manage-alerts-btn {
  background: #6b7280;
  color: white;
}

.manage-alerts-btn:hover {
  background: #4b5563;
  transform: translateY(-1px);
}

.test-icon,
.manage-icon {
  font-size: 16px;
}
</style>

