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
        <div v-for="it in list" :key="it.id" class="row item">
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
      </div>
    </div>
  </section>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { api } from '../api/api.js'

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
  try {
    // ✅ 用你封装好的方法
    const res = await api.listScheduledOrders()
    // 后端返回的是 { items: [...] }
    list.value = Array.isArray(res?.items) ? res.items : []
  } catch (e) {
    err.value = e?.message || '加载失败'
  }
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

    // ✅ 用你封装好的方法
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
    // ✅ 用你封装好的方法
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
  --text: #e5e7eb;
  --muted: #94a3b8;
  --border: rgba(255,255,255,.12);
}
.panel { max-width: 1100px; margin: 0 auto; padding: 18px; color: var(--text); }
.topbar { align-items: center; }
.row { display:flex; gap: 10px; align-items:center; }
.topbar .spacer { flex:1; }
.grid { display:grid; grid-template-columns: 520px 1fr; gap: 16px; }
.box { border:1px solid var(--border); border-radius: 12px; padding: 14px; background: #0b1320; }
.form { display:grid; grid-template-columns: 160px 1fr; gap: 10px; align-items:center; }
.form h4 { grid-column: 1 / -1; margin: 4px 0; color: var(--muted); }
label { text-align:right; color:#cbd5e1; }
input, select { height: 36px; border:1px solid var(--border); border-radius:8px; background:#0b1320; color:var(--text); padding:0 10px; }
.btn { height: 32px; padding: 0 12px; border-radius: 8px; border:1px solid var(--border); background:#1f2937; color:#e5e7eb; }
.btn.primary { background:#334155; }
.btn.danger { background:#7f1d1d; border-color:#b91c1c; }
.err { color:#ef4444; font-size:12px; }
.ok { color:#22c55e; font-size:12px; }
.item { align-items: center; gap: 10px; padding: 10px; border-bottom:1px dashed rgba(255,255,255,.06); }
.col1 { flex:1; }
.col2 { width: 240px; }
.col3 { width: 120px; display:flex; justify-content:flex-end; }
.small { font-size:12px; }
.muted { color: var(--muted); }
@media (max-width: 900px){
  .grid { grid-template-columns: 1fr; }
  .form { grid-template-columns: 1fr; }
  label { text-align:left; }
  .col3 { width: auto; }
  .item { flex-direction: column; align-items:flex-start; }
}
</style>
