<template>
  <section class="panel">
    <div class="row topbar">
      <h2>定时合约下单</h2>
      <div class="spacer"></div>
      <button class="btn" @click="load">刷新</button>
    </div>

    <div class="grid">
      <!-- 左：创建计划单 -->
      <div class="box">
        <h3>新建计划</h3>
        <div class="form">
          <label>交易所</label>
          <select v-model="form.exchange">
            <option value="binance_futures">Binance Futures</option>
          </select>

          <label>环境</label>
          <select v-model="form.testnet">
            <option :value="true">测试网</option>
            <option :value="false">正式网</option>
          </select>

          <label>交易对</label>
          <input v-model="form.symbol" placeholder="例如：ETHUSDT" />

          <label>方向</label>
          <select v-model="form.side">
            <option value="BUY">BUY</option>
            <option value="SELL">SELL</option>
          </select>

          <label>订单类型</label>
          <select v-model="form.order_type">
            <option value="MARKET">MARKET</option>
            <option value="LIMIT">LIMIT</option>
          </select>

          <label>数量（基础币）</label>
          <input v-model="form.quantity" placeholder="例如：0.010" />

          <label v-if="form.order_type==='LIMIT'">限价</label>
          <input v-if="form.order_type==='LIMIT'" v-model="form.price" placeholder="仅限价单必填" />

          <label>杠杆</label>
          <input v-model.number="form.leverage" type="number" min="0" placeholder="0 或 正整数" />

          <label>只减仓</label>
          <select v-model="form.reduce_only">
            <option :value="false">否</option>
            <option :value="true">是</option>
          </select>

          <!-- === 新增：一键三连（Bracket） === -->
          <label>启用一键三连</label>
          <select v-model="form.bracket_enabled">
            <option :value="false">否</option>
            <option :value="true">是</option>
          </select>

          <div v-if="form.bracket_enabled">
            <label>止盈(%)</label>
            <input v-model.number="form.tp_percent" type="number" min="0" step="0.01" placeholder="例如 2 表示 +2%" />

            <label>止损(%)</label>
            <input v-model.number="form.sl_percent" type="number" min="0" step="0.01" placeholder="例如 1 表示 -1%" />

            <small class="muted">或直接使用绝对价格（百分比优先）：</small>

            <label>止盈价</label>
            <input v-model="form.tp_price" placeholder="可选" />

            <label>止损价</label>
            <input v-model="form.sl_price" placeholder="可选" />

            <label>触发价格</label>
            <select v-model="form.working_type">
              <option value="MARK_PRICE">MARK_PRICE(默认)</option>
              <option value="CONTRACT_PRICE">CONTRACT_PRICE</option>
            </select>
          </div>

          <label>触发时间</label>
          <input type="datetime-local" v-model="triggerLocal" />

          <button class="btn primary" @click="create">创建</button>
          <div class="small" :class="err ? 'err' : 'ok'">{{ err || ok }}</div>
        </div>
      </div>

      <!-- 右：计划列表 -->
      <div class="box">
        <h3>计划列表</h3>
        <div v-if="loading" class="loading">加载中...</div>
        <div v-else-if="list.length === 0" class="empty">暂无计划</div>
        <div v-else>
          <div v-for="it in list" :key="it.id" class="row row2 item">
          <div class="col1">
            <div><b>{{ it.symbol }}</b> · <span class="small muted">{{ it.exchange }} {{ it.testnet ? '(testnet)' : '' }}</span></div>
            <div class="small">
              {{ it.side }}/{{ it.order_type }} · qty={{ it.quantity }}<span v-if="it.price"> · px={{ it.price }}</span>
              <span v-if="it.leverage"> · lev={{ it.leverage }}</span>
              <span v-if="it.reduce_only"> · reduceOnly</span>
            </div>
            <div class="small" v-if="it.bracket_enabled">
              Bracket: TP%={{ it.tp_percent || 0 }} SL%={{ it.sl_percent || 0 }}
              <span v-if="it.tp_price"> · TP={{ it.tp_price }}</span>
              <span v-if="it.sl_price">   · SL={{ it.sl_price }}</span>
                · WorkingType={{ it.working_type || 'MARK_PRICE' }}
            </div>
            <div class="small muted">触发：{{ toLocal(it.trigger_time) }}</div>
            <div class="small">状态：<b>{{ it.status }}</b>
<!--              · <span class="muted">{{ it.result }}</span>-->
            </div>
          </div>
          <div class="col3">
            <button v-if="['pending'     ,'processing'].includes(it.status)" class="btn danger" @click="cancel(it.id)">取消</button>
          </div>
        </div>
        
        <!-- 统一分页组件 -->
        <Pagination
          v-if="total > 0"
          v-model:page="page"
          v-model:pageSize="pageSize"
          :total="total"
          :totalPages="totalPages"
          :loading="loading"
          @change="onPaginationChange"
        />
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { api } from '../api/api.js'
import Pagination from '../components/Pagination.vue'

const form = reactive({
  exchange: 'binance_futures',
  testnet: true,
  symbol: 'ETHUSDT',
  side: 'BUY',
  order_type: 'MARKET',
  quantity: '0.010',
  price: '',
  leverage: 0,
  reduce_only: false,

  // === Bracket ===
  bracket_enabled: false,
  tp_percent: 0,
  sl_percent: 0,
  tp_price: '',
  sl_price: '',
  working_type: 'MARK_PRICE',
})

// 触发时间：默认当前时间 + 1 分钟
const triggerLocal = ref(new Date(Date.now() + 60_000).toISOString().slice(0, 16))

const err = ref('')
const ok = ref('')
const list = ref([])
const page = ref(1)
const pageSize = ref(50)
const total = ref(0)
const totalPages = ref(1)
const loading = ref(false)

function toRFC3339FromLocal(dtLocal) {
  const d = new Date(dtLocal)
  return d.toISOString()
}

function toLocal(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function load() {
  err.value = ''
  loading.value = true
  try {
    const res = await api.listScheduledOrders({ page: page.value, page_size: pageSize.value })
    list.value = Array.isArray(res?.items) ? res.items : []
    total.value = res?.total || 0
    totalPages.value = res?.total_pages || 1
    page.value = res?.page || page.value
  } catch (e) {
    err.value = e?.message || '加载失败'
  } finally {
    loading.value = false
  }
}

function onPaginationChange({ page: newPage, pageSize: newPageSize }) {
  page.value = newPage
  pageSize.value = newPageSize
  load()
}

async function create() {
  err.value = ''
  ok.value = ''
  try {
    const payload = {
      exchange: form.exchange,
      testnet: form.testnet,
      symbol: form.symbol,
      side: form.side,
      order_type: form.order_type,
      quantity: form.quantity,
      price: form.order_type === 'LIMIT' ? form.price : '',
      leverage: form.leverage,
      reduce_only: form.reduce_only,
      trigger_time: toRFC3339FromLocal(triggerLocal.value),

      // bracket
      bracket_enabled: form.bracket_enabled,
      tp_percent: form.tp_percent,
      sl_percent: form.sl_percent,
      tp_price: form.tp_price,
      sl_price: form.sl_price,
      working_type: form.working_type,
    }

    const r = await api.scheduleOrder(payload)
    ok.value = r?.id ? `创建成功（ID: ${r.id}）` : '创建成功'
    await load()
  } catch (e) {
    err.value = e?.message || '创建失败'
  }
}

async function cancel(id) {
  if (!confirm('确认取消该计划？')) return
  err.value = ''
  try {
    await api.cancelScheduledOrder(id)
    await load()
  } catch (e) {
    err.value = e?.message || '取消失败'
  }
}

onMounted(load)
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
.grid { display:grid; grid-template-columns: 520px 1fr; gap: 16px; }
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
@media (max-width: 1100px) {
  .grid { grid-template-columns: 1fr; }
  .form { grid-template-columns: 120px 1fr; }
  .topbar { flex-direction: column; align-items:flex-start; }
}
</style>
