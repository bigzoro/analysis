<template>
  <section class="panel" @click="closeDropdowns">
    <div class="row topbar">
      <h2>交易中心</h2>
      <div class="spacer"></div>
    </div>


    <!-- 选项卡切换 -->
    <div class="tabs">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'orders' }"
        @click="activeTab = 'orders'"
      >
        定时交易下单
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'order-list' }"
        @click="activeTab = 'order-list'"
      >
        订单列表
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'strategies' }"
        @click="activeTab = 'strategies'"
      >
        策略管理
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'market-analysis' }"
        @click="activeTab = 'market-analysis'"
      >
        市场分析
      </button>
    </div>

    <!-- 定时交易下单标签页 -->
    <div v-if="activeTab === 'orders'" class="tab-content">
      <TimedOrderForm @order-created="activeTab = 'order-list'" />
    </div>

            <!-- 策略管理标签页 -->
            <div v-if="activeTab === 'strategies'" class="tab-content">
              <StrategyManagement ref="strategyManagementRef" />
            </div>

    <!-- 市场分析标签页 -->
    <div v-if="activeTab === 'market-analysis'" class="tab-content">
      <MarketAnalysis />
    </div>

    <!-- 订单列表标签页 -->
    <div v-if="activeTab === 'order-list'" class="tab-content">
      <OrderList
        ref="orderListRef"
        @create-order="activeTab = 'orders'"
        @view-order-details="viewOrderDetails"
      />
    </div>



  </section>
</template>

<script setup>
import { reactive, ref, onMounted, computed, watch } from 'vue'
import { RouterLink, useRouter, useRoute } from 'vue-router'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'
import { executeStrategy, StrategyResult } from '../utils/strategy.js'
import { useAuth } from '../stores/auth.js'
import MarketAnalysis from '../components/analysis/MarketAnalysis.vue'
import TimedOrderForm from '../components/strategy/TimedOrderForm.vue'
import OrderList from '../components/strategy/OrderList.vue'
import StrategyManagement from '../components/strategy/StrategyManagement.vue'

const { isAuthed } = useAuth()
const router = useRouter()

// 标签页状态
const activeTab = ref('order-list') // 'orders', 'order-list' 或 'strategies'

// 组件引用
const orderListRef = ref(null)
const strategyManagementRef = ref(null)




// 订单创建回调函数现在由OrderList组件内部处理





// 加载数据（根据当前标签页）
async function loadData() {
  // 订单列表和策略管理的数据加载由各自组件自行处理
}









// 点击其他地方时关闭下拉菜单
function closeDropdowns() {
  // 关闭订单列表的下拉菜单
  if (orderListRef.value && orderListRef.value.closeDropdowns) {
    orderListRef.value.closeDropdowns()
  }

  // 关闭策略管理相关的下拉菜单（如果需要）
  if (strategyManagementRef.value && strategyManagementRef.value.closeDropdowns) {
    strategyManagementRef.value.closeDropdowns()
  }
}





// 查看订单详情
async function viewOrderDetails(id) {
  // 跳转到订单详情页面
  router.push(`/orders/schedule/${id}`)
}


// ===== 订单列表相关方法结束 =====


// 监听标签页切换
watch(activeTab, async (newTab) => {
  await loadData()
})

onMounted(async () => {
  // 检查URL参数中的tab设置
  const route = useRoute()
  const tabParam = route.query.tab
  if (tabParam && ['orders', 'order-list', 'strategies'].includes(tabParam)) {
    activeTab.value = tabParam
  }

  await loadData()
})


</script>

<style scoped>
:root{
  --text: #111827;
  --muted: #6b7280;
  --border: rgba(17,24,39,.12);
}
.panel { max-width: 1100px; margin: 0 auto; padding: 18px; color: var(--text); }
.topbar { align-items: center; }
.row { display:flex; gap: 10px; align-items:center; }
.row2{
  padding: 10px;
  border: 1px dashed darkgray;
  margin-bottom: 10px;
}
.topbar .spacer { flex:1; }
.single-column { display: flex; flex-direction: column; gap: 16px; }
.box {
  border:1px solid var(--border); border-radius: 12px; padding: 14px;
  background: var(--panel);
}
.form { display:grid; grid-template-columns: 160px 1fr; gap: 10px; align-items:center; }
.form h4 { grid-column: 1 / -1; margin: 4px 0; color: var(--muted); }
label { text-align:right; color:#4b5563; }
input, select {
  height: 36px; border:1px solid var(--border); border-radius:8px;
  background: var(--panel); color:var(--text); padding:0 10px;
}
.btn {
  height: 32px; padding: 0 12px; border-radius: 8px; border:1px solid var(--border);
  background:#f3f4f6; color:#111827; cursor:pointer;
}
.btn.primary { background:#2563eb; color:#fff; }
.btn.danger { background:#ef4444; color:#fff; }

/* 选项卡样式 */
.tabs {
  display: flex;
  border-bottom: 1px solid var(--border);
  margin-bottom: 16px;
}

.tab-btn {
  padding: 12px 24px;
  border: none;
  background: none;
  color: var(--muted);
  font-weight: 500;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s;
}

.tab-btn:hover {
  color: var(--text);
  background: rgba(0,0,0,0.05);
}

.tab-btn.active {
  color: var(--text);
  border-bottom-color: #2563eb;
}

.navigation-bar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 16px;
}

.nav-link {
  padding: 8px 16px;
  background: #2563eb;
  color: white;
  text-decoration: none;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 0.2s;
}

.nav-link:hover {
  background: #1d4ed8;
}

.tab-content {
  padding: 16px 0;
}


.condition-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.condition-tag {
  background: #e5e7eb;
  color: #374151;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 12px;
  white-space: nowrap;
}

/* 不同类型条件的标签颜色 */
.condition-tag.arb-strategy {
  background: #dbeafe;
  color: #1e40af;
  border: 1px solid #bfdbfe;
}

.condition-tag.tech-indicator {
  background: #f0f9ff;
  color: #0c4a6e;
  border: 1px solid #bae6fd;
}

.condition-tag.mean-reversion {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}

.condition-tag.risk-control {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}

.condition-tag.trading-direction {
  background: #f0fdf4;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.condition-tag.trading-config {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.condition-tag.timing-filter {
  background: #fefce8;
  color: #92400e;
  border: 1px solid #fde68a;
}


.preview-result {
  margin-top: 8px;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
}

.preview-result.buy {
  background: #dcfce7;
  border: 1px solid #16a34a;
  color: #166534;
}

.preview-result.sell {
  background: #fee2e2;
  border: 1px solid #dc2626;
  color: #991b1b;
}

.preview-result.skip {
  background: #fef3c7;
  border: 1px solid #d97706;
  color: #92400e;
}

.preview-result.no_op {
  background: #f3f4f6;
  border: 1px solid #6b7280;
  color: #374151;
}

.preview-result.error {
  background: #fef2f2;
  border: 1px solid #dc2626;
  color: #991b1b;
}

.preview-status {
  font-weight: bold;
  margin-bottom: 4px;
}

.preview-reason {
  margin-bottom: 4px;
}

.preview-multiplier {
  font-style: italic;
}

/* 符合条件的币种列表样式 */
.eligible-symbols-section {
  margin-top: 16px;
  padding: 16px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.symbols-header {
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.symbols-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
}

.symbol-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  transition: all 0.15s;
}

.symbol-item:hover {
  border-color: #2563eb;
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.1);
}

.symbol-item.selected {
  border-color: #2563eb;
  background: #eff6ff;
}

.symbol-checkbox {
  margin-right: 12px;
}

.symbol-checkbox input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.symbol-info {
  flex: 1;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

/* 批量操作按钮样式 */
.batch-actions {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
  display: flex;
  gap: 12px;
  justify-content: center;
}

.batch-create-btn {
  background: #2563eb !important;
  color: white !important;
  border-color: #2563eb !important;
  font-weight: 600;
}

.batch-create-btn:hover {
  background: #1d4ed8 !important;
  border-color: #1d4ed8 !important;
}

.batch-clear-btn {
  background: #f3f4f6 !important;
  color: #374151 !important;
  border-color: #d1d5db !important;
}

.batch-clear-btn:hover {
  background: #e5e7eb !important;
  border-color: #9ca3af !important;
}

.symbol-name {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
}

.symbol-details {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
}

.market-cap, .rank {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 三角套利路径样式 */
.triangle-path {
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 12px;
  width: 100%;
}

.path-label {
  font-weight: 500;
  color: #374151;
}

.path-symbols {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 3px;
  font-size: 11px;
  color: #1f2937;
}

.price-diff {
  font-weight: 600;
  font-size: 11px;
}

.price-diff.positive {
  color: #059669;
}

.price-diff.negative {
  color: #dc2626;
}


/* 策略管理样式已被移动到StrategyManagement组件中 */




/* 优化后的订单表单样式 */
.box-header {
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid #e5e7eb;
}

.box-header h3 {
  margin: 0 0 8px 0;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.box-description {
  color: #6b7280;
  font-size: 14px;
}

.order-form {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.form-section {
  padding: 20px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
}

.section-title {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  display: flex;
  align-items: center;
  gap: 8px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-hint {
  font-size: 11px;
  color: #6b7280;
  font-weight: 500;
  padding: 4px 8px;
  background-color: #f9fafb;
  border-radius: 4px;
  border: 1px solid #e5e7eb;
}

.form-warning {
  font-size: 11px;
  color: #dc2626;
  font-weight: 500;
  padding: 4px 8px;
  background-color: #fef2f2;
  border-radius: 4px;
  border: 1px solid #fecaca;
  margin-top: 4px;
}

.form-incomplete-notice {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background-color: #fefce8;
  border: 1px solid #fde047;
  border-radius: 6px;
  margin-top: 12px;
}

.notice-icon {
  font-size: 16px;
}

.notice-text {
  font-size: 13px;
  color: #92400e;
  font-weight: 500;
}

.form-label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 6px;
}

.required-mark {
  color: #dc2626;
  font-weight: 700;
  font-size: 14px;
}

.form-input,
.form-select {
  height: 40px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  color: #111827;
  padding: 0 12px;
  font-size: 14px;
  transition: border-color 0.15s;
}

.form-input:focus,
.form-select:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.form-select {
  cursor: pointer;
}

.bracket-settings {
  margin-top: 16px;
  padding: 16px;
  background: #ffffff;
  border: 1px solid #d1d5db;
  border-radius: 8px;
}

.bracket-notice {
  padding: 12px 16px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 6px;
  color: #1e40af;
  font-size: 13px;
  margin-bottom: 16px;
}

.bracket-divider {
  margin: 20px 0;
  text-align: center;
  position: relative;
}

.bracket-divider::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 0;
  right: 0;
  height: 1px;
  background: #e5e7eb;
}

.bracket-divider span {
  background: #f9fafb;
  padding: 0 12px;
  color: #6b7280;
  font-size: 12px;
  font-weight: 500;
}



.strategy-preview-section {
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.preview-header span {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.btn-outline {
  height: 32px;
  padding: 0 12px;
  border: 1px solid #d1d5db;
  background: #ffffff;
  color: #374151;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-outline:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.preview-content {
  flex: 1;
}

.preview-action {
  font-weight: 600;
  margin-bottom: 4px;
}

.preview-reason {
  font-size: 12px;
  opacity: 0.8;
}

.preview-multiplier {
  font-size: 12px;
  font-weight: 500;
  margin-top: 4px;
}

.form-actions {
  padding: 20px 0;
  border-top: 1px solid #e5e7eb;
  display: flex;
  justify-content: center;
  gap: 12px;
  flex-wrap: wrap;
}

.btn-large {
  height: 48px;
  padding: 0 32px;
  font-size: 16px;
  font-weight: 600;
  border-radius: 8px;
}

.form-message {
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  text-align: center;
}

.form-message.error {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.form-message.success {
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
}

/* 优化后的订单列表样式 */
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

.orders-list {
  display: flex;
  flex-direction: column;
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

.btn-warning {
  background: #d97706;
  color: white;
  border: 1px solid #d97706;
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

.btn-outline:hover {
  background: #f9fafb;
  border-color: #9ca3af;
}

.btn-outline:disabled {
  background: #f9fafb;
  color: #9ca3af;
  cursor: not-allowed;
}

/* 策略操作按钮样式 */

/* 双列布局样式 */
.two-column-layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.form-column {
  display: flex;
  flex-direction: column;
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
  backdrop-filter: blur(2px);
}



/* 启动策略表单样式 */

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: translateY(-20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 24px 32px 20px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.modal-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #111827;
}

.modal-close {
  background: none;
  border: none;
  font-size: 28px;
  color: #6b7280;
  cursor: pointer;
  padding: 0;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.modal-close:hover {
  background: #f3f4f6;
  color: #374151;
}

.modal-body {
  padding: 32px;
  padding-bottom: 0; /* 底部padding由form-actions提供 */
  flex: 1;
  overflow-y: auto;
  min-height: 0; /* 允许flex子项缩小 */
}

.strategy-form {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-label {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.form-input {
  height: 44px;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  background: #ffffff;
  color: #111827;
  padding: 0 16px;
  font-size: 14px;
  transition: border-color 0.15s;
}

.form-input:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.form-input::placeholder {
  color: #9ca3af;
}

.conditions-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  border-bottom: 2px solid #e5e7eb;
  padding-bottom: 8px;
}

.condition-group {
  margin-bottom: 24px;
}

.group-title {
  margin: 0 0 16px 0;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  display: flex;
  align-items: center;
  gap: 8px;
}

.condition-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.condition-row:last-child {
  margin-bottom: 0;
}

.checkbox-inline {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
  color: #6b7280;
  margin-left: 8px;
}

.inline-input.wide {
  min-width: 120px;
}

.inline-select {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  font-size: 14px;
  background: white;
  min-width: 80px;
}

/* 交易配置相关样式 */
.direction-selection {
  display: flex;
  gap: 16px;
  margin-top: 8px;
}

.direction-option {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  font-size: 14px;
  color: #374151;
}

.direction-option input[type="checkbox"] {
  position: absolute;
  opacity: 0;
  cursor: pointer;
}

.checkmark-small {
  width: 16px;
  height: 16px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  display: inline-block;
  position: relative;
  transition: all 0.2s;
}

.direction-option input[type="checkbox"]:checked + .checkmark-small {
  background-color: #10b981;
  border-color: #10b981;
}

.direction-option input[type="checkbox"]:checked + .checkmark-small::after {
  content: '✓';
  position: absolute;
  top: -2px;
  left: 2px;
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.leverage-config {
  margin-top: 8px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 14px;
}

.config-item label {
  min-width: 100px;
  color: #374151;
}

.config-note {
  margin-top: 8px;
  padding: 8px 12px;
  background: #f0f9ff;
  border: 1px solid #0ea5e9;
  border-radius: 6px;
  font-size: 13px;
  color: #0c4a6e;
}

.condition-card {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 20px;
  transition: all 0.15s;
}

.condition-card:hover {
  background: #f3f4f6;
  border-color: #d1d5db;
}

.condition-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.condition-checkbox {
  position: relative;
  cursor: pointer;
  display: flex;
  align-items: center;
}

.condition-checkbox input[type="checkbox"] {
  position: absolute;
  opacity: 0;
  cursor: pointer;
}

.checkmark {
  width: 20px;
  height: 20px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  background: white;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.condition-checkbox input[type="checkbox"]:checked + .checkmark {
  background: #2563eb;
  border-color: #2563eb;
}

.condition-checkbox input[type="checkbox"]:checked + .checkmark::after {
  content: '✓';
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.condition-title {
  font-size: 16px;
  font-weight: 500;
  color: #111827;
}

.condition-description {
  color: #6b7280;
  font-size: 14px;
  line-height: 1.5;
}

.inline-input {
  width: 60px;
  height: 28px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
  color: #111827;
  padding: 0 6px;
  font-size: 13px;
  text-align: center;
  transition: border-color 0.15s;
}

.inline-input:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.1);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 32px;
  border-top: 1px solid #e5e7eb;
  flex-shrink: 0; /* 防止按钮区域被压缩 */
  background: white; /* 确保按钮背景是白色 */
}

.btn {
  height: 40px;
  padding: 0 20px;
  border-radius: 8px;
  border: none;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.btn-secondary {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-secondary:hover {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.btn-primary {
  background: #2563eb;
  color: white;
}

.btn-primary:hover {
  background: #1d4ed8;
  transform: translateY(-1px);
}

.btn-batch-create {
  background: #16a34a;
  color: #fff;
  border: 1px solid #16a34a;
}

.btn-batch-create:hover {
  background: #15803d;
  border-color: #15803d;
  transform: translateY(-1px);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none !important;
}

.form-message {
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
}

.form-message.error {
  background: #fef2f2;
  color: #dc2626;
  border: 1px solid #fecaca;
}

.form-message.success {
  background: #f0fdf4;
  color: #16a34a;
  border: 1px solid #bbf7d0;
}

@media (max-width: 1100px) {
  .single-column { flex-direction: column; }
  .form { grid-template-columns: 120px 1fr; }
  .topbar { flex-direction: column; align-items:flex-start; }
  .two-column-layout { grid-template-columns: 1fr; gap: 16px; }

  /* 订单表单移动端优化 */
  .form-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .form-section {
    padding: 16px;
  }

  .section-title {
    font-size: 15px;
  }

  .bracket-settings {
    padding: 12px;
  }


  .preview-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .btn-large {
    width: 100%;
    height: 44px;
  }
}

@media (max-width: 640px) {
  .modal-overlay {
    padding: 16px;
  }


  .modal-header {
    padding: 20px 24px 16px;
  }

  .modal-header h3 {
    font-size: 18px;
  }

  .modal-body {
    padding: 24px 20px;
  }

  .condition-card {
    padding: 16px;
  }

  .condition-title {
    font-size: 15px;
  }

  .condition-description {
    font-size: 13px;
  }

  .inline-input {
    width: 50px;
    height: 26px;
    font-size: 12px;
  }

  .form-actions {
    flex-direction: column-reverse;
    gap: 8px;
  }

  .btn {
    width: 100%;
    height: 44px;
  }

  /* 订单表单移动端样式 */
  .box-header h3 {
    font-size: 16px;
  }

  .box-description {
    font-size: 13px;
  }

  .order-form {
    gap: 24px;
  }

  .form-section {
    padding: 12px;
  }

  .section-title {
    font-size: 14px;
    margin-bottom: 12px;
  }

  .bracket-notice {
    font-size: 12px;
    padding: 10px 12px;
  }

  /* 订单列表移动端样式 */
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

  /* 策略列表移动端样式 */
}

@media (max-width: 640px) {
  .modal-overlay {
    padding: 16px;
  }


  .modal-header {
    padding: 20px 24px 16px;
  }

  .modal-header h3 {
    font-size: 18px;
  }

  .modal-body {
    padding: 24px 20px;
  }

  .condition-card {
    padding: 16px;
  }

  .condition-title {
    font-size: 15px;
  }

  .condition-description {
    font-size: 13px;
  }

  .inline-input {
    width: 50px;
    height: 26px;
    font-size: 12px;
  }

  .form-actions {
    flex-direction: column-reverse;
    gap: 8px;
  }

  .btn {
    width: 100%;
    height: 44px;
  }
}

</style>
