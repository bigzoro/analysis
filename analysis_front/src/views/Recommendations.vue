<template>
  <!-- 标签页导航 -->
  <section class="panel">
    <div class="tabs">
      <button 
        :class="['tab', { active: activeTab === 'recommendations' }]"
        @click="activeTab = 'recommendations'"
      >
        推荐列表
      </button>
      <button 
        :class="['tab', { active: activeTab === 'backtest' }]"
        @click="activeTab = 'backtest'"
      >
        表现验证
      </button>
      <button 
        :class="['tab', { active: activeTab === 'simulation' }]"
        @click="activeTab = 'simulation'"
      >
        模拟交易
      </button>
    </div>
  </section>

  <!-- 推荐列表标签页 -->
  <div v-if="activeTab === 'recommendations'">
    <!-- 控制面板 -->
    <section class="panel">
      <div class="row">
        <div class="header-with-date">
        <h2>智能推荐</h2>
          <div class="date-picker-wrapper">
          <input
            type="date"
            v-model="selectedDate"
            :max="maxDate"
            :min="minDate"
            @change="handleDateChange"
              placeholder="选择日期查看历史"
              class="date-picker"
          />
          </div>
        </div>
        <div class="spacer"></div>
        <div class="control-group">
          <label style="margin-right: 8px;">类型：</label>
          <select v-model="kind" @change="handleKindChange">
            <option value="spot">现货</option>
            <option value="futures">期货</option>
          </select>
          <button
            class="primary"
            @click="loadData(true)"
            :disabled="loading || generating"
          >
            {{ loading || generating ? '加载中...' : (selectedDate ? '加载历史' : '刷新推荐') }}
          </button>
        </div>
      </div>
    </section>

    <!-- 推荐内容 -->
      <!-- 推荐算法表现概览 -->
      <section class="panel" v-if="performanceStats && !selectedDate">
        <div class="row">
          <h3>📊 推荐算法表现</h3>
          <div class="spacer"></div>
          <button @click="debouncedLoadPerformanceStats">刷新统计</button>
        </div>
      <div class="performance-overview">
        <div class="overview-item">
          <div class="overview-label">总推荐数</div>
          <div class="overview-value">{{ performanceStats.total || 0 }}</div>
        </div>
        <div class="overview-item">
          <div class="overview-label">策略平均收益</div>
          <div class="overview-value" :class="getPerformanceClass(performanceStats.avg_strategy_return)">
            {{ formatPercent(performanceStats.avg_strategy_return) }}
          </div>
        </div>
        <div class="overview-item">
          <div class="overview-label">策略胜率</div>
          <div class="overview-value">{{ formatPercent(performanceStats.strategy_win_rate) }}</div>
        </div>
        <div class="overview-item">
          <div class="overview-label">7天平均收益</div>
          <div class="overview-value" :class="getPerformanceClass(performanceStats.avg_return_7d)">
            {{ formatPercent(performanceStats.avg_return_7d) }}
          </div>
        </div>
        <div class="overview-item">
          <div class="overview-label">30天平均收益</div>
          <div class="overview-value" :class="getPerformanceClass(performanceStats.avg_return_30d)">
            {{ formatPercent(performanceStats.avg_return_30d) }}
          </div>
        </div>
      </div>
    </section>

  <!-- 加载状态：骨架屏 -->
  <section style="margin-top:12px;" class="panel" v-if="loading">
    <div class="skeleton-container">
      <div v-for="i in 5" :key="i" class="skeleton-card">
        <div class="skeleton-header">
          <div class="skeleton-avatar"></div>
          <div class="skeleton-content">
            <div class="skeleton-line" style="width: 80px; height: 20px;"></div>
            <div class="skeleton-line" style="width: 120px; height: 14px; margin-top: 8px;"></div>
          </div>
          <div class="skeleton-score"></div>
        </div>
        <div class="skeleton-body">
          <div class="skeleton-line" style="width: 100%; height: 16px;"></div>
          <div class="skeleton-line" style="width: 80%; height: 16px; margin-top: 12px;"></div>
          <div class="skeleton-chips">
            <div class="skeleton-chip"></div>
            <div class="skeleton-chip"></div>
            <div class="skeleton-chip"></div>
          </div>
        </div>
      </div>
    </div>
  </section>

  <section style="margin-top:12px;" class="panel" v-else-if="data && data.recommendations">
    <div class="meta-bar" style="margin-bottom: 16px;">
      <span class="chip">生成时间：{{ formatTime(data.generated_at) }}</span>
      <span class="chip" v-if="data.cached">使用缓存</span>
    </div>

    <div v-if="data.recommendations.length === 0" class="empty-state">
      <div class="empty-icon">📊</div>
      <p class="empty-text">暂无智能推荐数据</p>
      <p class="empty-hint">请稍后刷新或检查筛选条件</p>
      <button class="primary" @click="load(true)">刷新</button>
    </div>

    <div v-else class="recommendations-list">
      <div 
        v-for="rec in enrichedRecommendations" 
        :key="rec.rank"
        class="recommendation-card-compact"
        @click="viewDetail(rec)"
      >
        <div class="card-header-compact">
          <div class="rank-badge" :class="`rank-${rec.rank}`">
            #{{ rec.rank }}
          </div>
          <div class="symbol-info-compact">
            <h3>{{ rec.base_symbol }}</h3>
            <span class="symbol-pair">{{ rec.symbol }}</span>
          </div>
          <div class="score-and-strategy">
            <div class="total-score-compact">
              <div class="score-value">{{ rec.total_score.toFixed(1) }}</div>
              <div class="score-label">总分</div>
            </div>
            <div class="strategy-badge" v-if="rec.strategy_type" :class="`strategy-${rec.strategy_type.toLowerCase()}`">
              {{ getStrategyText(rec.strategy_type) }}
            </div>
          </div>
        </div>

        <div class="card-body-compact">
          <!-- 核心信息：当前价格、实时收益、风险等级 -->
          <div class="compact-info">
            <div class="info-item">
              <span class="info-label">当前价格</span>
              <span class="info-value" v-if="rec.data.price">${{ formatPrice(rec.data.price) }}</span>
              <span class="info-value loading-price" v-else>加载中...</span>
            </div>
            <div class="info-item" v-if="rec.performance && rec.performance.current_return !== null && rec.performance.current_return !== undefined">
              <span class="info-label">实时收益</span>
              <span class="info-value" :class="getPerformanceClass(rec.performance.current_return)">
                {{ formatPercent(rec.performance.current_return) }}
              </span>
            </div>
            <div class="info-item" v-if="rec.risk">
              <span class="info-label">风险等级</span>
              <span class="risk-badge-small" :class="`risk-${rec.risk.risk_level || 'medium'}`">
                {{ getRiskLevelText(rec.risk.risk_level) }}
              </span>
            </div>
          </div>
          
          <!-- 交易信号（新增） -->
          <div class="trading-signal-compact" v-if="rec.technical && rec.technical.trading_signal">
            <div class="signal-badge" :class="getSignalClass(rec.technical.trading_signal.signal)">
              <span class="signal-text">{{ getSignalText(rec.technical.trading_signal.signal) }}</span>
              <span class="signal-strength">{{ rec.technical.trading_signal.strength.toFixed(0) }}%</span>
            </div>
          </div>

          <!-- 因子得分概览（简化显示） -->
          <div class="scores-compact">
            <span class="score-chip">市场: {{ rec.scores.market.toFixed(1) }}</span>
            <span class="score-chip">资金流: {{ rec.scores.flow.toFixed(1) }}</span>
            <span class="score-chip">热度: {{ rec.scores.heat.toFixed(1) }}</span>
            <span class="score-chip">事件: {{ rec.scores.event.toFixed(1) }}</span>
            <span class="score-chip">情绪: {{ rec.scores.sentiment.toFixed(1) }}</span>
          </div>

          <!-- 生成时间 -->
          <div class="generation-time-compact" v-if="rec.generated_at">
            <span class="time-label">生成时间：</span>
            <span class="time-value">{{ formatTime(rec.generated_at) }}</span>
          </div>

          <button class="detail-btn" @click.stop="viewDetail(rec)">查看详情</button>
        </div>
      </div>
    </div>

    <!-- 详情模态框 -->
    <div v-if="selectedRecommendation" class="modal-overlay" @click="selectedRecommendation = null">
      <div class="modal-content large" @click.stop>
        <div class="modal-header">
          <h2>{{ selectedRecommendation.base_symbol }} 推荐详情</h2>
          <button class="close-btn" @click="selectedRecommendation = null">×</button>
        </div>
        <div class="modal-body" v-if="selectedRecommendation">
          <!-- 详情内容：显示所有原有信息 -->
          <div class="detail-section">
            <div class="detail-header">
              <div class="rank-badge" :class="`rank-${selectedRecommendation.rank}`">
                #{{ selectedRecommendation.rank }}
              </div>
              <div class="symbol-info">
                <h3>{{ selectedRecommendation.base_symbol }}</h3>
                <span class="symbol-pair">{{ selectedRecommendation.symbol }}</span>
              </div>
              <div class="total-score">
                <div class="score-value">{{ selectedRecommendation.total_score.toFixed(1) }}</div>
                <div class="score-label">总分</div>
              </div>
              <div class="generation-info" v-if="selectedRecommendation.generated_at">
                <div class="generation-time">
                  <span class="time-label">生成时间：</span>
                  <span class="time-value">{{ formatTime(selectedRecommendation.generated_at) }}</span>
                </div>
              </div>
            </div>

            <div class="detail-content">
              <!-- 因子得分详情 -->
              <div class="score-breakdown">
                <div class="score-item">
                  <span class="score-label">市场表现</span>
                  <span class="score-value">{{ selectedRecommendation.scores.market.toFixed(1) }}</span>
                </div>
                <div class="score-item">
                  <span class="score-label">资金流</span>
                  <span class="score-value">{{ selectedRecommendation.scores.flow.toFixed(1) }}</span>
                </div>
                <div class="score-item">
                  <span class="score-label">市场热度</span>
                  <span class="score-value">{{ selectedRecommendation.scores.heat.toFixed(1) }}</span>
                </div>
                <div class="score-item">
                  <span class="score-label">事件</span>
                  <span class="score-value">{{ selectedRecommendation.scores.event.toFixed(1) }}</span>
                </div>
                <div class="score-item">
                  <span class="score-label">情绪</span>
                  <span class="score-value">{{ selectedRecommendation.scores.sentiment.toFixed(1) }}</span>
                </div>
              </div>

              <!-- 数据信息 -->
              <div class="data-info">
                <div class="data-item" v-if="selectedRecommendation.data.price">
                  <span class="data-label">当前价格：</span>
                  <span class="data-value">${{ formatPrice(selectedRecommendation.data.price) }}</span>
                </div>
                <div class="data-item" v-if="selectedRecommendation.data.volume_24h !== null">
                  <span class="data-label">24h成交量：</span>
                  <span class="data-value">{{ formatVolume(selectedRecommendation.data.volume_24h) }}</span>
                </div>
                <div class="data-item" v-if="selectedRecommendation.data.market_cap_usd !== null">
                  <span class="data-label">市值：</span>
                  <span class="data-value">{{ formatUSD(selectedRecommendation.data.market_cap_usd) }}</span>
                </div>
                <div class="data-item" v-if="selectedRecommendation.data.net_flow_24h !== null && selectedRecommendation.data.net_flow_24h !== 0">
                  <span class="data-label">净流入：</span>
                  <span :class="['data-value', selectedRecommendation.data.net_flow_24h >= 0 ? 'positive' : 'negative']">
                    {{ selectedRecommendation.data.net_flow_24h >= 0 ? '+' : '' }}{{ formatUSD(selectedRecommendation.data.net_flow_24h) }}
                  </span>
                </div>
              </div>

              <!-- 风险评级 -->
              <div class="risk-section" v-if="selectedRecommendation.risk">
                <div class="risk-header">
                  <h4>风险评级</h4>
                  <span 
                    class="risk-badge" 
                    :class="`risk-${selectedRecommendation.risk.risk_level || 'medium'}`"
                  >
                    {{ getRiskLevelText(selectedRecommendation.risk.risk_level) }}
                  </span>
                </div>
                <div class="risk-metrics">
                  <div class="risk-item">
                    <span class="risk-label">综合风险：</span>
                    <span class="risk-value" :class="getRiskClass(selectedRecommendation.risk.overall_risk)">
                      {{ selectedRecommendation.risk.overall_risk?.toFixed(1) || 0 }}
                    </span>
                  </div>
                  <div class="risk-breakdown">
                    <div class="risk-metric">
                      <span class="metric-label">波动率</span>
                      <span class="metric-value">{{ selectedRecommendation.risk.volatility_risk?.toFixed(0) || 0 }}</span>
                    </div>
                    <div class="risk-metric">
                      <span class="metric-label">流动性</span>
                      <span class="metric-value">{{ selectedRecommendation.risk.liquidity_risk?.toFixed(0) || 0 }}</span>
                    </div>
                    <div class="risk-metric">
                      <span class="metric-label">市场</span>
                      <span class="metric-value">{{ selectedRecommendation.risk.market_risk?.toFixed(0) || 0 }}</span>
                    </div>
                    <div class="risk-metric">
                      <span class="metric-label">技术</span>
                      <span class="metric-value">{{ selectedRecommendation.risk.technical_risk?.toFixed(0) || 0 }}</span>
                    </div>
                  </div>
                </div>
                <div class="risk-warnings" v-if="selectedRecommendation.risk.risk_warnings && selectedRecommendation.risk.risk_warnings.length > 0">
                  <div class="warning-title">⚠️ 风险提示</div>
                  <ul class="warning-list">
                    <li v-for="(warning, idx) in selectedRecommendation.risk.risk_warnings" :key="idx">{{ warning }}</li>
                  </ul>
                </div>
              </div>

              <!-- 推荐理由 -->
              <div class="reasons" v-if="selectedRecommendation.reasons && selectedRecommendation.reasons.length > 0">
                <div class="reasons-title">推荐理由：</div>
                <ul class="reasons-list">
                  <li v-for="(reason, idx) in selectedRecommendation.reasons" :key="idx">{{ reason }}</li>
                </ul>
              </div>

              <!-- 交易策略 -->
              <div class="trading-strategy-section" v-if="selectedRecommendation.trading_strategy">
                <div class="strategy-header">
                  <h4>📈 交易策略</h4>
                  <span class="strategy-type-badge" :class="`strategy-${selectedRecommendation.trading_strategy.strategy_type?.toLowerCase() || 'long'}`">
                    {{ getStrategyTypeText(selectedRecommendation.trading_strategy.strategy_type) }}
                  </span>
                </div>

                <!-- 买卖方向 -->
                <div class="strategy-direction">
                  <div class="direction-item">
                    <span class="direction-label">买卖方向：</span>
                    <span class="direction-value" :class="`direction-${selectedRecommendation.trading_strategy.trading_direction?.toLowerCase() || 'long'}`">
                      {{ getTradingDirectionText(selectedRecommendation.trading_strategy.trading_direction) }}
                    </span>
                  </div>
                  <div class="direction-item">
                    <span class="direction-label">市场环境：</span>
                    <span class="market-condition">{{ getMarketConditionText(selectedRecommendation.trading_strategy.market_condition) }}</span>
                  </div>
                </div>

                <!-- 入场策略 -->
                <div class="entry-strategy">
                  <h5>🎯 入场策略</h5>
                  <div class="entry-timing">
                    <span class="timing-label">入场时机：</span>
                    <span class="timing-value">{{ selectedRecommendation.trading_strategy.entry_timing || '当前价格附近' }}</span>
                  </div>
                  <div class="entry-zone" v-if="selectedRecommendation.trading_strategy.entry_zone">
                    <span class="zone-label">入场区间：</span>
                    <span class="zone-range">
                      ${{ formatPrice(selectedRecommendation.trading_strategy.entry_zone.min) }} -
                      ${{ formatPrice(selectedRecommendation.trading_strategy.entry_zone.max) }}
                      <span class="zone-avg">(最佳: ${{ formatPrice(selectedRecommendation.trading_strategy.entry_zone.avg) }})</span>
                    </span>
                  </div>
                </div>

                <!-- 出场策略 -->
                <div class="exit-strategy">
                  <h5>🎯 出场策略</h5>
                  <div class="exit-timing">
                    <span class="timing-label">退场时机：</span>
                    <span class="timing-value">{{ selectedRecommendation.trading_strategy.exit_timing || '分批出场' }}</span>
                  </div>
                  <div class="exit-targets" v-if="selectedRecommendation.trading_strategy.exit_targets && selectedRecommendation.trading_strategy.exit_targets.length > 0">
                    <div class="targets-title">出场目标：</div>
                    <div class="target-list">
                      <div v-for="(target, idx) in selectedRecommendation.trading_strategy.exit_targets" :key="idx" class="target-item">
                        <span class="target-label">目标{{ idx + 1 }}：</span>
                        <span class="target-range">
                          ${{ formatPrice(target.min) }} - ${{ formatPrice(target.max) }}
                          <span class="target-desc">{{ target.description }}</span>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- 止损策略 -->
                <div class="stop-loss-strategy" v-if="selectedRecommendation.trading_strategy.stop_loss_levels && selectedRecommendation.trading_strategy.stop_loss_levels.length > 0">
                  <h5>🛡️ 止损策略</h5>
                  <div class="stop-loss-list">
                    <div v-for="(stopLoss, idx) in selectedRecommendation.trading_strategy.stop_loss_levels" :key="idx" class="stop-loss-item">
                      <span class="stop-loss-type" :class="`stop-type-${stopLoss.type?.toLowerCase() || 'initial'}`">{{ getStopLossTypeText(stopLoss.type) }}</span>
                      <span class="stop-loss-level">${{ formatPrice(stopLoss.level) }}</span>
                      <span class="stop-loss-condition">{{ stopLoss.condition }}</span>
                    </div>
                  </div>
                </div>

                <!-- 仓位管理 -->
                <div class="position-sizing" v-if="selectedRecommendation.trading_strategy.position_sizing">
                  <h5>📊 仓位管理</h5>
                  <div class="position-grid">
                    <div class="position-item">
                      <span class="position-label">建议仓位：</span>
                      <span class="position-value">{{ (selectedRecommendation.trading_strategy.position_sizing.adjusted_position * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="position-item">
                      <span class="position-label">最大仓位：</span>
                      <span class="position-value">{{ (selectedRecommendation.trading_strategy.position_sizing.max_position * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="position-item">
                      <span class="position-label">最小仓位：</span>
                      <span class="position-value">{{ (selectedRecommendation.trading_strategy.position_sizing.min_position * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="position-item">
                      <span class="position-label">仓位策略：</span>
                      <span class="position-strategy">{{ getPositionStrategyText(selectedRecommendation.trading_strategy.position_sizing.scaling_strategy) }}</span>
                    </div>
                  </div>
                </div>

                <!-- 风险管理 -->
                <div class="risk-management" v-if="selectedRecommendation.trading_strategy.risk_management">
                  <h5>⚠️ 风险管理</h5>
                  <div class="risk-grid">
                    <div class="risk-item">
                      <span class="risk-label">单笔最大亏损：</span>
                      <span class="risk-value">{{ (selectedRecommendation.trading_strategy.risk_management.max_loss_per_trade * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="risk-item">
                      <span class="risk-label">单日最大亏损：</span>
                      <span class="risk-value">{{ (selectedRecommendation.trading_strategy.risk_management.max_daily_loss * 100).toFixed(1) }}%</span>
                    </div>
                    <div class="risk-item" v-if="selectedRecommendation.trading_strategy.risk_management.volatility_adjustment">
                      <span class="risk-label">波动率调整：</span>
                      <span class="risk-value">启用</span>
                    </div>
                  </div>
                </div>

                <!-- 策略理由 -->
                <div class="strategy-rationale" v-if="selectedRecommendation.trading_strategy.strategy_rationale && selectedRecommendation.trading_strategy.strategy_rationale.length > 0">
                  <h5>💡 策略理由</h5>
                  <ul class="rationale-list">
                    <li v-for="(reason, idx) in selectedRecommendation.trading_strategy.strategy_rationale" :key="idx">{{ reason }}</li>
                  </ul>
                </div>
              </div>

              <!-- 实时表现追踪 -->
              <div class="performance-section" v-if="selectedRecommendation.performance">
                <div class="performance-header">
                  <h4>实时表现追踪</h4>
                  <span class="performance-status" :class="`status-${selectedRecommendation.performance.status || 'tracking'}`">
                    {{ getPerformanceStatusText(selectedRecommendation.performance.status) }}
                  </span>
                </div>
                <div class="performance-timeline">
                  <div class="timeline-item" v-if="selectedRecommendation.performance.return_1h !== null && selectedRecommendation.performance.return_1h !== undefined">
                    <span class="timeline-label">1h后：</span>
                    <span class="timeline-value" :class="getPerformanceClass(selectedRecommendation.performance.return_1h)">
                      {{ formatPercent(selectedRecommendation.performance.return_1h) }}
                    </span>
                  </div>
                  <div class="timeline-item" v-if="selectedRecommendation.performance.return_24h !== null && selectedRecommendation.performance.return_24h !== undefined">
                    <span class="timeline-label">24h后：</span>
                    <span class="timeline-value" :class="getPerformanceClass(selectedRecommendation.performance.return_24h)">
                      {{ formatPercent(selectedRecommendation.performance.return_24h) }}
                    </span>
                  </div>
                  <div class="timeline-item" v-if="selectedRecommendation.performance.return_7d !== null && selectedRecommendation.performance.return_7d !== undefined">
                    <span class="timeline-label">7天后：</span>
                    <span class="timeline-value" :class="getPerformanceClass(selectedRecommendation.performance.return_7d)">
                      {{ formatPercent(selectedRecommendation.performance.return_7d) }}
                    </span>
                  </div>
                  <div class="timeline-item" v-if="selectedRecommendation.performance.return_30d !== null && selectedRecommendation.performance.return_30d !== undefined">
                    <span class="timeline-label">30天后：</span>
                    <span class="timeline-value" :class="getPerformanceClass(selectedRecommendation.performance.return_30d)">
                      {{ formatPercent(selectedRecommendation.performance.return_30d) }}
                    </span>
                  </div>
                </div>
                <div class="performance-metrics" v-if="selectedRecommendation.performance.max_gain || selectedRecommendation.performance.max_drawdown">
                  <div class="metric-item" v-if="selectedRecommendation.performance.max_gain">
                    <span class="metric-label">最大涨幅：</span>
                    <span class="metric-value positive">{{ formatPercent(selectedRecommendation.performance.max_gain) }}</span>
                  </div>
                  <div class="metric-item" v-if="selectedRecommendation.performance.max_drawdown">
                    <span class="metric-label">最大回撤：</span>
                    <span class="metric-value negative">{{ formatPercent(selectedRecommendation.performance.max_drawdown) }}</span>
                  </div>
                </div>
              </div>

              <!-- 价格预测 -->
              <PricePrediction v-if="selectedRecommendation.prediction" :prediction="selectedRecommendation.prediction" />

              <!-- 交易信号和策略 -->
              <div class="trading-strategy-section" v-if="selectedRecommendation.prediction && selectedRecommendation.prediction.trading_strategy">
                <div class="strategy-header">
                  <h4>📈 交易策略</h4>
                </div>
                <div class="strategy-content">
                  <div class="strategy-item">
                    <span class="strategy-label">策略类型：</span>
                    <span class="strategy-value" :class="getStrategyClass(selectedRecommendation.prediction.trading_strategy.strategy_type)">
                      {{ getStrategyText(selectedRecommendation.prediction.trading_strategy.strategy_type) }}
                    </span>
                  </div>
                  <div class="strategy-item">
                    <span class="strategy-label">入场区间：</span>
                    <span class="strategy-value">
                      ${{ formatNumber(selectedRecommendation.prediction.trading_strategy.entry_zone.min) }} -
                      ${{ formatNumber(selectedRecommendation.prediction.trading_strategy.entry_zone.max) }}
                    </span>
                  </div>
                  <div class="strategy-item" v-if="selectedRecommendation.prediction.trading_strategy.exit_targets.length > 0">
                    <span class="strategy-label">目标价格：</span>
                    <span class="strategy-value positive">
                      ${{ formatNumber(selectedRecommendation.prediction.trading_strategy.exit_targets[0].avg) }}
                    </span>
                  </div>
                  <div class="strategy-item" v-if="selectedRecommendation.prediction.trading_strategy.stop_loss_levels.length > 0">
                    <span class="strategy-label">止损价格：</span>
                    <span class="strategy-value negative">
                      ${{ formatNumber(selectedRecommendation.prediction.trading_strategy.stop_loss_levels[0].level) }}
                    </span>
                  </div>
                  <div class="strategy-item">
                    <span class="strategy-label">建议仓位：</span>
                    <span class="strategy-value">
                      {{ (selectedRecommendation.prediction.trading_strategy.position_sizing.adjusted_position * 100).toFixed(1) }}%
                    </span>
                  </div>
                  <div class="strategy-item">
                    <span class="strategy-label">风险收益比：</span>
                    <span class="strategy-value" :class="selectedRecommendation.prediction.trading_strategy.risk_management.risk_reward_ratio >= 2 ? 'positive' : 'neutral'">
                      1:{{ selectedRecommendation.prediction.trading_strategy.risk_management.risk_reward_ratio.toFixed(1) }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- 技术指标 -->
              <div class="technical-section" v-if="selectedRecommendation.technical">
                <div class="technical-header">
                  <h4>技术指标</h4>
                </div>
                <div class="technical-metrics">
                  <!-- RSI -->
                  <div class="technical-item">
                    <span class="technical-label">RSI：</span>
                    <span class="technical-value" :class="getRSIClass(selectedRecommendation.technical.rsi)">
                      {{ selectedRecommendation.technical.rsi?.toFixed(2) || '-' }}
                    </span>
                    <span class="technical-hint" v-if="selectedRecommendation.technical.rsi">
                      <span v-if="selectedRecommendation.technical.rsi > 70">(超买)</span>
                      <span v-else-if="selectedRecommendation.technical.rsi < 30">(超卖)</span>
                      <span v-else>(正常)</span>
                    </span>
                  </div>
                  <!-- MACD -->
                  <div class="technical-item">
                    <span class="technical-label">MACD：</span>
                    <span class="technical-value">{{ selectedRecommendation.technical.macd?.toFixed(4) || '-' }}</span>
                  </div>
                  <div class="technical-item">
                    <span class="technical-label">信号线：</span>
                    <span class="technical-value">{{ selectedRecommendation.technical.macd_signal?.toFixed(4) || '-' }}</span>
                  </div>
                  <!-- 布林带 -->
                  <div class="technical-item" v-if="selectedRecommendation.technical.bb_position !== undefined">
                    <span class="technical-label">布林带位置：</span>
                    <span class="technical-value">{{ (selectedRecommendation.technical.bb_position * 100).toFixed(1) }}%</span>
                    <span class="technical-hint">
                      <span v-if="selectedRecommendation.technical.bb_position < 0.2">(接近下轨)</span>
                      <span v-else-if="selectedRecommendation.technical.bb_position > 0.8">(接近上轨)</span>
                      <span v-else>(正常)</span>
                    </span>
                  </div>
                  <!-- KDJ -->
                  <div class="technical-item" v-if="selectedRecommendation.technical.k !== undefined">
                    <span class="technical-label">KDJ：</span>
                    <span class="technical-value">
                      K:{{ selectedRecommendation.technical.k?.toFixed(1) || '-' }} 
                      D:{{ selectedRecommendation.technical.d?.toFixed(1) || '-' }} 
                      J:{{ selectedRecommendation.technical.j?.toFixed(1) || '-' }}
                    </span>
                  </div>
                  <!-- 均线 -->
                  <div class="technical-item" v-if="selectedRecommendation.technical.ma5 !== undefined && selectedRecommendation.technical.ma5 > 0">
                    <span class="technical-label">均线：</span>
                    <span class="technical-value">
                      MA5:{{ selectedRecommendation.technical.ma5?.toFixed(2) || '-' }}
                      MA20:{{ selectedRecommendation.technical.ma20?.toFixed(2) || '-' }}
                    </span>
                  </div>
                  <!-- 成交量 -->
                  <div class="technical-item" v-if="selectedRecommendation.technical.volume_ratio !== undefined">
                    <span class="technical-label">成交量比率：</span>
                    <span class="technical-value" :class="selectedRecommendation.technical.volume_ratio > 1 ? 'positive' : 'negative'">
                      {{ selectedRecommendation.technical.volume_ratio?.toFixed(2) || '-' }}x
                    </span>
                  </div>
                  <!-- 支撑位/阻力位 -->
                  <div class="technical-item" v-if="selectedRecommendation.technical.support_level !== undefined && selectedRecommendation.technical.support_level > 0">
                    <span class="technical-label">支撑位：</span>
                    <span class="technical-value">{{ selectedRecommendation.technical.support_level?.toFixed(2) || '-' }}</span>
                    <span class="technical-hint" v-if="selectedRecommendation.technical.distance_to_support !== undefined">
                      (距离{{ selectedRecommendation.technical.distance_to_support?.toFixed(1) }}%)
                    </span>
                  </div>
                  <div class="technical-item" v-if="selectedRecommendation.technical.resistance_level !== undefined && selectedRecommendation.technical.resistance_level > 0">
                    <span class="technical-label">阻力位：</span>
                    <span class="technical-value">{{ selectedRecommendation.technical.resistance_level?.toFixed(2) || '-' }}</span>
                    <span class="technical-hint" v-if="selectedRecommendation.technical.distance_to_resistance !== undefined">
                      (距离{{ selectedRecommendation.technical.distance_to_resistance?.toFixed(1) }}%)
                    </span>
                  </div>
                  <!-- 趋势 -->
                  <div class="technical-item">
                    <span class="technical-label">趋势：</span>
                    <span class="technical-value" :class="getTrendClass(selectedRecommendation.technical.trend)">
                      {{ getTrendText(selectedRecommendation.technical.trend) }}
                    </span>
                  </div>
                </div>

                <!-- 交易信号详情 -->
                <div class="trading-signal-section" v-if="selectedRecommendation.technical.trading_signal">
                  <div class="signal-header">
                    <h5>🎯 交易信号</h5>
                  </div>
                  <div class="signal-content">
                    <div class="signal-item">
                      <span class="signal-label">交易信号：</span>
                      <span class="signal-value" :class="getSignalClass(selectedRecommendation.technical.trading_signal.signal)">
                        {{ getSignalText(selectedRecommendation.technical.trading_signal.signal) }}
                      </span>
                    </div>
                    <div class="signal-item">
                      <span class="signal-label">信号强度：</span>
                      <span class="signal-value" :class="getSignalStrengthClass(selectedRecommendation.technical.trading_signal.strength)">
                        {{ selectedRecommendation.technical.trading_signal.strength.toFixed(1) }}%
                      </span>
                    </div>
                    <div class="signal-item" v-if="selectedRecommendation.technical.trading_signal.signal !== 'HOLD'">
                      <span class="signal-label">建议入场：</span>
                      <span class="signal-value">
                        ${{ formatNumber(selectedRecommendation.technical.trading_signal.entry_price) }}
                      </span>
                    </div>
                    <div class="signal-item" v-if="selectedRecommendation.technical.trading_signal.stop_loss > 0">
                      <span class="signal-label">止损价格：</span>
                      <span class="signal-value negative">
                        ${{ formatNumber(selectedRecommendation.technical.trading_signal.stop_loss) }}
                      </span>
                    </div>
                    <div class="signal-item" v-if="selectedRecommendation.technical.trading_signal.take_profit > 0">
                      <span class="signal-label">止盈价格：</span>
                      <span class="signal-value positive">
                        ${{ formatNumber(selectedRecommendation.technical.trading_signal.take_profit) }}
                      </span>
                    </div>
                    <div class="signal-item">
                      <span class="signal-label">风险等级：</span>
                      <span class="signal-value" :class="getRiskLevelClass(selectedRecommendation.technical.position_management.risk_level)">
                        {{ getRiskLevelText(selectedRecommendation.technical.position_management.risk_level) }}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
  </div>

  <!-- 回测分析标签页 -->
  <BacktestView v-if="activeTab === 'backtest'" />

  <!-- 模拟交易标签页 -->
  <SimulationView v-if="activeTab === 'simulation'" />

</template>

<script setup>
import { ref, onMounted, computed, defineAsyncComponent } from 'vue'
import { api } from '../api/api.js'

// 懒加载组件以提高初始加载性能
const BacktestView = defineAsyncComponent(() => import('./RecommendationAnalysis.vue'))
const SimulationView = defineAsyncComponent(() => import('./Simulation.vue'))
const PricePrediction = defineAsyncComponent(() => import('../components/PricePrediction.vue'))
const LineChart = defineAsyncComponent(() => import('../components/LineChart.vue'))

const activeTab = ref('recommendations')
const kind = ref('spot')
const limit = ref(5)
const data = ref(null)
const loading = ref(false)
const performanceStats = ref(null)
const performanceMap = ref({}) // 推荐ID -> 表现数据
const selectedRecommendation = ref(null) // 选中的推荐详情

// 日期选择相关变量
const selectedDate = ref('')
const historicalData = ref(null)
const generating = ref(false)
const maxDate = ref('')
const minDate = ref('')
const availableDates = ref([]) // 可用的历史日期列表

// 性能优化：数据缓存
const dataCache = new Map()
const CACHE_DURATION = 15 * 60 * 1000 // 15分钟缓存（配合后台预生成）
const PERFORMANCE_CACHE_DURATION = 10 * 60 * 1000 // 10分钟性能数据缓存

// 性能优化：防抖函数
const debounce = (func, delay) => {
  let timeoutId
  return (...args) => {
    clearTimeout(timeoutId)
    timeoutId = setTimeout(() => func.apply(null, args), delay)
  }
}

// 合并推荐数据和表现数据
const enrichedRecommendations = computed(() => {
  console.log('计算enrichedRecommendations，data:', data.value)
  if (!data.value || !data.value.recommendations) return []
  const result = data.value.recommendations.map(rec => ({
    ...rec,
    performance: performanceMap.value[rec.id] || performanceMap.value[rec.symbol] || null
  }))
  console.log('enrichedRecommendations结果:', result.length, '项')
  return result
})

// 准备趋势图表数据


function formatTime(timeStr) {
  if (!timeStr) return '-'

  // 如果时间字符串包含 'Z' 或时区偏移，说明是UTC时间
  // 否则当作本地时间处理
  let date
  if (timeStr.includes('Z') || timeStr.includes('+') || timeStr.includes('-')) {
    // 已经是带时区的时间字符串，直接解析
    date = new Date(timeStr)
  } else {
    // 没有时区信息，当作UTC时间处理
    date = new Date(timeStr + 'Z')
  }

  // 格式化为北京时间显示
  return date.toLocaleString('zh-CN', {
    timeZone: 'Asia/Shanghai',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatUSD(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1e9) {
    return (value / 1e9).toFixed(2) + 'B'
  } else if (value >= 1e6) {
    return (value / 1e6).toFixed(2) + 'M'
  } else if (value >= 1e3) {
    return (value / 1e3).toFixed(2) + 'K'
  }
  return value.toFixed(2)
}

function formatVolume(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1e9) {
    return (value / 1e9).toFixed(2) + 'B'
  } else if (value >= 1e6) {
    return (value / 1e6).toFixed(2) + 'M'
  }
  return value.toFixed(2)
}

function getRiskLevelText(level) {
  const map = {
    'low': '低风险',
    'medium': '中风险',
    'high': '高风险'
  }
  return map[level] || '未知'
}

// 获取策略类型样式类
function getStrategyClass(strategyType) {
  const classMap = {
    'LONG': 'positive',
    'SHORT': 'negative',
    'RANGE': 'neutral'
  }
  return classMap[strategyType] || 'neutral'
}

// 获取策略类型文本
function getStrategyText(strategyType) {
  const textMap = {
    'LONG': '多头策略',
    'SHORT': '空头策略',
    'RANGE': '震荡策略'
  }
  return textMap[strategyType] || strategyType
}

// 获取信号样式类
function getSignalClass(signal) {
  const classMap = {
    'BUY': 'positive',
    'SELL': 'negative',
    'HOLD': 'neutral'
  }
  return classMap[signal] || 'neutral'
}

// 获取信号文本
function getSignalText(signal) {
  const textMap = {
    'BUY': '买入',
    'SELL': '卖出',
    'HOLD': '观望'
  }
  return textMap[signal] || signal
}

// 获取信号强度样式类
function getSignalStrengthClass(strength) {
  if (strength >= 70) return 'positive'
  if (strength >= 40) return 'neutral'
  return 'negative'
}

// 获取风险等级样式类
function getRiskLevelClass(riskLevel) {
  const classMap = {
    'low': 'positive',
    'medium': 'neutral',
    'high': 'negative'
  }
  return classMap[riskLevel] || 'neutral'
}

function getRiskClass(risk) {
  if (!risk) return ''
  if (risk < 30) return 'risk-low'
  if (risk < 60) return 'risk-medium'
  return 'risk-high'
}

function getRSIClass(rsi) {
  if (!rsi) return ''
  if (rsi > 70) return 'rsi-overbought'
  if (rsi < 30) return 'rsi-oversold'
  return 'rsi-normal'
}

function getTrendText(trend) {
  const map = {
    'up': '上涨',
    'down': '下跌',
    'sideways': '震荡'
  }
  return map[trend] || '未知'
}

function getTrendClass(trend) {
  if (trend === 'up') return 'trend-up'
  if (trend === 'down') return 'trend-down'
  return 'trend-sideways'
}

function getPerformanceClass(value) {
  if (value === null || value === undefined) return ''
  return value >= 0 ? 'positive' : 'negative'
}

// 获取策略类型显示文本
function getStrategyTypeText(strategyType) {
  const textMap = {
    'LONG': '多头策略',
    'SHORT': '空头策略',
    'RANGE': '震荡策略'
  }
  return textMap[strategyType] || '未知策略'
}

// 获取买卖方向文本
function getTradingDirectionText(direction) {
  const textMap = {
    'long': '买入做多',
    'short': '卖出做空',
    'range': '区间交易',
    'LONG': '买入做多',
    'SHORT': '卖出做空',
    'RANGE': '区间交易'
  }
  return textMap[direction] || '观望'
}

// 获取市场环境文本
function getMarketConditionText(condition) {
  const textMap = {
    'bullish': '牛市环境',
    'bearish': '熊市环境',
    'neutral': '中性环境'
  }
  return textMap[condition] || '未知环境'
}

// 获取止损类型文本
function getStopLossTypeText(stopType) {
  const textMap = {
    'INITIAL': '初始止损',
    'TRAILING': '追踪止损',
    'MENTAL': '心理止损'
  }
  return textMap[stopType] || stopType
}

// 获取仓位策略文本
function getPositionStrategyText(strategy) {
  const textMap = {
    'FIXED': '固定仓位',
    'MARTINGALE': '马丁格尔',
    'ANTI_MARTINGALE': '反马丁格尔'
  }
  return textMap[strategy] || strategy
}

function formatPercent(value) {
  if (value === null || value === undefined) return '-'
  return (value >= 0 ? '+' : '') + value.toFixed(2) + '%'
}

function formatNumber(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else {
    return value.toFixed(6)
  }
}

function formatPrice(value) {
  if (value === null || value === undefined) return '-'
  if (value >= 1000000) {
    return (value / 1000000).toFixed(2) + 'M'
  } else if (value >= 1000) {
    return (value / 1000).toFixed(2) + 'K'
  } else {
    return value.toFixed(4)
  }
}

function getPerformanceStatusText(status) {
  const map = {
    'tracking': '追踪中',
    'completed': '已完成',
    'expired': '已过期'
  }
  return map[status] || '未知'
}

// 性能优化：防抖的性能统计加载
const debouncedLoadPerformanceStats = debounce(async function() {
  try {
    performanceStats.value = await api.getPerformanceStats({ days: 30 })
  } catch (error) {
    console.error('加载表现统计失败:', error)
  }
}, 300)


function viewDetail(rec) {
  selectedRecommendation.value = rec

  // 行为追踪：推荐查看
  import('@/utils/behaviorTracker.js').then(({ default: tracker }) => {
    const position = enrichedRecommendations.value.findIndex(r => r.id === rec.id)
    tracker.trackRecommendationView(rec, position + 1)
  })
}


function handleKindChange() {
  loadData(true)
}

// 处理日期选择
function handleDateChange() {
  if (selectedDate.value) {
    loadData(true)
  } else {
    // 如果清空日期，则加载实时推荐
    loadData(true)
  }
}

// 加载可用日期列表
async function loadAvailableDates() {
  try {
    const response = await api.getRecommendationTimeList({ kind: kind.value })
    if (response && response.dates) {
      availableDates.value = response.dates
      if (response.dates.length > 0) {
        maxDate.value = response.dates[0] // 最新的日期
        minDate.value = response.dates[response.dates.length - 1] // 最老的日期
        // 如果没有选择日期，默认选择最新的日期
        if (!selectedDate.value) {
          selectedDate.value = maxDate.value
        }
      }
    }
  } catch (error) {
    console.error('加载可用日期失败:', error)
  }
}




// 仅加载性能数据（用于缓存刷新）
// 异步加载推荐价格（在显示列表后）
async function loadRecommendationPrices() {
  if (!data.value || !data.value.recommendations) return

  const recommendations = data.value.recommendations.filter(rec =>
    rec.symbol && (!rec.data || !rec.data.price)
  )

  if (recommendations.length === 0) return

  console.log(`异步加载 ${recommendations.length} 个推荐的价格`)

  // 限制并发数量，避免过载
  const maxConcurrent = 2
  for (let i = 0; i < recommendations.length; i += maxConcurrent) {
    const batch = recommendations.slice(i, i + maxConcurrent)
    const pricePromises = batch.map(async (rec) => {
      try {
        // 使用价格历史API获取最新价格
        const history = await api.getMarketPriceHistory({
          symbol: rec.symbol,
          days: 1,
          interval: 'daily'
        })
        if (history && history.length > 0) {
          // 获取最新的价格
          const latestPrice = history[history.length - 1].close
          if (latestPrice && rec.data) {
            rec.data.price = latestPrice
          }
        }
      } catch (error) {
        console.warn(`获取 ${rec.symbol} 价格失败:`, error)
      }
    })

    await Promise.allSettled(pricePromises)

    // 小延迟避免过载
    if (i + maxConcurrent < recommendations.length) {
      await new Promise(resolve => setTimeout(resolve, 200))
    }
  }
}

async function loadPerformanceDataOnly() {
  const performanceCacheKey = `performance_stats_${kind.value}`
  const now = Date.now()

  try {
    // 加载表现统计数据
    const statsRes = await api.getPerformanceStats({ days: 30 })

    // 处理表现统计
    performanceStats.value = statsRes

    // 更新缓存
    const performanceData = {
      performanceStats: performanceStats.value,
      timestamp: now
    }
    dataCache.set(performanceCacheKey, performanceData)

    console.log('性能数据刷新完成')
  } catch (error) {
    console.warn('性能数据刷新失败:', error)
  }
}

async function loadRecommendationPerformance(recommendationId, symbol) {
  try {
    const perf = await api.getRecommendationPerformance({ 
      recommendation_id: recommendationId,
      symbol: symbol 
    })
    if (perf && perf.performances && perf.performances.length > 0) {
      performanceMap.value[recommendationId] = perf.performances[0]
    }
  } catch (error) {
    console.error('加载推荐表现失败:', error)
  }
}

// 性能优化：带缓存的数据加载函数
async function loadData(refresh = false) {
  // 根据是否有选择日期决定加载方式
  if (selectedDate.value) {
    return loadHistoricalData()
  } else {
    return loadLiveData(refresh)
  }
}

// 加载实时推荐数据
async function loadLiveData(refresh = false) {
  // 生成缓存键
  const cacheKey = `recommendations_${kind.value}_${limit.value}`
  const performanceCacheKey = `performance_stats_${kind.value}`

  // 检查完整缓存（非刷新模式）
  if (!refresh && dataCache.has(cacheKey)) {
    const cached = dataCache.get(cacheKey)
    const now = Date.now()

    if (now - cached.timestamp < CACHE_DURATION) {
      // 使用缓存数据
      data.value = cached.data
      performanceStats.value = cached.performanceStats
      performanceMap.value = cached.performanceMap

      console.log('使用缓存推荐数据，立即显示')

      // 后台异步刷新性能数据（如果需要）
      const performanceCached = dataCache.get(performanceCacheKey)
      if (!performanceCached || (now - performanceCached.timestamp >= PERFORMANCE_CACHE_DURATION)) {
        console.log('后台刷新性能数据')
        loadPerformanceDataOnly().catch(err => console.warn('后台性能数据更新失败:', err))
      }

      // 异步加载价格信息
      loadRecommendationPrices().catch(err => console.warn('异步价格加载失败:', err))

      return
    } else {
      // 缓存过期，清理
      dataCache.delete(cacheKey)
    }
  }

  loading.value = true
  try {
    // 并行加载推荐列表和表现统计
    const [recommendationsRes, statsRes] = await Promise.allSettled([
      api.getCoinRecommendations({ kind: kind.value, limit: limit.value, refresh }),
      api.getPerformanceStats({ days: 30 })
    ])
    
    // 处理推荐列表
    if (recommendationsRes.status === 'fulfilled') {
      data.value = recommendationsRes.value
      console.log('API响应数据:', data.value)
      console.log('推荐数量:', data.value?.recommendations?.length || 0)
    } else {
      throw new Error('加载推荐列表失败: ' + (recommendationsRes.reason?.message || '未知错误'))
    }
    
    // 处理表现统计
    if (statsRes.status === 'fulfilled') {
      performanceStats.value = statsRes.value
    } else {
      console.warn('加载表现统计失败:', statsRes.reason)
    }
    
    // 智能加载推荐的表现数据（只加载缺失的数据）
    if (data.value && data.value.recommendations) {
      const recommendations = data.value.recommendations

      // 找出需要加载表现数据的推荐（没有缓存或缓存过期）
      const needPerformanceData = recommendations.filter(rec => {
        if (rec.id && performanceMap.value[rec.id]) {
          return false // 已有ID缓存
        }
        if (rec.symbol && performanceMap.value[rec.symbol]) {
          return false // 已有symbol缓存
        }
        return true // 需要加载
      })

      console.log(`需要加载表现数据: ${needPerformanceData.length}/${recommendations.length} 条`)

      if (needPerformanceData.length > 0) {
        const recommendationIds = needPerformanceData
          .map(rec => rec.id)
          .filter(id => id != null && id !== undefined)

        if (recommendationIds.length > 0) {
          // 使用批量查询接口
          try {
            const batchPerf = await api.getBatchRecommendationPerformance({
              recommendation_ids: recommendationIds
            })
            if (batchPerf?.performances) {
              // 批量更新 performanceMap
              Object.keys(batchPerf.performances).forEach(id => {
                performanceMap.value[parseInt(id)] = batchPerf.performances[id]
              })
            }
          } catch (error) {
            console.warn('批量加载表现数据失败，降级到并行单个查询:', error)
            // 降级到并行单个查询，限制并发数量避免过载
            const maxConcurrent = 3 // 降低并发数
            const performancePromises = needPerformanceData
              .filter(rec => rec.id) // 只处理有ID的记录
              .map(rec => loadRecommendationPerformance(rec.id, rec.symbol))

            // 分批执行，避免一次性发起太多请求
            for (let i = 0; i < performancePromises.length; i += maxConcurrent) {
              const batch = performancePromises.slice(i, i + maxConcurrent)
              await Promise.allSettled(batch)
              // 小延迟避免服务器过载
              if (i + maxConcurrent < performancePromises.length) {
                await new Promise(resolve => setTimeout(resolve, 150))
              }
            }
          }
        }

        // 处理没有ID的记录（按symbol查询）
        const symbolOnlyRecs = needPerformanceData.filter(rec => !rec.id && rec.symbol)
        if (symbolOnlyRecs.length > 0) {
          const maxConcurrent = 3
          const performancePromises = symbolOnlyRecs.map(rec =>
            api.getRecommendationPerformance({ symbol: rec.symbol, limit: 1 })
              .then(perf => {
                if (perf?.performances?.length > 0) {
                  performanceMap.value[rec.symbol] = perf.performances[0]
                }
              })
              .catch(err => console.error(`加载推荐 ${rec.symbol} 表现失败:`, err))
          )

          // 分批执行
          for (let i = 0; i < performancePromises.length; i += maxConcurrent) {
            const batch = performancePromises.slice(i, i + maxConcurrent)
            await Promise.allSettled(batch)
            if (i + maxConcurrent < performancePromises.length) {
              await new Promise(resolve => setTimeout(resolve, 150))
            }
          }
        }
      } else {
        console.log('所有推荐的表现数据已在缓存中，跳过加载')
      }
    }

    // 存储到缓存
    const now = Date.now()
    if (data.value || performanceStats.value) {
      // 存储完整数据缓存
      dataCache.set(cacheKey, {
        data: data.value,
        performanceStats: performanceStats.value,
        performanceMap: performanceMap.value,
        timestamp: now
      })

      // 单独缓存性能数据（用于独立刷新）
      const performanceCacheKey = `performance_stats_${kind.value}`
      dataCache.set(performanceCacheKey, {
        performanceStats: performanceStats.value,
        timestamp: now
      })

      // 限制缓存大小
      if (dataCache.size > 15) { // 增加缓存大小限制
        // 清理最老的缓存
        const entries = Array.from(dataCache.entries())
          .sort((a, b) => a[1].timestamp - b[1].timestamp)
        const oldestKey = entries[0][0]
        dataCache.delete(oldestKey)
      }
    }

    // 异步加载推荐价格（不阻塞界面显示）- 暂时禁用，可能导致加载卡住
    // loadRecommendationPrices().catch(err => console.warn('异步价格加载失败:', err))

  } catch (error) {
    console.error('加载智能推荐失败:', error)

    // 特殊处理超时错误
    if (error.status === 408 || error.message.includes('超时')) {
      alert('加载推荐数据超时，请稍后重试或刷新页面。如果问题持续，请联系管理员。')
    } else if (error.status >= 500) {
      alert('服务器内部错误，请稍后重试或联系管理员。')
    } else if (error.status === 401 || error.status === 403) {
      alert('权限验证失败，请重新登录。')
      // 可以在这里添加跳转到登录页面的逻辑
    } else {
      alert('加载智能推荐失败: ' + (error.message || '未知错误'))
    }
  } finally {
    console.log('设置loading为false，数据:', data.value)
    loading.value = false
  }
}

// 加载历史推荐数据
async function loadHistoricalData() {
  if (!selectedDate.value) {
    alert('请选择日期')
    return
  }

  loading.value = true
  try {
    // 加载历史推荐数据
    const response = await api.getHistoricalRecommendations({
      kind: kind.value,
      date: selectedDate.value,
      includePerformance: true,
      page: 1,
      page_size: 50 // 增加页面大小以获取更多历史数据
    })

    // 处理历史数据格式，使其与实时数据兼容
    if (response && response.recommendations) {
      data.value = {
        recommendations: response.recommendations,
        generated_at: selectedDate.value + ' 00:00:00', // 使用选择的日期作为生成时间
        cached: false
      }

      // 清理之前的表现数据
      performanceMap.value = {}

      // 如果包含表现数据，直接使用
      if (response.performances) {
        Object.keys(response.performances).forEach(id => {
          performanceMap.value[parseInt(id)] = response.performances[id]
        })
      }
    } else {
      data.value = { recommendations: [] }
    }

    // 对于历史数据，我们不需要表现统计概览
    performanceStats.value = null

  } catch (error) {
    console.error('加载历史推荐失败:', error)

    // 特殊处理：如果没有历史数据，尝试生成
    if (error.status === 404) {
      const confirmGenerate = confirm(`没有找到 ${selectedDate.value} 的历史推荐数据，是否要为此日期生成推荐？`)
      if (confirmGenerate) {
        await generateHistoricalData()
        return
      }
    }

    alert('加载历史推荐失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 生成指定日期的历史推荐数据
async function generateHistoricalData() {
  if (!selectedDate.value) return

  generating.value = true
  try {
    await api.generateRecommendationsForDate({
      kind: kind.value,
      date: selectedDate.value,
      limit: limit.value
    })

    // 生成完成后重新加载数据
    await loadHistoricalData()

    alert('历史推荐数据生成完成')
  } catch (error) {
    console.error('生成历史推荐失败:', error)
    alert('生成历史推荐失败: ' + (error.message || '未知错误'))
  } finally {
    generating.value = false
  }
}


onMounted(() => {
  // 初始化为空（实时推荐）
  selectedDate.value = ''
  loadAvailableDates() // 加载可用的历史日期范围
  loadData() // 加载数据
})
</script>

<style scoped>
.recommendations-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 简化版推荐卡片 */
.recommendation-card-compact {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 12px;
  background: white;
  transition: all 0.2s;
  cursor: pointer;
}

.recommendation-card-compact:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  border-color: #3b82f6;
}

.card-header-compact {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.score-and-strategy {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
}

.strategy-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
}

.strategy-long {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.strategy-short {
  background: #fef2f2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.strategy-range {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}

.symbol-info-compact h3 {
  margin: 0;
  font-size: 18px;
  color: #333;
}

.symbol-info-compact .symbol-pair {
  font-size: 12px;
  color: #666;
}

.total-score-compact {
  margin-left: auto;
  text-align: center;
}

.card-body-compact {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.compact-info {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-label {
  font-size: 12px;
  color: #666;
}

.info-value {
  font-size: 14px;
  font-weight: bold;
}

.scores-compact {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.score-chip {
  padding: 4px 8px;
  background: #f0f9ff;
  border-radius: 4px;
  font-size: 12px;
  color: #1e40af;
}

.detail-btn {
  padding: 8px 16px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  align-self: flex-start;
}

.detail-btn:hover {
  background: #2563eb;
}

.risk-badge-small {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: bold;
}

/* 原有推荐卡片样式（保留用于模态框） */
.recommendation-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 20px;
  background: #fff;
  transition: box-shadow 0.2s;
}

.recommendation-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.rank-badge {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: bold;
  font-size: 18px;
  color: #fff;
}

.rank-1 { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.rank-2 { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.rank-3 { background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%); }
.rank-4 { background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%); }
.rank-5 { background: linear-gradient(135deg, #fa709a 0%, #fee140 100%); }

.symbol-info {
  flex: 1;
}

.symbol-info h3 {
  margin: 0;
  font-size: 24px;
  font-weight: bold;
}

.symbol-pair {
  color: #666;
  font-size: 14px;
}

.total-score {
  text-align: center;
}

.score-value {
  font-size: 32px;
  font-weight: bold;
  color: #667eea;
}

.score-label {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

.card-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.score-breakdown {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
  gap: 12px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
}

.score-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.score-item .score-label {
  font-size: 12px;
  color: #666;
}

.score-item .score-value {
  font-size: 18px;
  font-weight: bold;
  color: #333;
}

.data-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.data-item {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 4px;
}

.data-label {
  color: #666;
  font-size: 14px;
}

.data-value {
  font-weight: bold;
  font-size: 14px;
}

.data-value.positive {
  color: #10b981;
}

.data-value.negative {
  color: #ef4444;
}

.reasons {
  margin-top: 8px;
}

.reasons-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #333;
}

.reasons-list {
  margin: 0;
  padding-left: 20px;
  color: #666;
}

.reasons-list li {
  margin-bottom: 4px;
}

/* 推荐算法表现概览 */
.performance-overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 16px;
  margin-top: 16px;
}

.overview-item {
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
  text-align: center;
}

.overview-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 8px;
}

.overview-value {
  font-size: 20px;
  font-weight: bold;
  color: #333;
}

.overview-value.positive {
  color: #10b981;
}

.overview-value.negative {
  color: #ef4444;
}

/* 实时表现徽章 */
.performance-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px 12px;
  background: #f0f9ff;
  border-radius: 6px;
  margin-left: 12px;
}

.performance-label {
  font-size: 11px;
  color: #666;
  margin-bottom: 4px;
}

.performance-value {
  font-size: 16px;
  font-weight: bold;
}

/* 实时表现追踪区域 */
.performance-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e0e0e0;
}

.performance-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.performance-header h4 {
  margin: 0;
  color: #333;
  font-size: 1em;
}

.performance-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.85em;
  font-weight: bold;
}

.performance-status.status-tracking {
  background-color: #dbeafe;
  color: #1e40af;
}

.performance-status.status-completed {
  background-color: #d4edda;
  color: #155724;
}

.performance-status.status-expired {
  background-color: #f3f4f6;
  color: #6b7280;
}

.performance-timeline {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.timeline-item {
  display: flex;
  flex-direction: column;
  padding: 8px;
  background: #f8f9fa;
  border-radius: 4px;
}

.timeline-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}

.timeline-value {
  font-size: 14px;
  font-weight: bold;
}

.performance-metrics {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
}

.metric-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.metric-label {
  font-size: 12px;
  color: #666;
}

.metric-value {
  font-size: 14px;
  font-weight: bold;
}

/* 风险评级样式 */
.risk-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e0e0e0;
}

.risk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.risk-header h4 {
  margin: 0;
  color: #333;
  font-size: 1em;
}

.risk-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 0.85em;
  font-weight: bold;
}

.risk-badge.risk-low {
  background-color: #d4edda;
  color: #155724;
}

.risk-badge.risk-medium {
  background-color: #fff3cd;
  color: #856404;
}

.risk-badge.risk-high {
  background-color: #f8d7da;
  color: #721c24;
}

/* 模态框样式 */
.modal-overlay {
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
  overflow-y: auto;
  padding: 20px;
}

.modal-content {
  background: #fff;
  border-radius: 8px;
  max-width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  position: relative;
}

.modal-content.large {
  min-width: 800px;
  max-width: 95%;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e0e0e0;
  position: sticky;
  top: 0;
  background: #fff;
  z-index: 10;
}

.modal-header h2 {
  margin: 0;
  font-size: 20px;
  color: #333;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: #f0f0f0;
  border-radius: 50%;
  cursor: pointer;
  font-size: 24px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #666;
}

.close-btn:hover {
  background: #e0e0e0;
}

.modal-body {
  padding: 20px;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e0e0e0;
}

.generation-info {
  margin-left: auto;
}

.generation-time,
.generation-time-compact {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #666;
}

.time-label {
  font-weight: 500;
  color: #888;
}

.time-value {
  color: #666;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.risk-metrics {
  margin-bottom: 12px;
}

.risk-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.risk-label {
  font-size: 0.9em;
  color: #666;
}

.risk-value {
  font-size: 1.1em;
  font-weight: bold;
}

.risk-value.risk-low {
  color: #28a745;
}

.risk-value.risk-medium {
  color: #ffc107;
}

.risk-value.risk-high {
  color: #dc3545;
}

.risk-breakdown {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  margin-top: 8px;
}

.risk-metric {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.risk-metric .metric-label {
  font-size: 0.8em;
  color: #999;
  margin-bottom: 4px;
}

.risk-metric .metric-value {
  font-size: 1em;
  font-weight: bold;
  color: #333;
}

.risk-warnings {
  margin-top: 12px;
  padding: 12px;
  background-color: #fff3cd;
  border-left: 3px solid #ffc107;
  border-radius: 4px;
}

.warning-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #856404;
}

.warning-list {
  margin: 0;
  padding-left: 20px;
  list-style-type: disc;
}

.warning-list li {
  margin-bottom: 4px;
  font-size: 0.9em;
  color: #856404;
}

/* 技术指标样式 */
.technical-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e0e0e0;
}

.technical-header {
  margin-bottom: 12px;
}

.technical-header h4 {
  margin: 0;
  color: #333;
  font-size: 1em;
}

.technical-metrics {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.technical-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.technical-label {
  font-size: 0.9em;
  color: #666;
  font-weight: 500;
}

.technical-value {
  font-size: 1em;
  font-weight: bold;
  color: #333;
}

.technical-hint {
  font-size: 0.8em;
  color: #999;
}

/* 交易信号和策略样式 */
.trading-signal-compact {
  margin-bottom: 8px;
}

.signal-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.8em;
  font-weight: bold;
}

.signal-badge.positive {
  background: linear-gradient(135deg, #d4edda, #c3e6cb);
  color: #155724;
  border: 1px solid #c3e6cb;
}

.signal-badge.negative {
  background: linear-gradient(135deg, #f8d7da, #f5c6cb);
  color: #721c24;
  border: 1px solid #f5c6cb;
}

.signal-badge.neutral {
  background: linear-gradient(135deg, #e2e3e5, #d6d8db);
  color: #383d41;
  border: 1px solid #d6d8db;
}

.signal-text {
  font-weight: bold;
}

.signal-strength {
  opacity: 0.8;
}

.trading-strategy-section,
.trading-signal-section {
  margin-top: 20px;
  padding: 16px;
  background: linear-gradient(135deg, #f8f9fa 0%, #e9ecef 100%);
  border-radius: 8px;
  border: 1px solid #dee2e6;
}

.strategy-header h4,
.signal-header h5 {
  margin: 0 0 12px 0;
  font-size: 14px;
  color: #495057;
  font-weight: 600;
}

.strategy-content,
.signal-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 8px;
}

.strategy-item,
.signal-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 6px;
  border: 1px solid #e9ecef;
}

.strategy-label,
.signal-label {
  font-size: 11px;
  color: #6c757d;
  font-weight: 500;
}

.strategy-value,
.signal-value {
  font-size: 13px;
  font-weight: 600;
  color: #495057;
}

.strategy-value.positive,
.signal-value.positive {
  color: #28a745;
}

.strategy-value.negative,
.signal-value.negative {
  color: #dc3545;
}

.strategy-value.neutral,
.signal-value.neutral {
  color: #6c757d;
}

.rsi-overbought {
  color: #dc3545;
}

.rsi-oversold {
  color: #28a745;
}

.rsi-normal {
  color: #333;
}

.trend-up {
  color: #28a745;
}

.trend-down {
  color: #dc3545;
}

.trend-sideways {
  color: #666;
}

/* 骨架屏样式 */
.skeleton-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.skeleton-card {
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  padding: 16px;
  background: white;
}

.skeleton-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.skeleton-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
}

.skeleton-content {
  flex: 1;
}

.skeleton-score {
  width: 60px;
  height: 60px;
  border-radius: 8px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
}

.skeleton-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.skeleton-line {
  height: 16px;
  border-radius: 4px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
}

.skeleton-chips {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

.skeleton-chip {
  width: 80px;
  height: 24px;
  border-radius: 4px;
  background: linear-gradient(90deg, #f0f0f0 25%, #e0e0e0 50%, #f0f0f0 75%);
  background-size: 200% 100%;
  animation: skeleton-loading 1.5s ease-in-out infinite;
}

@keyframes skeleton-loading {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

/* 空状态样式 */
.empty-state {
  text-align: center;
  padding: 60px 20px;
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

.empty-text {
  font-size: 18px;
  color: #333;
  margin-bottom: 8px;
}

.empty-hint {
  font-size: 14px;
  color: #666;
  margin-bottom: 24px;
}

/* 控制面板样式 */
.control-group {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.control-group label {
  font-weight: 500;
  color: #374151;
  white-space: nowrap;
}

.control-group select,
.control-group input[type="date"] {
  padding: 6px 10px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
  font-size: 14px;
  min-width: 120px;
}

.control-group select:focus,
.control-group input[type="date"]:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

/* 标题和日期选择器布局 */
.header-with-date {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.header-with-date h2 {
  margin: 0;
  font-size: 20px;
  color: #1f2937;
  font-weight: 600;
  white-space: nowrap;
}

.date-picker-wrapper {
  display: flex;
  align-items: center;
}

.date-picker {
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  font-size: 14px;
  color: #374151;
  cursor: pointer;
  transition: border-color 0.2s, box-shadow 0.2s;
  min-width: 140px;
}

.date-picker:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.date-picker:hover {
  border-color: #9ca3af;
}

.date-picker::placeholder {
  color: #9ca3af;
  font-style: italic;
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
</style>

