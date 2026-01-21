<template>
  <div class="scheduled-order-detail">
    <!-- 加载状态 -->
    <div v-if="loading" class="loading">
      <div class="loading-spinner"></div>
      <p>加载订单详情中...</p>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="error-message">
      <h3>❌ 加载失败</h3>
      <p>{{ error }}</p>
      <button class="btn btn-primary" @click="loadOrderDetail">重试</button>
      <button class="btn btn-outline" @click="goBack">返回列表</button>
    </div>

    <!-- 订单详情 -->
    <div v-else-if="order" class="order-detail-content">
      <!-- 页面标题和概览卡片 -->
      <div class="page-header">
        <div class="header-main">
          <div class="title-section">
            <h1 class="page-title">订单详情</h1>
            <div class="order-badge">
              <span class="order-id-badge">#{{ order.id }}</span>
              <span class="symbol-badge">{{ order.symbol }}</span>
              <span class="exchange-badge" :class="{ testnet: order.testnet }">
                {{ order.exchange === 'binance_futures' ? '币安期货' : order.exchange }}
                {{ order.testnet ? '(测试网)' : '(正式网)' }}
              </span>
            </div>
          </div>
        </div>
        <div class="header-actions">
          <button class="btn btn-outline" @click="goBack">
            返回列表
          </button>
        </div>
      </div>

      <!-- 操作面板 -->
      <div class="action-panel" v-if="order && canOperateOrder()">
        <div class="panel-header">
          <h3>订单操作</h3>
        </div>
        <div class="panel-content">
          <div class="action-buttons">
            <!-- 取消订单 -->
            <button
              v-if="canCancelOrder()"
              class="btn btn-warning"
              @click="cancelOrder"
              :disabled="loading"
            >
              取消订单
            </button>

            <!-- 删除订单 -->
            <button
              class="btn btn-danger"
              @click="deleteOrder"
              :disabled="loading"
            >
              删除订单
            </button>

            <!-- 手动平仓 -->
            <button
              v-if="canClosePosition()"
              class="btn btn-primary"
              @click="closePosition"
              :disabled="loading"
            >
              手动平仓
            </button>
          </div>
        </div>
      </div>

      <!-- 订单基本信息卡片 -->
      <div class="info-cards">
        <!-- 交易信息卡片 -->
        <div class="info-card">
          <div class="card-header">
            <h3>交易信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="info-label">交易对</span>
              <span class="info-value symbol-value">{{ order.symbol }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">操作类型</span>
              <span class="info-value operation-badge" :class="getOperationClass(order.side, order.reduce_only)">
                {{ getOperationType(order.side, order.reduce_only) }}
              </span>
            </div>
            <div class="info-row">
              <span class="info-label">订单类型</span>
              <span class="info-value">{{ order.order_type === 'MARKET' ? '市价单' : '限价单' }}</span>
            </div>
            <div class="info-row" v-if="order.price">
              <span class="info-label">委托价格</span>
              <span class="info-value price-value">${{ order.price }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">委托数量</span>
              <span class="info-value quantity-value" :class="order.adjusted_quantity && order.adjusted_quantity !== order.quantity ? 'adjusted-quantity' : ''">
                {{ order.quantity }}
              </span>
              <span v-if="order.adjusted_quantity && order.adjusted_quantity !== order.quantity" class="adjusted-info">
                → {{ order.adjusted_quantity }}
              </span>
            </div>

            <!-- 名义价值 -->
            <div v-if="order.nominal_value" class="info-row">
              <span class="info-label">名义价值</span>
              <span class="info-value nominal-value">${{ formatNumber(order.nominal_value) }}</span>
              <span class="field-desc">合约总价值</span>
            </div>

            <!-- 保证金金额 -->
            <div v-if="order.margin_amount" class="info-row">
              <span class="info-label">保证金金额</span>
              <span class="info-value margin-amount">${{ formatNumber(order.margin_amount) }}</span>
              <span class="field-desc">用户实际投入</span>
            </div>

            <!-- 成交金额 -->
            <div v-if="order.deal_amount" class="info-row">
              <span class="info-label">成交金额</span>
              <span class="info-value deal-amount">${{ formatNumber(order.deal_amount) }}</span>
              <span class="field-desc">名义价值</span>
            </div>

            <!-- 计算说明 -->
            <div v-if="order.calculation_note" class="info-row">
              <span class="info-label">计算说明</span>
              <span class="info-value calculation-note">{{ order.calculation_note }}</span>
            </div>
          </div>
        </div>

        <!-- 配置信息卡片 -->
        <div class="info-card">
          <div class="card-header">
            <h3>配置信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="info-label">交易所</span>
              <span class="info-value exchange-value">
                {{ order.exchange === 'binance_futures' ? '币安期货' : order.exchange }}
                <span class="network-badge" :class="{ testnet: order.testnet }">
                  {{ order.testnet ? '测试网' : '正式网' }}
                </span>
              </span>
            </div>
            <div class="info-row" v-if="order.leverage">
              <span class="info-label">杠杆倍数</span>
              <span class="info-value leverage-value">{{ order.leverage }}x</span>
            </div>
            <div class="info-row">
              <span class="info-label">减仓模式</span>
              <span class="info-value" :class="order.reduce_only ? 'reduce-only-yes' : 'reduce-only-no'">
                {{ order.reduce_only ? '开启' : '关闭' }}
              </span>
            </div>
            <div class="info-row">
              <span class="info-label">仓位状态</span>
              <span class="info-value position-status" :class="getPositionStatusClass(order)">
                {{ getPositionStatusText(order) }}
              </span>
            </div>
            <div class="info-row">
              <span class="info-label">订单ID</span>
              <span class="info-value order-id-value">#{{ order.id }}</span>
            </div>
          </div>
        </div>

        <!-- 时间信息卡片 -->
        <div class="info-card">
          <div class="card-header">
            <h3>时间信息</h3>
          </div>
          <div class="card-content">
            <div class="info-row">
              <span class="info-label">创建时间</span>
              <span class="info-value time-value">{{ toLocal(order.created_at) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">触发时间</span>
              <span class="info-value time-value trigger-time">{{ toLocal(order.trigger_time) }}</span>
            </div>
            <div class="info-row" v-if="order.updated_at">
              <span class="info-label">最后更新</span>
              <span class="info-value time-value">{{ toLocal(order.updated_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 关联订单信息 -->
      <div v-if="hasRelatedOrders()" class="related-orders-section">
        <div class="section-header">
          <div class="section-icon">🔗</div>
          <h3>关联订单</h3>
          <div class="related-badge">交易链</div>
        </div>

        <div class="related-orders-content">
          <!-- 父订单（开仓订单） -->
          <div v-if="order.related_orders?.parent_order" class="related-order-card parent-order">
            <div class="order-header">
              <h4>开仓订单</h4>
              <span class="order-link" @click="viewOrder(order.related_orders.parent_order.id)">
                查看详情 →
              </span>
            </div>
            <div class="order-info">
              <div class="info-item">
                <span class="label">订单ID</span>
                <span class="value">#{{ order.related_orders.parent_order.id }}</span>
              </div>
              <div class="info-item">
                <span class="label">操作类型</span>
                <span class="value operation-type">{{ order.related_orders.parent_order.operation_type }}</span>
              </div>
              <div class="info-item">
                <span class="label">状态</span>
                <span class="value status" :class="getRelatedOrderStatusClass(order.related_orders.parent_order.status)">
                  {{ getRelatedOrderStatusText(order.related_orders.parent_order.status) }}
                </span>
              </div>
              <div class="info-item" v-if="order.related_orders.parent_order.executed_qty">
                <span class="label">成交数量</span>
                <span class="value">{{ formatNumber(order.related_orders.parent_order.executed_qty) }}</span>
              </div>
              <div class="info-item" v-if="order.related_orders.parent_order.avg_price">
                <span class="label">成交均价</span>
                <span class="value">${{ formatNumber(order.related_orders.parent_order.avg_price) }}</span>
              </div>
            </div>
          </div>

          <!-- 当前订单 -->
          <div class="related-order-card current-order">
            <div class="order-header">
              <h4>当前订单</h4>
              <span class="current-badge">正在查看</span>
            </div>
            <div class="order-info">
              <div class="info-item">
                <span class="label">订单ID</span>
                <span class="value">#{{ order.id }}</span>
              </div>
              <div class="info-item">
                <span class="label">操作类型</span>
                <span class="value operation-type">{{ getOperationType(order.side, order.reduce_only) }}</span>
              </div>
              <div class="info-item">
                <span class="label">状态</span>
                <span class="value status" :class="getRelatedOrderStatusClass(order.status)">
                  {{ getRelatedOrderStatusText(order.status) }}
                </span>
              </div>
              <div class="info-item" v-if="order.executed_quantity">
                <span class="label">成交数量</span>
                <span class="value">{{ formatNumber(order.executed_quantity) }}</span>
              </div>
              <div class="info-item" v-if="order.avg_price">
                <span class="label">成交均价</span>
                <span class="value">${{ formatNumber(order.avg_price) }}</span>
              </div>
            </div>
          </div>

          <!-- 平仓订单列表 -->
          <div v-if="order.related_orders?.close_orders?.length > 0" class="close-orders-group">
            <h4>平仓订单 ({{ order.related_orders.close_orders.length }})</h4>
            <div class="close-orders-list">
              <div
                v-for="closeOrder in order.related_orders.close_orders"
                :key="closeOrder.id"
                class="related-order-card close-order"
              >
                <div class="order-header">
                  <h5>平仓订单 #{{ closeOrder.id }}</h5>
                  <span class="order-link" @click="viewOrder(closeOrder.id)">
                    查看详情 →
                  </span>
                </div>
                <div class="order-info">
                  <div class="info-item">
                    <span class="label">操作类型</span>
                    <span class="value operation-type">{{ closeOrder.operation_type }}</span>
                  </div>
                  <div class="info-item">
                    <span class="label">状态</span>
                    <span class="value status" :class="getRelatedOrderStatusClass(closeOrder.status)">
                      {{ getRelatedOrderStatusText(closeOrder.status) }}
                    </span>
                  </div>
                  <div class="info-item" v-if="closeOrder.executed_qty">
                    <span class="label">成交数量</span>
                    <span class="value">{{ formatNumber(closeOrder.executed_qty) }}</span>
                  </div>
                  <div class="info-item" v-if="closeOrder.avg_price">
                    <span class="label">成交均价</span>
                    <span class="value">${{ formatNumber(closeOrder.avg_price) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 一键三连配置 -->
      <div v-if="order.bracket_enabled" class="bracket-section">
        <div class="section-header">
          <h3>一键三连配置</h3>
          <div class="bracket-badge">Bracket订单</div>
        </div>

        <div class="bracket-config">
          <!-- 止盈设置 -->
          <div class="bracket-panel profit-panel">
            <div class="panel-header">
              <h4>止盈设置</h4>
            </div>
            <div class="panel-content">
              <div v-if="calculateActualPercent(order).tpPercent !== null || order.tp_percent" class="config-item">
                <span class="config-label">止盈百分比</span>
                <span class="config-value profit-value"
                      :class="{ 'adjusted': calculateActualPercent(order).tpPercent && calculateActualPercent(order).tpPercent !== order.tp_percent }">
                  +{{ formatPercent(calculateActualPercent(order).tpPercent || order.tp_percent) }}%
                  <span v-if="calculateActualPercent(order).tpPercent && calculateActualPercent(order).tpPercent !== order.tp_percent"
                        class="original-value">(原: +{{ formatPercent(order.tp_percent) }}%)</span>
                </span>
              </div>
              <div v-if="order.tp_price" class="config-item">
                <span class="config-label">止盈价格</span>
                <span class="config-value profit-value">${{ order.tp_price }}</span>
              </div>
              <div v-else-if="!order.tp_percent" class="config-item">
                <span class="config-note">未设置止盈</span>
              </div>
            </div>
          </div>

          <!-- 止损设置 -->
          <div class="bracket-panel loss-panel">
            <div class="panel-header">
              <h4>止损设置</h4>
            </div>
            <div class="panel-content">
              <div v-if="calculateActualPercent(order).slPercent !== null || order.sl_percent" class="config-item">
                <span class="config-label">止损百分比</span>
                <span class="config-value loss-value"
                      :class="{ 'adjusted': calculateActualPercent(order).slPercent && calculateActualPercent(order).slPercent !== order.sl_percent }">
                  -{{ formatPercent(calculateActualPercent(order).slPercent || order.sl_percent) }}%
                  <span v-if="calculateActualPercent(order).slPercent && calculateActualPercent(order).slPercent !== order.sl_percent"
                        class="original-value">(原: -{{ formatPercent(order.sl_percent) }}%)</span>
                </span>
              </div>
              <div v-if="order.sl_price" class="config-item">
                <span class="config-label">止损价格</span>
                <span class="config-value loss-value">${{ order.sl_price }}</span>
              </div>
              <div v-else-if="!order.sl_percent" class="config-item">
                <span class="config-note">未设置止损</span>
              </div>
            </div>
          </div>

          <!-- 价格模式 -->
          <div class="bracket-panel mode-panel">
            <div class="panel-header">
              <h4>价格模式</h4>
            </div>
            <div class="panel-content">
              <div class="config-item">
                <span class="config-label">工作类型</span>
                <span class="config-value mode-value">
                  {{ order.working_type === 'MARK_PRICE' ? '标记价格' : '合约价格' }}
                </span>
              </div>
              <div class="mode-description">
                <small>
                  {{ order.working_type === 'MARK_PRICE'
                     ? '基于标记价格（更稳定，适合高杠杆）'
                     : '基于合约最新价格（更实时，适合快速交易）' }}
                </small>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行日志 -->
      <div v-if="order.result" class="execution-section">
        <div class="section-header">
          <div class="section-icon">📝</div>
          <h3>执行日志</h3>
          <div class="log-badge">系统日志</div>
        </div>

        <div class="execution-content">
          <div class="log-container">
            <pre class="log-text">{{ order.result }}</pre>
          </div>
        </div>
      </div>

      <!-- 交易所订单状态 -->
      <div v-if="order.order_status" class="exchange-section">
        <div class="section-header">
          <div class="section-icon">🏦</div>
          <h3>交易所状态</h3>
          <div class="exchange-badge" :class="getExchangeStatusClass(order.order_status.status)">
            {{ getStatusText(order.order_status.status) }}
          </div>
        </div>

        <div class="exchange-content">
          <div v-if="order.order_status.error" class="error-alert">
            <div class="alert-icon">⚠️</div>
            <div class="alert-content">
              <h4>交易所错误</h4>
              <p>{{ order.order_status.error }}</p>
            </div>
          </div>

          <div v-else class="exchange-metrics">
            <!-- 订单标识 -->
            <div class="metric-group">
              <h4>订单标识</h4>
              <div class="metric-grid">
                <div class="metric-item">
                  <span class="metric-label">客户端订单ID</span>
                  <span class="metric-value client-id">{{ order.order_status.client_order_id || '无' }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">交易所订单ID</span>
                  <span class="metric-value exchange-id">{{ order.order_status.order_id || '无' }}</span>
                </div>
              </div>
            </div>

            <!-- 成交信息 -->
            <div v-if="order.order_status.executed_qty || order.order_status.avg_price" class="metric-group">
              <h4>成交信息</h4>
              <div class="metric-grid">
                <div v-if="order.order_status.executed_qty" class="metric-item">
                  <span class="metric-label">已成交数量</span>
                  <span class="metric-value executed-qty">{{ formatNumber(order.order_status.executed_qty) }}</span>
                </div>
                <div v-if="order.order_status.avg_price" class="metric-item">
                  <span class="metric-label">平均成交价</span>
                  <span class="metric-value avg-price">${{ formatNumber(order.order_status.avg_price) }}</span>
                </div>
                <div v-if="order.order_status.executed_qty && order.order_status.avg_price" class="metric-item">
                  <span class="metric-label">成交金额</span>
                  <span class="metric-value total-value">${{ getTotalValue() }}</span>
                </div>
              </div>
            </div>

            <!-- 订单属性 -->
            <div class="metric-group">
              <h4>订单属性</h4>
              <div class="metric-grid">
                <div class="metric-item">
                  <span class="metric-label">交易方向</span>
                  <span class="metric-value" :class="order.order_status.side === 'BUY' ? 'buy-direction' : 'sell-direction'">
                    {{ order.order_status.side === 'BUY' ? '买入' : '卖出' }}
                  </span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">订单类型</span>
                  <span class="metric-value order-type">{{ order.order_status.type || 'MARKET' }}</span>
                </div>
                <div v-if="order.order_status.time" class="metric-item">
                  <span class="metric-label">更新时间</span>
                  <span class="metric-value update-time">{{ toLocal(order.order_status.time) }}</span>
                </div>
              </div>
            </div>

            <!-- 进度指示器 -->
            <div v-if="order.order_status.executed_qty && (order.adjusted_quantity || order.quantity)" class="progress-section">
              <div class="progress-header">
                <span class="progress-label">成交进度</span>
                <span class="progress-text">{{ getProgressPercentage() }}%</span>
              </div>
              <div class="progress-bar">
                <div
                  class="progress-fill"
                  :style="{ width: getProgressWidth() + '%' }"
                  :class="{ 'full': getProgressPercentage() >= 100 }"
                ></div>
              </div>
              <div class="progress-details">
                <span>{{ formatNumber(order.order_status.executed_qty) }} / {{ formatNumber(order.adjusted_quantity || order.quantity) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 财务分析面板 -->
      <div v-if="shouldShowProfitInfo() && profitInfo" class="finance-section">
        <div class="section-header">
          <h3>{{ getProfitSectionTitle() }}</h3>
          <div class="data-source-badge" :class="profitInfo.data_source === 'exchange' ? 'reliable' : 'estimated'">
            {{ getDataSourceText() }}
          </div>
        </div>

        <div v-if="profitInfo.error" class="error-panel">
          <div class="error-icon">⚠️</div>
          <div class="error-content">
            <h4>数据获取失败</h4>
            <p>{{ profitInfo.error }}</p>
          </div>
        </div>

        <div v-else class="finance-content">
          <!-- 主要指标 -->
          <div class="metrics-grid">
            <!-- 盈亏总额 -->
            <div class="metric-card main-metric">
              <div class="metric-header">
                <span class="metric-label">{{ getTotalPnlLabel() }}</span>
              </div>
              <div class="metric-value" :class="profitInfo.total_pnl >= 0 ? 'profit' : 'loss'">
                {{ profitInfo.total_pnl >= 0 ? '+' : '' }}{{ formatNumber(profitInfo.total_pnl) }} USDT
              </div>
              <div v-if="profitInfo.pnl_percentage !== undefined" class="metric-subvalue" :class="profitInfo.pnl_percentage >= 0 ? 'profit' : 'loss'">
                {{ profitInfo.pnl_percentage >= 0 ? '+' : '' }}{{ formatNumber(profitInfo.pnl_percentage) }}%
              </div>
              <div v-if="isTradeCompleted()" class="final-result-badge">
                📊 {{ getCompletionText() }}
              </div>
            </div>

            <!-- 持仓信息 -->
            <div class="metric-card position-card">
              <div class="metric-header">
                <span class="metric-label">持仓概览</span>
              </div>
              <div class="position-info">
                <div class="position-type" :class="profitInfo.position_type === 'long' ? 'long' : 'short'">
                  {{ profitInfo.position_type === 'long' ? '多头持仓' : '空头持仓' }}
                </div>
                <div class="position-size">
                  数量: {{ formatNumber(profitInfo.quantity) }}
                </div>
                <div class="position-value">
                  市值: ${{ formatNumber(profitInfo.position_value) }}
                </div>
                <div v-if="profitInfo.nominal_value" class="position-nominal">
                  名义价值: ${{ formatNumber(profitInfo.nominal_value) }}
                </div>
                <div v-if="profitInfo.margin_amount" class="position-margin">
                  保证金: ${{ formatNumber(profitInfo.margin_amount) }}
                </div>
                <div v-if="profitInfo.leverage" class="position-leverage">
                  杠杆: {{ profitInfo.leverage }}x
                </div>
              </div>
            </div>
          </div>

          <!-- 价格对比 -->
          <div class="price-comparison">
            <h4>价格对比</h4>
            <div class="price-grid">
              <div class="price-item">
                <div class="price-label">开仓价格</div>
                <div class="price-value entry-price">${{ formatNumber(profitInfo.entry_price) }}</div>
              </div>
              <div class="price-item">
                <div class="price-label">当前价格</div>
                <div class="price-value current-price">${{ formatNumber(profitInfo.current_price) }}</div>
                <div class="price-change" :class="profitInfo.current_price >= profitInfo.entry_price ? 'up' : 'down'">
                  {{ profitInfo.current_price >= profitInfo.entry_price ? '↗' : '↘' }}
                  {{ formatNumber(Math.abs(profitInfo.current_price - profitInfo.entry_price)) }}
                </div>
              </div>
            </div>
          </div>

          <!-- 详细指标 -->
          <div class="detailed-metrics">
            <h4>详细指标</h4>
            <div class="metrics-list">
              <!-- 已实现利润 -->
              <div v-if="profitInfo.realized_pnl !== undefined && profitInfo.realized_pnl !== 0" class="metric-row">
                <span class="metric-name">已实现盈亏</span>
                <span class="metric-value" :class="profitInfo.realized_pnl >= 0 ? 'profit' : 'loss'">
                  {{ profitInfo.realized_pnl >= 0 ? '+' : '' }}{{ formatNumber(profitInfo.realized_pnl) }} USDT
                </span>
                <span class="metric-desc">{{ order.reduce_only ? '相对于开仓价格的收益' : '基于平仓订单计算' }}</span>
              </div>

              <!-- 未实现利润 -->
              <div v-if="profitInfo.unrealized_pnl !== undefined && (profitInfo.actual_position_status !== 'closed' || profitInfo.unrealized_pnl !== 0)" class="metric-row">
                <span class="metric-name">未实现盈亏</span>
                <span class="metric-value" :class="profitInfo.unrealized_pnl >= 0 ? 'profit' : 'loss'">
                  {{ profitInfo.unrealized_pnl >= 0 ? '+' : '' }}{{ formatNumber(profitInfo.unrealized_pnl) }} USDT
                </span>
                <span class="metric-desc">{{ profitInfo.actual_position_status === 'closed' ? '持仓已平' : '基于当前价格估算' }}</span>
              </div>

              <!-- 持仓状态说明 -->
              <div v-if="profitInfo.actual_position_status" class="metric-row status-row">
                <span class="metric-name">持仓状态</span>
                <span class="metric-value position-status" :class="getProfitPositionStatusClass(profitInfo.actual_position_status)">
                  {{ getPositionStatusTextFromProfitInfo(profitInfo.actual_position_status) }}
                </span>
                <span v-if="profitInfo.actual_position_amt" class="metric-desc">
                  数量: {{ formatNumber(profitInfo.actual_position_amt) }}
                </span>
              </div>
            </div>
          </div>

          <!-- 说明信息 -->
          <div v-if="profitInfo.note" class="note-section">
            <div class="note-icon">💡</div>
            <div class="note-content">
              <p>{{ profitInfo.note }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- 执行时间轴 -->
      <div class="timeline-section">
        <div class="section-header">
          <div class="section-icon">📅</div>
          <h3>执行时间轴</h3>
        </div>

        <div class="timeline-content">
          <div class="timeline">
            <!-- 创建订单 -->
            <div class="timeline-item">
              <div class="timeline-marker created">
                <span class="marker-icon">📝</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">订单创建</div>
                <div class="timeline-time">{{ toLocal(order.created_at) }}</div>
                <div class="timeline-desc">订单已创建，等待执行</div>
              </div>
            </div>

            <!-- 触发执行 -->
            <div class="timeline-item">
              <div class="timeline-marker triggered">
                <span class="marker-icon">⏰</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">触发执行</div>
                <div class="timeline-time">{{ toLocal(order.trigger_time) }}</div>
                <div class="timeline-desc">到达预设时间，开始执行订单</div>
              </div>
            </div>

            <!-- 处理中 -->
            <div v-if="['processing', 'success', 'filled', 'completed', 'failed'].includes(order.status)" class="timeline-item">
              <div class="timeline-marker processing">
                <span class="marker-icon">⚙️</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">开始处理</div>
                <div class="timeline-time">处理中...</div>
                <div class="timeline-desc">系统正在处理订单请求</div>
              </div>
            </div>

            <!-- 提交交易所 -->
            <div v-if="['success', 'filled', 'completed'].includes(order.status)" class="timeline-item">
              <div class="timeline-marker submitted">
                <span class="marker-icon">📤</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">提交交易所</div>
                <div class="timeline-time">已提交</div>
                <div class="timeline-desc">订单已发送到交易所，等待确认</div>
              </div>
            </div>

            <!-- 交易所状态 -->
            <div v-if="order.order_status" class="timeline-item">
              <div class="timeline-marker" :class="getExchangeTimelineClass(order.order_status.status)">
                <span class="marker-icon">{{ getExchangeTimelineIcon(order.order_status.status) }}</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">交易所状态</div>
                <div class="timeline-time">{{ getStatusText(order.order_status.status) }}</div>
                <div class="timeline-desc">
                  <span v-if="order.order_status.executed_qty">已成交: {{ formatNumber(order.order_status.executed_qty) }}</span>
                  <span v-if="order.order_status.avg_price">均价: ${{ formatNumber(order.order_status.avg_price) }}</span>
                </div>
              </div>
            </div>

            <!-- 最终结果 -->
            <div class="timeline-item final">
              <div class="timeline-marker" :class="order.status">
                <span class="marker-icon">•</span>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">执行结果</div>
                <div class="timeline-time">{{ getSystemStatusText(order.status) }}</div>
                <div class="timeline-desc">
                  {{ getFinalResultDescription(order.status) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/api.js'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const error = ref('')
const order = ref(null)
const profitInfo = ref(null)

const orderId = ref(route.params.id)

// 加载订单详情
async function loadOrderDetail() {
  loading.value = true
  error.value = ''

  try {
    const response = await api.getScheduledOrderDetail(orderId.value)
    order.value = response

    // 如果订单已成交，获取盈利信息
    if (response.profit_info) {
      profitInfo.value = response.profit_info
    }
  } catch (e) {
    error.value = e?.message || '加载订单详情失败'
    console.error('加载订单详情失败:', e)
  } finally {
    loading.value = false
  }
}

// 取消订单
async function cancelOrder() {
  if (!confirm('确认取消该订单？')) return

  try {
    await api.cancelScheduledOrder(orderId.value)
    await loadOrderDetail() // 重新加载详情
  } catch (e) {
    alert('取消订单失败: ' + (e?.message || '未知错误'))
  }
}

// 删除订单
async function deleteOrder() {
  if (!confirm('确认删除该订单？此操作不可撤销。')) return

  try {
    await api.deleteScheduledOrder(orderId.value)
    router.push('/scheduled-orders') // 返回列表页
  } catch (e) {
    alert('删除订单失败: ' + (e?.message || '未知错误'))
  }
}

// 判断是否可以操作订单
function canOperateOrder() {
  if (!order.value) return false
  // 已完成或失败的订单不能操作
  return !['completed', 'failed'].includes(order.value.status)
}

// 判断是否可以取消订单
function canCancelOrder() {
  if (!order.value) return false
  // 只有待执行和执行中的订单可以取消
  return ['pending', 'processing', 'sent'].includes(order.value.status)
}

// 判断是否可以手动平仓
function canClosePosition() {
  if (!order.value) return false
  // 只有已成交的开仓订单可以手动平仓
  return !order.value.reduce_only &&
         ['filled', 'completed'].includes(order.value.status) &&
         getPositionStatusText(order.value) === '持仓中'
}

// 判断是否有关联订单
function hasRelatedOrders() {
  if (!order.value?.related_orders) return false
  return order.value.related_orders.parent_order ||
         (order.value.related_orders.close_orders && order.value.related_orders.close_orders.length > 0)
}

// 查看关联订单详情
function viewOrder(orderId) {
  // 使用编程式导航跳转到指定订单详情页
  router.push(`/orders/schedule/${orderId}`)
}

// 获取关联订单状态文本
function getRelatedOrderStatusText(status) {
  switch (status) {
    case 'pending': return '待执行'
    case 'processing': return '执行中'
    case 'filled':
    case 'completed': return '已完成'
    case 'failed': return '失败'
    case 'canceled':
    case 'cancelled': return '已取消'
    default: return status || '未知'
  }
}

// 获取关联订单状态样式类
function getRelatedOrderStatusClass(status) {
  switch (status) {
    case 'pending': return 'status-pending'
    case 'processing': return 'status-processing'
    case 'filled':
    case 'completed': return 'status-completed'
    case 'failed': return 'status-failed'
    case 'canceled':
    case 'cancelled': return 'status-cancelled'
    default: return 'status-unknown'
  }
}

// 获取总盈亏标签
function getTotalPnlLabel() {
  if (!profitInfo.value) return '总盈亏'

  const hasRealizedPnl = profitInfo.value.realized_pnl !== undefined && profitInfo.value.realized_pnl !== 0
  const isClosed = profitInfo.value.actual_position_status === 'closed'
  const isCloseOrder = order.value?.reduce_only
  // 检查是否有平仓订单关联（表示已被平仓）
  const hasCloseOrders = order.value?.related_orders?.close_orders && order.value.related_orders.close_orders.length > 0

  // 平仓订单的特殊处理
  if (isCloseOrder) {
    if (hasRealizedPnl) {
      return '平仓盈亏'
    } else {
      return '本次平仓'
    }
  }

  // 如果有平仓订单关联，说明已被平仓，应该显示总盈亏
  if (hasCloseOrders) {
    if (hasRealizedPnl) {
      return '交易总盈亏'
    } else {
      return '最终盈亏'
    }
  }

  if (isClosed && hasRealizedPnl) {
    return '交易总盈亏'
  } else if (isClosed) {
    return '最终盈亏'
  } else if (hasRealizedPnl) {
    return '累计盈亏'
  } else {
    return '当前盈亏'
  }
}

// 判断交易是否完成
function isTradeCompleted() {
  if (!order.value || !profitInfo.value) return false

  // 如果是平仓订单且已完成
  if (order.value.reduce_only && ['filled', 'completed'].includes(order.value.status)) {
    return true
  }

  // 如果是开仓订单且持仓已平仓
  if (!order.value.reduce_only && profitInfo.value.actual_position_status === 'closed') {
    return true
  }

  return false
}

// 获取完成状态文本
function getCompletionText() {
  if (order.value?.reduce_only) {
    return '平仓已完成'
  } else {
    return '交易已完成'
  }
}

// 从利润信息获取持仓状态文本
function getPositionStatusTextFromProfitInfo(status) {
  switch (status) {
    case 'closed': return '已平仓'
    case 'position_held': return '持仓中'
    case 'partially_closed': return '部分平仓'
    case 'no_position': return '无持仓'
    default: return '未知'
  }
}

// 获取持仓状态样式类（用于利润信息）
function getProfitPositionStatusClass(status) {
  switch (status) {
    case 'closed': return 'position-closed'
    case 'position_held': return 'position-open'
    case 'partially_closed': return 'position-partial'
    case 'no_position': return 'position-none'
    default: return 'position-unknown'
  }
}

// 判断是否应该显示利润信息
function shouldShowProfitInfo() {
  if (!order.value) return false

  // 已成交的订单
  if (['filled', 'completed', 'success'].includes(order.value.status)) {
    return true
  }

  // 有利润数据的订单（但不依赖actual_position_status来决定显示）
  return profitInfo.value && profitInfo.value.total_pnl !== undefined
}

// 获取利润信息区域的标题
function getProfitSectionTitle() {
  if (!profitInfo.value) return '财务分析'

  // 根据订单状态和类型来判断标题
  const isCompletedOrder = ['filled', 'completed'].includes(order.value?.status)
  const isCloseOrder = order.value?.reduce_only

  if (isCloseOrder && isCompletedOrder) {
    return '平仓结果'
  }

  if (isCompletedOrder) {
    return '交易结果'
  }

  // 默认显示实时分析
  return '实时分析'
}

// 获取数据源文本
function getDataSourceText() {
  if (!profitInfo.value) return '估算数据'

  const dataSource = profitInfo.value.data_source
  const isCompletedOrder = ['filled', 'completed'].includes(order.value?.status)

  if (isCompletedOrder) {
    return '最终数据'
  }

  return dataSource === 'exchange' ? '实时数据' : '估算数据'
}

// 手动平仓
async function closePosition() {
  if (!order.value) return

  const positionStatus = getPositionStatusText(order.value)
  const confirmMessage = `确认要手动平仓该订单吗？\n\n交易对: ${order.value.symbol}\n当前状态: ${positionStatus}\n\n系统将根据当前持仓自动创建平仓订单。`

  if (!confirm(confirmMessage)) return

  try {
    const result = await api.closePosition(orderId.value)
    alert(`平仓订单已创建！\n订单ID: ${result.close_order_id}\n交易对: ${result.symbol}\n方向: ${result.side}\n数量: ${result.quantity}`)
    await loadOrderDetail() // 重新加载详情
  } catch (e) {
    alert('创建平仓订单失败: ' + (e?.message || '未知错误'))
  }
}

// 返回列表
function goBack() {
  router.push('/scheduled-orders')
}

// 工具函数
function getOperationType(side, reduceOnly) {
  if (reduceOnly) {
    return side === 'BUY' ? '平空' : '平多'
  } else {
    return side === 'BUY' ? '开多' : '开空'
  }
}

function getOperationClass(side, reduceOnly) {
  if (reduceOnly) {
    return side === 'BUY' ? 'close-short' : 'close-long'
  } else {
    return side === 'BUY' ? 'open-long' : 'open-short'
  }
}

function getStatusClass(status) {
  // 对于已完成的订单，根据持仓状态返回不同的样式
  if (['filled', 'completed'].includes(status) && order.value && !order.value.reduce_only) {
    const positionStatus = getPositionStatusText(order.value)
    if (positionStatus === '已平仓') {
      return 'status-closed' // 已平仓状态使用特殊的样式
    }
  }

  // 其他状态保持原有逻辑
  switch (status) {
    case 'pending': return 'status-pending'
    case 'processing': return 'status-processing'
    case 'sent': return 'status-processing'
    case 'filled':
    case 'completed': return 'status-completed'
    case 'failed': return 'status-failed'
    case 'canceled':
    case 'cancelled': return 'status-cancelled'
    default: return 'status-unknown'
  }
}

// 交易所订单状态相关函数
function getStatusText(status) {
  switch (status) {
    case 'NEW': return '新建'
    case 'PARTIALLY_FILLED': return '部分成交'
    case 'FILLED': return '完全成交'
    case 'CANCELED': return '已取消'
    case 'PENDING_CANCEL': return '待取消'
    case 'REJECTED': return '已拒绝'
    case 'EXPIRED': return '已过期'
    default: return status || '未知'
  }
}

function getExchangeStatusClass(status) {
  switch (status) {
    case 'NEW': return 'status-pending'
    case 'PARTIALLY_FILLED': return 'status-processing'
    case 'FILLED': return 'status-completed'
    case 'CANCELED': case 'PENDING_CANCEL': return 'status-cancelled'
    case 'REJECTED': case 'EXPIRED': return 'status-failed'
    default: return 'status-unknown'
  }
}

function getSystemStatusText(status) {
  // 对于已完成的订单，根据订单类型和持仓状态显示更精确的状态
  if (['filled', 'completed'].includes(status)) {
    if (!order.value) return '已完成'

    const isReduceOnly = order.value.reduce_only

    if (isReduceOnly) {
      // 平仓订单：总是显示"已平仓"
      return '已平仓'
    } else {
      // 开仓订单：根据持仓状态显示
      const positionStatus = getPositionStatusText(order.value)

      // 如果有关联的平仓订单，说明已被平仓
      if (order.value.related_orders && order.value.related_orders.close_orders && order.value.related_orders.close_orders.length > 0) {
        return '已结束'
      }

      if (positionStatus === '已平仓') {
        return '已结束'  // 开仓订单被平仓后
      } else if (positionStatus === '持仓中') {
        return '开仓成功'  // 开仓订单当前持仓中
      } else {
        return '已完成'  // 其他情况
      }
    }
  }

  // 其他状态保持原有逻辑
  switch (status) {
    case 'pending': return '待执行'
    case 'processing': return '执行中'
    case 'sent': return '已发送'
    case 'failed': return '执行失败'
    case 'canceled':
    case 'cancelled': return '已取消'
    default: return '未知状态'
  }
}


function getActionStatusMessage(status) {
  switch (status) {
    case 'pending': return '订单等待执行，可随时取消'
    case 'processing': return '订单正在执行中，请耐心等待'
    case 'success': return '订单已提交到交易所，等待成交确认'
    case 'filled': return '订单已完全成交，持仓已建立'
    case 'completed': return '订单执行完成'
    case 'failed': return '订单执行失败，请检查错误信息'
    case 'cancelled': return '订单已被取消'
    default: return '未知状态'
  }
}

function getStatusDescription(status) {
  switch (status) {
    case 'pending': return '订单已创建，正在等待触发时间执行'
    case 'processing': return '订单正在本地处理中，包括参数验证和精度调整'
    case 'sent': return '订单已成功发送到币安API，等待确认'
    case 'filled': return '订单已在币安完全成交，所有委托数量都已执行'
    case 'completed': return '订单生命周期完成，所有相关操作已结束'
    case 'failed': return '订单执行过程中出现错误，已终止执行'
    case 'canceled':
    case 'cancelled': return '订单被主动取消或因其他原因停止执行'
    default: return '状态未知'
  }
}

function getStatusTooltip(status) {
  // 对于已完成的订单，根据持仓状态返回不同的提示
  if (['filled', 'completed'].includes(status) && order.value && !order.value.reduce_only) {
    const positionStatus = getPositionStatusText(order.value)
    if (positionStatus === '已平仓') {
      return '交易已完成并平仓'
    }
  }

  // 其他状态保持空字符串
  return ''
}

function getPositionStatusText(order) {
  // 优先使用实际持仓状态
  if (order.profit_info && order.profit_info.actual_position_status) {
    const actualStatus = order.profit_info.actual_position_status
    switch (actualStatus) {
      case 'closed': return '已平仓'
      case 'position_held': return '持仓中'
      case 'partially_closed': return '部分平仓'
      case 'no_position': return '无持仓'
    }
  }

  // 如果订单还未成交
  if (['pending', 'processing', 'cancelled', 'failed'].includes(order.status)) {
    return '未成交'
  }

  // 如果是平仓订单且已成交
  if (order.reduce_only && ['filled', 'completed', 'success'].includes(order.status)) {
    return '已平仓'
  }

  // 如果是开仓订单且已成交
  if (!order.reduce_only && ['filled', 'completed', 'success'].includes(order.status)) {
    return '持仓中'
  }

  return '未知'
}

function getPositionStatusClass(order) {
  const status = getPositionStatusText(order)
  switch (status) {
    case '已平仓': return 'position-closed'
    case '持仓中': return 'position-open'
    case '部分平仓': return 'position-partial'
    case '无持仓': return 'position-none'
    case '未成交': return 'position-pending'
    default: return 'position-unknown'
  }
}

function getExchangeTimelineClass(status) {
  switch (status) {
    case 'NEW': return 'pending'
    case 'PARTIALLY_FILLED': return 'processing'
    case 'FILLED': return 'completed'
    case 'CANCELED': case 'PENDING_CANCEL': return 'cancelled'
    case 'REJECTED': case 'EXPIRED': return 'failed'
    default: return 'unknown'
  }
}

function getExchangeTimelineIcon(status) {
  switch (status) {
    case 'NEW': return '🆕'
    case 'PARTIALLY_FILLED': return '📊'
    case 'FILLED': return '💰'
    case 'CANCELED': return '🚫'
    case 'PENDING_CANCEL': return '⏳'
    case 'REJECTED': return '❌'
    case 'EXPIRED': return '⏰'
    default: return '❓'
  }
}

function getFinalResultDescription(status) {
  switch (status) {
    case 'filled': return '订单完全成交，持仓已建立'
    case 'completed': return '订单执行完成'
    case 'failed': return '订单执行失败'
    case 'cancelled': return '订单已被取消'
    case 'success': return '订单已提交交易所'
    default: return '执行完成'
  }
}

function toLocal(timeStr) {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatNumber(num) {
  if (num === undefined || num === null) return '0.00'
  return Number(num).toLocaleString('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 8
  })
}

// 格式化百分比，保留5位小数
function formatPercent(num) {
  if (num === undefined || num === null) return '0.00000'
  return Number(num).toFixed(5)
}

// 计算成交进度百分比
function getProgressPercentage() {
  if (!order.value?.order_status?.executed_qty) {
    return 0
  }

  const executed = parseFloat(order.value.order_status.executed_qty) || 0
  // 对于bracket订单，使用调整后的数量；否则使用原始数量
  const totalStr = order.value.adjusted_quantity || order.value.quantity
  const total = parseFloat(totalStr) || 0

  if (total === 0) return 0

  const percentage = (executed / total) * 100
  return Math.round(Math.min(percentage, 100))
}

// 获取进度条宽度（百分比）
function getProgressWidth() {
  const percentage = getProgressPercentage()
  return Math.min(percentage, 100)
}

// 计算成交总金额
function getTotalValue() {
  if (!order.value?.order_status?.executed_qty || !order.value?.order_status?.avg_price) {
    return formatNumber(0)
  }

  const executed = parseFloat(order.value.order_status.executed_qty) || 0
  const avgPrice = parseFloat(order.value.order_status.avg_price) || 0

  return formatNumber(executed * avgPrice)
}

// 基于实际成交价格计算止盈止损百分比
function calculateActualPercent(order) {
  if (!order.avg_price || !order.tp_price && !order.sl_price) {
    return {
      tpPercent: order.actual_tp_percent || order.tp_percent,
      slPercent: order.actual_sl_percent || order.sl_percent
    }
  }

  const entryPrice = parseFloat(order.avg_price)
  if (!entryPrice || entryPrice <= 0) {
    return {
      tpPercent: order.actual_tp_percent || order.tp_percent,
      slPercent: order.actual_sl_percent || order.sl_percent
    }
  }

  const isLong = order.side === 'BUY'
  let tpPercent = null
  let slPercent = null

  // 计算止盈百分比
  if (order.tp_price) {
    const tpPrice = parseFloat(order.tp_price)
    if (tpPrice > 0) {
      if (isLong) {
        // 多头：止盈价格 > 入场价格
        if (tpPrice > entryPrice) {
          tpPercent = ((tpPrice - entryPrice) / entryPrice) * 100
        }
      } else {
        // 空头：止盈价格 < 入场价格
        if (tpPrice < entryPrice) {
          tpPercent = ((entryPrice - tpPrice) / entryPrice) * 100
        }
      }
    }
  }

  // 计算止损百分比
  if (order.sl_price) {
    const slPrice = parseFloat(order.sl_price)
    if (slPrice > 0) {
      if (isLong) {
        // 多头：止损价格 < 入场价格
        if (slPrice < entryPrice) {
          slPercent = ((entryPrice - slPrice) / entryPrice) * 100
        }
      } else {
        // 空头：止损价格 > 入场价格
        if (slPrice > entryPrice) {
          slPercent = ((slPrice - entryPrice) / entryPrice) * 100
        }
      }
    }
  }

  return {
    tpPercent: tpPercent !== null ? tpPercent : (order.actual_tp_percent || order.tp_percent),
    slPercent: slPercent !== null ? slPercent : (order.actual_sl_percent || order.sl_percent)
  }
}

onMounted(() => {
  loadOrderDetail()
})

// 监听路由参数变化，当订单ID改变时重新加载数据
watch(() => route.params.id, (newId, oldId) => {
  if (newId !== oldId) {
    orderId.value = newId
    loadOrderDetail()
  }
})
</script>

<style scoped>
.scheduled-order-detail {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.loading, .error-message {
  text-align: center;
  padding: 50px 20px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid #3498db;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin: 0 auto 20px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
  border-bottom: 1px solid #e5e7eb;
  padding-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #1f2937;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.detail-section {
  background: white;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.detail-section h3 {
  margin: 0 0 20px 0;
  color: #1f2937;
  font-size: 18px;
  font-weight: 600;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item label {
  font-weight: 500;
  color: #6b7280;
  font-size: 14px;
}

.info-item span {
  font-weight: 500;
  color: #1f2937;
  padding: 4px 8px;
  background: #f9fafb;
  border-radius: 4px;
  font-size: 14px;
}

.symbol {
  font-family: 'Monaco', 'Menlo', monospace;
  background: #dbeafe;
  color: #1e40af;
}

.trigger-time {
  background: #fef3c7;
  color: #92400e;
}

.adjusted-quantity {
  text-decoration: line-through;
  color: #9ca3af;
  background: #fef2f2;
}

.adjusted-info {
  color: #f59e0b;
  font-weight: 600;
  margin-left: 8px;
  background: #fffbeb;
  padding: 2px 6px;
  border-radius: 3px;
}

.bracket-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.order-status-info {
  background: #f8fafc;
  border-radius: 8px;
  padding: 16px;
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 16px;
}

.status-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.status-item label {
  font-size: 12px;
  color: #6b7280;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.status-item span {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.client-id, .order-id {
  font-family: 'Monaco', 'Menlo', monospace;
  background: #e5e7eb;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  word-break: break-all;
}

.executed-qty, .avg-price {
  color: #059669;
}

.status-error {
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 6px;
  padding: 12px;
}

.status-error p {
  color: #dc2626;
  margin: 0;
  font-size: 14px;
}

.bracket-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: #f9fafb;
  border-radius: 6px;
}

.bracket-item label {
  font-weight: 500;
  color: #6b7280;
}

.bracket-item span {
  font-weight: 600;
}

.bracket-item .profit {
  color: #16a34a;
}

.bracket-item .loss {
  color: #dc2626;
}

.result-info {
  background: #f9fafb;
  border-radius: 6px;
  padding: 16px;
}

.result-text {
  background: #1f2937;
  color: #f9fafb;
  padding: 12px;
  border-radius: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  overflow-x: auto;
}

.profit-analysis {
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border-radius: 8px;
  padding: 20px;
}

.profit-error {
  text-align: center;
  color: #dc2626;
  font-weight: 500;
}

.data-source-notice {
  text-align: center;
  margin-bottom: 16px;
}

.data-source-notice .reliable {
  color: #059669;
  font-weight: 600;
  background: #ecfdf5;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
}

.data-source-notice .estimated {
  color: #d97706;
  font-weight: 600;
  background: #fffbeb;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
}

.long-position {
  color: #059669;
  font-weight: 600;
  background: #ecfdf5;
  padding: 2px 8px;
  border-radius: 12px;
}

.short-position {
  color: #dc2626;
  font-weight: 600;
  background: #fef2f2;
  padding: 2px 8px;
  border-radius: 12px;
}

.entry-price, .current-price {
  font-weight: 600;
  color: #1f2937;
}

.position-value {
  font-weight: 600;
  color: #059669;
}

.profit {
  color: #059669;
  font-weight: 700;
}

.loss {
  color: #dc2626;
  font-weight: 700;
}

.profit-note {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.profit-note p {
  color: #6b7280;
  font-size: 14px;
  margin: 0;
  font-style: italic;
}

.profit-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 16px;
}

.profit-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: white;
  border-radius: 6px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.profit-item label {
  font-weight: 500;
  color: #6b7280;
  font-size: 14px;
}

.profit-item span {
  font-weight: 600;
  font-size: 14px;
}

.profit-item .price {
  color: #2563eb;
}

.profit-item .value {
  color: #7c3aed;
  font-family: 'Monaco', 'Menlo', monospace;
}

.profit-item .quantity {
  color: #059669;
}

.profit-item .profit {
  color: #16a34a;
}

.profit-item .loss {
  color: #dc2626;
}

.profit-note {
  text-align: center;
  color: #6b7280;
  font-style: italic;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

/* 状态样式 */
.status-pending { background: #fef3c7; color: #92400e; }
.status-processing { background: #dbeafe; color: #1e40af; }
.status-completed { background: #d1fae5; color: #065f46; }
.status-closed { background: #ecfdf5; color: #047857; border: 1px solid #a7f3d0; }
.status-failed { background: #fee2e2; color: #991b1b; }
.status-cancelled { background: #f3f4f6; color: #374151; }
.status-unknown { background: #f9fafb; color: #6b7280; }

/* 操作类型样式 */
.open-long { background: #dcfce7; color: #166534; }
.open-short { background: #fee2e2; color: #dc2626; }
.close-long { background: #fed7d7; color: #c53030; }
.close-short { background: #c6f6d5; color: #2f855a; }

/* 新的页面头部样式 */
.page-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 24px;
  border-radius: 12px;
  margin-bottom: 24px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.header-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.title-section h1 {
  margin: 0 0 8px 0;
  font-size: 28px;
  font-weight: 700;
}

.order-badge {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.order-id-badge {
  background: rgba(255, 255, 255, 0.2);
  padding: 4px 8px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
}

.symbol-badge {
  background: rgba(255, 255, 255, 0.15);
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 14px;
  font-weight: 600;
}

.exchange-badge {
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
}

.exchange-badge:not(.testnet) {
  background: rgba(34, 197, 94, 0.2);
  color: #dcfce7;
}

.exchange-badge.testnet {
  background: rgba(251, 191, 36, 0.2);
  color: #fef3c7;
}


.header-actions .btn {
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.3);
  color: white;
  backdrop-filter: blur(10px);
  transition: all 0.3s ease;
}

.header-actions .btn:hover {
  background: rgba(255, 255, 255, 0.25);
  transform: translateY(-1px);
}

.btn-icon {
  margin-right: 4px;
}

/* 信息卡片样式 */
.info-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.info-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  overflow: hidden;
  transition: all 0.3s ease;
}

.info-card:hover {
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  transform: translateY(-2px);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-bottom: 1px solid #e2e8f0;
}


.card-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #374151;
}

.card-content {
  padding: 20px 24px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #f3f4f6;
}

.info-row:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.info-value {
  font-size: 14px;
  color: #111827;
  font-weight: 600;
  text-align: right;
}

/* 新增字段样式 */
.nominal-value {
  color: #059669;
  font-weight: 700;
}

.margin-amount {
  color: #dc2626;
  font-weight: 700;
}

.deal-amount {
  color: #7c3aed;
  font-weight: 700;
}

.calculation-note {
  color: #6b7280;
  font-weight: 500;
  font-size: 13px;
  font-style: italic;
}

.field-desc {
  position: absolute;
  right: -120px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 11px;
  color: #9ca3af;
  font-weight: 400;
  font-style: italic;
  white-space: nowrap;
}

/* 特殊值样式 */
.symbol-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 16px;
  color: #1f2937;
}

.price-value, .quantity-value {
  font-family: 'Monaco', 'Menlo', monospace;
  color: #059669;
}

.leverage-value {
  color: #dc2626;
  font-weight: 700;
}

.operation-badge {
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.network-badge {
  display: inline-block;
  margin-left: 8px;
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 500;
}

.network-badge:not(.testnet) {
  background: #dcfce7;
  color: #166534;
}

.network-badge.testnet {
  background: #fef3c7;
  color: #92400e;
}

.order-id-value {
  font-family: 'Monaco', 'Menlo', monospace;
  color: #6b7280;
}

.time-value {
  font-size: 13px;
  color: #374151;
}

.trigger-time {
  color: #dc2626;
  font-weight: 600;
}

.reduce-only-yes {
  color: #dc2626;
}

.reduce-only-no {
  color: #059669;
}

/* 仓位状态样式 */
.position-status {
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
}

.position-closed {
  background: #dcfce7;
  color: #166534;
}

.position-open {
  background: #dbeafe;
  color: #1e40af;
}

.position-partial {
  background: #fef3c7;
  color: #92400e;
}

.position-none {
  background: #f3f4f6;
  color: #6b7280;
}

.position-pending {
  background: #f3f4f6;
  color: #374151;
}

.position-unknown {
  background: #fef3c7;
  color: #92400e;
}

.adjusted-quantity {
  text-decoration: line-through;
  color: #ef4444;
}

.adjusted-info {
  margin-left: 8px;
  padding: 2px 6px;
  background: #dbeafe;
  color: #1e40af;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
}

/* Bracket配置样式 */
.bracket-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}

.section-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #374151;
}

.bracket-badge {
  margin-left: auto;
  padding: 4px 12px;
  background: #dc2626;
  color: white;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.bracket-config {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  padding: 24px;
}

.bracket-panel {
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
  overflow: hidden;
  transition: all 0.3s ease;
}

.bracket-panel:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  background: white;
  border-bottom: 1px solid #e2e8f0;
}

.panel-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.panel-content {
  padding: 16px;
}

.config-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
}

.config-item:last-child {
  border-bottom: none;
}

.config-label {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.config-value {
  font-size: 14px;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
}

.profit-value {
  color: #059669;
}

.loss-value {
  color: #dc2626;
}

.adjusted {
  font-weight: 700;
}

.original-value {
  font-size: 12px;
  color: #9ca3af;
  font-weight: normal;
  text-decoration: none;
  margin-left: 8px;
}

.mode-value {
  color: #7c3aed;
}

.config-note {
  font-size: 13px;
  color: #9ca3af;
  font-style: italic;
  margin: 8px 0;
}

.mode-description {
  margin-top: 12px;
  padding: 12px;
  background: #f3f4f6;
  border-radius: 6px;
  border-left: 3px solid #7c3aed;
}

.mode-description small {
  color: #6b7280;
  line-height: 1.4;
}

/* 面板颜色区分 */
.profit-panel {
  border-top: 3px solid #059669;
}

.loss-panel {
  border-top: 3px solid #dc2626;
}

.mode-panel {
  border-top: 3px solid #7c3aed;
}

/* 财务分析样式 */
.finance-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.data-source-badge {
  margin-left: auto;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.data-source-badge.reliable {
  background: #dcfce7;
  color: #166534;
}

.data-source-badge.estimated {
  background: #fef3c7;
  color: #92400e;
}

.error-panel {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 24px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  margin: 24px;
}

.error-icon {
  font-size: 24px;
}

.error-content h4 {
  margin: 0 0 4px 0;
  color: #dc2626;
  font-size: 16px;
  font-weight: 600;
}

.error-content p {
  margin: 0;
  color: #991b1b;
}

.finance-content {
  padding: 24px;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.metric-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e2e8f0;
  transition: all 0.3s ease;
}

.metric-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}

.main-metric {
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 2px solid #0ea5e9;
}

.metric-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}


.metric-label {
  font-size: 14px;
  color: #6b7280;
  font-weight: 600;
}

.metric-value {
  font-size: 24px;
  font-weight: 700;
  font-family: 'Monaco', 'Menlo', monospace;
  margin-bottom: 4px;
}

.metric-value.profit {
  color: #059669;
}

.metric-value.loss {
  color: #dc2626;
}

.metric-subvalue {
  font-size: 16px;
  font-weight: 600;
  opacity: 0.8;
}

.position-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.position-type {
  font-size: 16px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 16px;
  text-align: center;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.position-type.long {
  background: #dcfce7;
  color: #166534;
}

.position-type.short {
  background: #fee2e2;
  color: #dc2626;
}

.position-size, .position-value {
  font-size: 14px;
  color: #374151;
  font-family: 'Monaco', 'Menlo', monospace;
}

.position-nominal {
  font-size: 14px;
  color: #059669;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
}

.position-margin {
  font-size: 14px;
  color: #dc2626;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
}

.position-leverage {
  font-size: 14px;
  color: #7c3aed;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 600;
}

.price-comparison {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 24px;
  border: 1px solid #e2e8f0;
}

.price-comparison h4 {
  margin: 0 0 16px 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

.price-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.price-item {
  text-align: center;
  padding: 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.price-label {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 8px;
  font-weight: 500;
}

.price-value {
  font-size: 20px;
  font-weight: 700;
  font-family: 'Monaco', 'Menlo', monospace;
  color: #1f2937;
  margin-bottom: 4px;
}

.entry-price {
  color: #7c3aed;
}

.current-price {
  color: #059669;
}

.price-change {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 12px;
  display: inline-block;
}

.price-change.up {
  background: #dcfce7;
  color: #166534;
}

.price-change.down {
  background: #fee2e2;
  color: #dc2626;
}

.detailed-metrics {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 24px;
  border: 1px solid #e2e8f0;
}

.detailed-metrics h4 {
  margin: 0 0 16px 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

.metrics-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.metric-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: white;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.metric-name {
  font-size: 14px;
  color: #6b7280;
  font-weight: 500;
}

.metric-row .metric-value {
  font-size: 16px;
  font-weight: 600;
  font-family: 'Monaco', 'Menlo', monospace;
}

.metric-desc {
  font-size: 12px;
  color: #6b7280;
  font-style: italic;
  margin-top: 2px;
}

.status-row .metric-value {
  font-size: 14px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 12px;
  text-align: center;
}

.status-row .metric-value.position-closed {
  background: #dcfce7;
  color: #166534;
}

.status-row .metric-value.position-open {
  background: #dbeafe;
  color: #1e40af;
}

.status-row .metric-value.position-partial {
  background: #fef3c7;
  color: #92400e;
}

.status-row .metric-value.position-none {
  background: #f3f4f6;
  color: #6b7280;
}

.note-section {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #fef3c7;
  border: 1px solid #f59e0b;
  border-radius: 8px;
  margin-top: 16px;
}

.note-icon {
  font-size: 18px;
  flex-shrink: 0;
}

.note-content p {
  margin: 0;
  color: #92400e;
  line-height: 1.5;
}


/* 响应式设计 */
@media (max-width: 1024px) {
  .info-cards {
    grid-template-columns: 1fr;
  }

  .bracket-config {
    grid-template-columns: 1fr;
  }

  .metrics-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .scheduled-order-detail {
    padding: 10px;
  }

  .page-header {
    padding: 16px;
  }

  .header-main {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }

  .title-section h1 {
    font-size: 24px;
  }

  .order-badge {
    flex-wrap: wrap;
    gap: 6px;
  }

  .symbol-badge,
  .exchange-badge {
    font-size: 11px;
    padding: 3px 8px;
  }

  .status-section {
    align-items: flex-start;
    width: 100%;
  }

  .main-status {
    align-self: flex-start;
    padding: 6px 12px;
    font-size: 13px;
  }

  .info-cards {
    gap: 16px;
  }

  .info-card {
    margin-bottom: 0;
  }

  .card-header {
    padding: 16px;
  }

  .card-content {
    padding: 16px;
  }

  .bracket-section {
    margin-bottom: 16px;
  }

  .section-header {
    padding: 16px;
  }

  .bracket-config {
    padding: 16px;
    gap: 12px;
  }

  .bracket-panel {
    min-width: 0;
  }

  .finance-section {
    margin-bottom: 16px;
  }

  .finance-content {
    padding: 16px;
  }

  .metrics-grid {
    gap: 16px;
  }

  .metric-card {
    padding: 16px;
  }

  .metric-value {
    font-size: 20px;
  }

  .price-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .actions-section {
    margin-bottom: 16px;
  }

  .actions-content {
    padding: 16px;
  }

}

@media (max-width: 480px) {
  .scheduled-order-detail {
    padding: 8px;
  }

  .page-header {
    padding: 12px;
    border-radius: 8px;
  }

  .title-section h1 {
    font-size: 20px;
  }

  .order-badge {
    justify-content: flex-start;
  }

  .card-header h3 {
    font-size: 14px;
  }

  .info-row {
    padding: 10px 0;
  }

  .info-label,
  .info-value {
    font-size: 13px;
  }

  .section-header h3 {
    font-size: 16px;
  }

  .panel-header h4 {
    font-size: 13px;
  }

  .metric-value {
    font-size: 18px;
  }

  .price-value {
    font-size: 16px;
  }

}

/* 执行日志样式 */
.execution-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.log-badge {
  margin-left: auto;
  padding: 4px 12px;
  background: #f3f4f6;
  color: #374151;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.execution-content {
  padding: 24px;
}

.log-container {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  max-height: 300px;
  overflow-y: auto;
}

.log-text {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: #374151;
  white-space: pre-wrap;
  word-break: break-word;
}

/* 交易所状态样式 */
.exchange-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.exchange-badge {
  margin-left: auto;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.exchange-content {
  padding: 24px;
}

.error-alert {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 8px;
  margin-bottom: 24px;
}

.alert-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.alert-content h4 {
  margin: 0 0 4px 0;
  color: #dc2626;
  font-size: 14px;
  font-weight: 600;
}

.alert-content p {
  margin: 0;
  color: #991b1b;
  font-size: 13px;
}

.exchange-metrics {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.metric-group h4 {
  margin: 0 0 12px 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}


.metric-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

.metric-item {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.metric-label {
  font-size: 12px;
  color: #6b7280;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.metric-value {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  font-family: 'Monaco', 'Menlo', monospace;
}

.client-id, .exchange-id {
  font-size: 14px;
  color: #7c3aed;
  word-break: break-all;
}

.executed-qty {
  color: #059669;
}

.avg-price, .total-value {
  color: #dc2626;
}

.buy-direction {
  color: #059669;
}

.sell-direction {
  color: #dc2626;
}


.order-type {
  color: #7c3aed;
}

.update-time {
  font-size: 13px;
  color: #374151;
}

/* 进度条样式 */
.progress-section {
  margin-top: 24px;
  padding: 20px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.progress-label {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.progress-text {
  font-size: 16px;
  font-weight: 700;
  color: #7c3aed;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #e2e8f0;
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 8px;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #10b981 0%, #059669 100%);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-fill.full {
  background: linear-gradient(90deg, #059669 0%, #047857 100%);
}

.progress-details {
  text-align: center;
  font-size: 12px;
  color: #6b7280;
  font-family: 'Monaco', 'Menlo', monospace;
}

@media (max-width: 768px) {
  .execution-content,
  .exchange-content {
    padding: 16px;
  }

  .log-container {
    max-height: 200px;
  }

  .metric-grid {
    grid-template-columns: 1fr;
  }

  .metric-item {
    padding: 12px;
  }

  .metric-value {
    font-size: 14px;
  }

  .progress-section {
    padding: 16px;
  }
}

/* 时间轴样式 */
.timeline-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.timeline-content {
  padding: 24px;
}

.timeline {
  position: relative;
  padding-left: 40px;
}

.timeline::before {
  content: '';
  position: absolute;
  left: 20px;
  top: 0;
  bottom: 0;
  width: 2px;
  background: linear-gradient(to bottom, #e5e7eb 0%, #e5e7eb 50%, transparent 50%);
  background-size: 2px 20px;
}

.timeline-item {
  position: relative;
  margin-bottom: 32px;
  padding-left: 32px;
}

.timeline-item:last-child {
  margin-bottom: 0;
}

.timeline-marker {
  position: absolute;
  left: -40px;
  top: 0;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 3px solid white;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 1;
}

.marker-icon {
  font-size: 16px;
}

.timeline-marker.created { background: #f59e0b; }
.timeline-marker.triggered { background: #3b82f6; }
.timeline-marker.processing { background: #f59e0b; }
.timeline-marker.submitted { background: #10b981; }
.timeline-marker.pending { background: #f59e0b; }
.timeline-marker.processing { background: #3b82f6; }
.timeline-marker.completed { background: #059669; }
.timeline-marker.filled { background: #059669; }
.timeline-marker.failed { background: #ef4444; }
.timeline-marker.cancelled { background: #6b7280; }
.timeline-marker.final { background: #7c3aed; }

.timeline-content {
  background: #f8fafc;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e2e8f0;
}

.timeline-title {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  margin-bottom: 4px;
}

.timeline-time {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 4px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.timeline-desc {
  font-size: 14px;
  color: #374151;
  line-height: 1.4;
}

.timeline-item.final .timeline-content {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  border: 2px solid #f59e0b;
}

@media (max-width: 768px) {
  .timeline {
    padding-left: 30px;
  }

  .timeline-item {
    padding-left: 24px;
    margin-bottom: 24px;
  }

  .timeline-marker {
    left: -35px;
    width: 32px;
    height: 32px;
  }

  .marker-icon {
    font-size: 14px;
  }

  .timeline-content {
    padding: 12px;
  }

  .timeline-title {
    font-size: 15px;
  }

  .timeline-desc {
    font-size: 13px;
  }
}

.status-tooltip {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 4px;
}

.final-result-badge {
  font-size: 11px;
  color: #059669;
  background: #ecfdf5;
  padding: 2px 6px;
  border-radius: 4px;
  margin-top: 4px;
  display: inline-block;
  border: 1px solid #a7f3d0;
}

/* 关联订单样式 */
.related-orders-section {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid #e5e7eb;
  margin-bottom: 24px;
  overflow: hidden;
}

.related-orders-content {
  padding: 24px;
}

.related-order-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
  transition: all 0.3s ease;
}

.related-order-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.related-order-card.parent-order {
  border-left: 4px solid #10b981;
}

.related-order-card.current-order {
  border-left: 4px solid #3b82f6;
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
}

.related-order-card.close-order {
  border-left: 4px solid #f59e0b;
}

.order-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.order-header h4, .order-header h5 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #374151;
}

.order-link {
  color: #3b82f6;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.2s ease;
}

.order-link:hover {
  color: #2563eb;
  text-decoration: underline;
}

.current-badge {
  background: #3b82f6;
  color: white;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.order-info {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-item .label {
  font-size: 12px;
  color: #6b7280;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-item .value {
  font-size: 14px;
  font-weight: 600;
  color: #111827;
}

.operation-type {
  color: #7c3aed;
  font-weight: 700;
}

.close-orders-group h4 {
  margin: 20px 0 12px 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

.close-orders-list {
  display: grid;
  gap: 12px;
}

/* 操作面板样式 */
.action-panel {
  margin-bottom: 24px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.panel-header {
  padding: 16px 20px;
  border-bottom: 1px solid #e5e7eb;
  background: #f9fafb;
  border-radius: 8px 8px 0 0;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #374151;
}

.panel-content {
  padding: 20px;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.action-buttons .btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.action-buttons .btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.action-buttons .btn-icon {
  font-size: 16px;
}
</style>
