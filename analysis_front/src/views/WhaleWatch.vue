<template>
  <div class="container">
    <!-- 现代化页面头部 -->
    <section class="page-header">
      <div class="header-gradient">
        <div class="header-content">
          <!-- 面包屑导航 -->
          <nav class="breadcrumb-nav">
            <div class="breadcrumb">
              <span class="breadcrumb-item">
                <i class="icon-chart">📊</i>
                数据监控
              </span>
              <span class="breadcrumb-separator">/</span>
              <span class="breadcrumb-item active">
                <i class="icon-whale">🐋</i>
                大户监控
              </span>
            </div>
          </nav>

          <!-- 标题区域 -->
          <div class="title-section">
            <div class="title-content">
              <h1 class="page-title">
                大户 & 机构地址监控
              </h1>
              <p class="page-subtitle">
                实时监控区块链大户和机构的资金流动，支持多数据源智能聚合分析
        </p>
      </div>
            <div class="title-visual">
              <div class="floating-shapes">
                <div class="shape shape-1"></div>
                <div class="shape shape-2"></div>
                <div class="shape shape-3"></div>
              </div>
            </div>
          </div>

          <!-- 紧凑的控制面板 -->
          <div class="header-controls">
            <div class="control-row">
              <!-- 数据源选择 -->
              <div class="control-item">
                <label class="control-label">
                  数据源
                </label>
                <div class="select-container">
                  <select v-model="dataSource" @change="onDataSourceChange" class="modern-select">
                    <option value="basic">基本监控</option>
                    <option value="arkham">Arkham</option>
                    <option value="nansen">Nansen</option>
        </select>
                  <i class="select-arrow">▼</i>
                </div>
              </div>

              <!-- 实体选择 -->
              <div class="control-item">
                <label class="control-label">
                  默认实体
                </label>
                <div class="select-container">
                  <select v-model="entity" @change="onEntityChange" class="modern-select">
          <option v-for="ent in entities" :key="ent" :value="ent">{{ ent }}</option>
        </select>
                  <i class="select-arrow">▼</i>
                </div>
              </div>

              <!-- 快速操作按钮 -->
              <div class="control-actions">
                <button
                  class="btn-primary btn-compact"
                  @click="refreshWatchEvents"
                  :disabled="loading"
                  :class="{ loading }"
                  title="刷新所有监控地址的最新数据"
                >
                  <i class="icon-refresh" :class="{ spinning: loading }">🔄</i>
                  <span class="btn-text">{{ loading ? '刷新中' : '刷新数据' }}</span>
        </button>
                <button
                  v-if="dataSource !== 'basic'"
                  class="btn-secondary btn-compact"
                  @click="syncExternalData"
                  :disabled="syncing"
                  :class="{ loading: syncing }"
                  title="从外部数据源同步最新数据"
                >
                  <i class="icon-sync" :class="{ spinning: syncing }">⚡</i>
                  <span class="btn-text">{{ syncing ? '同步中' : '外部同步' }}</span>
        </button>
      </div>
    </div>
          </div>
        </div>
    </div>
  </section>

    <!-- 现代化统计概览 -->
    <section class="stats-overview">
      <div class="stats-header">
        <h2 class="stats-title">
          监控概览
        </h2>
        <p class="stats-subtitle">实时监控状态与关键指标</p>
      </div>

      <div class="stats-grid">
        <!-- 监控地址卡片 -->
        <div class="stat-card primary" :class="{ 'pulse': summary.totalWatchers > 0 }">
          <div class="card-content">
            <div class="stat-details">
              <div class="stat-value animate-number" data-target="{{ summary.totalWatchers }}">
                {{ summary.totalWatchers }}
              </div>
              <div class="stat-label">地址</div>
              <div class="stat-meta">
                <span class="meta-indicator active"></span>
                正在监控
              </div>
            </div>
          </div>
        </div>

        <!-- 活跃地址卡片 -->
        <div class="stat-card success" :class="{ 'bounce': summary.activeWatchers > 0 }">
          <div class="card-content">
            <div class="stat-details">
              <div class="stat-value animate-number" data-target="{{ summary.activeWatchers }}">
                {{ summary.activeWatchers }}
              </div>
              <div class="stat-label">活跃地址</div>
              <div class="stat-meta">
                <span class="meta-indicator success"></span>
                最近交易
              </div>
            </div>
          </div>
        </div>

        <!-- 交易事件卡片 -->
        <div class="stat-card info">
          <div class="card-content">
            <div class="stat-details">
              <div class="stat-value animate-number" data-target="{{ summary.totalEvents }}">
                {{ summary.totalEvents }}
              </div>
              <div class="stat-label">交易事件</div>
              <div class="stat-meta">
                <span class="meta-indicator info"></span>
                累计命中
              </div>
            </div>
          </div>
        </div>

        <!-- 最大交易卡片 -->
        <div class="stat-card warning">
          <div class="card-content">
            <div class="stat-details">
              <div class="stat-value large-amount">
                {{ summary.largestLabel || '暂无' }}
              </div>
              <div class="stat-label">最大单笔</div>
              <div class="stat-meta">
                <span class="meta-indicator warning"></span>
                按金额排序
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 实时状态栏 -->
      <div class="status-dashboard">
        <div class="status-metrics">
          <div class="metric-item">
            <div class="metric-icon">
              <div class="status-pulse" :class="{ active: !loading }"></div>
            </div>
            <div class="metric-content">
              <div class="metric-label">同步状态</div>
              <div class="metric-value">{{ loading ? '更新中...' : '已同步' }}</div>
            </div>
          </div>

          <div class="metric-item">
            <div class="metric-icon">
              <i class="icon-time">🕐</i>
            </div>
            <div class="metric-content">
              <div class="metric-label">最后更新</div>
              <div class="metric-value">{{ summary.lastRefreshLabel || '从未更新' }}</div>
            </div>
          </div>

          <div class="metric-item">
            <div class="metric-icon">
              <i class="icon-source">🔗</i>
            </div>
            <div class="metric-content">
              <div class="metric-label">数据源</div>
              <div class="metric-value">{{ getDataSourceLabel(dataSource) }}</div>
            </div>
          </div>
        </div>

        <!-- 进度指示器 -->
        <div v-if="loading" class="progress-indicator">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="progress-text">{{ progressText }}</div>
        </div>
      </div>
    </section>

    <!-- 现代化智能查询面板 -->
    <section class="query-panel" :class="{ compact: !showQueryPanel }">
      <!-- 面板头部 -->
      <div class="panel-header">
        <div class="header-content">
          <div class="panel-title-section">
            <div class="panel-title">
              <div class="title-text">
                <h3>智能地址查询</h3>
                <p class="panel-subtitle">输入区块链地址进行快速查询，或将其添加到监控列表进行持续追踪</p>
              </div>
            </div>
          </div>
          <div class="header-actions">
            <button
              class="btn-link toggle-panel"
              @click="toggleQueryPanel"
            >
              <i class="icon-toggle">{{ showQueryPanel ? '🔽' : '🔼' }}</i>
              {{ showQueryPanel ? '收起查询' : '展开查询' }}
            </button>
          </div>
        </div>
      </div>

      <!-- 查询表单 -->
      <div v-show="showQueryPanel" class="query-form">
        <!-- 主要输入区域 -->
        <div class="form-primary">
          <div class="address-input-section">
            <div class="input-group featured">
              <div class="input-header">
                <label class="input-label required">
                  <i class="icon-address">📋</i>
                  区块链地址
                </label>
                <div class="input-badges">
                  <span v-if="addressValidation.isValid" class="badge valid">
                    <i class="icon-check">✓</i>
                    {{ addressValidation.chain }}
                  </span>
                  <span v-else-if="queryData.address && !addressValidation.isValid" class="badge invalid">
                    <i class="icon-warning">⚠</i>
                    格式无效
                  </span>
                </div>
              </div>

              <div class="input-container">
                <input
                  v-model.trim="queryData.address"
                  type="text"
                  placeholder="输入完整的区块链地址，支持 ETH、BSC、BTC、SOL 等多种链"
                  class="address-input"
                  :class="{ valid: addressValidation.isValid, invalid: queryData.address && !addressValidation.isValid }"
                  @paste="onAddressPaste"
                  @input="validateAddress"
                  ref="addressInput"
                />

                <div class="input-actions">
                  <button
                    v-if="queryData.address"
                    class="action-btn clear"
                    @click="queryData.address = ''"
                    title="清空地址"
                  >
                    <i class="icon-clear">✕</i>
                  </button>
                  <button
                    class="action-btn paste"
                    @click="pasteFromClipboard"
                    title="从剪贴板粘贴"
                  >
                    <i class="icon-paste">📄</i>
                  </button>
                </div>
              </div>

              <div class="input-footer">
                <div class="input-examples">
                  <span class="example-label">示例:</span>
                  <button
                    class="example-btn"
                    @click="setExampleAddress('ethereum')"
                    title="Ethereum 地址示例"
                  >
                    ETH: 0x3f5CE5FBFe3E9af3971dD833D26BA9b5C936f0bE
                  </button>
                  <button
                    class="example-btn"
                    @click="setExampleAddress('bitcoin')"
                    title="Bitcoin 地址示例"
                  >
                    BTC: 34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 快速操作按钮 -->
          <div class="quick-actions">
            <button
              class="btn-primary large"
              :disabled="!canQuery"
              @click="queryAddressOnce"
              :class="{ loading: queryLoading }"
            >
              <div class="btn-content">
                <i class="icon-search-btn" :class="{ spinning: queryLoading }">🔍</i>
                <span class="btn-text">{{ queryLoading ? '查询中...' : '立即查询' }}</span>
              </div>
            </button>

            <button
              class="btn-success large"
              :disabled="!canAdd"
              @click="addToWatchlist"
            >
              <div class="btn-content">
                <i class="icon-add">➕</i>
                <span class="btn-text">添加监控</span>
              </div>
            </button>
          </div>
        </div>

        <!-- 高级选项 -->
        <div class="form-advanced">
          <div class="advanced-toggle">
            <button
              class="toggle-btn"
              @click="showAdvanced = !showAdvanced"
              :class="{ active: showAdvanced }"
            >
              <i class="icon-settings">⚙️</i>
              <span>高级选项</span>
              <i class="icon-chevron" :class="{ rotated: showAdvanced }">▼</i>
            </button>
          </div>

          <div v-if="showAdvanced" class="advanced-options">
            <div class="options-grid">
              <div class="option-group">
                <label class="option-label">
                  <i class="icon-tag">🏷️</i>
                  地址标签
                </label>
                <input
                  v-model.trim="queryData.label"
                  type="text"
                  placeholder="为地址添加备注标签"
                  class="option-input"
                />
              </div>

              <div class="option-group">
                <label class="option-label">
                  <i class="icon-chain">⛓️</i>
                  指定链
                </label>
                <select v-model="queryData.chain" class="option-select">
                  <option value="">🤖 自动检测</option>
                  <option v-for="c in chainOptions" :key="c" :value="c">
                    {{ getChainIcon(c) }} {{ getChainName(c) }}
                  </option>
      </select>
              </div>

              <div class="option-group">
                <label class="option-label">
                  关联实体
                </label>
                <select v-model="queryData.entity" class="option-select">
                  <option value="">继承默认 ({{ entity }})</option>
        <option v-for="ent in entities" :key="ent" :value="ent">{{ ent }}</option>
      </select>
              </div>
    </div>

            <div class="advanced-actions">
              <button
                class="btn-outline small"
                @click="resetQueryForm"
              >
                <i class="icon-reset">🔄</i>
                重置表单
              </button>
    </div>
          </div>
        </div>

        <!-- 状态提示 -->
        <div v-if="formNotice" class="form-notice animate-slide-up" :class="noticeType">
          <div class="notice-content">
            <i :class="getNoticeIcon(noticeType)" class="notice-icon"></i>
            <span class="notice-text">{{ formNotice }}</span>
            <button
              class="notice-close"
              @click="formNotice = ''"
              title="关闭提示"
            >
              ✕
            </button>
          </div>
        </div>

      </div>
    </section>

    <!-- 查询结果显示 -->
    <div v-if="queryResult" class="query-result">
      <h4>查询结果</h4>
      <div class="result-card">
        <header class="result-header">
          <div>
            <h3>{{ queryResult.label || '查询地址' }}</h3>
            <p class="mono">{{ queryResult.address }}</p>
          </div>
        </header>

        <div class="result-meta">
          <span><strong>链：</strong>{{ queryResult.chain || '全部' }}</span>
          <span><strong>实体：</strong>{{ queryResult.entity || entity }}</span>
          <span v-if="queryResult.balance_usd"><strong>余额：</strong>${{ queryResult.balance_usd }}</span>
          <span v-if="queryResult.last_active_at"><strong>最后活跃：</strong>{{ fmtTime(queryResult.last_active_at) }}</span>
          <span><strong>查询时间：</strong>{{ fmtTime(queryResult.queried_at) }}</span>
          <span v-if="queryResult.dataSource" class="data-source-badge">{{ getDataSourceLabel(queryResult.dataSource) }}</span>
        </div>

        <!-- API错误提示 -->
        <div v-if="queryResult.api_error" class="api-error-notice">
          <p>⚠️ {{ queryResult.error_message }}</p>
        </div>

        <!-- 演示数据提示 -->
        <div v-if="queryResult.demo_data" class="demo-notice">
          <p>ℹ️ {{ queryResult.demo_message }}</p>
        </div>

        <div v-if="queryResult.transactions && queryResult.transactions.length > 0" class="result-transactions">
          <h5>最近交易</h5>
          <div class="transactions-list">
            <div v-for="tx in queryResult.transactions.slice(0, 5)" :key="tx.transaction_hash || tx.tx_hash" class="transaction-item">
              <div class="tx-top">
                <span class="pill" :class="tx.direction === 'in' ? 'in' : 'out'">
                  {{ tx.direction === 'in' ? '流入' : '流出' }}
                </span>
                <strong v-if="tx.symbol && tx.amount">
                  {{ fmtAmount(tx.amount) }} {{ tx.symbol }}
                </strong>
                <strong v-else-if="tx.volume_usd">
                  {{ fmtAmount(tx.volume_usd) }} USD
                </strong>
                <strong v-else>
                  {{ fmtAmount(tx.value_usd || '0') }} USD
                </strong>
                <span class="tx-time">{{ fmtTime(tx.occurred_at || tx.block_timestamp) }}</span>
              </div>
              <div class="tx-bottom">
                <div class="tx-hash">
                  <span>Hash: {{ shortAddress(tx.transaction_hash || tx.tx_hash) }}</span>
              </div>
                <!-- 显示代币转账详情 -->
                <div v-if="tx.tokens_received && tx.tokens_received.length > 0" class="token-details">
                  <div v-for="token in tx.tokens_received" :key="token.token_address" class="token-transfer">
                    <span class="token-info">
                      📥 {{ fmtAmount(token.token_amount) }} {{ token.token_symbol }}
                      <span v-if="token.value_usd">({{ fmtAmount(token.value_usd) }} USD)</span>
                    </span>
                    <span class="address-info">
                      <span v-if="token.from_address_label" class="address-label">{{ token.from_address_label }}</span>
                      → <span v-if="token.to_address_label" class="address-label">{{ token.to_address_label }}</span>
                    </span>
                  </div>
                </div>
                <div v-else-if="tx.tokens_sent && tx.tokens_sent.length > 0" class="token-details">
                  <div v-for="token in tx.tokens_sent" :key="token.token_address" class="token-transfer">
                    <span class="token-info">
                      📤 {{ fmtAmount(token.token_amount) }} {{ token.token_symbol }}
                      <span v-if="token.value_usd">({{ fmtAmount(token.value_usd) }} USD)</span>
                    </span>
                    <span class="address-info">
                      <span v-if="token.from_address_label" class="address-label">{{ token.from_address_label }}</span>
                      → <span v-if="token.to_address_label" class="address-label">{{ token.to_address_label }}</span>
                    </span>
                  </div>
                </div>
                <!-- 显示交易方法和来源类型 -->
                <div v-if="tx.method || tx.source_type" class="tx-meta">
                  <span v-if="tx.method" class="method">方法: {{ tx.method }}</span>
                  <span v-if="tx.source_type" class="source-type">来源: {{ tx.source_type }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="no-transactions">
          <p>暂无交易记录</p>
          <div class="sync-hint">
            <p class="muted">
              💡 提示：
              <span v-if="queryResult.dataSource === 'basic'">
                基本监控需要先同步数据，或尝试添加到监控列表后同步
              </span>
              <span v-else-if="queryResult.dataSource === 'arkham'">
                Arkham数据可能需要额外同步，或此地址近期无活跃交易
              </span>
              <span v-else-if="queryResult.dataSource === 'nansen'">
                Nansen API需要有效的API Key。如遇认证问题，可切换到基本监控或Arkham数据源
              </span>
              <span v-else>
                建议添加到监控列表并同步数据
              </span>
            </p>
            <div class="sync-actions">
              <button class="btn-sync" @click="syncExternalData" :disabled="syncing">
                {{ syncing ? '同步中...' : '🔄 同步数据' }}
              </button>
              <button class="btn-add" @click="addCurrentQueryToWatch" :disabled="!queryResult.address">
                📊 添加到监控
              </button>
              <button v-if="queryResult.dataSource === 'nansen'" class="btn-switch" @click="switchToBasicMonitoring">
                🔄 切换到基本监控
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 监控列表面板 -->
    <section class="watch-list-panel">
      <div class="stats-header">
        <h2 class="stats-title">
          监控地址列表
        </h2>
        <p class="stats-subtitle">{{ watchlist.length ? '正在监控的区块链地址和资金流动' : '开始监控您的第一个地址' }}</p>
      </div>

        <!-- 列表操作栏 -->
        <div class="list-controls">
          <!-- 搜索和过滤 -->
          <div class="search-filter">
            <div class="search-input-wrapper">
              <input
                v-model="searchQuery"
                type="text"
                placeholder="搜索地址或标签..."
                class="search-input"
              />
              <i class="search-icon">🔍</i>
            </div>
            <select v-model="filterChain" class="filter-select">
              <option value="">全部链</option>
              <option v-for="c in chainOptions" :key="c" :value="c">{{ getChainName(c) }}</option>
            </select>
          </div>

          <!-- 批量操作 -->
          <div class="bulk-actions">
            <button
              v-if="selectedAddresses.length > 0"
              class="btn-danger"
              @click="bulkRemove"
            >
              <i class="icon-delete">🗑️</i>
              删除选中 ({{ selectedAddresses.length }})
            </button>
          </div>
        </div>

        <!-- 状态标签 -->
        <div class="status-badges">
          <div class="status-badge" :class="{ active: summary.activeWatchers > 0 }">
            活跃监控 {{ summary.activeWatchers }}/{{ summary.totalWatchers || 0 }}
          </div>
          <div class="status-badge" :class="{ warning: summary.totalEvents === 0 }">
            交易事件 {{ summary.totalEvents }}
          </div>
          <div class="status-badge info">
            <i class="icon-last-update">🕐</i>
            {{ summary.lastRefreshLabel || '未刷新' }}
          </div>
        </div>

        <!-- 快速筛选按钮 -->
        <div class="quick-filters">
          <button
            class="filter-btn"
            :class="{ active: filterStatus === 'all' }"
            @click="setFilterStatus('all')"
          >
            全部 ({{ filteredWatchlist.length }})
          </button>
          <button
            class="filter-btn"
            :class="{ active: filterStatus === 'active' }"
            @click="setFilterStatus('active')"
          >
            活跃 ({{ summary.activeWatchers }})
          </button>
          <button
            class="filter-btn"
            :class="{ active: filterStatus === 'inactive' }"
            @click="setFilterStatus('inactive')"
          >
            未活跃 ({{ summary.totalWatchers - summary.activeWatchers }})
          </button>
          <button
            class="filter-btn"
            :class="{ active: filterStatus === 'error' }"
            @click="setFilterStatus('error')"
          >
            错误
          </button>
        </div>

      <!-- 空状态 -->
      <div v-if="filteredWatchlist.length === 0" class="empty-state">
        <div class="empty-icon">📭</div>
        <h4 class="empty-title">
          {{ watchlist.length === 0 ? '还没有监控地址' : '没有找到匹配的地址' }}
        </h4>
        <p class="empty-description">
          {{ watchlist.length === 0
            ? '添加您的第一个监控地址，开始追踪链上资金流动'
            : '调整搜索条件或清除过滤器'
          }}
        </p>
        <div class="empty-actions">
          <button class="btn-primary" @click="scrollToQuery">
            <i class="icon-add">➕</i>
            添加监控地址
          </button>
          <button v-if="searchQuery || filterChain" class="btn-outline" @click="clearFilters">
            <i class="icon-clear">🔄</i>
            清除过滤
          </button>
        </div>
      </div>

      <!-- 监控地址网格 -->
      <!-- 虚拟滚动容器 -->
      <div v-if="filteredWatchlist.length > 50" class="watch-grid virtual-scroll-wrapper">
        <div
          class="virtual-scroll-viewport"
          :style="{ height: containerHeight + 'px' }"
          @scroll="handleScroll"
        >
          <div class="virtual-scroll-content" :style="{ transform: `translateY(${virtualScrollOffset}px)` }">
            <article
              v-for="watch in visibleWatchlist"
              :key="watch.address"
              class="watch-card"
              :class="{ selected: isSelected(watch.address) }"
              :style="{ height: itemHeight + 'px' }"
            >
          <!-- 卡片头部 -->
          <div class="card-header">
            <div class="card-title-section">
              <div class="card-title">
                <h4 class="address-label">{{ watch.label || '未命名地址' }}</h4>
              </div>
              <div class="address-display">
                <code class="address-code">{{ shortAddress(watch.address) }}</code>
                <button
                  class="copy-btn"
                  @click="copyAddress(watch.address)"
                  :title="'复制完整地址: ' + watch.address"
                >
                  📋
                </button>
              </div>
            </div>

            <div class="card-actions">
              <button
                class="action-btn primary"
                @click="queryAddressOnceFromWatch(watch)"
                title="单独查询此地址"
              >
                🔍
              </button>
              <button
                class="action-btn danger"
                @click="removeWatch(watch.address)"
                title="移除监控"
              >
                🗑️
              </button>
            </div>
          </div>

          <!-- 地址信息面板 -->
          <div class="address-info">
            <div class="info-grid">
              <div class="info-item">
                <label class="info-label">实体</label>
                <span class="info-value entity">{{ watch.entity || entity }}</span>
              </div>
              <div class="info-item">
                <label class="info-label">链</label>
                <span class="info-value chain">{{ getChainName(watch.chain) || '全部链' }}</span>
              </div>
              <div v-if="dataSource !== 'basic' && watch.balance_usd" class="info-item">
                <label class="info-label">余额</label>
                <span class="info-value balance">${{ fmtAmount(watch.balance_usd) }}</span>
              </div>
              <div class="info-item">
                <label class="info-label">最后活跃</label>
                <span class="info-value last-active">
                  {{ (dataSource !== 'basic' && watch.last_active_at)
                    ? fmtTime(watch.last_active_at)
                    : '未知' }}
        </span>
      </div>
    </div>
          </div>

          <!-- 状态指示器 -->
          <div class="card-status">
            <div class="status-indicator">
              <div
                class="status-dot"
                :class="getWatchStatus(watch.address)"
              ></div>
              <span class="status-text">
                {{ getWatchStatusText(watch.address) }}
              </span>
            </div>
            <div class="last-update">
              更新于 {{ watchEvents[watch.address]?.updated_at ? fmtTime(watchEvents[watch.address].updated_at) : '从未' }}
            </div>
        </div>

        <div class="event-list">
          <h4>最近交易</h4>
          <div v-if="watchEvents[watch.address]?.error" class="error">{{ watchEvents[watch.address].error }}</div>
          <div v-else-if="!(watchEvents[watch.address]?.items?.length)">
            <p class="muted">当前还没有命中，点击按钮刷新最新数据。</p>
          </div>
          <div v-else class="events">
            <div v-for="it in watchEvents[watch.address].items" :key="it.id + '-' + it.txid" class="event-row">
              <div class="event-top">
                <span class="pill" :class="it.direction === 'in' ? 'in' : 'out'">
                  {{ it.direction === 'in' ? '流入' : '流出' }}
                </span>
                <strong v-if="it.symbol && it.amount">
                  {{ fmtAmount(it.amount) }} {{ it.symbol }}
                </strong>
                <strong v-else>
                  {{ fmtAmount(it.amount) }} {{ it.coin || 'USD' }}
                </strong>
                <span class="event-time">{{ fmtTime(it.occurred_at || it.block_timestamp) }} UTC</span>
              </div>
              <div class="event-bottom">
                <div class="event-addresses">
                  <span v-if="it.from">From: {{ shortAddress(it.from) }}</span>
                  <span v-if="it.to">To: {{ shortAddress(it.to) }}</span>
                  <!-- 显示Nansen特有的地址标签信息 -->
                  <div v-if="it.tokens_received && it.tokens_received.length > 0" class="token-summary">
                    <span v-for="token in it.tokens_received.slice(0, 2)" :key="token.token_address" class="token-tag">
                      {{ token.token_symbol }} {{ fmtAmount(token.token_amount) }}
                    </span>
                  </div>
                </div>
                <a class="link" :href="txLink(it.chain, it.txid || it.transaction_hash)" target="_blank" rel="noreferrer">查看 Tx</a>
              </div>
            </div>
          </div>
        </div>
      </article>
          </div>
        </div>
      </div>

      <!-- 普通网格渲染（数据量少时） -->
      <div v-else class="watch-grid">
        <article
          v-for="watch in filteredWatchlist"
          :key="watch.address"
          class="watch-card"
          :class="{ selected: isSelected(watch.address) }"
        >
          <!-- 卡片头部 -->
          <div class="card-header">
            <div class="card-title-section">
              <div class="card-title">
                <h4 class="address-label">{{ watch.label || '未命名地址' }}</h4>
              </div>
              <div class="address-display">
                <code class="address-code">{{ shortAddress(watch.address) }}</code>
                <button
                  class="copy-btn"
                  @click="copyAddress(watch.address)"
                  :title="'复制完整地址: ' + watch.address"
                >
                  📋
                </button>
              </div>
            </div>

            <div class="card-actions">
              <button
                class="action-btn primary"
                @click="queryAddressOnceFromWatch(watch)"
                title="单独查询此地址"
              >
                🔍
              </button>
              <button
                class="action-btn danger"
                @click="removeWatch(watch.address)"
                title="移除监控"
              >
                🗑️
              </button>
            </div>
          </div>

          <!-- 地址信息面板 -->
          <div class="address-info">
            <div class="info-grid">
              <div class="info-item">
                <label class="info-label">实体</label>
                <span class="info-value entity">{{ watch.entity || entity }}</span>
              </div>
              <div class="info-item">
                <label class="info-label">链</label>
                <span class="info-value chain">{{ getChainName(watch.chain) || '全部链' }}</span>
              </div>
              <div v-if="dataSource !== 'basic' && watch.balance_usd" class="info-item">
                <label class="info-label">余额</label>
                <span class="info-value balance">${{ fmtAmount(watch.balance_usd) }}</span>
              </div>
              <div class="info-item">
                <label class="info-label">最后活跃</label>
                <span class="info-value last-active">
                  {{ (dataSource !== 'basic' && watch.last_active_at)
                    ? fmtTime(watch.last_active_at)
                    : '未知' }}
        </span>
      </div>
    </div>
          </div>

          <!-- 状态指示器 -->
          <div class="card-status">
            <div class="status-indicator">
              <div
                class="status-dot"
                :class="getWatchStatus(watch.address)"
              ></div>
              <span class="status-text">
                {{ getWatchStatusText(watch.address) }}
              </span>
            </div>
            <div class="last-update">
              更新于 {{ watchEvents[watch.address]?.updated_at ? fmtTime(watchEvents[watch.address].updated_at) : '从未' }}
            </div>
        </div>

        <div class="event-list">
          <h4>最近交易</h4>
          <div v-if="watchEvents[watch.address]?.error" class="error">{{ watchEvents[watch.address].error }}</div>
          <div v-else-if="!(watchEvents[watch.address]?.items?.length)">
            <p class="muted">当前还没有命中，点击按钮刷新最新数据。</p>
          </div>
          <div v-else class="events">
            <div v-for="it in watchEvents[watch.address].items" :key="it.id + '-' + it.txid" class="event-row">
              <div class="event-top">
                <span class="pill" :class="it.direction === 'in' ? 'in' : 'out'">
                  {{ it.direction === 'in' ? '流入' : '流出' }}
                </span>
                <strong v-if="it.symbol && it.amount">
                  {{ fmtAmount(it.amount) }} {{ it.symbol }}
                </strong>
                <strong v-else>
                  {{ fmtAmount(it.amount) }} {{ it.coin || 'USD' }}
                </strong>
                <span class="event-time">{{ fmtTime(it.occurred_at || it.block_timestamp) }} UTC</span>
              </div>
              <div class="event-bottom">
                <div class="event-addresses">
                  <span v-if="it.from">From: {{ shortAddress(it.from) }}</span>
                  <span v-if="it.to">To: {{ shortAddress(it.to) }}</span>
                  <!-- 显示Nansen特有的地址标签信息 -->
                  <div v-if="it.tokens_received && it.tokens_received.length > 0" class="token-summary">
                    <span v-for="token in it.tokens_received.slice(0, 2)" :key="token.token_address" class="token-tag">
                      {{ token.token_symbol }} {{ fmtAmount(token.token_amount) }}
                    </span>
                  </div>
                </div>
                <a class="link" :href="txLink(it.chain, it.txid || it.transaction_hash)" target="_blank" rel="noreferrer">查看 Tx</a>
              </div>
            </div>
          </div>
        </div>
      </article>
    </div>
    </section>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { api } from '../api/api.js'
import { fmtAmount } from '../utils/utils.js'

const WATCHLIST_STORAGE_KEY = 'whale_watchlist'
const SAMPLE_WATCHLIST = [
  {
    label: 'Binance 冷钱包 (ETH)',
    address: '0x3f5CE5FBFe3E9af3971dD833D26BA9b5C936f0bE',
    chain: 'ethereum',
    entity: 'binance',
  },
  {
    label: 'Binance BTC 热钱包',
    address: '0x0000000000000000000000000000000000001004',
    chain: 'bitcoin',
    entity: 'binance',
  },
  {
    label: 'Binance Solana',
    address: '34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo',
    chain: 'solana',
    entity: 'binance',
  },
].map((w) => ({ ...w }))

const chainOptions = ['bitcoin', 'ethereum', 'bsc', 'solana', 'tron', 'arbitrum', 'optimism', 'polygon']

const entities = ref([])
const entity = ref('binance')
const dataSource = ref('basic') // 'basic', 'arkham', 'nansen'
const watchlist = ref(loadWatchlist())
const watchEvents = ref({})
const loading = ref(false)
const syncing = ref(false)
const lastRefresh = ref('')
const formNotice = ref('')
const noticeType = ref('info')

// 进度指示器相关
const progressPercent = ref(0)
const progressText = ref('准备刷新数据...')

// 性能优化：节流控制
const refreshThrottle = ref(false)
const lastRefreshTime = ref(0)
const REFRESH_THROTTLE_MS = 5000 // 5秒内只能刷新一次

// 虚拟滚动相关
const itemHeight = 200 // 每个卡片的高度
const containerHeight = 600 // 容器高度
const scrollTop = ref(0)

const queryData = ref({
  label: '',
  address: '',
  chain: '',
  entity: '',
})

const queryResult = ref(null)
const queryLoading = ref(false)
const searchQuery = ref('')
const filterChain = ref('')
const selectedAddresses = ref([])
const addressValidation = ref({ isValid: false, chain: '' })
const showAdvanced = ref(false)
const filterStatus = ref('all') // 'all', 'active', 'inactive', 'error'
const showQueryPanel = ref(false) // 查询面板默认收起

function loadWatchlist() {
  if (typeof window === 'undefined') {
    return SAMPLE_WATCHLIST.map((item) => ({ ...item }))
  }
  try {
    const raw = window.localStorage.getItem(WATCHLIST_STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed) && parsed.length) {
        return parsed
      }
    }
  } catch (error) {
    console.warn('读取追踪列表失败', error)
  }
  return SAMPLE_WATCHLIST.map((item) => ({ ...item }))
}

function persistWatchlist() {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(WATCHLIST_STORAGE_KEY, JSON.stringify(watchlist.value))
  } catch (error) {
    console.warn('保存追踪列表失败', error)
  }
}

const addDisabled = computed(() => !newWatch.value.address.trim())

const summary = computed(() => {
  // 使用缓存优化计算性能
  const cache = new Map()
  let active = 0
  let total = 0
  let largest = { amount: 0, coin: '' }

  for (const watch of watchlist.value) {
    const cacheKey = watch.address
    if (!cache.has(cacheKey)) {
      const data = watchEvents.value[watch.address]
      const items = data?.items || []
      cache.set(cacheKey, { hasItems: items.length > 0, items })
    }

    const cached = cache.get(cacheKey)
    if (cached.hasItems) active += 1

    // 限制计算的交易数量，避免性能问题
    const itemsToCheck = cached.items.slice(0, 100) // 只检查最近100笔交易
    for (const it of itemsToCheck) {
      const amount = Number(it.amount) || Number(it.value_usd) || 0
      total += 1
      if (amount > largest.amount) {
        largest = { amount, coin: it.symbol || it.coin || 'USD' }
      }
    }
  }

  return {
    totalWatchers: watchlist.value.length,
    activeWatchers: active,
    totalEvents: total,
    largestLabel: largest.amount ? `${largest.coin} ${fmtAmount(largest.amount)}` : '-',
    lastRefreshLabel: lastRefresh.value ? fmtTime(lastRefresh.value) : '',
  }
})

// 过滤后的监控列表
const filteredWatchlist = computed(() => {
  return watchlist.value.filter(watch => {
    // 搜索过滤
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase()
      const matchesLabel = watch.label?.toLowerCase().includes(query)
      const matchesAddress = watch.address.toLowerCase().includes(query)
      const matchesEntity = watch.entity?.toLowerCase().includes(query)
      if (!matchesLabel && !matchesAddress && !matchesEntity) {
        return false
      }
    }

    // 链过滤
    if (filterChain.value && watch.chain !== filterChain.value) {
      return false
    }

    // 状态过滤
    if (filterStatus.value !== 'all') {
      const status = getWatchStatus(watch.address)
      if (filterStatus.value === 'active' && status !== 'active') {
        return false
      }
      if (filterStatus.value === 'inactive' && status !== 'inactive') {
        return false
      }
      if (filterStatus.value === 'error' && status !== 'error') {
        return false
      }
    }

    return true
  })
})

// 虚拟滚动计算属性
const virtualScrollHeight = computed(() => {
  return filteredWatchlist.value.length * itemHeight
})

const visibleRange = computed(() => {
  const start = Math.floor(scrollTop.value / itemHeight)
  const end = Math.min(
    start + Math.ceil(containerHeight / itemHeight) + 2, // 多渲染2个缓冲
    filteredWatchlist.value.length
  )
  return { start: Math.max(0, start), end }
})

const visibleWatchlist = computed(() => {
  const { start, end } = visibleRange.value
  return filteredWatchlist.value.slice(start, end)
})

const virtualScrollOffset = computed(() => {
  return visibleRange.value.start * itemHeight
})

// 查询按钮是否可用
const canQuery = computed(() => {
  return queryData.value.address.trim() && !queryLoading.value
})

// 添加按钮是否可用
const canAdd = computed(() => {
  return queryData.value.address.trim() &&
         !watchlist.value.some(w => w.address.toLowerCase() === queryData.value.address.toLowerCase().trim())
})

const heroStats = computed(() => {
  const s = summary.value
  return [
    { label: '追踪地址', value: s.totalWatchers, note: '保存的链上地址' },
    { label: '活跃地址', value: s.activeWatchers, note: '最近有命中' },
    { label: '事件总数', value: s.totalEvents, note: '被拉取的交易' },
    { label: '最大单笔', value: s.largestLabel || '-', note: '按金额排序' },
    { label: '上次刷新', value: s.lastRefreshLabel || '-', note: '拉取基础数据' },
  ]
})

function getDataSourceLabel(dataSource) {
  const labels = {
    basic: '📊 基本监控',
    arkham: '🔍 Arkham',
    nansen: '📈 Nansen'
  }
  return labels[dataSource] || dataSource
}

function fmtTime(value) {
  if (!value) return '-'
  try {
    const d = new Date(value)
    const pad = (v) => String(v).padStart(2, '0')
    return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())} ${pad(
      d.getUTCHours(),
    )}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())} UTC`
  } catch {
    return value
  }
}

function shortAddress(value) {
  if (!value) return '-'
  const str = String(value)
  if (str.length <= 16) return str
  return `${str.slice(0, 6)}…${str.slice(-4)}`
}

function txLink(chain, txid) {
  if (!txid) return '#'
  const c = String(chain || '').toLowerCase()
  if (c.includes('btc') || c === 'bitcoin') return `https://mempool.space/tx/${txid}`
  if (c.includes('eth') || c === 'ethereum') return `https://etherscan.io/tx/${txid}`
  if (c.includes('sol')) return `https://solscan.io/tx/${txid}`
  if (c.includes('tron')) return `https://tronscan.org/#/transaction/${txid}`
  return '#'
}

// 获取链的图标
function getChainIcon(chain) {
  const icons = {
    ethereum: '🔷',
    bsc: '🟡',
    solana: '🟣',
    bitcoin: '🟠',
    polygon: '🟣',
    arbitrum: '🔵',
    optimism: '🔴',
    avalanche: '🔴',
    tron: '🟢'
  }
  return icons[chain] || '⛓️'
}

// 获取链的显示名称
function getChainName(chain) {
  const names = {
    ethereum: 'Ethereum',
    bsc: 'BSC',
    solana: 'Solana',
    bitcoin: 'Bitcoin',
    polygon: 'Polygon',
    arbitrum: 'Arbitrum',
    optimism: 'Optimism',
    avalanche: 'Avalanche',
    tron: 'Tron'
  }
  return names[chain] || chain || '全部链'
}



// 获取通知图标
function getNoticeIcon(type) {
  const icons = {
    success: '✅',
    error: '❌',
    warning: '⚠️',
    info: 'ℹ️'
  }
  return icons[type] || 'ℹ️'
}

// 复制地址到剪贴板
async function copyAddress(address) {
  try {
    await navigator.clipboard.writeText(address)
    // 这里可以添加一个临时的成功提示
  } catch (error) {
    console.warn('复制失败:', error)
  }
}

// 检查地址是否被选中
function isSelected(address) {
  return selectedAddresses.value.includes(address)
}

// 切换地址选择状态
function toggleSelection(address) {
  const index = selectedAddresses.value.indexOf(address)
  if (index > -1) {
    selectedAddresses.value.splice(index, 1)
  } else {
    selectedAddresses.value.push(address)
  }
}

// 获取监控地址的状态
function getWatchStatus(address) {
  const data = watchEvents.value[address]
  if (!data) return 'unknown'
  if (data.error) return 'error'
  if (data.items && data.items.length > 0) return 'active'
  return 'inactive'
}

// 获取监控地址的状态文本
function getWatchStatusText(address) {
  const status = getWatchStatus(address)
  const texts = {
    active: '活跃',
    inactive: '无活动',
    error: '查询失败',
    unknown: '未知'
  }
  return texts[status] || '未知'
}

// 从监控列表查询地址
function queryAddressOnceFromWatch(watch) {
  queryData.value = {
    label: watch.label,
    address: watch.address,
    chain: watch.chain,
    entity: watch.entity
  }
  // 滚动到查询表单
  document.querySelector('.query-panel')?.scrollIntoView({ behavior: 'smooth' })
}

// 滚动到查询表单
function scrollToQuery() {
  document.querySelector('.query-panel')?.scrollIntoView({ behavior: 'smooth' })
}

// 清除过滤条件
function clearFilters() {
  searchQuery.value = ''
  filterChain.value = ''
}

// 虚拟滚动事件处理
function handleScroll(event) {
  scrollTop.value = event.target.scrollTop
}

// 切换查询面板显示
function toggleQueryPanel() {
  showQueryPanel.value = !showQueryPanel.value
}

// 设置状态筛选
function setFilterStatus(status) {
  filterStatus.value = status
}

// 批量删除选中的地址
function bulkRemove() {
  if (!selectedAddresses.value.length) return

  if (confirm(`确定要删除 ${selectedAddresses.value.length} 个监控地址吗？`)) {
    selectedAddresses.value.forEach(address => {
      removeWatch(address)
    })
    selectedAddresses.value = []
  }
}


function onEntityChange() {
  formNotice.value = ''
}

// 地址验证函数
function validateAddress() {
  const address = queryData.value.address.trim()
  if (!address) {
    addressValidation.value = { isValid: false, chain: '' }
    return
  }

  // 检查各种区块链地址格式
  const validations = [
    { chain: 'ethereum', pattern: /^0x[a-fA-F0-9]{40}$/ },
    { chain: 'bitcoin', pattern: /^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$/ },
    { chain: 'solana', pattern: /^[1-9A-HJ-NP-Za-km-z]{32,44}$/ },
    { chain: 'tron', pattern: /^T[a-km-zA-HJ-NP-Z1-9]{33}$/ },
    { chain: 'bsc', pattern: /^0x[a-fA-F0-9]{40}$/ },
  ]

  for (const validation of validations) {
    if (validation.pattern.test(address)) {
      addressValidation.value = { isValid: true, chain: validation.chain }
      return
    }
  }

  addressValidation.value = { isValid: false, chain: '' }
}

// 粘贴地址功能
async function pasteFromClipboard() {
  try {
    const text = await navigator.clipboard.readText()
    queryData.value.address = text.trim()
    validateAddress()
    formNotice.value = '已从剪贴板粘贴地址'
    noticeType.value = 'success'
  } catch (error) {
    formNotice.value = '无法访问剪贴板，请手动输入'
    noticeType.value = 'warning'
  }
}

// 粘贴事件处理
function onAddressPaste(event) {
  // 延迟验证，让v-model更新完成
  setTimeout(() => {
    validateAddress()
  }, 0)
}

// 设置示例地址
function setExampleAddress(chain) {
  const examples = {
    ethereum: '0x3f5CE5FBFe3E9af3971dD833D26BA9b5C936f0bE',
    bitcoin: '34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo',
  }
  queryData.value.address = examples[chain] || ''
  queryData.value.chain = chain
  validateAddress()
}

async function onDataSourceChange() {
  formNotice.value = ''
  // 切换数据源时重新加载监控列表
  await loadWatchlistForDataSource()
  // 切换数据源后刷新事件数据
  await refreshWatchEvents()
}

async function syncExternalData() {
  if (syncing.value) return

  syncing.value = true
  try {
    if (dataSource.value === 'arkham') {
      await api.syncArkhamData()
    } else if (dataSource.value === 'nansen') {
      await api.syncNansenData()
    }
    formNotice.value = '外部数据同步完成'
    noticeType.value = 'success'
    // 同步完成后重新加载数据
    await loadWatchlistForDataSource()
  } catch (error) {
    formNotice.value = `同步失败: ${error.message}`
    noticeType.value = 'error'
  } finally {
    syncing.value = false
  }
}

async function loadWatchlistForDataSource() {
  try {
    let response
    if (dataSource.value === 'arkham') {
      response = await api.listArkhamWatches()
    } else if (dataSource.value === 'nansen') {
      response = await api.listNansenWatches()
    } else {
      response = await api.listWhaleWatches()
    }

    // 处理后端返回的数据结构
    let items = []
    if (response.watchlist) {
      // 基本监控的数据结构
      items = response.watchlist
    } else if (response.items) {
      // Arkham/Nansen的数据结构
      items = response.items
    } else if (Array.isArray(response)) {
      // 直接返回数组的情况
      items = response
    }
    // 转换数据格式以保持一致性
    watchlist.value = items.map(item => ({
      id: item.id,
      label: item.label || item.address,
      address: item.address,
      chain: item.chain,
      entity: item.entity,
      balance_usd: item.balance_usd,
      last_active_at: item.last_active_at,
      created_at: item.created_at,
      // Nansen特有的字段
      transactions_json: item.transactions_json,
      metadata_json: item.metadata_json,
      last_snapshot_at: item.last_snapshot_at
    }))
  } catch (error) {
    console.warn('加载监控列表失败', error)
    watchlist.value = []
  }
}

async function refreshWatchEvents() {
  if (!watchlist.value.length) {
    watchEvents.value = {}
    lastRefresh.value = ''
    return
  }

  // 节流控制：防止过于频繁的刷新
  const now = Date.now()
  if (refreshThrottle.value && (now - lastRefreshTime.value) < REFRESH_THROTTLE_MS) {
    const remaining = Math.ceil((REFRESH_THROTTLE_MS - (now - lastRefreshTime.value)) / 1000)
    formNotice.value = `请等待 ${remaining} 秒后再刷新`
    noticeType.value = 'warning'
    return
  }

  loading.value = true
  refreshThrottle.value = true
  lastRefreshTime.value = now
  progressPercent.value = 0
  progressText.value = '准备刷新数据...'
  formNotice.value = ''

  try {
    const totalItems = watchlist.value.length
    let completedItems = 0

    // 并发处理，但限制并发数量
    const concurrencyLimit = 3
    for (let i = 0; i < totalItems; i += concurrencyLimit) {
      const batch = watchlist.value.slice(i, i + concurrencyLimit)
      progressText.value = `正在刷新 ${Math.min(i + concurrencyLimit, totalItems)}/${totalItems} 个地址...`

    await Promise.all(
        batch.map(async (watch) => {
        try {
          let items = []
          let total = 0
          let error = ''

          if (dataSource.value === 'nansen') {
            // 对于Nansen数据源，从已同步的数据中获取交易记录
            const nansenWatch = watchlist.value.find(w => w.address.toLowerCase() === watch.address.toLowerCase())
            if (nansenWatch && nansenWatch.transactions_json) {
              try {
                let transactions
                // 处理API返回的数据格式（可能是对象或字符串）
                if (typeof nansenWatch.transactions_json === 'string') {
                  transactions = JSON.parse(nansenWatch.transactions_json)
                } else if (Array.isArray(nansenWatch.transactions_json)) {
                  transactions = nansenWatch.transactions_json
                } else {
                  throw new Error('未知的数据格式')
                }

                if (transactions && transactions.length > 0) {
                  // 将Nansen交易数据转换为前端期望的格式
                  items = transactions.slice(0, 3).map(tx => ({
                    id: tx.transaction_hash,
                    direction: tx.direction,
                    symbol: tx.symbol,
                    amount: tx.amount,
                    txid: tx.transaction_hash,
                    from: tx.tokens_received?.[0]?.from_address || tx.tokens_sent?.[0]?.from_address,
                    to: tx.tokens_received?.[0]?.to_address || tx.tokens_sent?.[0]?.to_address,
                    occurred_at: tx.occurred_at,
                    chain: tx.chain
                  }))
                  total = transactions.length
                } else {
                  error = '暂无交易记录'
                }
              } catch (parseErr) {
                error = '交易数据解析失败'
                console.warn('Failed to parse Nansen transactions:', parseErr, nansenWatch.transactions_json)
              }
            } else {
              error = '等待数据同步完成...'
            }
          } else {
            // 对于其他数据源，使用通用的交易查询API
            const params = {
              keyword: watch.address,
              page: 1,
              page_size: 3,
              entity: watch.entity || entity.value,
              chain: watch.chain || undefined,
            }
            const res = await api.recentTransfers(params)
            items = res.items || []
            total = res.total || 0
          }

          setWatchEvents(watch.address, {
            items: items,
            updated_at: new Date().toISOString(),
            error: error,
            total: total,
          })
        } catch (err) {
          setWatchEvents(watch.address, {
            items: [],
            updated_at: new Date().toISOString(),
            error: err?.message || '请求失败',
            total: 0,
          })
        }
      }),
    )

      completedItems += batch.length
      progressPercent.value = Math.round((completedItems / totalItems) * 100)
    }

    progressText.value = '数据刷新完成'
    lastRefresh.value = new Date().toISOString()

    // 3秒后隐藏进度条
    setTimeout(() => {
      if (!loading.value) {
        progressPercent.value = 0
      }
    }, 3000)

  } finally {
    loading.value = false
    // 重置节流控制
    setTimeout(() => {
      refreshThrottle.value = false
    }, REFRESH_THROTTLE_MS)
  }
}

function setWatchEvents(address, payload) {
  watchEvents.value = { ...watchEvents.value, [address]: payload }
}

async function removeWatch(address) {
  try {
    if (dataSource.value === 'arkham') {
      await api.deleteArkhamWatch(address)
    } else if (dataSource.value === 'nansen') {
      await api.deleteNansenWatch(address)
    } else {
      await api.deleteWhaleWatch(address)
    }

    const filtered = watchlist.value.filter((item) => item.address !== address)
    watchlist.value = filtered
    persistWatchlist()
    const next = { ...watchEvents.value }
    delete next[address]
    watchEvents.value = next
  } catch (error) {
    formNotice.value = `删除失败: ${error.message}`
    noticeType.value = 'error'
  }
}

function resetQueryForm() {
  queryData.value = {
    label: '',
    address: '',
    chain: '',
    entity: '',
  }
  queryResult.value = null
  formNotice.value = ''
}

async function queryAddressOnce() {
  const address = queryData.value.address.trim()
  if (!address) {
    formNotice.value = '请输入地址'
    noticeType.value = 'error'
    return
  }

  loading.value = true
  formNotice.value = ''
  queryResult.value = null

  try {
    let result = null

    const queryPayload = {
      address: address,
      chain: queryData.value.chain,
      entity: queryData.value.entity || entity.value,
    }

    if (dataSource.value === 'arkham') {
      // 调用Arkham查询接口获取实时数据
      result = await api.queryArkhamAddress(queryPayload)
      result.dataSource = 'arkham'
    } else if (dataSource.value === 'nansen') {
      // 调用Nansen查询接口获取实时数据
      result = await api.queryNansenAddress(queryPayload)
      result.dataSource = 'nansen'
    } else {
      // 基本监控：使用现有的转账查询接口
      const params = {
        keyword: address,
        page: 1,
        page_size: 10,
        entity: queryData.value.entity || entity.value,
        chain: queryData.value.chain || undefined,
      }

      const res = await api.recentTransfers(params)

      result = {
        label: queryData.value.label || `查询: ${shortAddress(address)}`,
        address: address,
        chain: queryData.value.chain,
        entity: queryData.value.entity || entity.value,
        transactions: res.items || [],
        queried_at: new Date().toISOString(),
        total: res.total || 0,
        dataSource: 'basic'
      }
    }

    // 统一结果格式
    queryResult.value = {
      ...result,
      label: queryData.value.label || result.label || `查询: ${shortAddress(address)}`,
      queried_at: result.queried_at || new Date().toISOString()
    }

    formNotice.value = '查询完成'
    noticeType.value = 'success'

  } catch (error) {
    formNotice.value = `查询失败: ${error.message}`
    noticeType.value = 'error'
  } finally {
    loading.value = false
  }
}

async function addToWatchlist() {
  const address = queryData.value.address.trim()
  if (!address) {
    formNotice.value = '请输入地址'
    noticeType.value = 'error'
    return
  }

  // 检查是否已在监控列表中
  if (watchlist.value.some((item) => item.address.toLowerCase() === address.toLowerCase())) {
    formNotice.value = '该地址已在监控列表中'
    noticeType.value = 'error'
    return
  }

  const entry = {
    label: queryData.value.label.trim() || `监控: ${shortAddress(address)}`,
    address,
    chain: queryData.value.chain || '',
    entity: queryData.value.entity || entity.value,
  }

  try {
    if (dataSource.value === 'arkham') {
      await api.createArkhamWatch(entry)
    } else if (dataSource.value === 'nansen') {
      await api.createNansenWatch(entry)
    } else {
      await api.createWhaleWatch(entry)
    }

    watchlist.value = [entry, ...watchlist.value]
    persistWatchlist()
    formNotice.value = '已添加到监控列表'
    noticeType.value = 'success'
    // 添加成功后重置表单，但保留查询结果
    resetQueryForm()

  } catch (error) {
    formNotice.value = `添加失败: ${error.message}`
    noticeType.value = 'error'
  }
}

async function bulkImportAddresses() {
  if (!bulkImportData.value.trim()) {
    formNotice.value = '请输入要导入的数据'
    noticeType.value = 'error'
    return
  }

  try {
    const addresses = JSON.parse(bulkImportData.value.trim())
    if (!Array.isArray(addresses)) {
      throw new Error('数据格式错误，应为地址数组')
    }

    let successCount = 0
    let errorCount = 0

    for (const addr of addresses) {
      try {
        // 检查必填字段
        if (!addr.address || !addr.address.trim()) {
          errorCount++
          continue
        }

        // 检查是否已存在
        if (watchlist.value.some((item) => item.address.toLowerCase() === addr.address.toLowerCase())) {
          errorCount++
          continue
        }

        const entry = {
          label: addr.label || `导入: ${shortAddress(addr.address)}`,
          address: addr.address.trim(),
          chain: addr.chain || '',
          entity: addr.entity || entity.value,
        }

        // 调用相应的API
        if (dataSource.value === 'arkham') {
          await api.createArkhamWatch(entry)
        } else if (dataSource.value === 'nansen') {
          await api.createNansenWatch(entry)
        } else {
          await api.createWhaleWatch(entry)
        }

        watchlist.value = [entry, ...watchlist.value]
        successCount++

      } catch (error) {
        console.warn('导入地址失败:', addr, error)
        errorCount++
      }
    }

    persistWatchlist()

    if (successCount > 0) {
      formNotice.value = `批量导入完成: 成功 ${successCount} 个${errorCount > 0 ? `, 失败 ${errorCount} 个` : ''}`
      noticeType.value = errorCount > 0 ? 'info' : 'success'
    } else {
      formNotice.value = '批量导入失败，请检查数据格式'
      noticeType.value = 'error'
    }

  } catch (error) {
    formNotice.value = `数据解析失败: ${error.message}`
    noticeType.value = 'error'
  }
}

function clearBulkImport() {
  bulkImportData.value = ''
}

async function addCurrentQueryToWatch() {
  if (!queryResult.value || !queryResult.value.address) {
    formNotice.value = '没有可添加的查询结果'
    noticeType.value = 'error'
    return
  }

  const address = queryResult.value.address
  const existing = watchlist.value.find(item => item.address.toLowerCase() === address.toLowerCase())

  if (existing) {
    formNotice.value = '该地址已在监控列表中'
    noticeType.value = 'info'
    return
  }

  const entry = {
    label: queryResult.value.label || `监控: ${shortAddress(address)}`,
    address: address,
    chain: queryResult.value.chain || '',
    entity: queryResult.value.entity || entity.value,
  }

  try {
    if (dataSource.value === 'arkham') {
      await api.createArkhamWatch(entry)
    } else if (dataSource.value === 'nansen') {
      await api.createNansenWatch(entry)
    } else {
      await api.createWhaleWatch(entry)
    }

    watchlist.value = [entry, ...watchlist.value]
    persistWatchlist()
    formNotice.value = '已添加到监控列表，现在可以同步数据查看交易记录'
    noticeType.value = 'success'

  } catch (error) {
    formNotice.value = `添加失败: ${error.message}`
    noticeType.value = 'error'
  }
}

function switchToBasicMonitoring() {
  dataSource.value = 'basic'
  formNotice.value = '已切换到基本监控模式，无需API Key即可使用'
  noticeType.value = 'success'
}

async function addWatch() {
  const address = newWatch.value.address.trim()
  if (!address) {
    formNotice.value = '请输入地址'
    noticeType.value = 'error'
    return
  }
  if (watchlist.value.some((item) => item.address.toLowerCase() === address.toLowerCase())) {
    formNotice.value = '该地址已在追踪列表中'
    noticeType.value = 'error'
    return
  }

  const entry = {
    label: newWatch.value.label.trim(),
    address,
    chain: newWatch.value.chain || '',
    entity: newWatch.value.entity || entity.value,
  }

  try {
    if (dataSource.value === 'arkham') {
      await api.createArkhamWatch(entry)
    } else if (dataSource.value === 'nansen') {
      await api.createNansenWatch(entry)
      // 对于Nansen，添加后立即同步数据
      try {
        await api.syncNansenData()
        // 同步完成后重新加载监控列表以获取最新数据
        await loadWatchlistForDataSource()
      } catch (syncErr) {
        console.warn('Nansen数据同步失败:', syncErr)
      }
    } else {
      await api.createWhaleWatch(entry)
    }

    watchlist.value = [entry, ...watchlist.value]
    persistWatchlist()
    formNotice.value = '地址已添加'
    noticeType.value = 'success'
    resetForm()
    refreshWatchEvents()
  } catch (error) {
    formNotice.value = `添加失败: ${error.message}`
    noticeType.value = 'error'
  }
}

async function loadEntities() {
  try {
    const res = await api.listEntities()
    if (res?.entities?.length) {
      entities.value = res.entities
      if (!entities.value.includes(entity.value)) {
        entity.value = entities.value[0]
      }
    }
  } catch (error) {
    console.warn('加载实体列表失败', error)
  }
}

onMounted(async () => {
  await loadEntities()
  await loadWatchlistForDataSource()
  await refreshWatchEvents()

  // 移除页面加载时的自动同步，现在需要手动同步
})
</script>

<style scoped lang="scss">
.topbar .label-inline {
  font-weight: 500;
  margin-right: 6px;
}
.helper-text {
  font-size: 12px;
  margin: 6px 0 0;
}
.section-title h3 {
  margin: 0;
}
.form-grid {
  margin-top: 12px;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px 18px;
  align-items: center;
}
.form-grid label {
  margin: 0;
  font-weight: 500;
}
.form-grid input,
.form-grid select {
  width: 100%;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: #fff;
}
.form-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
button.secondary {
  background: #f3f4f6;
  border-color: var(--border);
  color: var(--text);
}
.form-helper {
  margin-top: 6px;
  font-size: 13px;
}
.form-helper.success {
  color: #16a34a;
}
.form-helper.error {
  color: #ef4444;
}
.hero-panel {
  background: linear-gradient(135deg, rgba(37,99,235,0.08), rgba(99,102,241,0.15));
  border: 1px solid rgba(15,23,42,0.15);
}
.hero-main {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  justify-content: space-between;
  align-items: flex-start;
}
.hero-main h2 {
  margin: 4px 0;
}
.hero-subtitle {
  margin: 0;
  color: rgba(15, 23, 42, 0.75);
  max-width: 640px;
}
.hero-subtitle code {
  background: rgba(146, 196, 255, 0.3);
  padding: 2px 6px;
  border-radius: 6px;
  font-size: 13px;
}
.hero-actions {
  display: flex;
  align-items: flex-end;
}

/* 现代化页面头部样式 */
.page-header {
  margin-bottom: 2rem;
  overflow: hidden;
}

.header-gradient {
  background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
  border: 1px solid #475569;
  border-radius: 16px;
  position: relative;
}

.header-gradient::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: radial-gradient(circle at 20% 80%, rgba(120, 119, 198, 0.3) 0%, transparent 50%),
              radial-gradient(circle at 80% 20%, rgba(255, 119, 198, 0.15) 0%, transparent 50%);
  border-radius: 16px;
  pointer-events: none;
}

/* 暂时移除@keyframes以修复语法错误 */

.header-content {
  padding: 3rem 2rem;
  position: relative;
  z-index: 2;
}

/* 面包屑导航 - 隐藏 */
.breadcrumb-nav {
  display: none;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
}

.breadcrumb-item {
  color: rgba(255, 255, 255, 0.8) !important;
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-weight: 500;
}

.breadcrumb-item.active {
  color: white;
  font-weight: 600;
}

.breadcrumb-separator {
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.75rem;
}

/* 标题区域 */
.title-section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 2rem;
  gap: 2rem;
}

.title-content {
  flex: 1;
}

.page-title {
  font-size: 2.5rem;
  font-weight: 700;
  color: white !important;
  margin: 0 0 0.5rem 0;
  line-height: 1.2;
}

.page-subtitle {
  font-size: 1.125rem;
  color: rgba(255, 255, 255, 0.9) !important;
  margin: 0;
  line-height: 1.6;
  max-width: 600px;
}

/* 浮动装饰元素 */
.title-visual {
  flex-shrink: 0;
  position: relative;
  width: 120px;
  height: 120px;
}

.floating-shapes {
  position: relative;
  width: 100%;
  height: 100%;
}

.shape {
  position: absolute;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  /* 暂时移除动画以修复语法错误 */
}

.shape-1 {
  width: 40px;
  height: 40px;
  top: 20px;
  left: 30px;
  animation-delay: 0s;
}

.shape-2 {
  width: 25px;
  height: 25px;
  top: 60px;
  right: 20px;
  animation-delay: 2s;
}

.shape-3 {
  width: 15px;
  height: 15px;
  bottom: 30px;
  left: 50px;
  animation-delay: 4s;
}

/* 暂时移除@keyframes以修复语法错误 */

/* 控制面板 */
.header-controls {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.2);
  border-radius: 16px;
  padding: 1.5rem;
}

.control-row {
  display: flex;
  align-items: flex-end;
  gap: 2rem;
  flex-wrap: wrap;
}

.control-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-width: 180px;
}

.control-label {
  font-size: 0.875rem;
  font-weight: 600;
  color: white;
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.select-container {
  position: relative;
}

.modern-select {
  width: 100%;
  padding: 0.75rem 1rem;
  background: rgba(255, 255, 255, 0.15);
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: 8px;
  color: white;
  font-size: 0.875rem;
  font-weight: 500;
  appearance: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modern-select:focus {
  outline: none;
  border-color: rgba(255, 255, 255, 0.8);
  background: rgba(255, 255, 255, 0.25);
  box-shadow: 0 0 0 3px rgba(255, 255, 255, 0.1);
}

.modern-select option {
  background: white;
  color: #374151;
  padding: 0.5rem;
}

.select-arrow {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: rgba(255, 255, 255, 0.8);
  font-size: 0.75rem;
  pointer-events: none;
}

/* 操作按钮 */
.control-actions {
  display: flex;
  gap: 0.75rem;
  margin-left: auto;
}

.btn-compact {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  border: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.btn-primary {
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #d1d5db;
}

.btn-primary:hover:not(:disabled) {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.1);
  color: white;
  border: 1px solid rgba(255, 255, 255, 0.3);
}

.btn-secondary:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.5);
}

.btn-compact:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.btn-compact.loading {
  position: relative;
}

/* 暂时移除动画以修复语法错误 */

.btn-text {
  display: inline-block;
}

.btn-compact.loading .btn-text {
  opacity: 0;
}

/* 暂时移除@keyframes以修复语法错误 */

/* 响应式设计 */
@media (max-width: 1024px) {
  .header-content {
    padding: 2rem 1.5rem;
  }

  .page-title {
    font-size: 2rem;
  }

  .control-row {
    gap: 1.5rem;
  }

  .control-item {
    min-width: 160px;
  }
}

@media (max-width: 768px) {
  .title-section {
    flex-direction: column;
    align-items: flex-start;
    gap: 1.5rem;
  }

  .title-visual {
    width: 80px;
    height: 80px;
  }

  .page-title {
    font-size: 1.75rem;
  }

  .control-row {
    flex-direction: column;
    align-items: stretch;
    gap: 1rem;
  }

  .control-item {
    min-width: auto;
  }

  .control-actions {
    margin-left: 0;
    justify-content: center;
  }

  .btn-compact {
    flex: 1;
    justify-content: center;
  }
}
/* 现代化统计概览样式 */
.stats-overview {
  margin-bottom: 2.5rem;
}

.stats-header {
  margin-bottom: 1.5rem;
}

.stats-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: #1f2937;
  margin: 0 0 0.25rem 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.stats-subtitle {
  font-size: 0.875rem;
  color: #6b7280;
  margin: 0;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.25rem;
  margin-bottom: 1.5rem;
}

/* 现代化统计卡片 */
.stat-card {
  position: relative;
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1), 0 1px 2px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15), 0 4px 10px rgba(0, 0, 0, 0.1);
}

.stat-card.primary {
  background: #f0f9ff;
  border-color: #0ea5e9;
  color: #0c4a6e;
}

.stat-card.success {
  background: #f0fdf4;
  border-color: #22c55e;
  color: #166534;
}

.stat-card.info {
  background: #eff6ff;
  border-color: #3b82f6;
  color: #1e40af;
}

.stat-card.warning {
  background: #fffbeb;
  border-color: #f59e0b;
  color: #92400e;
}

/* 移除不必要的背景装饰 */

/* 暂时移除@keyframes以修复语法错误 */

/* 卡片内容 */
.card-content {
  position: relative;
  z-index: 2;
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 0.5rem;
}

/* 隐藏统计图标 */
.stat-icon {
  display: none;
}

.stat-details {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.stat-value {
  font-size: 2.5rem;
  font-weight: 800;
  line-height: 1;
  color: inherit;
}

.stat-card .stat-value {
  color: inherit;
}

.large-amount {
  font-size: 1.125rem !important;
  font-weight: 700 !important;
  word-break: break-word;
  line-height: 1.3;
}

.stat-label {
  font-size: 1rem;
  font-weight: 600;
  opacity: 0.9;
  color: inherit;
}

.stat-meta {
  display: none;
}

.meta-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.meta-indicator.active {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
}

.meta-indicator.success {
  background: #10b981;
}

.meta-indicator.info {
  background: #3b82f6;
}

.meta-indicator.warning {
  background: #f59e0b;
}

/* 移除不必要的卡片装饰 */

/* 动画效果 */
/* 暂时移除@keyframes以修复语法错误 */

/* 暂时移除动画以修复语法错误 */

/* 状态仪表板 */
.status-dashboard {
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  padding: 1.5rem;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.status-metrics {
  display: flex;
  gap: 2rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.metric-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.metric-icon {
  width: 32px;
  height: 32px;
  background: #f3f4f6;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1rem;
}

.metric-content {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.metric-label {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.metric-value {
  font-size: 0.875rem;
  color: #1f2937;
  font-weight: 600;
}

/* 状态脉冲动画 */
.status-pulse {
  width: 12px;
  height: 12px;
  background: #ef4444;
  border-radius: 50%;
  /* 暂时移除动画以修复语法错误 */
}

.status-pulse.active {
  background: #10b981;
}

/* 进度指示器 */
.progress-indicator {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.progress-bar {
  flex: 1;
  height: 4px;
  background: #e5e7eb;
  border-radius: 2px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 0.875rem;
  color: #6b7280;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
    gap: 1rem;
  }

  .card-content {
    padding: 1.25rem;
    gap: 0.75rem;
  }

  .stat-icon {
    width: 40px;
    height: 40px;
    font-size: 1rem;
  }

  .stat-value {
    font-size: 1.5rem;
  }

  .status-metrics {
    gap: 1rem;
  }

  .metric-item {
    flex: 1;
    min-width: 120px;
  }
}
.stat-card .stat-note {
  margin: 0;
  font-size: 12px;
  color: rgba(15, 23, 42, 0.7);
}
.watch-form {
  margin-top: 12px;
  border-radius: 16px;
}
.watch-list {
  margin-top: 12px;
}
.watch-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
  margin-top: 12px;
}

/* 虚拟滚动样式 */
.virtual-scroll-container {
  position: relative;
  overflow: hidden;
}

.virtual-scroll-viewport {
  overflow-y: auto;
  overflow-x: hidden;
}

.virtual-scroll-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 14px;
  position: relative;
}

/* 快速筛选按钮 */
.quick-filters {
  display: flex;
  gap: 0.75rem;
  margin-top: 1rem;
  flex-wrap: wrap;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: white;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-btn:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
  color: #374151;
}

.filter-btn.active {
  background: #667eea;
  border-color: #667eea;
  color: white;
}

.filter-btn.active:hover {
  background: #5a67d8;
  border-color: #5a67d8;
}

/* 查询面板折叠样式 */
.query-panel.compact .query-form {
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.3s ease;
}

.query-panel:not(.compact) .query-form {
  max-height: none;
}
.watch-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}
.watch-header h3 {
  margin: 0;
}
.watch-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 13px;
  color: rgba(15, 23, 42, 0.62);
  margin: 10px 0;
}
.events {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.event-top {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.event-time {
  margin-left: auto;
  font-size: 12px;
  color: rgba(15, 23, 42, 0.6);
}
.event-bottom {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: rgba(15, 23, 42, 0.6);
  margin-top: 6px;
}
.link {
  color: #2563eb;
  text-decoration: none;
  font-size: 13px;
}
.link:hover {
  text-decoration: underline;
}
.watch-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 8px;
  gap: 10px;
}
.watch-actions {
  display: flex;
  gap: 8px;
}
.badge {
  border-radius: 999px;
  padding: 4px 12px;
  font-size: 12px;
  border: 1px solid rgba(15, 23, 42, 0.15);
  color: rgba(15, 23, 42, 0.8);
}
.badge.active {
  background: rgba(16, 185, 129, 0.12);
  border-color: rgba(16, 185, 129, 0.3);
  color: #047857;
}
.badge.warn {
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.3);
  color: #b91c1c;
}
.empty {
  text-align: center;
  padding: 26px 0;
  color: var(--muted);
}
.watch-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 16px;
  padding: 16px;
  background: #fff;
  box-shadow: 0 12px 22px -18px rgba(15, 23, 42, 0.5);
}

.query-result {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
}

.query-result h4 {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.result-card {
  border: 1px solid rgba(15, 23, 42, 0.08);
  border-radius: 16px;
  padding: 16px;
  background: #fff;
  box-shadow: 0 8px 16px -12px rgba(15, 23, 42, 0.3);
}

.result-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;
}

.result-header h3 {
  margin: 0;
  font-size: 16px;
}

.result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  font-size: 13px;
  color: rgba(15, 23, 42, 0.65);
  margin-bottom: 12px;
  padding: 8px 0;
  border-bottom: 1px solid rgba(15, 23, 42, 0.04);
}

.result-meta span {
  font-size: 13px;
  color: rgba(15, 23, 42, 0.65);
}

.result-transactions {
  margin-top: 12px;
}

.result-transactions h5 {
  margin: 0 0 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}

.transactions-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.transaction-item {
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid rgba(15, 23, 42, 0.06);
  background: #f9fafb;
}

.tx-top {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 4px;
}

.tx-top strong {
  font-size: 14px;
}

.tx-time {
  margin-left: auto;
  font-size: 11px;
  color: rgba(15, 23, 42, 0.5);
}

.tx-bottom {
  font-size: 11px;
  color: rgba(15, 23, 42, 0.6);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tx-hash {
  margin-bottom: 2px;
}

.token-details {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
  padding: 6px 8px;
  background: rgba(15, 23, 42, 0.03);
  border-radius: 4px;
  border: 1px solid rgba(15, 23, 42, 0.05);
}

.token-transfer {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 10px;
}

.token-info {
  font-weight: 500;
  color: rgba(15, 23, 42, 0.8);
}

.address-info {
  color: rgba(15, 23, 42, 0.6);
  font-size: 9px;
}

.address-label {
  color: #6b7280;
}

.tx-meta {
  display: flex;
  gap: 8px;
  margin-top: 4px;
  font-size: 10px;
  color: rgba(15, 23, 42, 0.5);
}

.method {
  background: rgba(16, 185, 129, 0.1);
  color: #047857;
  padding: 1px 4px;
  border-radius: 3px;
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.source-type {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
  padding: 1px 4px;
  border-radius: 3px;
  border: 1px solid rgba(245, 158, 11, 0.2);
}

.no-transactions {
  padding: 16px;
  text-align: center;
  color: var(--muted);
  font-size: 14px;
}

.sync-hint {
  margin-top: 12px;
  padding: 12px;
  background: rgba(37, 99, 235, 0.05);
  border: 1px solid rgba(37, 99, 235, 0.1);
  border-radius: 8px;
}

.sync-hint p {
  margin: 0 0 8px;
  font-size: 13px;
}

.btn-sync {
  padding: 6px 12px;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-sync:hover:not(:disabled) {
  background: #1d4ed8;
}

.btn-sync:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.sync-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
  flex-wrap: wrap;
}

.btn-add {
  padding: 6px 12px;
  background: #16a34a;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-add:hover:not(:disabled) {
  background: #15803d;
}

.btn-add:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-switch {
  padding: 6px 12px;
  background: #f59e0b;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-switch:hover:not(:disabled) {
  background: #d97706;
}

.data-source-badge {
  background: rgba(37, 99, 235, 0.1);
  color: #2563eb;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
  border: 1px solid rgba(37, 99, 235, 0.2);
}

.api-error-notice {
  margin-top: 12px;
  padding: 8px 12px;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: 6px;
  color: #dc2626;
}

.api-error-notice p {
  margin: 0;
  font-size: 13px;
}

.demo-notice {
  margin-top: 12px;
  padding: 8px 12px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: 6px;
  color: #1d4ed8;
}

.demo-notice p {
  margin: 0;
  font-size: 13px;
}

.form-actions .outline {
  background: #f3f4f6;
  border-color: var(--border);
  color: var(--text);
}

.form-actions .outline:hover {
  background: #e5e7eb;
}

.bulk-import-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid rgba(15, 23, 42, 0.08);
}

.bulk-import-section h4 {
  margin: 0 0 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}

.bulk-import-section .muted {
  margin: 0 0 12px;
  color: var(--muted);
  font-size: 14px;
}

.bulk-import-textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: #fff;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
  line-height: 1.4;
  resize: vertical;
  min-height: 120px;
}

.bulk-import-textarea:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
}

.bulk-import-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.watch-meta span {
  font-size: 13px;
  color: rgba(15, 23, 42, 0.65);
}
.event-list h4 {
  margin: 0 0 0.75rem 0;
  font-size: 1rem;
  font-weight: 600;
  color: #1f2937;
}
.event-row {
  padding: 10px;
  border-radius: 12px;
  border: 1px solid rgba(15, 23, 42, 0.08);
  background: #f9fafb;
}
.event-row + .event-row {
  margin-top: 8px;
}
.event-top strong {
  font-size: 16px;
}
.event-bottom {
  margin-top: 6px;
  font-size: 12px;
  color: rgba(15, 23, 42, 0.6);
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 4px;
}

.event-addresses {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.token-summary {
  display: flex;
  gap: 4px;
  margin-top: 2px;
  flex-wrap: wrap;
}

.token-tag {
  background: rgba(59, 130, 246, 0.1);
  color: #1d4ed8;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 10px;
  border: 1px solid rgba(59, 130, 246, 0.2);
}
.btn-clear {
  border: 1px solid rgba(15, 23, 42, 0.2);
  padding: 4px 12px;
  border-radius: 8px;
}
.hero-panel code,
.watch-form code {
  font-size: 12px;
}

/* 现代化查询面板样式 */
.query-panel {
  margin-bottom: 2.5rem;
  background: white;
  border-radius: 16px;
  border: 1px solid rgba(0, 0, 0, 0.05);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  color: #1f2937; /* 确保文字颜色正确 */
}

/* 监控列表面板样式 */
.watch-list-panel {
  margin-top: 2rem;
}

/* 列表控制栏样式 */
.list-controls {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 2rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.search-filter {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex: 1;
  min-width: 300px;
}

.search-input-wrapper {
  position: relative;
  flex: 1;
}

.search-input {
  width: 100%;
  padding: 0.75rem 3rem 0.75rem 1rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 0.875rem;
  transition: border-color 0.2s ease;
}

.search-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.search-icon {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%);
  color: #6b7280;
  font-size: 1rem;
  pointer-events: none;
}

.filter-select {
  padding: 0.75rem 2.5rem 0.75rem 1rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 0.875rem;
  background: white;
  cursor: pointer;
  min-width: 120px;
}

.filter-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.bulk-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.bulk-actions button {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  color: #374151;
  font-size: 0.875rem;
  cursor: pointer;
  transition: all 0.2s ease;
}

.bulk-actions button:hover:not(:disabled) {
  background: #f3f4f6;
  border-color: #9ca3af;
}

.bulk-actions button:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-danger {
  background: #fef2f2 !important;
  border-color: #fca5a5 !important;
  color: #dc2626 !important;
}

.btn-danger:hover:not(:disabled) {
  background: #fee2e2 !important;
  border-color: #f87171 !important;
}

/* 状态标签样式 */
.status-badges {
  display: flex;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.875rem;
  font-weight: 500;
  background: #f3f4f6;
  color: #374151;
  border: 1px solid #e5e7eb;
}

.status-badge.active {
  background: #f0fdf4;
  color: #166534;
  border-color: #bbf7d0;
}

.status-badge.warning {
  background: #fef3c7;
  color: #92400e;
  border-color: #fcd34d;
}

/* 快速筛选按钮样式 */
.quick-filters {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.filter-btn {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 1rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  background: white;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-btn:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
  color: #374151;
}

.filter-btn.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: white;
}

.filter-btn.active:hover {
  background: #2563eb;
  border-color: #2563eb;
}

/* 空状态样式 */
.empty-state {
  text-align: center;
  padding: 3rem 2rem;
  background: white;
  border-radius: 16px;
  border: 2px dashed #e5e7eb;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
  display: block;
}

.empty-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 0.5rem 0;
}

.empty-description {
  color: #6b7280;
  margin: 0 0 1.5rem 0;
  line-height: 1.5;
}

.empty-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}

/* 监控地址网格样式 */
.watch-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.25rem;
}

/* 虚拟滚动容器样式 */
.virtual-scroll-container {
  position: relative;
  overflow: hidden;
}

.virtual-scroll-viewport {
  overflow-y: auto;
  overflow-x: hidden;
  max-height: 600px; /* 限制高度 */
}

.virtual-scroll-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 1.25rem;
  position: relative;
}

/* 监控卡片样式 */
.watch-card {
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  padding: 1.5rem;
  background: white;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
  cursor: pointer;
}

.watch-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

.watch-card.selected {
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

/* 卡片头部样式 */
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
  gap: 1rem;
}


.card-title-section {
  flex: 1;
  min-width: 0;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}

.chain-badge {
  display: none;
}

.chain-badge.ethereum {
  background: #627eea;
  color: white;
}

.chain-badge.bsc {
  background: #f3ba2f;
  color: white;
}

.chain-badge.solana {
  background: #9945ff;
  color: white;
}

.chain-badge.bitcoin {
  background: #f7931a;
  color: white;
}

.chain-badge.polygon {
  background: #8247e5;
  color: white;
}

.chain-badge.arbitrum {
  background: #28a0f0;
  color: white;
}

.chain-badge.optimism {
  background: #ff0420;
  color: white;
}

.address-label {
  font-size: 1.125rem;
  font-weight: 600;
  color: #1f2937;
  margin: 0;
}

.address-display {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.address-code {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  color: #6b7280;
  background: #f3f4f6;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.copy-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 0.25rem;
  color: #6b7280;
  border-radius: 4px;
  transition: background-color 0.2s ease;
  flex-shrink: 0;
}

.copy-btn:hover {
  background: #f3f4f6;
  color: #374151;
}

.card-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.2s ease;
}

.action-btn.primary {
  background: #3b82f6;
  color: white;
}

.action-btn.primary:hover {
  background: #2563eb;
}

.action-btn.danger {
  background: #fef2f2;
  color: #dc2626;
}

.action-btn.danger:hover {
  background: #fee2e2;
}

/* 地址信息样式 */
.address-info {
  margin-bottom: 1rem;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.info-label {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-value {
  font-size: 0.875rem;
  font-weight: 600;
  color: #1f2937;
}

.info-value.entity {
  color: #7c3aed;
}

.info-value.chain {
  color: #059669;
}

.info-value.balance {
  color: #dc2626;
}

.info-value.last-active {
  color: #6b7280;
}

/* 状态指示器样式 */
.card-status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 1rem;
  border-top: 1px solid #f3f4f6;
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.active {
  background: #10b981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.5);
}

.status-dot.inactive {
  background: #6b7280;
}

.status-dot.error {
  background: #ef4444;
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.5);
}

.status-dot.unknown {
  background: #d1d5db;
}

.status-text {
  font-size: 0.875rem;
  font-weight: 500;
  color: #374151;
}

.last-update {
  font-size: 0.75rem;
  color: #6b7280;
}

/* 事件列表样式 */
.event-list {
  margin-top: 1rem;
}

.event-list h4 {
  font-size: 1rem;
  font-weight: 600;
  color: #1f2937;
  margin: 0 0 0.75rem 0;
}

.event-list .muted {
  color: #6b7280;
  font-style: italic;
  padding: 1rem;
  text-align: center;
  background: #f9fafb;
  border-radius: 8px;
  border: 1px solid #f3f4f6;
}

.events {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.event-row {
  padding: 0.75rem;
  border-radius: 8px;
  border: 1px solid #f3f4f6;
  background: #f9fafb;
  transition: background-color 0.2s ease;
}

.event-row:hover {
  background: #f3f4f6;
}

.event-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.pill {
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.pill.in {
  background: #dcfce7;
  color: #166534;
}

.pill.out {
  background: #fef2f2;
  color: #dc2626;
}

.event-time {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.event-bottom {
  font-size: 0.875rem;
  color: #6b7280;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.event-addresses {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
  transition: color 0.2s ease;
}

.link:hover {
  color: #2563eb;
  text-decoration: underline;
}

.token-summary {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.25rem;
  flex-wrap: wrap;
}

.token-tag {
  background: #e0e7ff;
  color: #3730a3;
  padding: 0.125rem 0.375rem;
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 500;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .list-controls {
    flex-direction: column;
    align-items: stretch;
    gap: 1rem;
  }

  .search-filter {
    flex-direction: column;
    gap: 0.75rem;
  }

  .bulk-actions {
    justify-content: center;
  }
}

@media (max-width: 768px) {
  .watch-grid,
  .virtual-scroll-content {
    grid-template-columns: 1fr;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }

  .card-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .card-actions {
    align-self: flex-end;
  }

  .event-top {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .status-badges {
    flex-direction: column;
    align-items: stretch;
  }

  .quick-filters {
    flex-direction: column;
    align-items: stretch;
  }

  .quick-filters .filter-btn {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .virtual-scroll-viewport {
    max-height: 400px;
  }

  .card-title {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .address-display {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.25rem;
  }

  .address-code {
    width: 100%;
    text-align: center;
  }
}

/* 查询面板头部 */
.panel-header {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  padding: 1.5rem;
  color: #1f2937; /* 确保文字在浅色背景上可见 */
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
}

.panel-title-section {
  flex: 1;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.5rem;
}


.title-text h3 {
  font-size: 1.25rem;
  font-weight: 700;
  color: #1f2937 !important;
  margin: 0 0 0.25rem 0;
}

.panel-subtitle {
  font-size: 0.875rem;
  color: #6b7280 !important;
  margin: 0;
  line-height: 1.5;
}

.header-actions {
  flex-shrink: 0;
}

.btn-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-link:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
  color: #374151;
}

/* 查询表单 */
.query-form {
  padding: 1.5rem;
}

/* 主要输入区域 */
.form-primary {
  margin-bottom: 2rem;
}

.address-input-section {
  margin-bottom: 1.5rem;
}

.input-group.featured {
  background: #f8fafc;
  border: 2px solid #e2e8f0;
  border-radius: 16px;
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.input-group.featured:focus-within {
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.input-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.input-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.input-label.required::after {
  content: '*';
  color: #ef4444;
  font-weight: 700;
}

.input-badges {
  display: flex;
  gap: 0.5rem;
}

.badge {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge.valid {
  background: rgba(16, 185, 129, 0.1);
  color: #047857;
}

.badge.invalid {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
}

.input-container {
  position: relative;
  margin-bottom: 1rem;
}

.address-input {
  width: 100%;
  padding: 1rem 3rem 1rem 1rem;
  border: none;
  background: white;
  border-radius: 8px;
  font-size: 1rem;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  color: #1f2937 !important; /* 强制文字颜色 */
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.address-input:focus {
  outline: none;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.address-input.valid {
  background: rgba(16, 185, 129, 0.05);
  border-color: #10b981;
}

.address-input.invalid {
  background: rgba(239, 68, 68, 0.05);
  border-color: #ef4444;
}

.input-actions {
  position: absolute;
  right: 0.5rem;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  gap: 0.25rem;
}

.action-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #6b7280;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: #e5e7eb;
  color: #374151;
}

.input-footer {
  margin-top: 0.75rem;
}

.input-examples {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.example-label {
  font-size: 0.75rem;
  color: #6b7280;
  font-weight: 500;
}

.example-btn {
  padding: 0.25rem 0.5rem;
  background: rgba(102, 126, 234, 0.1);
  color: #667eea;
  border: 1px solid rgba(102, 126, 234, 0.2);
  border-radius: 6px;
  font-size: 0.75rem;
  font-family: monospace;
  cursor: pointer;
  transition: all 0.2s ease;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.example-btn:hover {
  background: rgba(102, 126, 234, 0.2);
  border-color: rgba(102, 126, 234, 0.4);
}

/* 快速操作按钮 */
.quick-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  align-items: center;
  min-height: 60px; /* 确保最小高度以保持垂直居中 */
}

.btn-primary.large,
.btn-success.large {
  flex: 1;
  max-width: 200px;
  padding: 1rem 1.5rem;
  border: none;
  border-radius: 12px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 48px; /* 确保按钮有足够的高度 */
}

.btn-primary.large {
  background: #3b82f6;
  color: white;
  border: 1px solid #2563eb;
}

.btn-primary.large:hover:not(:disabled) {
  background: #2563eb;
  border-color: #1d4ed8;
}

.btn-success.large {
  background: #10b981;
  color: white;
  border: 1px solid #059669;
}

.btn-success.large:hover:not(:disabled) {
  background: #059669;
  border-color: #047857;
}

.btn-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

/* 隐藏按钮图标 */
.btn-content .icon-search-btn,
.btn-content .icon-add {
  display: none;
}

/* 确保按钮文本垂直居中 */
.btn-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.btn-text {
  font-size: 0.875rem;
}

/* 高级选项 */
.form-advanced {
  border-top: 1px solid #e5e7eb;
  padding-top: 1.5rem;
}

.advanced-toggle {
  margin-bottom: 1rem;
}

.toggle-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: #f3f4f6;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.toggle-btn:hover {
  background: #e5e7eb;
  border-color: #9ca3af;
}

.toggle-btn.active {
  background: #667eea;
  border-color: #667eea;
  color: white;
}

.icon-chevron {
  transition: transform 0.2s ease;
}

.icon-chevron.rotated {
  transform: rotate(180deg);
}

.advanced-options {
  background: #f8fafc;
  border-radius: 12px;
  padding: 1.5rem;
  border: 1px solid #e2e8f0;
}

.options-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.option-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.option-label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.option-input,
.option-select {
  padding: 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.option-input:focus,
.option-select:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.advanced-actions {
  display: flex;
  justify-content: flex-end;
}

.btn-outline.small {
  padding: 0.5rem 1rem;
  background: transparent;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #6b7280;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-outline.small:hover {
  background: #f3f4f6;
  border-color: #9ca3af;
}

/* 状态提示 */
.form-notice {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 12px;
  border-left: 4px solid;
  animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.form-notice.success {
  background: rgba(16, 185, 129, 0.1);
  border-left-color: #10b981;
  color: #047857;
}

.form-notice.error {
  background: rgba(239, 68, 68, 0.1);
  border-left-color: #ef4444;
  color: #dc2626;
}

.form-notice.warning {
  background: rgba(245, 158, 11, 0.1);
  border-left-color: #f59e0b;
  color: #d97706;
}

.form-notice.info {
  background: rgba(59, 130, 246, 0.1);
  border-left-color: #3b82f6;
  color: #2563eb;
}

.notice-content {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.notice-icon {
  font-size: 1.125rem;
  flex-shrink: 0;
}

.notice-text {
  flex: 1;
  font-weight: 500;
}

.notice-close {
  background: none;
  border: none;
  color: currentColor;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
  transition: background-color 0.2s ease;
}

.notice-close:hover {
  background: rgba(0, 0, 0, 0.1);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .query-form {
    padding: 1rem;
  }

  .header-content {
    flex-direction: column;
    align-items: stretch;
    gap: 1rem;
  }

  .panel-title {
    flex-direction: column;
    text-align: center;
    gap: 0.75rem;
  }

  .title-text {
    text-align: center;
  }

  .header-actions {
    text-align: center;
  }

  .address-input {
    font-size: 0.875rem;
    padding: 0.875rem 2.5rem 0.875rem 0.875rem;
  }

  .input-actions {
    right: 0.25rem;
  }

  .action-btn {
    width: 28px;
    height: 28px;
    font-size: 0.75rem;
  }

  .quick-actions {
    flex-direction: column;
    gap: 0.75rem;
  }

  .btn-primary.large,
  .btn-success.large {
    max-width: none;
  }

  .options-grid {
    grid-template-columns: 1fr;
    gap: 0.75rem;
  }

  .input-examples {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.5rem;
  }

  .example-btn {
    max-width: none;
    align-self: stretch;
    text-align: left;
  }
}
</style>

