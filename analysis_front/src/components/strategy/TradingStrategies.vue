<template>
  <div class="tab-pane">
    <div v-if="validationErrors.strategy" class="tab-error">{{ validationErrors.strategy }}</div>

    <div class="config-grid">
      <!-- 传统交易策略 -->
      <div class="config-card">
        <h5 class="card-title">传统交易策略</h5>

        <!-- 不开空限制 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.no_short_below_market_cap" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">不开空市值限制</span>
          </div>
          <div class="condition-description">
            市值低于
            <input
              v-model.number="conditions.market_cap_limit_short"
              class="inline-input"
              type="number"
              min="0"
              step="100"
              placeholder="5000"
            /> 万不开空
          </div>
        </div>

        <!-- 资金费率要求 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.funding_rate_filter_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">资金费率过滤</span>
          </div>
          <div class="condition-description">
            资金费率高于
            <input
              v-model.number="conditions.min_funding_rate"
              class="inline-input"
              type="number"
              min="-1"
              max="1"
              step="0.001"
              placeholder="0.01"
            /> % 时才执行策略
            <div class="config-note" style="margin-top: 8px;">
              💡 资金费率是期货合约的融资成本，正值表示资金成本较高，负值表示可以获得资金补贴
            </div>
          </div>
        </div>

        <!-- 合约涨幅排名过滤 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.futures_price_rank_filter_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">合约涨幅排名过滤</span>
          </div>
          <div class="condition-description">
            仅在合约涨幅排名前
            <input
              v-model.number="conditions.max_futures_price_rank"
              class="inline-input"
              type="number"
              min="1"
              max="500"
              placeholder="5"
            /> 名以内执行策略
            <div class="config-note" style="margin-top: 8px;">
              💡 限制策略仅在合约市场涨幅排名靠前的币种上执行，避免在冷门币种上交易
            </div>
          </div>
        </div>

        <!-- 开空条件 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.short_on_gainers" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">涨幅开空</span>
          </div>
          <div class="condition-description">
            市值高于
            <input
              v-model.number="conditions.market_cap_limit_short"
              class="inline-input"
              type="number"
              min="0"
              step="100"
              placeholder="5000"
            /> 万，如果进入涨幅前
            <input
              v-model.number="conditions.gainers_rank_limit"
              class="inline-input"
              type="number"
              min="1"
              max="100"
              placeholder="7"
            /> 位，直接开空
            <input
              v-model.number="conditions.short_multiplier"
              class="inline-input"
              type="number"
              min="0.1"
              max="10"
              step="0.1"
              placeholder="3.0"
            /> 倍杠杆
          </div>
        </div>

        <!-- 合约涨幅开空策略 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.futures_price_short_strategy_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">合约涨幅开空策略</span>
          </div>
          <div class="condition-description">
            市值高于
            <input
              v-model.number="conditions.futures_price_short_min_market_cap"
              class="inline-input"
              type="number"
              min="0"
              step="0.01"
              placeholder="1000"
            /> 万，涨幅排名前
            <input
              v-model.number="conditions.futures_price_short_max_rank"
              class="inline-input"
              type="number"
              min="1"
              max="100"
              placeholder="5"
            /> 名以内，资金费率高于
            <input
              v-model.number="conditions.futures_price_short_min_funding_rate"
              class="inline-input"
              type="number"
              min="-1"
              max="1"
              step="0.001"
              placeholder="-0.005"
            /> %，直接开空
            <input
              v-model.number="conditions.futures_price_short_leverage"
              class="inline-input"
              type="number"
              min="0.1"
              max="10"
              step="0.1"
              placeholder="3.0"
            /> 倍杠杆
          </div>
        </div>

        <!-- 开多条件 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.long_on_small_gainers" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">小市值涨幅开多</span>
          </div>
          <div class="condition-description">
            市值低于
            <input
              v-model.number="conditions.market_cap_limit_long"
              class="inline-input"
              type="number"
              min="0"
              step="100"
              placeholder="500"
            /> 万，如果进入涨幅前
            <input
              v-model.number="conditions.gainers_rank_limit_long"
              class="inline-input"
              type="number"
              min="1"
              max="100"
              placeholder="20"
            /> 位，直接开多
            <input
              v-model.number="conditions.long_multiplier"
              class="inline-input"
              type="number"
              min="0.1"
              max="10"
              step="0.1"
              placeholder="1.0"
            /> 倍杠杆
          </div>
        </div>
      </div>

      <!-- 技术指标策略 -->
      <div class="config-card">
        <h5 class="card-title">📈 技术指标策略</h5>

        <!-- 均线策略 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.moving_average_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">
              均线策略
              <span class="help-tooltip" data-tooltip="基于移动平均线交叉和趋势的交易策略">?</span>
            </span>
          </div>
          <div class="condition-description">
            <div v-if="conditions.moving_average_enabled" class="ma-config">
              <!-- 信号模式选择 -->
              <div class="config-item">
                <label>信号模式：</label>
                <select v-model="conditions.ma_signal_mode" class="inline-select">
                  <option value="QUALITY_FIRST">质量优先 (高品质，低数量)</option>
                  <option value="QUANTITY_FIRST">数量优先 (中等品质，高数量)</option>
                </select>
              </div>

              <!-- 模式说明 -->
              <div class="config-item mode-description">
                <div v-if="conditions.ma_signal_mode === 'QUALITY_FIRST'" class="quality-mode">
                  <strong>🎯 质量优先模式</strong><br>
                  • 信号质量极高 (胜率80-90%)<br>
                  • 假信号极少<br>
                  • 适合保守投资者
                </div>
                <div v-else-if="conditions.ma_signal_mode === 'QUANTITY_FIRST'" class="quantity-mode">
                  <strong>🚀 数量优先模式</strong><br>
                  • 信号数量充足 (每天5-15个)<br>
                  • 资金利用高效<br>
                  • 适合活跃交易者
                </div>
              </div>

              <!-- 均线类型选择 -->
              <div class="config-item">
                <label>均线类型：</label>
                <select v-model="conditions.ma_type" class="inline-select">
                  <option value="SMA">简单移动平均线 (SMA)</option>
                  <option value="EMA">指数移动平均线 (EMA)</option>
                  <option value="WMA">加权移动平均线 (WMA)</option>
                </select>
              </div>

              <!-- 均线周期设置 -->
              <div class="config-item">
                <label>短期均线：</label>
                <input
                  v-model.number="conditions.short_ma_period"
                  class="inline-input small"
                  type="number"
                  min="5"
                  max="50"
                  step="1"
                  placeholder="5"
                /> 日
              </div>
              <div class="config-item">
                <label>长期均线：</label>
                <input
                  v-model.number="conditions.long_ma_period"
                  class="inline-input small"
                  type="number"
                  min="10"
                  max="200"
                  step="1"
                  placeholder="20"
                /> 日
              </div>

              <!-- 交叉信号 -->
              <div class="config-item">
                <label>交叉信号：</label>
                <select v-model="conditions.ma_cross_signal" class="inline-select">
                  <option value="GOLDEN_CROSS">金叉买入 (短期上穿长期)</option>
                  <option value="DEATH_CROSS">死叉卖出 (短期下穿长期)</option>
                  <option value="BOTH">双向交易 (金叉买入+死叉卖出)</option>
                </select>
              </div>

              <!-- 趋势过滤 -->
              <div class="condition-sub-item">
                <label class="condition-checkbox small">
                  <input type="checkbox" v-model="conditions.ma_trend_filter" />
                  <span class="checkmark-small"></span>
                </label>
                <span class="condition-title small">趋势过滤</span>
                <select v-if="conditions.ma_trend_filter" v-model="conditions.ma_trend_direction" class="inline-select small">
                  <option value="UP">仅上涨趋势</option>
                  <option value="DOWN">仅下跌趋势</option>
                  <option value="BOTH">双向趋势</option>
                </select>
              </div>
            </div>
            <div class="config-note">
              💡 均线策略适合趋势性行情，金叉和死叉是经典的技术分析信号
            </div>
          </div>
        </div>
      </div>

      <!-- 均值回归策略 -->
      <div class="config-card">
        <h5 class="card-title">🔄 增强均值回归策略</h5>
        <p class="section-description">智能市场适应，支持保守和激进两种模式，特别适合震荡行情</p>

        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.mean_reversion_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">
              增强均值回归策略
              <span class="help-tooltip" data-tooltip="基于价格向均值回归理论的增强版本，支持智能市场环境检测、动态参数调整和多重风险控制">?</span>
            </span>
          </div>
          <div class="condition-description">
            <div v-if="conditions.mean_reversion_enabled" class="mr-config">
              <!-- 策略模式选择 -->
              <div class="config-item">
                <label>策略模式：</label>
                <select v-model="conditions.mean_reversion_mode" class="inline-select">
                  <option value="basic">基础模式 (传统)</option>
                  <option value="enhanced">增强模式 (智能)</option>
                </select>
              </div>

              <!-- 增强模式子模式选择 -->
              <div v-if="conditions.mean_reversion_mode === 'enhanced'" class="config-item">
                <label>交易风格：</label>
                <select v-model="conditions.mean_reversion_sub_mode" class="inline-select">
                  <option value="conservative">保守模式 (高胜率)</option>
                  <option value="aggressive">激进模式 (高频交易)</option>
                  <option value="adaptive">自适应模式 (智能平衡)</option>
                </select>
              </div>

              <!-- 模式说明 -->
              <div v-if="conditions.mean_reversion_mode === 'enhanced'" class="mode-description">
                <div v-if="conditions.mean_reversion_sub_mode === 'conservative'" class="conservative-mode">
                  <strong>🛡️ 保守模式</strong><br>
                  • 信号确认度: 80% (极高)<br>
                  • 交易频率: 低 (每周1-3次)<br>
                  • 风险控制: 极严格 (1.5%仓位, 3倍止损, 6%止盈)<br>
                  • 适合: 风险偏好低，追求稳定收益
                </div>
                <div v-else-if="conditions.mean_reversion_sub_mode === 'aggressive'" class="aggressive-mode">
                  <strong>🚀 激进模式</strong><br>
                  • 信号确认度: 25% (适中)<br>
                  • 交易频率: 高 (每天3-8次)<br>
                  • 风险控制: 激进 (4%仓位, 2倍止损, 20%止盈)<br>
                  • 适合: 风险偏好高，追求高收益
                </div>
                <div v-else-if="conditions.mean_reversion_sub_mode === 'adaptive'" class="adaptive-mode">
                  <strong>🧠 自适应模式 (推荐)</strong><br>
                  • 信号确认度: 动态调整 (15%-85%)<br>
                  • 交易频率: 高 (每天4-8次)<br>
                  • 风险控制: 智能平衡 (2.5%仓位, 2.5倍止损, 12%止盈)<br>
                  • 适合: 全市场环境，追求高收益<br>
                  • <span style="color: #10b981; font-weight: bold;">⚡ 基于大数据优化，表现最佳</span>
                </div>
              </div>

              <!-- 基础参数 -->
              <div class="config-item">
                <label>计算周期：</label>
                <input
                  v-model.number="conditions.mr_period"
                  class="inline-input small"
                  type="number"
                  min="10"
                  max="50"
                  step="1"
                  placeholder="20"
                /> 日
                <span class="unit">{{ getOptimizedParamDisplay(conditions, 'period') }}</span>
              </div>

              <!-- 指标启用选项 -->
              <div class="mr-indicators">
                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.mr_bollinger_bands_enabled" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">布林带均值回归</span>
                  <div v-if="conditions.mr_bollinger_bands_enabled" class="sub-config">
                    倍数:
                    <input
                      v-model.number="conditions.mr_bollinger_multiplier"
                      class="inline-input tiny"
                      type="number"
                      min="1.5"
                      max="3.0"
                      step="0.1"
                      placeholder="2.0"
                    />
                    <span class="unit">{{ getOptimizedParamDisplay(conditions, 'bollinger') }}</span>
                  </div>
                </div>

                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.mr_rsi_enabled" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">RSI均值回归</span>
                  <div v-if="conditions.mr_rsi_enabled" class="sub-config">
                    超买:
                    <input
                      v-model.number="conditions.mr_rsi_overbought"
                      class="inline-input tiny"
                      type="number"
                      min="60"
                      max="80"
                      step="1"
                      placeholder="70"
                    />
                    超卖:
                    <input
                      v-model.number="conditions.mr_rsi_oversold"
                      class="inline-input tiny"
                      type="number"
                      min="20"
                      max="40"
                      step="1"
                      placeholder="30"
                    />
                    <span class="unit">{{ getOptimizedParamDisplay(conditions, 'rsi') }}</span>
                  </div>
                </div>

                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.mr_price_channel_enabled" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">价格通道均值回归</span>
                  <div v-if="conditions.mr_price_channel_enabled" class="sub-config">
                    周期:
                    <input
                      v-model.number="conditions.mr_channel_period"
                      class="inline-input tiny"
                      type="number"
                      min="10"
                      max="30"
                      step="1"
                      placeholder="20"
                    />
                  </div>
                </div>
              </div>

              <!-- 回归强度要求 -->
              <div class="config-item">
                <label>最小回归强度：</label>
                <input
                  v-model.number="conditions.mr_min_reversion_strength"
                  class="inline-input small"
                  type="number"
                  step="0.01"
                  placeholder="0.15"
                />
                <span class="unit">{{ getOptimizedParamDisplay(conditions, 'strength') || '(0.1-1.0，建议0.15)' }}</span>
              </div>

              <!-- 增强模式专用设置 -->
              <div v-if="conditions.mean_reversion_mode === 'enhanced'" class="enhanced-settings">
                <h6 class="enhanced-title">⚙️ 增强功能配置</h6>

                <!-- 市场环境检测 -->
                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.market_environment_detection" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">智能市场环境检测</span>
                  <span class="help-tooltip small" data-tooltip="自动识别震荡、趋势、高波动等市场环境，动态调整策略参数">?</span>
                </div>

                <!-- 智能权重系统 -->
                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.intelligent_weights" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">智能信号权重系统</span>
                  <span class="help-tooltip small" data-tooltip="根据市场环境动态调整各技术指标的权重，提高信号质量">?</span>
                </div>

                <!-- 高级风险管理 -->
                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.advanced_risk_management" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">高级风险管理系统</span>
                  <span class="help-tooltip small" data-tooltip="动态止损止盈、仓位管理、每日损失限制等全方位风险控制">?</span>
                </div>

                <!-- 性能监控 -->
                <div class="config-item">
                  <label class="condition-checkbox small">
                    <input type="checkbox" v-model="conditions.performance_monitoring" />
                    <span class="checkmark-small"></span>
                  </label>
                  <span class="condition-title small">实时性能监控</span>
                  <span class="help-tooltip small" data-tooltip="实时跟踪胜率、盈亏、持仓时间等关键指标，自动优化策略">?</span>
                </div>
              </div>

              <!-- 风险管理参数配置 -->
              <div class="mr-risk-management">
                <h6 class="risk-title">🛡️ 风险管理参数</h6>

                <div class="config-item">
                  <label>止损倍数：</label>
                  <input
                    v-model.number="conditions.mr_stop_loss_multiplier"
                    class="inline-input small"
                    type="number"
                    min="1.1"
                    max="5.0"
                    step="0.1"
                    placeholder="2.0"
                  />
                  <span class="unit">倍标准差</span>
                  <span class="help-tooltip small" data-tooltip="价格偏离均值的标准差倍数作为止损点，例如2.0表示偏离2倍标准差时止损">?</span>
                </div>

                <div class="config-item">
                  <label>止盈倍数：</label>
                  <input
                    v-model.number="conditions.mr_take_profit_multiplier"
                    class="inline-input small"
                    type="number"
                    min="1.01"
                    max="3.0"
                    step="0.01"
                    placeholder="1.08"
                  />
                  <span class="unit">倍标准差</span>
                  <span class="help-tooltip small" data-tooltip="价格偏离均值的标准差倍数作为止盈点，例如1.08表示偏离1.08倍标准差时止盈">?</span>
                </div>

                <div class="config-item">
                  <label>最大仓位：</label>
                  <input
                    v-model.number="conditions.mr_max_position_size"
                    class="inline-input small"
                    type="number"
                    min="0.005"
                    max="0.1"
                    step="0.005"
                    placeholder="0.02"
                  />
                  <span class="unit">%</span>
                  <span class="help-tooltip small" data-tooltip="单个交易对的最大仓位比例，建议2%以内控制风险">?</span>
                </div>

                <div class="config-item">
                  <label>最大持仓：</label>
                  <input
                    v-model.number="conditions.mr_max_hold_hours"
                    class="inline-input small"
                    type="number"
                    min="1"
                    max="168"
                    step="1"
                    placeholder="24"
                  />
                  <span class="unit">小时</span>
                  <span class="help-tooltip small" data-tooltip="单笔交易的最大持仓时间，超过此时间将强制平仓">?</span>
                </div>
              </div>
            </div>
            <div class="config-note">
              💡 增强均值回归策略集成了智能市场环境检测、动态参数调整和多重风险控制，在震荡市中能够显著提高胜率和收益稳定性
            </div>
          </div>
        </div>
      </div>

      <!-- 套利策略 -->
      <div class="config-card">
        <h5 class="card-title">套利策略</h5>

        <!-- 跨交易所套利 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.cross_exchange_arb_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">
              跨交易所套利
              <span class="help-tooltip" data-tooltip="利用不同交易所间的价格差异进行无风险套利">?</span>
            </span>
          </div>
          <div class="condition-description">
            价差超过
            <input
              v-model.number="conditions.price_diff_threshold"
              class="inline-input"
              type="number"
              min="0.01"
              max="10"
              step="0.01"
              placeholder="0.5"
            /> % 且套利金额大于
            <input
              v-model.number="conditions.min_arb_amount"
              class="inline-input"
              type="number"
              min="1"
              step="1"
              placeholder="100"
            /> USDT时执行套利
          </div>
        </div>

        <!-- 现货-合约套利 -->
        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.spot_future_arb_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">
              现货-合约套利
              <span class="help-tooltip" data-tooltip="利用现货和合约价格差异及资金费率进行套利">?</span>
            </span>
          </div>
          <div class="condition-description">
            基差超过
            <input
              v-model.number="conditions.basis_threshold"
              class="inline-input"
              type="number"
              min="0.01"
              max="5"
              step="0.01"
              placeholder="0.2"
            /> % 或资金费率超过
            <input
              v-model.number="conditions.funding_rate_threshold"
              class="inline-input"
              type="number"
              min="0.001"
              max="1"
              step="0.001"
              placeholder="0.01"
            /> % 时执行套利
          </div>
        </div>
      </div>

      <!-- 网格交易策略 -->
      <div class="config-card" @click="onGridTradingClick" :class="{ 'clickable': conditions.grid_trading_enabled && availableSymbols.length === 0 }">
        <h5 class="card-title">
          网格交易策略
        </h5>

        <!-- 增强币种选择器 -->
        <div class="symbol-selector-section">
          <div class="grid-param-row">
            <label class="grid-param-label">选择币种：</label>
            <div class="symbol-selector-wrapper">
              <!-- 币种选择下拉框 -->
              <div class="symbol-dropdown-container">
                <div
                  class="symbol-dropdown-trigger"
                  @click="toggleSymbolDropdown"
                >
                  <div class="selected-symbol-display">
                    <span v-if="selectedGridSymbol" class="selected-symbol-text">
                      {{ getSelectedSymbolDisplay() }}
                    </span>
                    <span v-else class="placeholder-text">请选择币种...</span>
                  </div>
                  <div class="dropdown-arrow" :class="{ 'rotated': showSymbolDropdown }">
                    ▼
                  </div>
                </div>

                <!-- 下拉菜单 -->
                <div v-if="showSymbolDropdown" class="symbol-dropdown-menu">
                  <!-- 加载状态提示 -->
                  <div v-if="loadingSymbols" class="loading-indicator">
                    <div class="loading-spinner"></div>
                    <span>正在加载币种列表...</span>
                  </div>

                  <!-- 搜索框 -->
                  <div class="symbol-search-container" v-else>
                    <input
                      v-model="symbolSearchQuery"
                      @input="filterSymbols"
                      type="text"
                      class="symbol-search-input"
                      placeholder="搜索币种..."
                    />
                    <div class="search-icon">🔍</div>
                  </div>

                  <!-- 排序选项 -->
                  <div class="sort-options">
                    <button
                      v-for="option in sortOptions"
                      :key="option.key"
                      @click.prevent="setSortOption(option.key)"
                      class="sort-option-btn"
                      :class="{ 'active': currentSort === option.key }"
                    >
                      {{ option.label }}
                    </button>
                  </div>

                  <!-- 币种列表 -->
                  <div class="symbol-list-container">
                    <div
                      v-for="symbol in filteredSymbols"
                      :key="symbol.symbol"
                      @click.prevent="selectSymbol(symbol.symbol)"
                      class="symbol-list-item"
                      :class="{ 'selected': selectedGridSymbol === symbol.symbol }"
                    >
                      <div class="symbol-info">
                        <div class="symbol-name">
                          <span class="symbol-code">{{ symbol.symbol }}</span>
                          <span class="symbol-price" v-if="symbol.current_price > 0">
                            ${{ formatPrice(symbol.current_price) }}
                          </span>
                        </div>
                        <div class="symbol-details" v-if="conditions.grid_trading_enabled">
                          <!-- 网格交易策略显示网格评分信息 -->
                          <div class="grid-scores">
                            <span class="grid-score-item">
                              <span class="score-label">综合:</span>
                              <span class="score-value" :class="getScoreClass(symbol.grid_overall_score)">
                                {{ (symbol.grid_overall_score || 0).toFixed(2) }}
                              </span>
                            </span>
                            <span class="grid-score-item">
                              <span class="score-label">波动:</span>
                              <span class="score-value" :class="getScoreClass(symbol.grid_volatility_score)">
                                {{ (symbol.grid_volatility_score || 0).toFixed(2) }}
                              </span>
                            </span>
                            <span class="grid-score-item">
                              <span class="score-label">流动性:</span>
                              <span class="score-value" :class="getScoreClass(symbol.grid_liquidity_score)">
                                {{ (symbol.grid_liquidity_score || 0).toFixed(2) }}
                              </span>
                            </span>
                          </div>
                        </div>
                        <div class="symbol-details" v-else>
                          <!-- 其他策略显示传统信息 -->
                          <span
                            class="price-change"
                            :class="{
                              'positive': symbol.price_change_percent > 0,
                              'negative': symbol.price_change_percent < 0
                            }"
                            v-if="symbol.price_change_percent"
                          >
                            {{ formatPercent(symbol.price_change_percent) }}
                          </span>
                          <span class="volume" v-if="symbol.volume_24h">Vol: {{ formatVolume(symbol.volume_24h) }}</span>
                        </div>
                      </div>
                      <div class="symbol-market-cap">
                        市值: {{ formatMarketCap(symbol.market_cap_usd) }}
                      </div>
                    </div>

                    <!-- 加载更多 -->
                    <div v-if="canLoadMoreSymbols" class="load-more-container">
                      <button @click.prevent="loadMoreSymbols" class="load-more-btn" :disabled="loadingSymbols">
                        <span v-if="loadingSymbols">加载中...</span>
                        <span v-else>加载更多</span>
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 分析按钮 -->
              <button
                v-if="selectedGridSymbol && conditions.grid_trading_enabled"
                @click.prevent="analyzeSymbolForGrid"
                :disabled="analyzingSymbol"
                class="analyze-symbol-btn"
              >
                <span v-if="analyzingSymbol">分析中...</span>
                <span v-else>🔍 自动分析</span>
              </button>
            </div>
          </div>

          <!-- 最近使用的币种 -->
          <div v-if="recentSymbols.length > 0" class="recent-symbols-section">
            <div class="recent-symbols-label">最近使用:</div>
            <div class="recent-symbols-list">
              <button
                v-for="symbol in recentSymbols"
                :key="symbol"
                @click.prevent="selectSymbol(symbol)"
                class="recent-symbol-btn"
                :class="{ 'active': selectedGridSymbol === symbol }"
              >
                {{ symbol }}
              </button>
            </div>
          </div>

          <!-- 分析结果显示 -->
          <div v-if="symbolAnalysis" class="symbol-analysis-result">
            <div class="analysis-summary">
              <h6>{{ selectedGridSymbol }} 分析结果</h6>
              <div class="analysis-metrics">
                <div class="metric-item">
                  <span class="metric-label">当前价格:</span>
                  <span class="metric-value">{{ symbolAnalysis.currentPrice.toFixed(4) }} USDT</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">波动率:</span>
                  <span class="metric-value">{{ (symbolAnalysis.volatility * 100).toFixed(2) }}%</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">推荐网格层数:</span>
                  <span class="metric-value">{{ symbolAnalysis.recommendedLevels }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">价格区间:</span>
                  <span class="metric-value">{{ symbolAnalysis.recommendedLower.toFixed(4) }} - {{ symbolAnalysis.recommendedUpper.toFixed(4) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="condition-card">
          <div class="condition-header">
            <label class="condition-checkbox">
              <input type="checkbox" v-model="conditions.grid_trading_enabled" />
              <span class="checkmark"></span>
            </label>
            <span class="condition-title">
              网格交易策略
              <span class="help-tooltip" data-tooltip="在价格区间内设置多个买卖点，通过低买高卖获得稳定收益，特别适合震荡行情。选择后将自动加载适合网格交易的币种列表">?</span>
            </span>
          </div>

          <div class="condition-description">
            <!-- 只有当网格交易策略启用时才显示配置区域 -->
            <div v-if="conditions.grid_trading_enabled" class="grid-config-section">
              <div class="grid-param-row">
                <label class="grid-param-label">价格区间：</label>
                <input
                  v-model.number="conditions.grid_lower_price"
                  class="grid-price-input"
                  type="number"
                  min="0.00000001"
                  step="0.00000001"
                  placeholder="下限价格"
                />
                <span class="price-separator">-</span>
                <input
                  v-model.number="conditions.grid_upper_price"
                  class="grid-price-input"
                  type="number"
                  min="0.00000001"
                  step="0.00000001"
                  placeholder="上限价格"
                />
                <span class="price-unit">USDT</span>
              </div>

              <div class="grid-param-row">
                <label class="grid-param-label">网格层数：</label>
                <input
                  v-model.number="conditions.grid_levels"
                  class="grid-number-input"
                  type="number"
                  min="2"
                  max="100"
                  placeholder="10"
                />
                <span class="param-unit">层</span>
              </div>

              <div class="grid-param-row">
                <label class="grid-param-label">利润百分比：</label>
                <input
                  v-model.number="conditions.grid_profit_percent"
                  class="grid-number-input"
                  type="number"
                  min="0.01"
                  max="10"
                  step="0.01"
                  placeholder="1.0"
                />
                <span class="param-unit">%</span>
              </div>

              <div class="grid-param-row">
                <label class="grid-param-label">投资金额：</label>
                <input
                  v-model.number="conditions.grid_investment_amount"
                  class="grid-number-input"
                  type="number"
                  min="10"
                  step="10"
                  placeholder="1000"
                />
                <span class="param-unit">USDT</span>
              </div>

              <div class="grid-options-row">
                <label class="condition-checkbox small">
                  <input type="checkbox" v-model="conditions.grid_rebalance_enabled" />
                  <span class="checkmark small"></span>
                </label>
                <span class="option-label">启用网格再平衡</span>

                <label class="condition-checkbox small">
                  <input type="checkbox" v-model="conditions.grid_stop_loss_enabled" />
                  <span class="checkmark small"></span>
                </label>
                <span class="option-label">启用网格止损</span>

                <div v-if="conditions.grid_stop_loss_enabled" class="stop-loss-config">
                  <input
                    v-model.number="conditions.grid_stop_loss_percent"
                    class="grid-number-input small"
                    type="number"
                    min="1"
                    max="50"
                    step="0.1"
                    placeholder="10"
                  />
                  <span class="param-unit">%</span>
                </div>
              </div>
            </div>


            <!-- 网格参数预览 -->
            <div v-if="conditions.grid_trading_enabled && conditions.grid_upper_price > 0 && conditions.grid_lower_price > 0 && conditions.grid_levels > 0" class="grid-preview-section">
              <h6 class="preview-title">📊 网格参数预览</h6>

              <!-- 网格可视化图表 -->
              <div class="grid-visualization">
                <div class="grid-chart">
                  <div class="price-axis">
                    <div class="price-label">{{ conditions.grid_upper_price.toFixed(4) }}</div>
                    <div class="grid-lines">
                      <div
                        v-for="(level, index) in generateGridLevels()"
                        :key="index"
                        class="grid-line"
                        :style="{ bottom: level.position + '%' }"
                      >
                        <div class="grid-price">{{ level.price.toFixed(4) }}</div>
                        <div class="grid-marker" :class="{ 'buy-marker': level.isBuy, 'sell-marker': level.isSell }"></div>
                      </div>
                    </div>
                    <div class="price-label">{{ conditions.grid_lower_price.toFixed(4) }}</div>
                  </div>
                  <div class="grid-info">
                    <div class="info-item">
                      <span class="info-label">网格范围:</span>
                      <span class="info-value">{{ conditions.grid_lower_price.toFixed(4) }} - {{ conditions.grid_upper_price.toFixed(4) }} USDT</span>
                    </div>
                    <div class="info-item">
                      <span class="info-label">波动区间:</span>
                      <span class="info-value">{{ getPriceRangePercent() }}%</span>
                    </div>
                    <div class="info-item">
                      <span class="info-label">每格间距:</span>
                      <span class="info-value">{{ getGridSpacing().toFixed(4) }} USDT ({{ getGridSpacingPercent() }}%)</span>
                    </div>
                  </div>
                </div>
              </div>

              <div class="preview-grid">
                <div class="preview-item">
                  <span class="preview-label">网格层数:</span>
                  <span class="preview-value">{{ conditions.grid_levels }}层</span>
                </div>
                <div class="preview-item">
                  <span class="preview-label">利润率:</span>
                  <span class="preview-value">{{ conditions.grid_profit_percent }}%</span>
                </div>
                <div class="preview-item">
                  <span class="preview-label">总投资:</span>
                  <span class="preview-value">{{ conditions.grid_investment_amount }} USDT</span>
                </div>
                <div class="preview-item">
                  <span class="preview-label">每格投资:</span>
                  <span class="preview-value">{{ (conditions.grid_investment_amount / conditions.grid_levels).toFixed(2) }} USDT</span>
                </div>
                <div class="preview-item">
                  <span class="preview-label">预期单程收益:</span>
                  <span class="preview-value">{{ calculateExpectedProfit() }}</span>
                </div>
                <div class="preview-item">
                  <span class="preview-label">潜在最大收益:</span>
                  <span class="preview-value">{{ calculateMaxPotentialProfit() }}</span>
                </div>
              </div>

              <!-- 参数验证提示 -->
              <div class="validation-messages">
                <div v-for="message in getGridValidationMessages()" :key="message.text" :class="['validation-item', message.type]">
                  {{ message.icon }} {{ message.text }}
                </div>
              </div>

              <!-- 智能参数建议 -->
              <div v-if="getGridValidationMessages().length > 0" class="parameter-suggestions">
                <h6 class="suggestions-title">💡 优化建议</h6>
                <div class="suggestions-list">
                  <div v-for="suggestion in getParameterSuggestions()" :key="suggestion.id" class="suggestion-item">
                    <div class="suggestion-content">
                      <span class="suggestion-icon">{{ suggestion.icon }}</span>
                      <span class="suggestion-text">{{ suggestion.text }}</span>
                      <button v-if="suggestion.action" @click.prevent="applySuggestion(suggestion)" class="apply-btn">
                        应用
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="config-note">
            网格交易策略在价格震荡区间内设置多层买卖点，通过频繁小额交易获得稳定收益。适合横盘震荡行情，不适合单边趋势行情。
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, watch, computed, nextTick, onMounted } from 'vue'
import { api } from '../../api/api.js'

// Props
const props = defineProps({
  conditions: {
    type: Object,
    required: true
  },
  validationErrors: {
    type: Object,
    default: () => ({})
  }
})

// Emits
const emit = defineEmits(['update:conditions'])

// 初始化白名单字段 - 网格策略默认启用白名单
props.conditions.use_symbol_whitelist = true
// 确保symbol_whitelist始终是一个数组，使用深拷贝避免引用问题
if (!props.conditions.symbol_whitelist || !Array.isArray(props.conditions.symbol_whitelist)) {
  props.conditions.symbol_whitelist = []
} else {
  // 创建数组的深拷贝，避免Vue响应式系统可能导致的类型转换问题
  props.conditions.symbol_whitelist = [...props.conditions.symbol_whitelist]
}

// 初始化均值回归增强模式字段
if (!props.conditions.mean_reversion_mode) {
  props.conditions.mean_reversion_mode = 'basic'
}
if (!props.conditions.mean_reversion_sub_mode) {
  props.conditions.mean_reversion_sub_mode = 'conservative'
}

// 初始化增强功能字段
if (typeof props.conditions.market_environment_detection === 'undefined') {
  props.conditions.market_environment_detection = true
}
if (typeof props.conditions.intelligent_weights === 'undefined') {
  props.conditions.intelligent_weights = true
}
if (typeof props.conditions.advanced_risk_management === 'undefined') {
  props.conditions.advanced_risk_management = true
}
if (typeof props.conditions.performance_monitoring === 'undefined') {
  props.conditions.performance_monitoring = false
}

// 初始化所有策略相关参数
initializeStrategyParams(props.conditions)

// 初始化网格参数：如果网格交易被禁用，清空所有网格相关参数
if (!props.conditions.grid_trading_enabled) {
  props.conditions.grid_upper_price = 0
  props.conditions.grid_lower_price = 0
  props.conditions.grid_levels = 0
  props.conditions.grid_profit_percent = 0
  props.conditions.grid_investment_amount = 0
  props.conditions.grid_rebalance_enabled = false
  props.conditions.grid_stop_loss_enabled = false
  props.conditions.grid_stop_loss_percent = 0
}

// 初始化合约涨幅排名过滤参数
if (typeof props.conditions.futures_price_rank_filter_enabled === 'undefined') {
  props.conditions.futures_price_rank_filter_enabled = false
}
if (!props.conditions.max_futures_price_rank) {
  props.conditions.max_futures_price_rank = 10
}

// 初始化合约涨幅开空策略参数
if (typeof props.conditions.futures_price_short_strategy_enabled === 'undefined') {
  props.conditions.futures_price_short_strategy_enabled = false
}
if (typeof props.conditions.futures_price_short_min_market_cap === 'undefined') {
  props.conditions.futures_price_short_min_market_cap = 1000 // 默认1000万市值
}
if (!props.conditions.futures_price_short_max_rank) {
  props.conditions.futures_price_short_max_rank = 5
}
if (typeof props.conditions.futures_price_short_min_funding_rate === 'undefined') {
  props.conditions.futures_price_short_min_funding_rate = -0.005
}
if (!props.conditions.futures_price_short_leverage) {
  props.conditions.futures_price_short_leverage = 3.0
}

// 网格交易币种选择状态
const selectedGridSymbol = ref('')
const availableSymbols = ref([])
const analyzingSymbol = ref(false)
const symbolAnalysis = ref(null)
const showSymbolDropdown = ref(false)
const symbolSearchQuery = ref('')
const filteredSymbols = ref([])
// 当前排序方式（动态默认值）
const currentSort = ref('market_cap')

// 监听策略变化，自动调整默认排序
watch(() => props.conditions.grid_trading_enabled, (enabled) => {
  if (enabled) {
    // 启用网格交易时，默认按网格适应性排序
    currentSort.value = 'grid_overall'
  } else {
    // 其他策略时，默认按市值排序
    currentSort.value = 'market_cap'
  }
})
const loadingSymbols = ref(false)
const symbolsPage = ref(1)
const canLoadMoreSymbols = ref(true)
const recentSymbols = ref([])

// 网格交易相关状态
const onGridSymbolChange = () => {
  // 清空之前的分析结果
  symbolAnalysis.value = null
}

// 白名单相关函数 - 自动化管理
const addSymbolToWhitelist = (symbol = null) => {
  const targetSymbol = symbol || selectedGridSymbol.value
  if (!targetSymbol) {
    return false
  }

  // 确保symbol_whitelist是数组，使用深拷贝
  if (!props.conditions.symbol_whitelist || !Array.isArray(props.conditions.symbol_whitelist)) {
    props.conditions.symbol_whitelist = []
  } else {
    props.conditions.symbol_whitelist = [...props.conditions.symbol_whitelist]
  }

  if (symbolInWhitelist(targetSymbol)) {
    return false // 已经在白名单中
  }

  props.conditions.symbol_whitelist.push(targetSymbol)
  return true
}

const symbolInWhitelist = (symbol) => {
  return props.conditions.symbol_whitelist &&
         Array.isArray(props.conditions.symbol_whitelist) &&
         props.conditions.symbol_whitelist.includes(symbol)
}

// 自动管理白名单
const autoManageWhitelist = () => {
  if (!selectedGridSymbol.value) {
    return // 没有选中币种
  }

  const added = addSymbolToWhitelist(selectedGridSymbol.value)
  if (added) {
    console.log(`自动将 ${selectedGridSymbol.value} 添加到白名单`)
  }
}


// 网格交易区域点击时的预加载
const onGridTradingClick = async () => {
  // 只有当用户已经选择网格交易策略，且没有币种数据时，才预加载
  if (props.conditions.grid_trading_enabled && availableSymbols.value.length === 0 && !loadingSymbols.value) {
    console.log('用户点击网格交易区域，开始预加载网格交易币种...')
    await loadGridTradingSymbols()
  }
}

// 策略相关的排序选项（动态生成）
const sortOptions = computed(() => {
  if (props.conditions.grid_trading_enabled) {
    // 网格交易策略的排序选项
    return [
      { key: 'grid_overall', label: '网格适应性' },
      { key: 'grid_volatility', label: '波动率' },
      { key: 'grid_liquidity', label: '流动性' },
      { key: 'grid_stability', label: '稳定性' },
      { key: 'market_cap', label: '市值' }
    ]
  } else {
    // 其他策略的排序选项（暂时保持通用排序）
    return [
      { key: 'market_cap', label: '市值' },
      { key: 'price_change', label: '涨跌幅' },
      { key: 'volume', label: '成交量' },
      { key: 'alphabetical', label: '字母' }
    ]
  }
})

// 初始化所有策略相关参数的函数
function initializeStrategyParams(conditions) {
  // 初始化合约涨幅排名过滤参数
  if (typeof conditions.futures_price_rank_filter_enabled === 'undefined') {
    conditions.futures_price_rank_filter_enabled = false
  }
  if (conditions.max_futures_price_rank === null || conditions.max_futures_price_rank === undefined || conditions.max_futures_price_rank === 0) {
    conditions.max_futures_price_rank = 10
  }

  // 初始化合约涨幅开空策略参数
  if (typeof conditions.futures_price_short_strategy_enabled === 'undefined') {
    conditions.futures_price_short_strategy_enabled = false
  }
  if (conditions.futures_price_short_min_market_cap === null || conditions.futures_price_short_min_market_cap === undefined) {
    conditions.futures_price_short_min_market_cap = 1000 // 默认1000万市值
  }
  if (conditions.futures_price_short_max_rank === null || conditions.futures_price_short_max_rank === undefined || conditions.futures_price_short_max_rank === 0) {
    conditions.futures_price_short_max_rank = 5
  }
  if (typeof conditions.futures_price_short_min_funding_rate === 'undefined') {
    conditions.futures_price_short_min_funding_rate = -0.005
  }
  if (conditions.futures_price_short_leverage === null || conditions.futures_price_short_leverage === undefined || conditions.futures_price_short_leverage === 0) {
    conditions.futures_price_short_leverage = 3.0
  }

  // 初始化均值回归基础参数
  if (typeof conditions.mr_min_reversion_strength === 'undefined' || conditions.mr_min_reversion_strength === null ||
      conditions.mr_min_reversion_strength === 0 || conditions.mr_min_reversion_strength < 0.1 || conditions.mr_min_reversion_strength > 1.0) {
    conditions.mr_min_reversion_strength = 0.15 // 默认回归强度0.15，确保在0.1-1.0范围内
  }

  // 初始化均值回归风险管理参数
  if (typeof conditions.mr_stop_loss_multiplier === 'undefined' || conditions.mr_stop_loss_multiplier === null || conditions.mr_stop_loss_multiplier === 0) {
    conditions.mr_stop_loss_multiplier = 2.5
  }
  if (typeof conditions.mr_take_profit_multiplier === 'undefined' || conditions.mr_take_profit_multiplier === null || conditions.mr_take_profit_multiplier === 0) {
    conditions.mr_take_profit_multiplier = 1.12
  }
  if (typeof conditions.mr_max_position_size === 'undefined' || conditions.mr_max_position_size === null || conditions.mr_max_position_size === 0) {
    conditions.mr_max_position_size = 0.025
  }
  if (typeof conditions.mr_max_hold_hours === 'undefined' || conditions.mr_max_hold_hours === null || conditions.mr_max_hold_hours === 0) {
    conditions.mr_max_hold_hours = 36
  }
}

// 监听条件变化
watch(() => props.conditions, (newConditions, oldConditions) => {
  // 确保symbol_whitelist始终是数组
  if (!newConditions.symbol_whitelist || !Array.isArray(newConditions.symbol_whitelist)) {
    console.warn('symbol_whitelist不是数组，重置为空数组:', newConditions.symbol_whitelist)
    newConditions.symbol_whitelist = []
  }

  // 初始化所有策略相关参数
  initializeStrategyParams(newConditions)

  // 初始化时或数据加载时，如果网格交易被禁用，清空网格参数
  if (!newConditions.grid_trading_enabled) {
    newConditions.grid_upper_price = 0
    newConditions.grid_lower_price = 0
    newConditions.grid_levels = 0
    newConditions.grid_profit_percent = 0
    newConditions.grid_investment_amount = 0
    newConditions.grid_rebalance_enabled = false
    newConditions.grid_stop_loss_enabled = false
    newConditions.grid_stop_loss_percent = 0
  }

  // 初始化合约涨幅排名过滤参数
  if (typeof newConditions.futures_price_rank_filter_enabled === 'undefined') {
    newConditions.futures_price_rank_filter_enabled = false
  }
  if (!newConditions.max_futures_price_rank) {
    newConditions.max_futures_price_rank = 10
  }

  // 初始化时或数据加载时，如果合约涨幅开空策略被禁用，清空相关参数
  if (!newConditions.futures_price_short_strategy_enabled) {
    newConditions.futures_price_short_min_market_cap = 1000 // 默认1000万市值
    newConditions.futures_price_short_max_rank = 5
    newConditions.futures_price_short_min_funding_rate = -0.005
    newConditions.futures_price_short_leverage = 3.0
  }

  // 如果网格交易策略状态发生变化
  if (oldConditions && newConditions.grid_trading_enabled !== oldConditions.grid_trading_enabled) {
    console.log('检测到网格交易策略状态变化:', newConditions.grid_trading_enabled)
    if (newConditions.grid_trading_enabled) {
      // 用户选择启用网格交易策略，立即加载网格交易专用币种
      console.log('用户选择启用网格交易策略，开始加载网格交易币种...')
      loadGridTradingSymbols().catch(error => {
        console.warn('加载网格交易币种失败:', error.message)
      })
    } else {
      // 关闭网格交易，清空币种列表和网格参数，避免显示不相关的币种和验证错误
      availableSymbols.value = []
      filteredSymbols.value = []
      selectedGridSymbol.value = ''
      // 清空网格参数，避免在禁用状态下仍然显示验证错误
      newConditions.grid_upper_price = 0
      newConditions.grid_lower_price = 0
      newConditions.grid_levels = 0
      newConditions.grid_profit_percent = 0
      newConditions.grid_investment_amount = 0
      newConditions.grid_rebalance_enabled = false
      newConditions.grid_stop_loss_enabled = false
      newConditions.grid_stop_loss_percent = 0
      console.log('网格交易策略已关闭，清空币种数据和网格参数')
    }
  }

  // 如果均值回归子模式发生变化，更新风险管理参数
  if (oldConditions && newConditions.mean_reversion_sub_mode !== oldConditions.mean_reversion_sub_mode) {
    console.log('检测到均值回归子模式变化:', newConditions.mean_reversion_sub_mode)
    updateMRRiskParamsForSubMode(newConditions)
  }


  emit('update:conditions', newConditions)
}, { deep: true })

// 监听选中币种变化 - 自动管理白名单
watch(selectedGridSymbol, (newSymbol, oldSymbol) => {
  if (newSymbol && newSymbol !== oldSymbol) {
    console.log('检测到币种选择变化:', newSymbol)
    // 确保symbol_whitelist是数组后再添加
    if (!props.conditions.symbol_whitelist || !Array.isArray(props.conditions.symbol_whitelist)) {
      props.conditions.symbol_whitelist = []
    }
    const added = addSymbolToWhitelist(newSymbol)
    if (added) {
      showSuccessMessage(`已自动将 ${newSymbol} 添加到白名单`)
    }
  }
})

// ===========================================
// 数据加载策略说明：
// 1. 组件初始化时不预加载任何币种数据
// 2. 只有当用户明确选择网格交易策略时，才加载网格交易专用币种
// 3. 其他策略不进行预加载，用户需要手动触发
// ===========================================

// 切换币种下拉框显示状态
const toggleSymbolDropdown = async () => {
  const wasOpen = showSymbolDropdown.value
  showSymbolDropdown.value = !showSymbolDropdown.value

  // 如果是打开下拉框，且没有可用币种数据，自动加载对应策略的币种
  if (!wasOpen && showSymbolDropdown.value && availableSymbols.value.length === 0) {
    console.log('下拉框打开时检测到无币种数据，开始自动加载...')
    await loadAvailableSymbols()
  }
}

const filterSymbols = () => {
  // 简单的币种过滤逻辑
  if (availableSymbols.value.length === 0) {
    filteredSymbols.value = []
    return
  }

  const query = symbolSearchQuery.value.toLowerCase()
  if (!query) {
    filteredSymbols.value = [...availableSymbols.value]
  } else {
    filteredSymbols.value = availableSymbols.value.filter(symbol =>
      symbol.symbol.toLowerCase().includes(query)
    )
  }

  // 过滤后重新排序
  sortSymbols()
}

const setSortOption = (key) => {
  currentSort.value = key
  // 使用统一的排序函数
  sortSymbols()
}

const selectSymbol = (symbol) => {
  selectedGridSymbol.value = symbol
  showSymbolDropdown.value = false
  // 添加到最近使用
  if (!recentSymbols.value.includes(symbol)) {
    recentSymbols.value.unshift(symbol)
    if (recentSymbols.value.length > 5) {
      recentSymbols.value = recentSymbols.value.slice(0, 5)
    }
  }
}



const getSelectedSymbolDisplay = () => {
  // 实现获取选中币种显示逻辑
  return selectedGridSymbol.value || ''
}

const formatPrice = (price) => {
  return price.toFixed(4)
}

const formatPercent = (percent) => {
  return (percent >= 0 ? '+' : '') + percent.toFixed(2) + '%'
}

// 获取评分显示的颜色类名
const getScoreClass = (score) => {
  if (score >= 0.8) return 'score-excellent'  // 优秀
  if (score >= 0.6) return 'score-good'       // 良好
  if (score >= 0.4) return 'score-fair'       // 一般
  return 'score-poor'                         // 较差
}

const formatVolume = (volume) => {
  if (volume >= 1000000) {
    return (volume / 1000000).toFixed(1) + 'M'
  } else if (volume >= 1000) {
    return (volume / 1000).toFixed(1) + 'K'
  }
  return volume.toString()
}

const formatMarketCap = (marketCap) => {
  if (marketCap >= 1000000000) {
    return (marketCap / 1000000000).toFixed(1) + 'B'
  } else if (marketCap >= 1000000) {
    return (marketCap / 1000000).toFixed(1) + 'M'
  }
  return marketCap.toString()
}

const generateGridLevels = () => {
  // 实现网格层级生成逻辑
  if (!props.conditions.grid_upper_price || !props.conditions.grid_lower_price || !props.conditions.grid_levels) {
    return []
  }

  const levels = []
  const range = props.conditions.grid_upper_price - props.conditions.grid_lower_price
  const spacing = range / (props.conditions.grid_levels - 1)

  for (let i = 0; i < props.conditions.grid_levels; i++) {
    const price = props.conditions.grid_lower_price + (spacing * i)
    const position = (i / (props.conditions.grid_levels - 1)) * 100
    levels.push({
      price,
      position,
      isBuy: i === 0,
      isSell: i === props.conditions.grid_levels - 1
    })
  }

  return levels
}

const getPriceRangePercent = () => {
  if (props.conditions.grid_upper_price && props.conditions.grid_lower_price) {
    return (((props.conditions.grid_upper_price - props.conditions.grid_lower_price) / props.conditions.grid_lower_price) * 100).toFixed(2)
  }
  return '0.00'
}

const getGridSpacing = () => {
  if (props.conditions.grid_upper_price && props.conditions.grid_lower_price && props.conditions.grid_levels) {
    return (props.conditions.grid_upper_price - props.conditions.grid_lower_price) / (props.conditions.grid_levels - 1)
  }
  return 0
}

const getGridSpacingPercent = () => {
  const spacing = getGridSpacing()
  if (spacing && props.conditions.grid_lower_price) {
    return ((spacing / props.conditions.grid_lower_price) * 100).toFixed(2)
  }
  return '0.00'
}

const calculateExpectedProfit = () => {
  // 实现预期收益计算逻辑
  if (props.conditions.grid_investment_amount && props.conditions.grid_profit_percent) {
    const profit = (props.conditions.grid_investment_amount * props.conditions.grid_profit_percent) / 100
    return profit.toFixed(2) + ' USDT'
  }
  return '0.00 USDT'
}

const calculateMaxPotentialProfit = () => {
  // 实现最大潜在收益计算逻辑
  if (props.conditions.grid_investment_amount && props.conditions.grid_levels) {
    const maxProfit = (props.conditions.grid_investment_amount * props.conditions.grid_levels * props.conditions.grid_profit_percent) / 100
    return maxProfit.toFixed(2) + ' USDT'
  }
  return '0.00 USDT'
}

const getGridValidationMessages = () => {
  // 实现参数验证逻辑
  const messages = []

  // 只有当网格交易策略启用时才进行验证
  if (!props.conditions.grid_trading_enabled) {
    return messages
  }

  if (props.conditions.grid_upper_price && props.conditions.grid_lower_price) {
    if (props.conditions.grid_upper_price <= props.conditions.grid_lower_price) {
      messages.push({
        text: '上限价格必须高于下限价格',
        type: 'error',
        icon: '❌'
      })
    } else {
      const rangePercent = ((props.conditions.grid_upper_price - props.conditions.grid_lower_price) / props.conditions.grid_lower_price) * 100
      if (rangePercent < 1) {
        messages.push({
          text: '价格区间过小，建议至少1%的波动区间',
          type: 'warning',
          icon: '⚠️'
        })
      } else if (rangePercent > 50) {
        messages.push({
          text: '价格区间过大，风险较高',
          type: 'warning',
          icon: '⚠️'
        })
      } else {
        messages.push({
          text: '参数设置合理',
          type: 'success',
          icon: '✅'
        })
      }
    }
  }

  return messages
}

const getParameterSuggestions = () => {
  // 实现参数建议逻辑
  const suggestions = []

  // 只有当网格交易策略启用时才提供建议
  if (!props.conditions.grid_trading_enabled) {
    return suggestions
  }

  if (props.conditions.grid_upper_price && props.conditions.grid_lower_price) {
    const rangePercent = ((props.conditions.grid_upper_price - props.conditions.grid_lower_price) / props.conditions.grid_lower_price) * 100

    if (rangePercent < 2) {
      suggestions.push({
        id: 'expand_range',
        icon: '📈',
        text: '扩大价格区间以获得更多交易机会',
        action: () => {
          // 扩大区间10%
          const currentRange = props.conditions.grid_upper_price - props.conditions.grid_lower_price
          const expansion = currentRange * 0.1
          emit('update:conditions', {
            ...props.conditions,
            grid_lower_price: props.conditions.grid_lower_price - expansion / 2,
            grid_upper_price: props.conditions.grid_upper_price + expansion / 2
          })
        }
      })
    } else if (rangePercent > 30) {
      suggestions.push({
        id: 'reduce_range',
        icon: '📉',
        text: '缩小价格区间以降低风险',
        action: () => {
          // 缩小区间10%
          const currentRange = props.conditions.grid_upper_price - props.conditions.grid_lower_price
          const reduction = currentRange * 0.1
          emit('update:conditions', {
            ...props.conditions,
            grid_lower_price: props.conditions.grid_lower_price + reduction / 2,
            grid_upper_price: props.conditions.grid_upper_price - reduction / 2
          })
        }
      })
    }
  }

  return suggestions
}

const applySuggestion = (suggestion) => {
  // 实现应用建议逻辑
  if (suggestion.action) {
    suggestion.action()
  }
}

// 关闭币种下拉菜单
const closeSymbolDropdown = () => {
  showSymbolDropdown.value = false
}

// 网格交易相关方法

// ===========================================
// 币种加载相关方法
// ===========================================

// 加载网格交易专用币种列表
// 专门为网格交易策略筛选符合条件的币种
const loadGridTradingSymbols = async () => {
  loadingSymbols.value = true
  try {
    console.log('加载网格交易专用币种列表...')
    const data = await api.getGridTradingSymbols({ kind: 'spot', limit: 50, page: 1 })

    if (data.symbols && data.symbols.length > 0) {
      availableSymbols.value = data.symbols
      filteredSymbols.value = [...data.symbols]
      canLoadMoreSymbols.value = data.symbols.length >= 50
      symbolsPage.value = 1
      sortSymbols()
    } else {
      await loadFallbackSymbols()
    }
  } catch (error) {
    console.error('加载网格交易币种失败:', error)
    await loadFallbackSymbols()
  } finally {
    loadingSymbols.value = false
  }
}

// 加载通用市值筛选币种列表
// 为其他策略提供通用的市值排序币种列表
const loadGeneralSymbols = async () => {
  loadingSymbols.value = true
  try {
    console.log('加载通用市值筛选币种列表...')
    const data = await api.getSymbolsWithMarketCap({ kind: 'spot', limit: 50, page: 1 })

    if (data.symbols && data.symbols.length > 0) {
      availableSymbols.value = data.symbols
      filteredSymbols.value = [...data.symbols]
      canLoadMoreSymbols.value = data.symbols.length >= 50
      symbolsPage.value = 1
      sortSymbols()
    } else {
      await loadFallbackSymbols()
    }
  } catch (error) {
    console.error('加载通用币种失败:', error)
    await loadFallbackSymbols()
  } finally {
    loadingSymbols.value = false
  }
}

// 统一的加载入口（根据当前策略选择对应的专用加载函数）
const loadAvailableSymbols = async () => {
  if (props.conditions.grid_trading_enabled) {
    await loadGridTradingSymbols()
  } else {
    await loadGeneralSymbols()
  }
}

// 加载备用币种列表
const loadFallbackSymbols = async () => {
  console.log('开始加载备用币种列表...')
  try {
    const data = await api.getAvailableSymbols({ kind: 'spot', limit: 100 })
    console.log('备用API响应:', data)

    if (data.success && data.data && data.data.length > 0) {
      console.log('使用备用API数据:', data.data.length, '个币种')
      // 将简单符号转换为丰富格式
      availableSymbols.value = data.data.map(symbol => ({
        symbol: symbol,
        current_price: 0,
        price_change_percent: 0,
        volume_24h: 0,
        market_cap_usd: 0
      }))
      filteredSymbols.value = [...availableSymbols.value]
    } else {
      console.log('备用API也失败，使用硬编码默认列表')
      // 使用默认币种列表
      setDefaultSymbols()
    }
  } catch (error) {
    console.error('备用API也失败，使用硬编码默认列表:', error)
    // 使用默认币种列表
    setDefaultSymbols()
  }
}

// 设置默认币种列表
const setDefaultSymbols = () => {
  console.log('设置默认币种列表')
  const defaultSymbols = [
    'BTCUSDT', 'ETHUSDT', 'BNBUSDT', 'ADAUSDT', 'SOLUSDT',
    'DOTUSDT', 'LINKUSDT', 'LTCUSDT', 'XRPUSDT', 'DOGEUSDT',
    'AVAXUSDT', 'MATICUSDT', 'ALGOUSDT', 'VETUSDT', 'ICPUSDT'
  ]

  availableSymbols.value = defaultSymbols.map(symbol => ({
    symbol: symbol,
    current_price: Math.random() * 100, // 模拟价格
    price_change_percent: (Math.random() - 0.5) * 10, // 模拟涨跌幅
    volume_24h: Math.random() * 1000000, // 模拟成交量
    market_cap_usd: Math.random() * 10000000000 // 模拟市值
  }))

  filteredSymbols.value = [...availableSymbols.value]
  console.log('默认币种列表设置完成:', availableSymbols.value.length, '个币种')
}


// 过滤币种

// 设置排序选项

// 排序币种
const sortSymbols = () => {
  filteredSymbols.value.sort((a, b) => {
    switch (currentSort.value) {
      // 网格交易专用排序
      case 'grid_overall':
        // 按综合网格适应性评分排序（从高到低）
        return (b.grid_overall_score || 0) - (a.grid_overall_score || 0)
      case 'grid_volatility':
        // 按波动率评分排序（网格交易需要适中波动率）
        return (b.grid_volatility_score || 0) - (a.grid_volatility_score || 0)
      case 'grid_liquidity':
        // 按流动性评分排序（从高到低）
        return (b.grid_liquidity_score || 0) - (a.grid_liquidity_score || 0)
      case 'grid_stability':
        // 按稳定性评分排序（从高到低）
        return (b.grid_stability_score || 0) - (a.grid_stability_score || 0)

      // 通用排序（保持兼容性）
      case 'market_cap':
        return (b.market_cap_usd || 0) - (a.market_cap_usd || 0)
      case 'price_change':
        return (b.price_change_percent || 0) - (a.price_change_percent || 0)
      case 'volume':
        return (b.volume_24h || 0) - (a.volume_24h || 0)
      case 'alphabetical':
        return a.symbol.localeCompare(b.symbol)
      default:
        // 默认排序：如果是网格交易，按综合评分；否则按市值
        if (props.conditions.grid_trading_enabled) {
          return (b.grid_overall_score || 0) - (a.grid_overall_score || 0)
        } else {
          return (b.market_cap_usd || 0) - (a.market_cap_usd || 0)
        }
    }
  })
}

// 加载更多网格交易币种
// 分页加载网格交易专用币种列表
const loadMoreGridTradingSymbols = async () => {
  if (loadingSymbols.value || !canLoadMoreSymbols.value) return

  loadingSymbols.value = true
  symbolsPage.value++

  try {
    const data = await api.getGridTradingSymbols({ kind: 'spot', limit: 50, page: symbolsPage.value })

    if (data.symbols && data.symbols.length > 0) {
      availableSymbols.value.push(...data.symbols)
      filterSymbols()
      canLoadMoreSymbols.value = data.symbols.length >= 50
    } else {
      canLoadMoreSymbols.value = false
    }
  } catch (error) {
    console.error('加载更多网格交易币种失败:', error)
    canLoadMoreSymbols.value = false
  } finally {
    loadingSymbols.value = false
  }
}

// 加载更多通用币种
// 分页加载通用市值筛选币种列表
const loadMoreGeneralSymbols = async () => {
  if (loadingSymbols.value || !canLoadMoreSymbols.value) return

  loadingSymbols.value = true
  symbolsPage.value++

  try {
    const data = await api.getSymbolsWithMarketCap({ kind: 'spot', limit: 50, page: symbolsPage.value })

    if (data.symbols && data.symbols.length > 0) {
      availableSymbols.value.push(...data.symbols)
      filterSymbols()
      canLoadMoreSymbols.value = data.symbols.length >= 50
    } else {
      canLoadMoreSymbols.value = false
    }
  } catch (error) {
    console.error('加载更多通用币种失败:', error)
    canLoadMoreSymbols.value = false
  } finally {
    loadingSymbols.value = false
  }
}

// 统一的加载更多入口（根据当前策略选择对应的专用加载函数）
const loadMoreSymbols = async () => {
  if (props.conditions.grid_trading_enabled) {
    await loadMoreGridTradingSymbols()
  } else {
    await loadMoreGeneralSymbols()
  }
}

// 添加到最近使用记录
const addToRecentSymbols = (symbol) => {
  const index = recentSymbols.value.indexOf(symbol)
  if (index > -1) {
    recentSymbols.value.splice(index, 1)
  }
  recentSymbols.value.unshift(symbol)
  if (recentSymbols.value.length > 5) {
    recentSymbols.value = recentSymbols.value.slice(0, 5)
  }
  // 保存到本地存储
  localStorage.setItem('recentGridSymbols', JSON.stringify(recentSymbols.value))
}

// 加载最近使用记录
const loadRecentSymbols = () => {
  try {
    const stored = localStorage.getItem('recentGridSymbols')
    if (stored) {
      recentSymbols.value = JSON.parse(stored)
    }
  } catch (error) {
    console.error('加载最近使用币种失败:', error)
  }
}

// 获取选中币种的显示文本

// 格式化价格

// 格式化百分比

// 格式化成交量

// 格式化市值

// 分析币种并自动填充网格参数
const analyzeSymbolForGrid = async () => {
  if (!selectedGridSymbol.value) return

  analyzingSymbol.value = true
  symbolAnalysis.value = null

  try {
    // 获取币种的历史价格和技术指标
    const data = await api.analyzeSymbolForGridTrading(selectedGridSymbol.value)

    if (data.success && data.data) {
      const analysis = data.data
      symbolAnalysis.value = {
        currentPrice: analysis.current_price || 0,
        volatility: analysis.volatility || 0.05,
        recommendedLevels: analysis.recommended_levels || 10,
        recommendedLower: analysis.recommended_lower || 0,
        recommendedUpper: analysis.recommended_upper || 0,
        historicalPrices: analysis.historical_prices || []
      }

      // 自动填充网格参数
      autoFillGridParameters(analysis)
    }
  } catch (error) {
    console.error('分析币种失败:', error)
    alert('分析币种失败，请稍后重试')
  } finally {
    analyzingSymbol.value = false
  }
}

// 自动填充网格参数
const autoFillGridParameters = (analysis) => {
  if (!analysis) return

  // 基于分析结果填充参数
  props.conditions.grid_lower_price = Math.max(0, analysis.recommended_lower || 0)
  props.conditions.grid_upper_price = Math.max(0, analysis.recommended_upper || 0)
  props.conditions.grid_levels = Math.max(2, analysis.recommended_levels || 10)

  // 根据波动率调整利润百分比（网格间距百分比）
  const volatility = analysis.volatility || 0.05
  const priceRange = analysis.recommended_upper - analysis.recommended_lower
  const avgPrice = (analysis.recommended_upper + analysis.recommended_lower) / 2

  if (priceRange > 0 && avgPrice > 0) {
    // 基于价格范围计算合适的利润率
    const rangePercent = (priceRange / avgPrice) * 100
    if (rangePercent > 20) {
      props.conditions.grid_profit_percent = 0.3 // 大幅震荡，降低利润率
    } else if (rangePercent > 10) {
      props.conditions.grid_profit_percent = 0.5 // 中等震荡
    } else if (rangePercent > 5) {
      props.conditions.grid_profit_percent = 1.0 // 小幅震荡
    } else {
      props.conditions.grid_profit_percent = 1.5 // 稳定震荡
    }
  } else {
    // 基于波动率的默认设置
    if (volatility > 0.1) {
      props.conditions.grid_profit_percent = 0.5
    } else if (volatility > 0.05) {
      props.conditions.grid_profit_percent = 1.0
    } else {
      props.conditions.grid_profit_percent = 1.5
    }
  }

  // 根据币种价格设置合适的投资金额
  const currentPrice = analysis.current_price || 1
  if (currentPrice > 1000) {
    props.conditions.grid_investment_amount = 100 // 高价币种，减少投资
  } else if (currentPrice > 100) {
    props.conditions.grid_investment_amount = 500 // 中价币种
  } else if (currentPrice > 10) {
    props.conditions.grid_investment_amount = 1000 // 普通币种
  } else {
    props.conditions.grid_investment_amount = 2000 // 低价币种，可增加投资
  }

  // 根据波动率和价格范围调整止损百分比
  const rangePercent = priceRange > 0 && avgPrice > 0 ? (priceRange / avgPrice) * 100 : 10
  if (volatility > 0.15 || rangePercent > 30) {
    props.conditions.grid_stop_loss_percent = 20.0 // 高风险，增加止损
  } else if (volatility > 0.1 || rangePercent > 15) {
    props.conditions.grid_stop_loss_percent = 15.0 // 中高风险
  } else if (volatility > 0.05 || rangePercent > 8) {
    props.conditions.grid_stop_loss_percent = 10.0 // 中等风险
  } else {
    props.conditions.grid_stop_loss_percent = 5.0 // 低风险
  }

  // 显示成功提示
  showSuccessMessage(`已自动填充${analysis.symbol || selectedGridSymbol.value}的网格参数`)
}

// 计算预期收益

// 计算潜在最大收益


// 获取网格间距



// 获取参数优化建议


// 获取网格参数验证消息

// 显示成功消息的辅助函数
const showSuccessMessage = (message) => {
  // 这里可以触发一个成功消息的显示
  console.log('Success:', message)
}

// 获取均线交叉信号的显示文本
const getMACrossSignalText = (signal) => {
  const signalMap = {
    'GOLDEN_CROSS': '金叉买入',
    'DEATH_CROSS': '死叉卖出',
    'BOTH': '双向交易'
  }
  return signalMap[signal] || signal
}

// 获取均线趋势方向的显示文本
const getMATrendDirectionText = (direction) => {
  const directionMap = {
    'UP': '上涨趋势',
    'DOWN': '下跌趋势',
    'BOTH': '双向趋势'
  }
  return directionMap[direction] || direction
}

// 获取均值回归启用的指标文本
const getMREnabledIndicators = (conditions) => {
  const indicators = []
  if (conditions.mr_bollinger_bands_enabled) {
    indicators.push('布林带')
  }
  if (conditions.mr_rsi_enabled) {
    indicators.push('RSI')
  }
  if (conditions.mr_price_channel_enabled) {
    indicators.push('价格通道')
  }

  if (indicators.length === 0) {
    return '无指标'
  }

  return indicators.join('+')
}

// 获取增强模式的优化参数显示
const getOptimizedParamDisplay = (conditions, paramType) => {
  if (conditions.mean_reversion_mode !== 'enhanced') {
    return ''
  }

  switch (paramType) {
    case 'period':
      if (conditions.mean_reversion_sub_mode === 'adaptive') {
        return ' (已优化为20天 - 平衡周期)'
      } else if (conditions.mean_reversion_sub_mode === 'aggressive') {
        return ' (已优化为12天 - 快速响应)'
      }
      break
    case 'bollinger':
      if (conditions.mean_reversion_sub_mode === 'adaptive') {
        return ' (已优化为2.0倍 - 标准区间)'
      } else if (conditions.mean_reversion_sub_mode === 'aggressive') {
        return ' (已优化为1.5倍 - 灵敏区间)'
      }
      break
    case 'rsi':
      if (conditions.mean_reversion_sub_mode === 'adaptive') {
        return ' (已优化为超卖25/超买75 - 扩大范围)'
      } else if (conditions.mean_reversion_sub_mode === 'aggressive') {
        return ' (已优化为超卖20/超买80 - 激进范围)'
      }
      break
    case 'strength':
      if (conditions.mean_reversion_sub_mode === 'adaptive') {
        return ' (已优化为15% - 高频交易)'
      } else if (conditions.mean_reversion_sub_mode === 'conservative') {
        return ' (已优化为80% - 高质量)'
      } else if (conditions.mean_reversion_sub_mode === 'aggressive') {
        return ' (已优化为25% - 快速信号)'
      }
      break
  }
  return ''
}

// 根据均值回归子模式更新风险管理参数
const updateMRRiskParamsForSubMode = (conditions) => {
  const subMode = conditions.mean_reversion_sub_mode

  switch (subMode) {
    case 'conservative':
      // 保守模式：高止损倍数、低止盈倍数、小仓位、长持仓
      conditions.mr_stop_loss_multiplier = 3.0   // 3倍标准差，宽松止损
      conditions.mr_take_profit_multiplier = 1.06 // 6%止盈，保守收益目标
      conditions.mr_max_position_size = 0.015    // 1.5%仓位，严格控制风险
      conditions.mr_max_hold_hours = 48          // 48小时，等待合适时机
      break

    case 'aggressive':
      // 激进模式：低止损倍数、高止盈倍数、大仓位、短持仓
      conditions.mr_stop_loss_multiplier = 2.0   // 2倍标准差，严格止损
      conditions.mr_take_profit_multiplier = 1.20 // 20%止盈，激进收益目标
      conditions.mr_max_position_size = 0.04     // 4%仓位，充分利用资金
      conditions.mr_max_hold_hours = 12          // 12小时，快速进出
      break

    case 'adaptive':
    default:
      // 自适应模式：平衡参数，智能调整
      conditions.mr_stop_loss_multiplier = 2.5   // 2.5倍标准差，中等宽松
      conditions.mr_take_profit_multiplier = 1.12 // 12%止盈，中等收益目标
      conditions.mr_max_position_size = 0.025    // 2.5%仓位，中等仓位控制
      conditions.mr_max_hold_hours = 36          // 36小时，中等持仓时间
      break
  }
}

// 获取当前模式的描述信息
const getCurrentModeDescription = (conditions) => {
  if (conditions.mean_reversion_mode !== 'enhanced') {
    return '基础均值回归策略，适合传统交易需求'
  }

  if (conditions.mean_reversion_sub_mode === 'conservative') {
    return '保守模式：高确认度信号，严格风险控制，适合稳健投资者'
  } else if (conditions.mean_reversion_sub_mode === 'adaptive') {
    return '自适应模式：大数据优化参数，智能市场适应，高频稳定收益，推荐选择'
  } else {
    return '激进模式：高频交易，低确认度要求，适合活跃投资者'
  }
}

// 组件挂载时确保所有策略参数被初始化
onMounted(() => {
  console.log('TradingStrategies组件挂载，初始化策略参数')
  initializeStrategyParams(props.conditions)
  // 触发一次emit确保父组件收到更新
  emit('update:conditions', props.conditions)
})
</script>

<style scoped>
/* 交易策略组件的样式 */


/* 网格参数行样式 */
.grid-options-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 8px;
  flex-wrap: wrap;
}

/* 参数单位样式 */
.param-unit {
  font-size: 12px;
  color: #9ca3af;
  margin-left: 4px;
}

/* 加载指示器样式 */
.loading-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  color: #6b7280;
  font-size: 14px;
}

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid #e5e7eb;
  border-top: 2px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 网格评分样式 */
.grid-scores {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.grid-score-item {
  display: flex;
  align-items: center;
  gap: 2px;
  font-size: 11px;
}

.score-label {
  color: #6b7280;
  font-weight: 500;
}

.score-value {
  font-weight: 600;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
}

.score-value.score-excellent {
  background: #dcfce7;
  color: #166534;
}

.score-value.score-good {
  background: #dbeafe;
  color: #1e40af;
}

.score-value.score-fair {
  background: #fef3c7;
  color: #92400e;
}

.score-value.score-poor {
  background: #fee2e2;
  color: #991b1b;
}

/* 小型复选框样式 */
.condition-checkbox.small {
  margin-right: 4px;
}

.condition-checkbox.small .checkmark {
  width: 16px;
  height: 16px;
}

.condition-checkbox.small .checkmark::after {
  left: 5px;
  top: 2px;
  width: 4px;
  height: 8px;
}

/* 选项标签样式 */
.option-label {
  font-size: 13px;
  color: #374151;
  margin-right: 12px;
}

/* 止损配置样式 */
.stop-loss-config {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 均线策略信号模式样式 */
.mode-description {
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  padding: 12px;
  margin: 8px 0;
  border-left: 4px solid var(--primary-color);
  font-size: 13px;
  line-height: 1.5;
}

.mode-description strong {
  color: var(--primary-color);
  font-weight: 600;
}

.quality-mode {
  border-left-color: #10b981;
}

.quantity-mode {
  border-left-color: #f59e0b;
}

/* 均值回归策略样式 */
.mr-config {
  margin-top: 12px;
}

.mr-indicators {
  margin: 8px 0;
}

.mr-indicators .config-item {
  margin-bottom: 8px;
}

.sub-config {
  display: inline-block;
  margin-left: 8px;
  font-size: 12px;
  color: #6b7280;
}

.sub-config input {
  width: 50px;
  margin: 0 4px;
}

.unit {
  font-size: 12px;
  color: #9ca3af;
  margin-left: 4px;
}
.tab-pane {
  padding: 20px 0;
}

.tab-error {
  background: #fee;
  color: #c33;
  padding: 12px 16px;
  border-radius: 6px;
  margin-bottom: 20px;
  border: 1px solid #fcc;
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
  gap: 24px;
}

.config-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
  transition: all 0.2s;
}

.config-card:hover {
  border-color: #cbd5e1;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}

.config-card.clickable {
  cursor: pointer;
  border-color: #3b82f6;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.config-card.clickable:hover {
  border-color: #2563eb;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 0 0 1px #3b82f6;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #e2e8f0;
}


.section-description {
  font-size: 14px;
  color: #64748b;
  margin-bottom: 16px;
  font-style: italic;
}

.condition-card {
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 16px;
  transition: all 0.2s;
}

.condition-card:hover {
  border-color: #d1d5db;
  box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
}

.condition-card:last-child {
  margin-bottom: 0;
}

.condition-header {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.condition-checkbox {
  display: flex;
  align-items: center;
  margin-right: 12px;
  cursor: pointer;
  user-select: none;
}

.condition-checkbox input {
  display: none;
}

.checkmark {
  width: 20px;
  height: 20px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  margin-right: 8px;
  position: relative;
  transition: all 0.2s;
}

.condition-checkbox input:checked + .checkmark {
  background: #3b82f6;
  border-color: #3b82f6;
}

.condition-checkbox input:checked + .checkmark::after {
  content: '✓';
  position: absolute;
  top: -2px;
  left: 2px;
  color: white;
  font-size: 14px;
  font-weight: bold;
}

.condition-title {
  font-weight: 500;
  color: #374151;
}

.condition-description {
  font-size: 14px;
  color: #6b7280;
  line-height: 1.5;
}

.condition-sub-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  padding-left: 20px;
}

.condition-checkbox.small input {
  display: none;
}

.checkmark-small {
  width: 16px;
  height: 16px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  position: relative;
  transition: all 0.2s;
}

.condition-checkbox.small input:checked + .checkmark-small {
  background: #10b981;
  border-color: #10b981;
}

.condition-checkbox.small input:checked + .checkmark-small::after {
  content: '✓';
  position: absolute;
  top: -2px;
  left: 1px;
  color: white;
  font-size: 12px;
  font-weight: bold;
}

.condition-title.small {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.inline-input {
  width: 80px;
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  text-align: center;
}

.inline-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.inline-input.small {
  width: 60px;
}

.inline-input.tiny {
  width: 50px;
}

.inline-select {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  background: white;
}

.inline-select:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.inline-select.small {
  font-size: 12px;
  padding: 2px 4px;
}

.ma-config {
  margin-bottom: 12px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.config-item label {
  min-width: 80px;
  font-size: 14px;
  color: #555;
}

.config-item.mode-description {
  margin-top: 12px;
  margin-bottom: 12px;
}

.quality-mode, .quantity-mode {
  background: #f3f4f6;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.4;
}

.mr-config {
  margin-bottom: 12px;
}

.mr-indicators {
  margin: 12px 0;
}

.sub-config {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 8px;
  font-size: 13px;
  color: #666;
}

.unit {
  font-size: 12px;
  color: #888;
  margin-left: 4px;
}

/* 增强模式样式 */
.enhanced-settings {
  margin-top: 20px;
  padding: 16px;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}


.enhanced-title {
  margin: 0 0 12px 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  display: flex;
  align-items: center;
  gap: 6px;
}

.mr-risk-management {
  margin-top: 20px;
  padding: 16px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 2px solid #0ea5e9;
  border-radius: 12px;
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(14, 165, 233, 0.1);
}

.risk-title {
  margin: 0 0 16px 0;
  color: #0c4a6e;
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
  text-shadow: 0 1px 2px rgba(12, 74, 110, 0.1);
}

.conservative-mode, .aggressive-mode, .adaptive-mode {
  background: #f3f4f6;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.4;
  margin-top: 8px;
}

.conservative-mode {
  border-left: 4px solid #10b981;
}

.aggressive-mode {
  border-left: 4px solid #f59e0b;
}

.adaptive-mode {
  border-left: 4px solid #8b5cf6;
}

.conservative-mode strong, .aggressive-mode strong, .adaptive-mode strong {
  color: #1e293b;
  font-weight: 600;
}

.help-tooltip.small {
  font-size: 11px;
  margin-left: 4px;
  vertical-align: middle;
}

.help-tooltip.small::after {
  font-size: 11px;
  padding: 6px 8px;
}

.config-note {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-muted);
  font-style: italic;
  display: flex;
  align-items: flex-start;
  gap: 4px;
}

.config-note::before {
  content: '💡';
  font-size: 11px;
  flex-shrink: 0;
  margin-top: 1px;
}

.help-tooltip {
  position: relative;
  display: inline-block;
  margin-left: 6px;
  cursor: help;
  color: var(--text-secondary);
  font-size: 12px;
  vertical-align: middle;
}

.help-tooltip:hover::after {
  content: attr(data-tooltip);
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--text-primary);
  color: white;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  white-space: nowrap;
  z-index: 1000;
}


.symbol-selector-section {
  margin-bottom: 20px;
}

.grid-param-row {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  gap: 8px;
}

.grid-param-label {
  min-width: 80px;
  font-weight: 500;
  color: #374151;
}

.symbol-selector-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.symbol-dropdown-container {
  position: relative;
  flex: 1;
}

.symbol-dropdown-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: white;
  cursor: pointer;
  min-height: 36px;
}

.selected-symbol-display {
  flex: 1;
}

.selected-symbol-text {
  font-weight: 500;
  color: #1f2937;
}

.placeholder-text {
  color: #9ca3af;
}

.dropdown-arrow {
  transition: transform 0.2s;
  color: #6b7280;
}

.dropdown-arrow.rotated {
  transform: rotate(180deg);
}

.symbol-dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: white;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
  z-index: 1000;
  max-height: 300px;
  overflow: hidden;
}

.symbol-search-container {
  position: relative;
  padding: 8px;
  border-bottom: 1px solid #e5e7eb;
}

.symbol-search-input {
  width: 100%;
  padding: 6px 8px 6px 32px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
}

.search-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: #9ca3af;
  font-size: 14px;
}

.sort-options {
  display: flex;
  padding: 8px;
  border-bottom: 1px solid #e5e7eb;
  gap: 4px;
}

.sort-option-btn {
  padding: 4px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  background: white;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.sort-option-btn:hover {
  background: #f3f4f6;
}

.sort-option-btn.active {
  background: #3b82f6;
  color: white;
  border-color: #3b82f6;
}

.sort-hint {
  font-size: 12px;
  color: #059669;
  font-weight: 500;
  margin-bottom: 6px;
  padding: 4px 8px;
  background: #ecfdf5;
  border-radius: 4px;
  border: 1px solid #a7f3d0;
}

.symbol-list-container {
  max-height: 200px;
  overflow-y: auto;
}

.symbol-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid #f3f4f6;
  transition: background-color 0.2s;
}

.symbol-list-item:hover {
  background: #f9fafb;
}

.symbol-list-item.selected {
  background: #dbeafe;
}

.symbol-info {
  flex: 1;
}

.symbol-name {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
}

.symbol-code {
  font-weight: 500;
  color: #1f2937;
}

.symbol-price {
  font-size: 12px;
  color: #6b7280;
}

.symbol-details {
  display: flex;
  gap: 12px;
  font-size: 12px;
}

.price-change.positive {
  color: #dc2626;
}

.price-change.negative {
  color: #16a34a;
}

.volume {
  color: #6b7280;
}

.symbol-market-cap {
  font-size: 12px;
  color: #9ca3af;
  text-align: right;
}

.load-more-container {
  padding: 8px 12px;
  text-align: center;
}

.load-more-btn {
  padding: 6px 12px;
  background: #f3f4f6;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: background-color 0.2s;
}

.load-more-btn:hover:not(:disabled) {
  background: #e5e7eb;
}

.load-more-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.analyze-symbol-btn {
  padding: 8px 16px;
  background: #3b82f6;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: background-color 0.2s;
}

.analyze-symbol-btn:hover:not(:disabled) {
  background: #2563eb;
}

.analyze-symbol-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.recent-symbols-section {
  margin-top: 8px;
}

.recent-symbols-label {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}

.recent-symbols-list {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.recent-symbol-btn {
  padding: 2px 6px;
  background: #f3f4f6;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.recent-symbol-btn:hover {
  background: #e5e7eb;
}

.recent-symbol-btn.active {
  background: #dbeafe;
  border-color: #3b82f6;
  color: #1d4ed8;
}

.symbol-analysis-result {
  margin-top: 12px;
  padding: 12px;
  background: #f0f9ff;
  border: 1px solid #bae6fd;
  border-radius: 6px;
}

.analysis-summary h6 {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e40af;
}

.analysis-metrics {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.metric-label {
  color: #374151;
}

.metric-value {
  font-weight: 500;
  color: #1f2937;
}

.grid-config-section {
  margin-bottom: 16px;
}

.grid-price-input {
  flex: 1;
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
}

.price-separator {
  margin: 0 8px;
  color: #6b7280;
}

.price-unit {
  margin-left: 4px;
  color: #6b7280;
  font-size: 14px;
}

.grid-number-input {
  width: 80px;
  padding: 6px 8px;
  border: 1px solid #d1d5db;
  border-radius: 4px;
  font-size: 14px;
  text-align: center;
}

.grid-number-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.grid-number-input.small {
  width: 60px;
}

.param-unit {
  margin-left: 4px;
  color: #6b7280;
  font-size: 14px;
}

.grid-options-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 8px;
}

.option-label {
  font-size: 14px;
  color: #374151;
}

.stop-loss-config {
  display: flex;
  align-items: center;
  gap: 4px;
}

.grid-preview-section {
  margin-top: 16px;
  padding: 16px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.preview-title {
  margin: 0 0 12px 0;
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.grid-visualization {
  margin-bottom: 16px;
}

.grid-chart {
  display: flex;
  height: 120px;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background: #fafafa;
  position: relative;
}

.price-axis {
  width: 60px;
  padding: 8px 4px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  font-size: 10px;
  color: #6b7280;
  border-right: 1px solid #e5e7eb;
}

.grid-lines {
  position: relative;
  height: 100%;
  margin: 12px 0;
}

.grid-line {
  position: absolute;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  transition: all 0.3s ease;
}

.grid-line:hover .grid-price {
  opacity: 1;
  transform: translateX(0);
}

.grid-price {
  position: absolute;
  right: 95px;
  font-size: 12px;
  color: #0c4a6e;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.98) 0%, rgba(248, 250, 252, 0.98) 100%);
  padding: 6px 10px;
  border-radius: 8px;
  border: 2px solid #bae6fd;
  white-space: nowrap;
  font-weight: 700;
  opacity: 0;
  transform: translateX(15px);
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(14, 165, 233, 0.15);
  backdrop-filter: blur(8px);
}

.grid-marker {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  margin-left: 8px;
  border: 3px solid white;
  box-shadow: 0 3px 8px rgba(0, 0, 0, 0.25);
  transition: all 0.3s ease;
  position: relative;
}

.grid-marker::before {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: 50%;
  background: inherit;
  opacity: 0.3;
  z-index: -1;
}

.grid-marker:hover {
  transform: scale(1.3);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.buy-marker {
  background: linear-gradient(135deg, #10b981, #059669);
  border-color: #10b981;
}

.sell-marker {
  background: linear-gradient(135deg, #ef4444, #dc2626);
  border-color: #ef4444;
}

.grid-info {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  border: 1px solid #e0f2fe;
  border-radius: 10px;
  font-size: 12px;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.02);
}

.info-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.info-label {
  color: #374151;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
}


.info-value {
  color: #1e293b;
  font-weight: 700;
  font-size: 13px;
}

.preview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.preview-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 8px;
  background: white;
  border: 1px solid #e5e7eb;
  border-radius: 4px;
  font-size: 12px;
}

.preview-label {
  color: #6b7280;
}

.preview-value {
  font-weight: 500;
  color: #1f2937;
}

.validation-messages {
  margin-bottom: 12px;
}

.validation-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 4px;
  font-size: 12px;
  margin-bottom: 4px;
}

.validation-item.success {
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
}

.validation-item.warning {
  background: #fef3c7;
  color: #92400e;
  border: 1px solid #fde68a;
}

.validation-item.error {
  background: #fee2e2;
  color: #991b1b;
  border: 1px solid #fecaca;
}

.parameter-suggestions {
  margin-top: 24px;
  padding: 24px;
  background: linear-gradient(135deg, #f0f9ff 0%, #e0f2fe 100%);
  border: 2px solid #0ea5e9;
  border-radius: 12px;
  position: relative;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(14, 165, 233, 0.1);
  transition: all 0.3s ease;
}

.parameter-suggestions:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(14, 165, 233, 0.15);
}


.suggestions-title {
  margin: 0 0 20px 0;
  color: #0c4a6e;
  font-size: 18px;
  font-weight: 800;
  display: flex;
  align-items: center;
  gap: 10px;
  text-shadow: 0 1px 2px rgba(12, 74, 110, 0.1);
}


.suggestions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.suggestion-item {
  padding: 4px 0;
}

.suggestion-content {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.95) 0%, rgba(248, 250, 252, 0.95) 100%);
  border: 2px solid #bae6fd;
  border-radius: 12px;
  backdrop-filter: blur(8px);
  box-shadow: 0 4px 12px rgba(224, 231, 255, 0.3);
  transition: all 0.3s ease;
}

.suggestion-content:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 16px rgba(186, 230, 253, 0.4);
  border-color: #7dd3fc;
}

.suggestion-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.suggestion-text {
  flex: 1;
  font-size: 14px;
  color: #1e293b;
  font-weight: 500;
  line-height: 1.4;
}

.apply-btn {
  padding: 8px 16px;
  background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.apply-btn:hover {
  background: linear-gradient(135deg, #2563eb, #1e40af);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.apply-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.3);
}

@media (max-width: 768px) {
  .config-grid {
    grid-template-columns: 1fr;
  }

  .grid-config-section {
    padding: 16px;
    margin-top: 16px;
    border-radius: 12px;
  }

  .grid-param-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 16px;
    border-radius: 10px;
  }

  .grid-param-row:hover {
    transform: none;
    box-shadow: 0 4px 12px rgba(59, 130, 246, 0.08);
  }

  .grid-param-label {
    min-width: auto;
    width: 100%;
    font-size: 15px;
    font-weight: 700;
    color: #1f2937;
    margin-bottom: 4px;
  }

  .grid-price-input {
    width: 100%;
    padding: 14px 16px;
    font-size: 16px;
    border-radius: 8px;
  }

  .grid-number-input {
    width: 100%;
    padding: 14px 16px;
    font-size: 16px;
    border-radius: 8px;
  }

  .symbol-selector-wrapper {
    width: 100%;
  }

  .grid-options-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 16px;
    border-radius: 10px;
  }

  .grid-visualization {
    padding: 16px;
    border-radius: 12px;
    grid-template-columns: 1fr;
  }

  .grid-chart {
    flex-direction: column;
    gap: 16px;
    padding: 12px;
    border-radius: 8px;
  }

  .price-axis {
    width: 100%;
    height: 150px;
    border-radius: 8px;
  }

  .grid-lines {
    margin: 16px 0;
  }

  .grid-price {
    right: 100px;
    font-size: 12px;
  }

  .preview-grid {
    grid-template-columns: 1fr;
  }

  .grid-info {
    grid-template-columns: 1fr;
    gap: 8px;
  }

  .info-item {
    padding: 8px 12px;
    font-size: 12px;
  }

  .analysis-metrics {
    grid-template-columns: 1fr;
  }

  .parameter-suggestions {
    padding: 16px;
    margin-top: 16px;
  }

  .suggestions-title {
    font-size: 16px;
    margin-bottom: 16px;
  }

  .suggestion-content {
    padding: 10px 12px;
  }

  .suggestion-text {
    font-size: 13px;
  }

  .apply-btn {
    padding: 6px 12px;
    font-size: 12px;
  }

}
</style>