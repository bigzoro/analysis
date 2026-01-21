<template>
  <div class="order-list-tab-content">
    <!-- 加载状态 -->
    <div v-if="orderListLoading" class="loading-state">
      <div class="loading-spinner"></div>
      <span>加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="orderList.length === 0" class="empty-state">
      <div class="empty-title">暂无定时订单</div>
      <div class="empty-description">创建您的第一个定时交易订单</div>
      <button class="btn btn-primary" @click="$emit('create-order')">
        创建定时订单
      </button>
    </div>

    <!-- 筛选条件 -->
    <div v-else class="filters-section" :class="{ expanded: filtersExpanded }">
      <div class="filters-header" @click="filtersExpanded = !filtersExpanded">
        <span class="filters-title">筛选条件</span>
        <button class="toggle-btn" :class="{ expanded: filtersExpanded }">
          <span class="arrow">{{ filtersExpanded ? '▲' : '▼' }}</span>
          {{ filtersExpanded ? '收起筛选' : '展开筛选' }}
        </button>
      </div>
      <div v-show="filtersExpanded" class="filters-content">
        <div class="filters-grid">
          <div class="filter-group">
            <label class="filter-label">订单状态</label>
            <select v-model="orderFilters.status" class="filter-select" @change="onOrderFilterChange">
              <option value="">全部状态</option>
              <option value="pending">等待执行</option>
              <option value="processing">执行中</option>
              <option value="completed">已完成</option>
              <option value="filled">已成交</option>
              <option value="已结束">已结束</option>
              <option value="failed">执行失败</option>
              <option value="cancelled">已取消</option>
            </select>
          </div>

          <div class="filter-group">
            <label class="filter-label">操作类型</label>
            <select v-model="orderFilters.operation_type" class="filter-select" @change="onOrderFilterChange">
              <option value="">全部操作</option>
              <option value="开多">开多</option>
              <option value="开空">开空</option>
            </select>
          </div>

          <div class="filter-group">
            <label class="filter-label">交易对</label>
            <input
              v-model="orderFilters.symbol"
              class="filter-input"
              placeholder="例如：ETHUSDT"
              @input="debounceFilterChange"
            />
          </div>

          <div class="filter-group">
            <label class="filter-label">交易所</label>
            <select v-model="orderFilters.exchange" class="filter-select" @change="onOrderFilterChange">
              <option value="">全部交易所</option>
              <option value="binance_futures">Binance Futures</option>
            </select>
          </div>

          <div class="filter-group">
            <label class="filter-label">环境</label>
            <select v-model="orderFilters.testnet" class="filter-select" @change="onOrderFilterChange">
              <option value="">全部环境</option>
              <option :value="true">测试网</option>
              <option :value="false">正式网</option>
            </select>
          </div>

          <div class="filter-group">
            <label class="filter-label">开始日期</label>
            <input
              v-model="orderFilters.date_from"
              type="date"
              class="filter-input"
              @change="onOrderFilterChange"
            />
          </div>

          <div class="filter-group">
            <label class="filter-label">结束日期</label>
            <input
              v-model="orderFilters.date_to"
              type="date"
              class="filter-input"
              @change="onOrderFilterChange"
            />
          </div>

          <div class="filter-actions">
            <button class="btn btn-outline" @click="clearOrderFilters">
              清除筛选
            </button>
          </div>
        </div>
      </div>
    </div>


    <!-- 订单列表 -->
    <div class="orders-container">
      <div v-for="order in processedOrderList" :key="order.id" class="order-wrapper">
        <!-- 主要订单卡片 -->
        <div class="order-card main-card" :class="{ 'has-children': order.childOrders && order.childOrders.length > 0 }">
          <div class="order-header">
            <div class="order-symbol">
              <span class="symbol-text">{{ order.symbol }}</span>
              <span class="exchange-badge" :class="{ testnet: order.testnet }">
                {{ order.exchange }} {{ order.testnet ? '(测试网)' : '(正式网)' }}
              </span>
              <!-- 交易链标识（如果有关联订单） -->
              <div v-if="order.related_orders?.trade_chain" class="chain-indicator">
                <span class="chain-badge">{{ order.related_orders.trade_chain }}</span>
              </div>
            </div>
            <div class="order-status" :class="getOrderStatusClass(order)">
              <span class="status-text">{{ getEnhancedStatusText(order) }}</span>
            </div>
          </div>

          <div class="order-details">
            <div class="detail-row">
              <span class="detail-label">操作:</span>
              <span class="detail-value" :class="order.operation_class || getOperationClass(order.side, order.reduce_only)">
                {{ order.operation_type || getOperationType(order.side, order.reduce_only) }}
              </span>
              <span class="detail-description" :title="order.operation_desc || getOperationDescription(order.side, order.reduce_only)">
                ({{ order.operation_desc || getOperationDescription(order.side, order.reduce_only) }})
              </span>
            </div>

            <div class="detail-row">
              <span class="detail-label">类型:</span>
              <span class="detail-value">
                {{ order.order_type === 'MARKET' ? '市价' : '限价' }}
              </span>
            </div>

            <div class="detail-row">
              <span class="detail-label">数量:</span>
              <span class="detail-value">
                <span :class="{'adjusted-quantity': order.adjusted_quantity && order.adjusted_quantity !== order.quantity}">
                  {{ order.quantity }}
                </span>
                <span v-if="order.adjusted_quantity && order.adjusted_quantity !== order.quantity" class="adjusted-info">
                  → {{ order.adjusted_quantity }}
                </span>
              </span>
            </div>

            <div v-if="order.price" class="detail-row">
              <span class="detail-label">价格:</span>
              <span class="detail-value">{{ order.price }}</span>
            </div>

            <div v-if="order.leverage" class="detail-row">
              <span class="detail-label">杠杆:</span>
              <span class="detail-value">{{ order.leverage }}x</span>
            </div>

            <div v-if="order.reduce_only" class="detail-row">
              <span class="detail-label">只减仓:</span>
              <span class="detail-value">是</span>
            </div>

            <div v-if="order.bracket_enabled" class="bracket-info">
              <div class="bracket-title">一键三连设置</div>
              <div class="bracket-details">
                <div class="bracket-item">
                  <span>止盈: {{ order.actual_tp_percent || order.tp_percent || 0 }}%</span>
                  <span v-if="order.tp_price"> ({{ order.tp_price }})</span>
                </div>
                <div class="bracket-item">
                  <span>止损: {{ order.actual_sl_percent || order.sl_percent || 0 }}%</span>
                  <span v-if="order.sl_price"> ({{ order.sl_price }})</span>
                </div>
                <div class="bracket-item">
                  <span>触发类型: {{ order.working_type || 'MARK_PRICE' }}</span>
                </div>
              </div>
            </div>

            <div class="detail-row trigger-time">
              <span class="detail-label">触发时间:</span>
              <span class="detail-value">{{ formatDateTime(order.trigger_time) }}</span>
            </div>

            <div class="order-actions">
              <button
                class="btn btn-primary btn-small"
                @click="viewOrderDetails(order.id)"
              >
                查看详情
              </button>
              <button
                v-if="['pending', 'processing'].includes(order.status)"
                class="btn btn-danger btn-small"
                @click="cancelOrder(order.id)"
              >
                🚫 取消订单
              </button>
              <button
                v-if="order.status === 'completed'"
                class="btn btn-outline btn-small"
                disabled
              >
                已完成
              </button>
              <button
                v-if="!['processing'].includes(order.status)"
                class="btn btn-danger btn-small"
                @click="removeOrder(order.id)"
              >
                删除订单
              </button>
            </div>

            <!-- 展开/折叠指示器（在有任何关联订单时显示） -->
            <div v-if="hasRelatedOrders(order)" class="expand-indicator bottom" @click="toggleOrderExpansion(order.id)">
              <span class="expand-icon">{{ isOrderExpanded(order.id) ? '▼' : '▶' }}</span>
              <span class="expand-text">{{ isOrderExpanded(order.id) ? '收起' : '展开' }}交易链 ({{ getRelatedOrderCount(order) }})</span>
            </div>
          </div>
        </div>

        <!-- 展开的交易链详情 -->
        <div v-if="isOrderExpanded(order.id) && hasRelatedOrders(order)" class="trade-chain-container">
          <div class="trade-chain-header">
            <h5>交易链详情</h5>
          </div>

          <!-- 父订单（如果有） -->
          <div v-if="order.related_orders && order.related_orders.parent_order" class="trade-chain-section">
            <div class="section-title">父订单</div>
            <div class="trade-chain-list">
              <div class="trade-chain-item parent-order-item">
                <div class="trade-chain-card">
                  <div class="trade-chain-main">
                    <span class="trade-chain-type" :class="order.related_orders.parent_order.operation_class || getOperationClass(order.related_orders.parent_order.side, order.related_orders.parent_order.reduce_only)">
                      {{ order.related_orders.parent_order.operation_type || getOperationType(order.related_orders.parent_order.side, order.related_orders.parent_order.reduce_only) }}
                    </span>
                    <span class="trade-chain-quantity">
                      {{ order.related_orders.parent_order.quantity }}
                      <span v-if="order.related_orders.parent_order.adjusted_quantity && order.related_orders.parent_order.adjusted_quantity !== order.related_orders.parent_order.quantity" class="adjusted-info">
                        → {{ order.related_orders.parent_order.adjusted_quantity }}
                      </span>
                    </span>
                    <span v-if="order.related_orders.parent_order.price" class="trade-chain-price">{{ order.related_orders.parent_order.price }}</span>
                    <span class="trade-chain-time">{{ formatDateTime(order.related_orders.parent_order.trigger_time) }}</span>
                    <span class="trade-chain-status" :class="getOrderStatusClass(order.related_orders.parent_order)">
                      {{ getEnhancedStatusText(order.related_orders.parent_order) }}
                    </span>
                  </div>
                  <div class="trade-chain-actions">
                    <button class="btn-link small" @click="viewOrderDetails(order.related_orders.parent_order.id)">
                      详情
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 子订单（平仓和加仓） -->
          <div v-if="(order.childOrders && order.childOrders.length > 0) || (order.related_orders && order.related_orders.close_orders && order.related_orders.close_orders.length > 0)" class="trade-chain-section">
            <div class="section-title">子订单 ({{
              (order.childOrders ? order.childOrders.length : 0) +
              (order.related_orders?.close_orders ? order.related_orders.close_orders.length : 0)
            }})</div>
            <div class="trade-chain-list">
              <!-- 显示childOrders中的订单 -->
              <div v-for="childOrder in order.childOrders" :key="childOrder.id" class="trade-chain-item child-order-item">
                <div class="trade-chain-card">
                  <div class="trade-chain-main">
                    <span class="trade-chain-type" :class="childOrder.operation_class || getOperationClass(childOrder.side, childOrder.reduce_only)">
                      {{ childOrder.operation_type || getOperationType(childOrder.side, childOrder.reduce_only) }}
                    </span>
                    <span class="trade-chain-quantity">
                      {{ childOrder.quantity }}
                      <span v-if="childOrder.adjusted_quantity && childOrder.adjusted_quantity !== childOrder.quantity" class="adjusted-info">
                        → {{ childOrder.adjusted_quantity }}
                      </span>
                    </span>
                    <span v-if="childOrder.price" class="trade-chain-price">{{ childOrder.price }}</span>
                    <span class="trade-chain-time">{{ formatDateTime(childOrder.trigger_time) }}</span>
                    <span class="trade-chain-status" :class="getOrderStatusClass(childOrder)">
                      {{ getEnhancedStatusText(childOrder) }}
                    </span>
                  </div>
                  <div class="trade-chain-actions">
                    <button class="btn-link small" @click="viewOrderDetails(childOrder.id)">
                      详情
                    </button>
                    <button
                      v-if="['pending', 'processing'].includes(childOrder.status)"
                      class="btn-danger small"
                      @click="cancelOrder(childOrder.id)"
                    >
                      取消
                    </button>
                    <button
                      v-if="!['processing'].includes(childOrder.status)"
                      class="btn-danger small"
                      @click="removeOrder(childOrder.id)"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>

              <!-- 显示related_orders.close_orders中的订单 -->
              <div v-for="closeOrder in order.related_orders?.close_orders" :key="closeOrder.id" class="trade-chain-item child-order-item">
                <div class="trade-chain-card">
                  <div class="trade-chain-main">
                    <span class="trade-chain-type" :class="closeOrder.operation_class || getOperationClass(closeOrder.side, closeOrder.reduce_only)">
                      {{ closeOrder.operation_type || getOperationType(closeOrder.side, closeOrder.reduce_only) }}
                    </span>
                    <span class="trade-chain-quantity">
                      {{ closeOrder.quantity }}
                      <span v-if="closeOrder.adjusted_quantity && closeOrder.adjusted_quantity !== closeOrder.quantity" class="adjusted-info">
                        → {{ closeOrder.adjusted_quantity }}
                      </span>
                    </span>
                    <span v-if="closeOrder.price" class="trade-chain-price">{{ closeOrder.price }}</span>
                    <span class="trade-chain-time">{{ formatDateTime(closeOrder.trigger_time) }}</span>
                    <span class="trade-chain-status" :class="getOrderStatusClass(closeOrder)">
                      {{ getEnhancedStatusText(closeOrder) }}
                    </span>
                  </div>
                  <div class="trade-chain-actions">
                    <button class="btn-link small" @click="viewOrderDetails(closeOrder.id)">
                      详情
                    </button>
                    <button
                      v-if="['pending', 'processing'].includes(closeOrder.status)"
                      class="btn-danger small"
                      @click="cancelOrder(closeOrder.id)"
                    >
                      取消
                    </button>
                    <button
                      v-if="!['processing'].includes(closeOrder.status)"
                      class="btn-danger small"
                      @click="removeOrder(closeOrder.id)"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Bracket订单（TP/SL） -->
          <div v-if="order.related_orders && order.related_orders.bracket_orders && order.related_orders.bracket_orders.has_bracket" class="trade-chain-section">
            <div class="section-title">止盈止损订单</div>
            <div class="trade-chain-list">
              <!-- TP订单 -->
              <div v-if="order.related_orders.bracket_orders.tp_order" class="trade-chain-item bracket-order-item tp-order">
                <div class="trade-chain-card">
                  <div class="trade-chain-main">
                    <span class="trade-chain-type tp-type">
                      止盈
                    </span>
                    <span class="trade-chain-quantity">
                      {{ order.related_orders.bracket_orders.tp_order.quantity }}
                      <span v-if="order.related_orders.bracket_orders.tp_order.adjusted_quantity && order.related_orders.bracket_orders.tp_order.adjusted_quantity !== order.related_orders.bracket_orders.tp_order.quantity" class="adjusted-info">
                        → {{ order.related_orders.bracket_orders.tp_order.adjusted_quantity }}
                      </span>
                    </span>
                    <span v-if="order.related_orders.bracket_orders.tp_order.price" class="trade-chain-price">{{ order.related_orders.bracket_orders.tp_order.price }}</span>
                    <span class="trade-chain-time">{{ formatDateTime(order.related_orders.bracket_orders.tp_order.trigger_time) }}</span>
                    <span class="trade-chain-status" :class="getOrderStatusClass(order.related_orders.bracket_orders.tp_order)">
                      {{ getEnhancedStatusText(order.related_orders.bracket_orders.tp_order) }}
                    </span>
                  </div>
                  <div class="trade-chain-actions">
                    <button class="btn-link small" @click="viewOrderDetails(order.related_orders.bracket_orders.tp_order.id)">
                      详情
                    </button>
                    <button
                      v-if="['pending', 'processing'].includes(order.related_orders.bracket_orders.tp_order.status)"
                      class="btn-danger small"
                      @click="cancelOrder(order.related_orders.bracket_orders.tp_order.id)"
                    >
                      取消
                    </button>
                  </div>
                </div>
              </div>

              <!-- SL订单 -->
              <div v-if="order.related_orders.bracket_orders.sl_order" class="trade-chain-item bracket-order-item sl-order">
                <div class="trade-chain-card">
                  <div class="trade-chain-main">
                    <span class="trade-chain-type sl-type">
                      止损
                    </span>
                    <span class="trade-chain-quantity">
                      {{ order.related_orders.bracket_orders.sl_order.quantity }}
                      <span v-if="order.related_orders.bracket_orders.sl_order.adjusted_quantity && order.related_orders.bracket_orders.sl_order.adjusted_quantity !== order.related_orders.bracket_orders.sl_order.quantity" class="adjusted-info">
                        → {{ order.related_orders.bracket_orders.sl_order.adjusted_quantity }}
                      </span>
                    </span>
                    <span v-if="order.related_orders.bracket_orders.sl_order.price" class="trade-chain-price">{{ order.related_orders.bracket_orders.sl_order.price }}</span>
                    <span class="trade-chain-time">{{ formatDateTime(order.related_orders.bracket_orders.sl_order.trigger_time) }}</span>
                    <span class="trade-chain-status" :class="getOrderStatusClass(order.related_orders.bracket_orders.sl_order)">
                      {{ getEnhancedStatusText(order.related_orders.bracket_orders.sl_order) }}
                    </span>
                  </div>
                  <div class="trade-chain-actions">
                    <button class="btn-link small" @click="viewOrderDetails(order.related_orders.bracket_orders.sl_order.id)">
                      详情
                    </button>
                    <button
                      v-if="['pending', 'processing'].includes(order.related_orders.bracket_orders.sl_order.status)"
                      class="btn-danger small"
                      @click="cancelOrder(order.related_orders.bracket_orders.sl_order.id)"
                    >
                      取消
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 统一分页组件（移出网格容器） -->
    <div class="pagination-container">
      <Pagination
        v-if="orderTotal > 0"
        v-model:page="orderPage"
        v-model:pageSize="orderPageSize"
        :total="orderTotal"
        :totalPages="orderTotalPages"
        :loading="orderListLoading"
        @change="onOrderPaginationChange"
      />
    </div>

    <!-- 删除确认对话框 -->
    <div v-if="deleteDialogVisible" class="delete-confirmation-dialog-overlay" @click="closeDeleteDialog">
      <div class="delete-confirmation-dialog" @click.stop>
        <div class="dialog-header">
          <h3 class="dialog-title">确认删除订单</h3>
          <button class="dialog-close-btn" @click="closeDeleteDialog">
            <span>×</span>
          </button>
        </div>

        <div class="dialog-body">
          <!-- 要删除的开仓订单信息 -->
          <div class="order-to-delete">
            <div class="order-info-header">
              <span class="order-type-badge" :class="deleteDialogData.order.operation_class || getOperationClass(deleteDialogData.order.side, deleteDialogData.order.reduce_only)">
                {{ deleteDialogData.order.operation_type || getOperationType(deleteDialogData.order.side, deleteDialogData.order.reduce_only) }}
              </span>
              <span class="order-symbol">{{ deleteDialogData.order.symbol }}</span>
            </div>
            <div class="order-details">
              <div class="detail-item">
                <span class="label">数量:</span>
                <span class="value">{{ deleteDialogData.order.quantity }}</span>
              </div>
              <div class="detail-item" v-if="deleteDialogData.order.price">
                <span class="label">价格:</span>
                <span class="value">{{ deleteDialogData.order.price }}</span>
              </div>
              <div class="detail-item">
                <span class="label">状态:</span>
                <span class="value status-text" :class="getOrderStatusClass(deleteDialogData.order)">
                  {{ getEnhancedStatusText(deleteDialogData.order) }}
                </span>
              </div>
            </div>
          </div>

          <!-- 级联删除选项 -->
          <div v-if="deleteDialogData.hasCloseOrders || deleteDialogData.hasBracketOrders" class="cascade-options">
            <h4 class="options-title">删除选项</h4>

            <div class="option-group">
              <label class="option-radio">
                <input
                  type="radio"
                  v-model="deleteOption"
                  value="single"
                  @change="onDeleteOptionChange"
                />
                <span class="radio-mark"></span>
                <span class="option-text">
                  <strong>仅删除开仓订单</strong>
                  <span class="option-desc">保留所有关联的平仓订单{{ deleteDialogData.hasBracketOrders ? '和止盈止损订单' : '' }}</span>
                </span>
              </label>

              <label class="option-radio">
                <input
                  type="radio"
                  v-model="deleteOption"
                  value="cascade"
                  @change="onDeleteOptionChange"
                />
                <span class="radio-mark"></span>
                <span class="option-text">
                  <strong>删除整个交易链</strong>
                  <span class="option-desc">同时删除开仓订单{{ deleteDialogData.hasCloseOrders ? '和关联的平仓订单' : '' }}{{ deleteDialogData.hasBracketOrders ? '及止盈止损订单' : '' }}</span>
                </span>
              </label>
            </div>

            <!-- 关联平仓订单列表 -->
            <div v-if="deleteDialogData.closeOrders && deleteDialogData.closeOrders.length > 0" class="related-orders">
              <h5 class="related-title">关联的平仓订单 ({{ deleteDialogData.closeOrders.length }}个)</h5>
              <div class="related-orders-list">
                <div
                  v-for="closeOrder in deleteDialogData.closeOrders"
                  :key="closeOrder.id"
                  class="related-order-item"
                  :class="{ 'will-be-deleted': deleteOption === 'cascade', 'will-be-kept': deleteOption === 'single' }"
                >
                  <div class="related-order-info">
                    <span class="related-order-type" :class="closeOrder.operation_class || getOperationClass(closeOrder.side, closeOrder.reduce_only)">
                      {{ closeOrder.operation_type || getOperationType(closeOrder.side, closeOrder.reduce_only) }}
                    </span>
                    <span class="related-order-quantity">{{ closeOrder.quantity }}</span>
                    <span class="related-order-status" :class="getOrderStatusClass(closeOrder)">
                      {{ getEnhancedStatusText(closeOrder) }}
                    </span>
                  </div>
                  <div class="related-order-action">
                    <span class="action-text" :class="deleteOption === 'cascade' ? 'delete-action' : 'keep-action'">
                      {{ deleteOption === 'cascade' ? '将删除' : '将保留' }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Bracket止盈止损订单列表 -->
            <div v-if="deleteDialogData.hasBracketOrders" class="bracket-orders">
              <h5 class="related-title">关联的止盈止损订单</h5>
              <div class="bracket-orders-list">
                <div v-if="deleteDialogData.hasTpOrder"
                     class="bracket-order-item"
                     :class="{ 'will-be-deleted': deleteOption === 'cascade', 'will-be-kept': deleteOption === 'single' }">
                  <div class="bracket-order-info">
                    <span class="bracket-order-type tp-order">止盈单</span>
                    <span class="bracket-order-symbol">{{ deleteDialogData.tpOrder.symbol }}</span>
                    <span class="bracket-order-price" v-if="deleteDialogData.tpOrder.trigger_price">触发价: {{ deleteDialogData.tpOrder.trigger_price }}</span>
                    <span class="bracket-order-status" :class="getOrderStatusClass(deleteDialogData.tpOrder)">
                      {{ getEnhancedStatusText(deleteDialogData.tpOrder) }}
                    </span>
                  </div>
                  <div class="bracket-order-action">
                    <span class="action-text" :class="deleteOption === 'cascade' ? 'delete-action' : 'keep-action'">
                      {{ deleteOption === 'cascade' ? '将删除' : '将保留' }}
                    </span>
                  </div>
                </div>

                <div v-if="deleteDialogData.hasSlOrder"
                     class="bracket-order-item"
                     :class="{ 'will-be-deleted': deleteOption === 'cascade', 'will-be-kept': deleteOption === 'single' }">
                  <div class="bracket-order-info">
                    <span class="bracket-order-type sl-order">止损单</span>
                    <span class="bracket-order-symbol">{{ deleteDialogData.slOrder.symbol }}</span>
                    <span class="bracket-order-price" v-if="deleteDialogData.slOrder.trigger_price">触发价: {{ deleteDialogData.slOrder.trigger_price }}</span>
                    <span class="bracket-order-status" :class="getOrderStatusClass(deleteDialogData.slOrder)">
                      {{ getEnhancedStatusText(deleteDialogData.slOrder) }}
                    </span>
                  </div>
                  <div class="bracket-order-action">
                    <span class="action-text" :class="deleteOption === 'cascade' ? 'delete-action' : 'keep-action'">
                      {{ deleteOption === 'cascade' ? '将删除' : '将保留' }}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 警告信息 -->
            <div v-if="deleteDialogData.hasCompletedCloseOrders || deleteDialogData.tpOrderCompleted || deleteDialogData.slOrderCompleted" class="warning-message">
              <div class="warning-icon">🚨</div>
              <div class="warning-content">
                <strong>警告：</strong>
                <span v-if="deleteDialogData.hasCompletedCloseOrders">部分平仓订单已成交，</span>
                <span v-if="deleteDialogData.tpOrderCompleted || deleteDialogData.slOrderCompleted">部分止盈止损订单已执行，</span>
                包含重要的交易记录、盈亏数据和历史信息。删除整个交易链将<strong>永久删除</strong>这些历史数据，无法恢复。
              </div>
            </div>

          </div>
        </div>

        <div class="dialog-footer">
          <button class="btn btn-outline" @click="closeDeleteDialog">取消</button>
          <button
            class="btn btn-danger"
            @click="confirmDelete"
            :disabled="deleteDialogLoading"
          >
            <span v-if="deleteDialogLoading" class="loading-spinner small"></span>
            {{ deleteOption === 'cascade' ? '删除交易链' : '删除订单' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from 'vue'
import Pagination from '../Pagination.vue'
import { api } from '../../api/api.js'

// 订单列表相关状态
const orderList = ref([])
const orderPage = ref(1)
const orderPageSize = ref(5)
const orderTotal = ref(0)
const orderTotalPages = ref(1)
const orderListLoading = ref(false)

// 订单列表筛选条件
const orderFilters = reactive({
  status: '', // 订单状态筛选
  operation_type: '', // 操作类型筛选 (开多/开空/平多/平空)
  symbol: '', // 交易对筛选
  exchange: '', // 交易所筛选
  testnet: '', // 环境筛选
  date_from: '', // 开始日期
  date_to: '' // 结束日期
})

// 筛选区域展开/折叠状态（默认折叠）
const filtersExpanded = ref(false)

// 订单列表关联订单下拉菜单状态
const orderActiveDropdown = ref(null)

// 展开/折叠状态管理
const expandedOrders = ref(new Set()) // 展开的订单（显示平仓订单）

// 删除确认对话框状态
const deleteDialogVisible = ref(false)
const deleteDialogLoading = ref(false)
const deleteOption = ref('single') // 'single' 或 'cascade'
const deleteDialogData = ref({
  order: null,
  hasCloseOrders: false,
  hasCompletedCloseOrders: false,
  closeOrders: [],
  closeOrderCount: 0,
  completedCloseOrderCount: 0
})

// 计算属性：处理订单关联关系
const processedOrderList = computed(() => {
  const orders = [...orderList.value]

  // 为每个订单添加关联的子订单信息
  orders.forEach(order => {
    const childOrders = []

    // 查找关联的平仓订单
    if (!order.reduce_only && order.related_orders?.has_close && order.related_orders.close_ids) {
      const closeOrders = orders.filter(o =>
        o.reduce_only && order.related_orders.close_ids.includes(o.id)
      )
      childOrders.push(...closeOrders)
    }

    // 查找关联的加仓订单
    if (!order.reduce_only && order.related_orders?.has_scaling && order.related_orders.scaling_ids) {
      const scalingOrders = orders.filter(o =>
        !o.reduce_only && order.related_orders.scaling_ids.includes(o.id)
      )
      childOrders.push(...scalingOrders)
    }

    // 备用逻辑：通过parent_order_id查找子订单（包括加仓和平仓订单）
    if (!order.reduce_only) {
      const parentChildOrders = orders.filter(o =>
        o.parent_order_id === order.id
      )
      // 合并两个来源的子订单，避免重复
      const existingIds = new Set(childOrders.map(o => o.id))
      parentChildOrders.forEach(childOrder => {
        if (!existingIds.has(childOrder.id)) {
          childOrders.push(childOrder)
        }
      })
    }

    order.childOrders = childOrders
  })

  // 只显示独立的开仓订单，平仓订单和加仓订单在对应开仓订单的展开区域中显示
  const independentOrders = orders.filter(order =>
    !order.reduce_only && !order.parent_order_id
  )

  // 按时间倒序排序
  return independentOrders.sort((a, b) => {
    const aTime = new Date(a.created_at || a.trigger_time)
    const bTime = new Date(b.created_at || b.trigger_time)
    return bTime - aTime
  })
})

// 计算属性：直接显示API返回的订单列表（API已在后端完成分页）- 保持兼容性
const filteredOrderList = computed(() => {
  // 由于分页已在后端完成，这里直接返回API数据
  // TODO: 如果需要前端筛选功能，需要：
  // 1. 修改API支持筛选参数
  // 2. 或者在前端缓存所有数据然后筛选
  return orderList.value
})

// 使用后端返回的总数（已修复分页组件使用正确的总数）
const filteredOrderTotal = computed(() => {
  return orderTotal.value
})

// 使用后端返回的总页数
const filteredOrderTotalPages = computed(() => {
  return orderTotalPages.value
})

// 防抖的筛选条件变化处理（用于输入框）
let filterDebounceTimer = null
function debounceFilterChange() {
  clearTimeout(filterDebounceTimer)
  filterDebounceTimer = setTimeout(() => {
    onOrderFilterChange()
  }, 500) // 500ms防抖
}

// 订单列表分页变化处理
function onOrderPaginationChange(paginationData) {
  console.log('分页变化事件触发:', paginationData)
  const { page: newPage, pageSize: newPageSize } = paginationData
  console.log('新的分页参数:', { newPage, newPageSize })

  orderPage.value = newPage
  orderPageSize.value = newPageSize

  console.log('更新后的状态:', { orderPage: orderPage.value, orderPageSize: orderPageSize.value })
  loadOrderList()
}

// 筛选条件变化处理
function onOrderFilterChange() {
  // 由于现在不支持前端筛选（因为分页在后端），重置分页并重新加载数据
  console.log('筛选条件变化，重置分页并重新加载数据')
  orderPage.value = 1
  loadOrderList()
}

// 清除筛选条件
function clearOrderFilters() {
  orderFilters.status = ''
  orderFilters.operation_type = ''
  orderFilters.symbol = ''
  orderFilters.exchange = ''
  orderFilters.testnet = ''
  orderFilters.date_from = ''
  orderFilters.date_to = ''
  onOrderFilterChange()
}

// 检查订单是否符合筛选条件
function matchesOrderFilters(order) {
  // 状态筛选
  if (orderFilters.status) {
    const enhancedStatus = getEnhancedStatusText(order)
    // 检查原始状态或增强状态是否匹配
    if (order.status !== orderFilters.status && enhancedStatus !== orderFilters.status) {
      return false
    }
  }

  // 操作类型筛选
  if (orderFilters.operation_type) {
    const operationType = getOperationType(order.side, order.reduce_only)
    if (operationType !== orderFilters.operation_type) {
      return false
    }
  }

  // 交易对筛选（支持模糊匹配）
  if (orderFilters.symbol && !order.symbol.toUpperCase().includes(orderFilters.symbol.toUpperCase())) {
    return false
  }

  // 交易所筛选
  if (orderFilters.exchange && order.exchange !== orderFilters.exchange) {
    return false
  }

  // 环境筛选
  if (orderFilters.testnet !== '' && order.testnet !== (orderFilters.testnet === 'true')) {
    return false
  }

  // 日期范围筛选
  if (orderFilters.date_from || orderFilters.date_to) {
    const orderDate = new Date(order.created_at || order.trigger_time).toISOString().split('T')[0]
    if (orderFilters.date_from && orderDate < orderFilters.date_from) {
      return false
    }
    if (orderFilters.date_to && orderDate > orderFilters.date_to) {
      return false
    }
  }

  return true
}

// 加载订单列表
async function loadOrderList() {
  console.log('开始加载订单列表，当前状态:', {
    page: orderPage.value,
    pageSize: orderPageSize.value,
    total: orderTotal.value,
    totalPages: orderTotalPages.value
  })

  orderListLoading.value = true
  try {
    console.log('调用listScheduledOrders API, page:', orderPage.value, 'page_size:', orderPageSize.value)
    const res = await api.listScheduledOrders({ page: orderPage.value, page_size: orderPageSize.value })
    console.log('API响应完整信息:', {
      status: 'success',
      response: res,
      hasItems: Array.isArray(res?.items),
      itemsLength: res?.items?.length || 0,
      total: res?.total,
      totalPages: res?.total_pages,
      page: res?.page
    })

    orderList.value = Array.isArray(res?.items) ? res.items : []
    console.log('更新订单列表，新的列表长度:', orderList.value.length)

    // 更新分页信息
    const oldTotal = orderTotal.value
    const oldTotalPages = orderTotalPages.value

    orderTotal.value = res?.total || 0
    orderTotalPages.value = res?.total_pages || 1
    orderPage.value = res?.page || orderPage.value

    console.log('分页信息更新:', {
      old: { total: oldTotal, totalPages: oldTotalPages },
      new: { total: orderTotal.value, totalPages: orderTotalPages.value, page: orderPage.value }
    })

    // 检查是否有数据
    if (orderList.value.length === 0 && orderTotal.value > 0) {
      console.warn('警告: API返回total > 0但items为空数组', {
        total: orderTotal.value,
        page: orderPage.value,
        pageSize: orderPageSize.value
      })
    }

  } catch (e) {
    console.error('加载订单列表失败:', e)
    console.error('错误详情:', {
      message: e.message,
      stack: e.stack,
      page: orderPage.value,
      pageSize: orderPageSize.value
    })
  } finally {
    orderListLoading.value = false
    console.log('订单列表加载完成')
  }
}

// 查看订单详情
async function viewOrderDetails(id) {
  // 通过事件向父组件传递
  emit('view-order-details', id)
}

// 取消订单
async function cancelOrder(id) {
  if (!confirm('确认取消该计划？')) return
  try {
    await api.cancelScheduledOrder(id)
    await loadOrderList()
  } catch (e) {
    alert('取消失败: ' + (e?.message || '未知错误'))
  }
}

// 删除订单 - 显示确认对话框
async function removeOrder(id) {
  // 查找要删除的订单（基本信息）
  const orderToDelete = orderList.value.find(o => o.id === id)
  if (!orderToDelete) {
    alert('未找到要删除的订单')
    return
  }

  try {
    // 获取订单的完整详细信息（包括Bracket订单信息）
    const response = await api.getScheduledOrderDetail(id)
    const detailedOrder = response

    // 准备对话框数据
    const hasCloseOrders = !detailedOrder.reduce_only && detailedOrder.related_orders && detailedOrder.related_orders.close_orders && detailedOrder.related_orders.close_orders.length > 0
    const closeOrders = hasCloseOrders ? detailedOrder.related_orders.close_orders : []
    const completedCloseOrders = closeOrders.filter(co => ['filled', 'completed'].includes(co.status))

    // 检查Bracket订单的TP/SL订单
    const hasBracketOrders = detailedOrder.related_orders && detailedOrder.related_orders.bracket_orders && detailedOrder.related_orders.bracket_orders.has_bracket
    const bracketOrders = hasBracketOrders ? detailedOrder.related_orders.bracket_orders : null
    const tpOrder = bracketOrders ? bracketOrders.tp_order : null
    const slOrder = bracketOrders ? bracketOrders.sl_order : null
    const hasTpOrder = tpOrder !== null
    const hasSlOrder = slOrder !== null
    const tpOrderCompleted = tpOrder && ['filled', 'completed'].includes(tpOrder.status)
    const slOrderCompleted = slOrder && ['filled', 'completed'].includes(slOrder.status)

    // 设置对话框数据
    deleteDialogData.value = {
      order: detailedOrder, // 使用详细订单信息
      hasCloseOrders,
      hasCompletedCloseOrders: completedCloseOrders.length > 0,
      closeOrders,
      closeOrderCount: closeOrders.length,
      completedCloseOrderCount: completedCloseOrders.length,
      hasBracketOrders,
      tpOrder,
      slOrder,
      hasTpOrder,
      hasSlOrder,
      tpOrderCompleted,
      slOrderCompleted
    }

    // 默认选择逻辑：
    // 1. 如果有Bracket订单，默认选择级联删除（包括TP/SL）
    // 2. 否则如果有关联平仓订单，默认选择级联删除
    // 3. 否则只删除单个订单
    if (hasBracketOrders || hasCloseOrders) {
      deleteOption.value = 'cascade'
    } else {
      deleteOption.value = 'single'
    }

    // 显示删除确认对话框
    deleteDialogVisible.value = true

  } catch (error) {
    console.error('获取订单详情失败:', error)
    alert('获取订单详情失败，无法显示删除确认信息')
  }

  // 显示对话框
  deleteDialogVisible.value = true
}

// 关闭删除确认对话框
function closeDeleteDialog() {
  deleteDialogVisible.value = false
  deleteDialogLoading.value = false
  deleteOption.value = 'single'
}

// 删除选项变化处理
function onDeleteOptionChange() {
  // 可以在这里添加选项变化时的逻辑
  console.log('删除选项变更为:', deleteOption.value)
}

// 确认删除操作
async function confirmDelete() {
  if (deleteDialogLoading.value) return

  const orderId = deleteDialogData.value.order.id
  const isCascadeDelete = deleteOption.value === 'cascade'

  deleteDialogLoading.value = true

  try {
    console.log(`开始${isCascadeDelete ? '级联' : '单个'}删除订单:`, orderId)
    console.log('删除选项:', deleteOption.value, 'isCascadeDelete:', isCascadeDelete)

    // 准备删除参数
    const deleteParams = { cascade: isCascadeDelete }

    // 如果是级联删除，传递所有相关的平仓订单ID列表
    // 让后端来决定哪些可以删除（未成交的），哪些需要保留（已成交的）
    if (isCascadeDelete && deleteDialogData.value.closeOrders && deleteDialogData.value.closeOrders.length > 0) {
      const closeOrderIds = deleteDialogData.value.closeOrders.map(co => co.id)
      deleteParams.closeOrderIds = closeOrderIds
      console.log('将检查删除的平仓订单IDs:', closeOrderIds)
    }

    console.log('传递给API的参数:', deleteParams)
    const response = await api.deleteScheduledOrder(orderId, deleteParams)

    console.log('删除API调用成功:', response)

    // 显示成功消息
    const message = response.message || '删除成功'
    alert(message)

    // 关闭对话框
    closeDeleteDialog()

    // 重新加载订单列表
    await loadOrderList()
    console.log('订单列表重新加载完成')

  } catch (e) {
    console.error('删除操作失败:', e)
    alert('删除失败: ' + (e?.message || '未知错误'))
  } finally {
    deleteDialogLoading.value = false
  }
}

// 切换订单关联下拉菜单
function toggleOrderRelationDropdown(orderId) {
  if (orderActiveDropdown.value === orderId) {
    orderActiveDropdown.value = null
  } else {
    orderActiveDropdown.value = orderId
  }
}

// 处理订单平仓订单点击
function handleCloseOrderClick(closeId) {
  console.log('点击平仓订单:', closeId, typeof closeId)

  // 确保ID是数字类型
  let id
  if (typeof closeId === 'string') {
    id = parseInt(closeId, 10)
  } else if (typeof closeId === 'number') {
    id = closeId
  } else {
    console.error('无效的ID类型:', typeof closeId, closeId)
    return
  }

  console.log('转换后的ID:', id, typeof id)

  if (isNaN(id) || id <= 0) {
    console.error('无效的订单ID:', id)
    return
  }

  viewOrderDetails(id)
}

// 处理订单平仓订单点击（带参数）
function handleOrderCloseOrderClick(closeId) {
  handleCloseOrderClick(closeId)
}

// ===== 订单状态和操作类型相关函数 =====

// 获取状态文本
function getStatusText(status) {
  const statusMap = {
    'pending': '等待执行',
    'processing': '执行中',
    'sent': '已发送',
    'success': '已提交',
    'filled': '已完成',
    'completed': '已完成',
    'failed': '执行失败',
    'cancelled': '已取消'
  }
  return statusMap[status] || status
}

// 获取增强的状态文本（考虑订单类型）
function getEnhancedStatusText(order) {
  const baseStatus = getStatusText(order.status)

  // 对于已完成的开仓订单，检查是否已被平仓
  if (['filled', 'completed'].includes(order.status) && !order.reduce_only) {
    // 检查是否有已完成的平仓订单
    if (order.related_orders && order.related_orders.has_close && order.related_orders.close_count > 0) {
      return '已结束'
    }
    return baseStatus
  }

  // 对于平仓订单，明确标识
  if (['filled', 'completed'].includes(order.status) && order.reduce_only) {
    return '已平仓'
  }

  return baseStatus
}

// 获取订单状态的CSS类（考虑订单类型）
function getOrderStatusClass(order) {
  // 对于已完成的开仓订单，检查是否已被平仓
  if (['filled', 'completed'].includes(order.status) && !order.reduce_only) {
    // 检查是否有已完成的平仓订单
    if (order.related_orders && order.related_orders.has_close && order.related_orders.close_count > 0) {
      return 'finished' // 已结束状态
    }
  }

  // 对于平仓订单，使用特殊的样式
  if (['filled', 'completed'].includes(order.status) && order.reduce_only) {
    return 'closed'
  }

  // 其他情况使用原有的状态
  return order.status
}

// 获取操作类型
function getOperationType(side, reduce_only) {
  if (reduce_only) {
    return side === 'BUY' ? '平空' : '平多'
  } else {
    return side === 'BUY' ? '开多' : '开空'
  }
}

// 获取操作描述
function getOperationDescription(side, reduce_only) {
  if (reduce_only) {
    return side === 'BUY' ? '买入平空仓位' : '卖出平多仓位'
  } else {
    return side === 'BUY' ? '买入开多仓位' : '卖出开空仓位'
  }
}

// 获取操作类型的CSS类
function getOperationClass(side, reduce_only) {
  if (reduce_only) {
    return side === 'BUY' ? 'close-short' : 'close-long'
  } else {
    return side === 'BUY' ? 'open-long' : 'open-short'
  }
}

// 格式化时间显示
function formatDateTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

// 切换订单展开状态
async function toggleOrderExpansion(orderId) {
  if (expandedOrders.value.has(orderId)) {
    // 收起
    expandedOrders.value.delete(orderId)
  } else {
    // 展开 - 获取完整订单详细信息
    try {
      const response = await api.getScheduledOrderDetail(orderId)
      const detailedOrder = response

      // 更新订单列表中的这个订单，添加完整的related_orders信息
      const orderIndex = orderList.value.findIndex(o => o.id === orderId)
      if (orderIndex !== -1) {
        orderList.value[orderIndex] = { ...orderList.value[orderIndex], ...detailedOrder }
      }

      // 添加到展开列表
      expandedOrders.value.add(orderId)
    } catch (error) {
      console.error('获取订单详情失败:', error)
      alert('获取订单详情失败，无法显示完整的交易链信息')
    }
  }
}


// 判断订单是否展开
function isOrderExpanded(orderId) {
  return expandedOrders.value.has(orderId)
}

// 判断订单是否有任何关联订单
function hasRelatedOrders(order) {
  // 检查子订单（平仓和加仓）
  if (order.childOrders && order.childOrders.length > 0) {
    return true
  }

  // 检查Bracket订单（TP/SL）
  if (order.related_orders && order.related_orders.has_bracket) {
    return true
  }

  // 检查其他关联订单
  if (order.related_orders && (
    (order.related_orders.close_orders && order.related_orders.close_orders.length > 0) ||
    (order.related_orders.parent_order) ||
    (order.related_orders.scaling_orders && order.related_orders.scaling_orders.length > 0)
  )) {
    return true
  }

  return false
}

// 获取关联订单总数
function getRelatedOrderCount(order) {
  let count = 0

  // 子订单数量（平仓和加仓）
  if (order.childOrders && order.childOrders.length > 0) {
    count += order.childOrders.length
  }

  // Bracket订单数量（TP/SL）
  if (order.related_orders && order.related_orders.has_bracket) {
    count += order.related_orders.bracket_count || 0
  }

  return count
}

// 点击其他地方时关闭下拉菜单
function closeDropdowns() {
  // 关闭订单列表的下拉菜单
  orderActiveDropdown.value = null
}

// 定义组件事件
const emit = defineEmits(['create-order', 'view-order-details'])

// 组件挂载时加载数据
onMounted(async () => {
  await loadOrderList()
})

// 暴露一些方法给父组件使用
defineExpose({
  loadOrderList,
  closeDropdowns
})
</script>

<style scoped>

/* ===== Bracket止盈止损订单样式 ===== */
.bracket-orders {
  margin-top: 16px;
}

.bracket-orders-list {
  margin-top: 8px;
}

.bracket-order-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  margin: 4px 0;
  border-radius: 6px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  transition: all 0.2s ease;
}

.bracket-order-item.will-be-deleted {
  background: #fef2f2;
  border-color: #fecaca;
}

.bracket-order-item.will-be-kept {
  background: #f0f9ff;
  border-color: #bae6fd;
}

.bracket-order-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.bracket-order-type {
  font-size: 12px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  color: white;
}

.bracket-order-type.tp-order {
  background: #059669;
}

.bracket-order-type.sl-order {
  background: #dc2626;
}

.bracket-order-symbol {
  font-weight: 500;
  color: #374151;
}

.bracket-order-price {
  font-size: 12px;
  color: #6b7280;
}

.bracket-order-status {
  font-size: 12px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
}

.bracket-order-action {
  font-size: 12px;
}

.action-text.delete-action {
  color: #dc2626;
  font-weight: 600;
}

.action-text.keep-action {
  color: #059669;
  font-weight: 600;
}

/* ===== 订单包装器样式 ===== */
.orders-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.order-wrapper {
  display: flex;
  flex-direction: column;
}

/* ===== 主订单卡片样式 ===== */
.main-card {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  transition: all 0.2s ease;
}

.main-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.main-card.has-children {
  border-left: 4px solid #2563eb;
}

/* ===== 展开指示器样式 ===== */
.expand-indicator {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f8fafc;
  border-top: 1px solid #e5e7eb;
  cursor: pointer;
  font-size: 14px;
  color: #374151;
  transition: background-color 0.2s ease;
  margin-top: 12px;
}

.expand-indicator:hover {
  background: #f0f4f8;
}

.expand-indicator.bottom {
  border-top: 1px solid #e5e7eb;
  margin-top: 12px;
}

.expand-icon {
  font-size: 12px;
  font-weight: bold;
  color: #6b7280;
}

.expand-text {
  font-weight: 500;
}

/* ===== 交易链标识样式 ===== */
.chain-indicator {
  margin-top: 4px;
}

.chain-badge {
  display: inline-block;
  padding: 2px 8px;
  background: #dbeafe;
  color: #1e40af;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

/* ===== 父订单样式 ===== */
.parent-order {
  margin-bottom: 16px;
}

.parent-card {
  border: 2px solid #2563eb !important;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
}

.parent-card .order-header {
  background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
  border-bottom: 2px solid #2563eb;
}

/* ===== 交易链容器样式 ===== */
.trade-chain-container {
  margin-left: 24px;
  border-left: 2px solid #e5e7eb;
  background: #f9fafb;
  border-radius: 6px;
  overflow: hidden;
}

.trade-chain-header {
  padding: 12px 16px;
  background: #f3f4f6;
  border-bottom: 1px solid #e5e7eb;
}

.trade-chain-header h5 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.trade-chain-section {
  border-bottom: 1px solid #e5e7eb;
}

.trade-chain-section:last-child {
  border-bottom: none;
}

.section-title {
  padding: 8px 16px;
  background: #f8fafc;
  font-size: 13px;
  font-weight: 600;
  color: #374151;
  border-bottom: 1px solid #e5e7eb;
}

.trade-chain-list {
  padding: 8px 0;
}

.trade-chain-item {
  padding: 8px 16px;
  border-bottom: 1px solid #f3f4f6;
}

.trade-chain-item:last-child {
  border-bottom: none;
}

.trade-chain-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.trade-chain-main {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  font-size: 13px;
}

.trade-chain-type {
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  text-transform: uppercase;
}

.trade-chain-type.open-long {
  background: #dcfce7;
  color: #166534;
}

.trade-chain-type.open-short {
  background: #fef2f2;
  color: #991b1b;
}

.trade-chain-type.close-long {
  background: #dbeafe;
  color: #1e40af;
}

.trade-chain-type.close-short {
  background: #fef3c7;
  color: #92400e;
}

.trade-chain-type.tp-type {
  background: #dcfce7;
  color: #166534;
}

.trade-chain-type.sl-type {
  background: #fef2f2;
  color: #991b1b;
}

.trade-chain-quantity {
  font-weight: 500;
  color: #111827;
}

.trade-chain-price {
  color: #6b7280;
}

.trade-chain-time {
  color: #9ca3af;
  font-size: 12px;
}

.trade-chain-status {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: 500;
}

.trade-chain-actions {
  display: flex;
  gap: 6px;
}

.btn-link.small {
  background: none;
  border: none;
  color: #3b82f6;
  cursor: pointer;
  font-size: 12px;
  padding: 2px 6px;
  text-decoration: underline;
}

.btn-link.small:hover {
  color: #2563eb;
}

.btn-danger.small {
  background: #dc2626;
  color: white;
  border: 1px solid #dc2626;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.btn-danger.small:hover {
  background: #b91c1c;
  border-color: #b91c1c;
}

/* ===== 订单列表筛选样式 ===== */

.filters-section {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  margin-bottom: 16px;
  overflow: hidden;
  transition: all 0.3s ease;
}

.filters-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  user-select: none;
  background: #f9fafb;
  border-bottom: 1px solid #e5e7eb;
  transition: background-color 0.2s ease;
}

.filters-header:hover {
  background: #f3f4f6;
}

.filters-title {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.toggle-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  background: none;
  border: none;
  color: #6b7280;
  font-size: 12px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.toggle-btn:hover {
  background: #e5e7eb;
  color: #374151;
}

.arrow {
  font-size: 10px;
  transition: transform 0.3s ease;
}

.toggle-btn.expanded .arrow {
  transform: rotate(180deg);
}

.filters-content {
  padding: 12px 16px;
}

.filters-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
  align-items: end;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.filter-label {
  font-size: 12px;
  font-weight: 500;
  color: #6b7280;
  margin-bottom: 2px;
}

.filter-select,
.filter-input {
  height: 32px;
  padding: 0 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 13px;
  background: #ffffff;
  color: #374151;
}

.filter-select:focus,
.filter-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.filter-actions {
  display: flex;
  justify-content: flex-end;
  align-items: flex-end;
}

.filter-actions .btn {
  height: 30px;
  padding: 0 10px;
  font-size: 12px;
  border-radius: 4px;
}

/* ===== 分页容器样式 ===== */
.pagination-container {
  margin-top: 24px;
  display: flex;
  justify-content: center;
}

.orders-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.order-card {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.15s;
}

.order-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.order-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f3f4f6;
}

.order-symbol {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  flex: 1;
}

.symbol-text {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

/* 关联订单指示器样式 */
.relation-indicator {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
}

.trade-chain {
  font-size: 11px;
  color: #7c3aed;
  background: #f3e8ff;
  padding: 2px 6px;
  border-radius: 10px;
  font-weight: 600;
  align-self: flex-start;
}

.relation-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 8px;
  font-weight: 500;
  display: inline-block;
}

.relation-badge.parent {
  background: #dbeafe;
  color: #1e40af;
  border: 1px solid #bfdbfe;
}

.relation-badge.close {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde047;
}

.exchange-badge {
  padding: 4px 8px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 500;
  text-transform: uppercase;
}

.exchange-badge:not(.testnet) {
  background: #dcfce7;
  color: #166534;
}

.exchange-badge.testnet {
  background: #fef3c7;
  color: #92400e;
}

.order-id {
  font-size: 11px;
  color: #9ca3af;
  font-weight: 500;
  font-family: 'Monaco', 'Menlo', monospace;
  margin-left: 8px;
}

.order-status {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 500;
}

.order-status.pending {
  background: #fef3c7;
  color: #92400e;
}

.order-status.processing {
  background: #dbeafe;
  color: #1e40af;
}

.order-status.completed {
  background: #dcfce7;
  color: #166534;
}

.order-status.closed {
  background: #ecfdf5;
  color: #047857;
  border: 1px solid #a7f3d0;
}

.order-status.finished {
  background: #f3e8ff;
  color: #6b21a8;
  border: 1px solid #c4b5fd;
}

.order-status.success {
  background: #fef3c7;
  color: #92400e;
}

.order-status.sent {
  background: #dbeafe;
  color: #1e40af;
}

.order-status.filled {
  background: #f0f9ff;
  color: #0c4a6e;
}

.order-status.failed {
  background: #fee2e2;
  color: #dc2626;
}

.order-status.cancelled {
  background: #f3f4f6;
  color: #6b7280;
}

.status-icon {
  font-size: 14px;
}

.order-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.detail-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.detail-label {
  color: #6b7280;
  font-weight: 500;
  min-width: 60px;
}

.detail-value {
  color: #374151;
  font-weight: 500;
}

.adjusted-quantity {
  text-decoration: line-through;
  color: #9ca3af;
}

.adjusted-info {
  color: #f59e0b;
  font-weight: 600;
  margin-left: 8px;
}

.detail-value.buy {
  color: #16a34a;
}

.detail-value.sell {
  color: #dc2626;
}

/* 新增的操作类型样式 */
.detail-value.open-long {
  color: #16a34a;
  font-weight: 600;
}

.detail-value.open-short {
  color: #dc2626;
  font-weight: 600;
}

.detail-value.close-long {
  color: #059669;
  font-weight: 600;
}

.detail-value.close-short {
  color: #b91c1c;
  font-weight: 600;
}

.detail-description {
  color: #6b7280;
  font-size: 12px;
  margin-left: 8px;
  font-weight: normal;
}

.trigger-time {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #f3f4f6;
}

.trigger-time .detail-value {
  color: #2563eb;
  font-weight: 600;
}

.bracket-info {
  margin-top: 12px;
  padding: 12px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.bracket-title {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 8px;
}

.bracket-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.bracket-item {
  font-size: 12px;
  color: #6b7280;
}

.order-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  padding-top: 16px;
  border-top: 1px solid #f3f4f6;
  flex-wrap: wrap;
}

.btn-small {
  height: 32px;
  padding: 0 12px;
  font-size: 12px;
  font-weight: 500;
  border-radius: 6px;
}

.btn-danger {
  background: #dc2626;
  color: white;
  border: 1px solid #dc2626;
}

.btn-danger:hover {
  background: #b91c1c;
  border-color: #b91c1c;
}

.btn-outline {
  background: #ffffff;
  color: #6b7280;
  border: 1px solid #d1d5db;
}

.btn-outline:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

/* 关联订单跳转按钮的特殊样式 */
.btn-outline[title*="开仓订单"],
.btn-outline[title*="平仓订单"] {
  background: #f8fafc;
  color: #3b82f6;
  border: 1px solid #bfdbfe;
}

.btn-outline[title*="开仓订单"]:hover,
.btn-outline[title*="平仓订单"]:hover {
  background: #eff6ff;
  border-color: #93c5fd;
}

/* 关联订单下拉菜单样式 */
.relation-dropdown-container {
  position: relative;
  display: inline-block;
}

.relation-dropdown {
  position: absolute;
  top: 100%;
  right: 0;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 180px;
  margin-top: 4px;
}

.dropdown-item {
  padding: 8px 12px;
  cursor: pointer;
  font-size: 12px;
  color: #374151;
  border-bottom: 1px solid #f3f4f6;
  transition: background-color 0.15s;
}

.dropdown-item:hover {
  background: #f8fafc;
}

.dropdown-item:last-child {
  border-bottom: none;
}

.btn:disabled {
  background: #f9fafb;
  color: #9ca3af;
  cursor: not-allowed;
}

/* ===== 加载和空状态样式 ===== */

.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #6b7280;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #e5e7eb;
  border-top: 3px solid #2563eb;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 12px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
  color: #6b7280;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.6;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 8px;
}

.empty-description {
  font-size: 14px;
  color: #9ca3af;
}

/* ===== 移动端样式 ===== */
@media (max-width: 768px) {
  .filters-header {
    padding: 10px 12px;
  }

  .filters-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .filter-group {
    gap: 3px;
  }

  .filter-actions {
    justify-content: center;
    margin-top: 6px;
  }

  .toggle-btn {
    font-size: 11px;
    padding: 3px 6px;
  }

  .orders-list {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .order-card {
    padding: 16px;
  }

  .order-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .order-symbol {
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
  }

  .symbol-text {
    font-size: 16px;
  }

  .order-details {
    gap: 6px;
  }

  .detail-row {
    font-size: 13px;
  }

  .detail-label {
    min-width: 50px;
  }

  .bracket-details {
    gap: 6px;
  }

  .order-actions {
    padding-top: 12px;
  }

  .btn-small {
    height: 36px;
    font-size: 13px;
  }

  /* ===== 移动端订单展示优化 ===== */

  .orders-container {
    grid-template-columns: 1fr; /* 移动端单列显示 */
    gap: 12px;
  }

  .expand-indicator {
    padding: 10px 12px;
    font-size: 13px;
  }

  .expand-text {
    font-size: 13px;
  }

  .expand-indicator.bottom {
    margin-top: 8px;
  }

  /* ===== 平板等中等屏幕优化 ===== */
  @media (max-width: 1024px) {
    .orders-container {
      grid-template-columns: 1fr; /* 平板也单列显示 */
    }
  }

  .trade-chain-container {
    margin-left: 16px;
  }

  .trade-chain-header {
    padding: 8px 12px;
  }

  .trade-chain-header h5 {
    font-size: 13px;
  }

  .trade-chain-card {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .trade-chain-main {
    flex-wrap: wrap;
    gap: 8px;
    width: 100%;
  }

  .trade-chain-actions {
    align-self: flex-end;
  }

  .pagination-container {
    margin-top: 20px;
  }
}

/* ===== 删除确认对话框样式 ===== */
.delete-confirmation-dialog-overlay {
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
  padding: 20px;
}

.delete-confirmation-dialog {
  background: #ffffff;
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  max-width: 500px;
  width: 100%;
  max-height: 90vh;
  overflow-y: auto;
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px 16px;
  border-bottom: 1px solid #e5e7eb;
}

.dialog-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.dialog-close-btn {
  background: none;
  border: none;
  font-size: 24px;
  color: #6b7280;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s ease;
}

.dialog-close-btn:hover {
  background: #f3f4f6;
  color: #374151;
}

.dialog-body {
  padding: 20px 24px;
}

.order-to-delete {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 20px;
}

.order-info-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.order-type-badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
}

.order-type-badge.open-long {
  background: #dcfce7;
  color: #166534;
}

.order-type-badge.open-short {
  background: #fef2f2;
  color: #991b1b;
}

.order-symbol {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.order-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-item .label {
  color: #6b7280;
  font-weight: 500;
  min-width: 50px;
}

.detail-item .value {
  color: #374151;
  font-weight: 500;
}

.status-text {
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
}

.status-text.pending {
  background: #fef3c7;
  color: #92400e;
}

.status-text.processing {
  background: #dbeafe;
  color: #1e40af;
}

.status-text.completed,
.status-text.filled {
  background: #dcfce7;
  color: #166534;
}

.status-text.failed {
  background: #fee2e2;
  color: #dc2626;
}

.cascade-options {
  border-top: 1px solid #e5e7eb;
  padding-top: 20px;
}

.options-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.option-group {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.option-radio {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
  padding: 12px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  transition: all 0.2s ease;
}

.option-radio:hover {
  border-color: #d1d5db;
  background: #f9fafb;
}

.option-radio input[type="radio"] {
  display: none;
}

.option-radio input[type="radio"]:checked + .radio-mark {
  background: #2563eb;
  border-color: #2563eb;
}

.option-radio input[type="radio"]:checked + .radio-mark::after {
  opacity: 1;
}

.radio-mark {
  width: 18px;
  height: 18px;
  border: 2px solid #d1d5db;
  border-radius: 50%;
  background: #ffffff;
  flex-shrink: 0;
  position: relative;
  transition: all 0.2s ease;
}

.radio-mark::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 8px;
  height: 8px;
  background: #ffffff;
  border-radius: 50%;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.option-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.option-text strong {
  font-size: 14px;
  color: #111827;
}

.option-desc {
  font-size: 13px;
  color: #6b7280;
}

.related-orders {
  margin-top: 16px;
}

.related-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
}

.related-orders-list {
  border: 1px solid #e5e7eb;
  border-radius: 6px;
  overflow: hidden;
}

.related-order-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #f3f4f6;
  transition: all 0.2s ease;
}

.related-order-item:last-child {
  border-bottom: none;
}

.related-order-item.will-be-deleted {
  background: #fef2f2;
  border-left: 4px solid #dc2626;
}

.related-order-item.will-be-kept {
  background: #f0fdf4;
  border-left: 4px solid #16a34a;
}

.related-order-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.related-order-type {
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
}

.related-order-type.close-long {
  background: #dbeafe;
  color: #1e40af;
}

.related-order-type.close-short {
  background: #fef3c7;
  color: #92400e;
}

.related-order-quantity {
  font-weight: 500;
  color: #374151;
}

.related-order-status {
  padding: 2px 6px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
}

.related-order-status.completed,
.related-order-status.filled {
  background: #dcfce7;
  color: #166534;
}

.related-order-status.pending {
  background: #fef3c7;
  color: #92400e;
}

.related-order-action {
  flex-shrink: 0;
}

.action-text {
  font-size: 12px;
  font-weight: 500;
  padding: 4px 8px;
  border-radius: 4px;
}

.action-text.delete-action {
  background: #dc2626;
  color: #ffffff;
}

.action-text.keep-action {
  background: #16a34a;
  color: #ffffff;
}

.warning-message {
  display: flex;
  gap: 12px;
  padding: 16px;
  background: #fef3c7;
  border: 1px solid #f59e0b;
  border-radius: 8px;
  margin-top: 16px;
}

.warning-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.warning-content {
  font-size: 13px;
  color: #92400e;
  line-height: 1.5;
}

.warning-content strong {
  color: #78350f;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px 20px;
  border-top: 1px solid #e5e7eb;
}

.dialog-footer .btn {
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 6px;
}

.loading-spinner.small {
  width: 14px;
  height: 14px;
  border: 2px solid #e5e7eb;
  border-top: 2px solid #dc2626;
  margin-right: 8px;
}

/* ===== 移动端对话框样式 ===== */
@media (max-width: 768px) {
  .delete-confirmation-dialog-overlay {
    padding: 10px;
  }

  .delete-confirmation-dialog {
    max-width: none;
    width: 100%;
    margin: 10px 0;
  }

  .dialog-header {
    padding: 16px 20px 12px;
  }

  .dialog-body {
    padding: 16px 20px;
  }

  .order-info-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .option-radio {
    padding: 10px;
  }

  .related-order-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .related-order-info {
    width: 100%;
    justify-content: space-between;
  }

  .dialog-footer {
    padding: 12px 20px 16px;
    flex-direction: column;
  }

  .dialog-footer .btn {
    width: 100%;
  }
}
</style>