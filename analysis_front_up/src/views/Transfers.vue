<template>
  <section class="panel">
    <div class="row">
      <h2>转账实时</h2>
      <div class="spacer"></div>

      <label>交易所：</label>
      <select v-model="entity" @change="reload">
        <option v-for="e in entities" :key="e" :value="e">{{ e }}</option>
      </select>

      <label>链：</label>
      <select v-model="chain" @change="reload">
        <option value="">全部</option>
        <option v-for="c in chainOptions" :key="c" :value="c">{{ c }}</option>
      </select>

      <label>币种：</label>
      <select v-model="coin" @change="reload">
        <option value="">全部</option>
        <option v-for="s in coinOptions" :key="s" :value="s">{{ s }}</option>
      </select>

      <label>显示条数：</label>
      <select v-model.number="limit" @change="reload">
        <option :value="20">20</option>
        <option :value="50">50</option>
        <option :value="100">100</option>
      </select>
    </div>
  </section>

  <section class="panel" style="margin-top:12px;">
    <div class="row" style="margin-bottom:10px;">
      <div class="badge">连接：<span :class="{'ok': connected, 'err': !connected}">{{ connected ? '已连接' : '未连接' }}</span></div>
      <div class="spacer"></div>
      <button class="primary" @click="reconnect">重连</button>
    </div>

    <div class="table-wrap">
      <table class="table">
        <thead>
        <tr>
          <th>时间 (UTC)</th>
          <th>链</th>
          <th>币种</th>
          <th>方向</th>
          <th>数量</th>
          <th>命中地址</th>
          <th>From → To</th>
          <th>TxID</th>
        </tr>
        </thead>
        <tbody>
        <tr v-for="it in items" :key="it.id">
          <td>{{ fmtTime(it.occurred_at) }}</td>
          <td>{{ it.chain }}</td>
          <td>{{ it.coin }}</td>
          <td><span :class="['pill', it.direction === 'in' ? 'in' : 'out']">{{ it.direction }}</span></td>
          <td class="mono">{{ fmtAmount(it.amount) }}</td>
          <td class="mono">{{ short(it.address) }}</td>
          <td class="mono">{{ short(it.from) }} → {{ short(it.to) }}</td>
          <td class="mono">
            <a class="link" :href="txLink(it.chain, it.txid)" target="_blank" rel="noreferrer">{{ short(it.txid) }}</a>
          </td>
        </tr>
        </tbody>
      </table>
    </div>

    <div class="row" style="justify-content:center; margin-top:10px;">
      <button class="primary" :disabled="loadingMore || !nextCursor" @click="loadMore">
        {{ nextCursor ? (loadingMore ? '加载中…' : '加载更多') : '没有更多了' }}
      </button>
    </div>
  </section>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { api } from '../api/api.js'

const entities = ref([])
const entity = ref('binance')

/** 新增筛选：链/币种 */
const chainOptions = ['bitcoin', 'ethereum', 'tron', 'solana']
const coinOptions  = ['BTC', 'ETH', 'USDT', 'USDC', 'SOL']
const chain = ref('')  // '' 表示全部
const coin  = ref('')  // '' 表示全部

const items = ref([])
const limit = ref(50)

let ws = null
const connected = ref(false)

const nextCursor = ref(null)
const loadingMore = ref(false)

const fmtTime = (s) => new Date(s).toISOString().replace('T',' ').replace('Z','')
const fmtAmount = (s) => Number(s).toLocaleString(undefined, { maximumFractionDigits: 8 })
const short = (s) => !s ? '-' : (String(s).length > 12 ? String(s).slice(0,6) + '…' + String(s).slice(-6) : s)
const txLink = (chain, txid) => {
  const c = String(chain).toLowerCase()
  if (c.includes('btc') || c === 'bitcoin') return `https://mempool.space/tx/${txid}`
  if (c.includes('eth') || c === 'ethereum') return `https://etherscan.io/tx/${txid}`
  if (c.includes('sol')) return `https://solscan.io/tx/${txid}`
  if (c.includes('tron')) return `https://tronscan.org/#/transaction/${txid}`
  return '#'
}

async function initEntities() {
  const r = await api.listEntities()
  entities.value = r.entities || []
  if (!entities.value.includes(entity.value) && entities.value.length) {
    entity.value = entities.value[0]
  }
}

async function loadRecent() {
  const r = await api.recentTransfers({
    entity: entity.value,
    chain: chain.value || undefined,
    coin : coin.value  || undefined,
    limit: limit.value
  })
  items.value = (r.items || [])
  nextCursor.value = r.next_cursor || null
}

async function loadMore() {
  if (!nextCursor.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const r = await api.recentTransfers({
      entity: entity.value,
      chain: chain.value || undefined,
      coin : coin.value  || undefined,
      limit: limit.value,
      before_ts: nextCursor.value.before_ts,
      before_id: nextCursor.value.before_id,
    })
    const more = r.items || []
    const seen = new Set(items.value.map(it => it.id))
    for (const it of more) {
      if (!seen.has(it.id)) items.value.push(it)
    }
    nextCursor.value = r.next_cursor || null
  } finally {
    loadingMore.value = false
  }
}

function openWS() {
  const url = api.wsTransfersURL({ entity: entity.value })
  if (ws) { ws.close(); ws = null }
  ws = new WebSocket(url)
  ws.onopen = () => { connected.value = true }
  ws.onclose = () => { connected.value = false }
  ws.onerror = () => { connected.value = false }
  ws.onmessage = (ev) => {
    try {
      const msg = JSON.parse(ev.data)
      if (msg.type === 'transfers' && Array.isArray(msg.data)) {
        // 前端再做一次链/币种过滤，避免筛选为 BTC 时仍插入 ETH
        const filtered = msg.data.filter(it => {
          if (chain.value && String(it.chain).toLowerCase() !== chain.value) return false
          if (coin.value  && String(it.coin).toUpperCase()   !== coin.value)  return false
          return true
        })
        if (filtered.length) {
          items.value = [...filtered, ...items.value].slice(0, limit.value)
        }
      }
    } catch (e) {}
  }
}

async function reload() { await loadRecent(); openWS() }
function reconnect(){ openWS() }

onMounted(async () => { await initEntities(); await reload() })
onBeforeUnmount(() => { if (ws) ws.close() })
</script>

<style scoped lang="scss">
.table-wrap { overflow: auto; }
.table { width: 100%; border-collapse: collapse; font-size: 14px; }
.table th, .table td { padding: 10px 8px; border-bottom: 1px solid var(--border); white-space: nowrap; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace; }
.pill { padding: 2px 8px; border-radius: 999px; font-size: 12px; border: 1px solid var(--border); }
.in { background: rgba(16,185,129,.12); border-color: rgba(16,185,129,.35); color: #16a34a; }
.out { background: rgba(239,68,68,.12); border-color: rgba(239,68,68,.35); color: #ef4444; }
.badge { background:#1a2233; border:1px solid var(--border); padding:4px 8px; border-radius:8px; font-size:12px; color:var(--muted); }
.badge .ok { color: #22c55e; } .badge .err { color: #ef4444; }
</style>
