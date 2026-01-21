/**
 * 交易策略执行引擎
 */

// 策略执行结果
export const StrategyResult = {
  SKIP: 'skip',           // 跳过，不执行交易
  BUY: 'buy',            // 执行买入
  SELL: 'sell',           // 执行卖出
  NO_OP: 'no_op'         // 无操作
}

/**
 * 执行策略判断
 * @param {Object} strategy - 策略对象
 * @param {Object} marketData - 市场数据
 * @param {string} symbol - 交易对
 * @returns {Object} { result: StrategyResult, reason: string, multiplier: number }
 */
export function executeStrategy(strategy, marketData, symbol) {
  if (!strategy || !strategy.conditions) {
    return { result: StrategyResult.NO_OP, reason: '无策略配置', multiplier: 1.0 }
  }

  const conditions = strategy.conditions

  // 检查是否有现货+合约条件
  if (conditions.spot_contract) {
    const hasSpot = checkHasSpot(symbol, marketData)
    const hasFutures = checkHasFutures(symbol, marketData)

    if (!hasSpot || !hasFutures) {
      return {
        result: StrategyResult.SKIP,
        reason: `需要现货+合约，但${!hasSpot ? '缺少现货' : ''}${!hasSpot && !hasFutures ? '+' : ''}${!hasFutures ? '缺少合约' : ''}`,
        multiplier: 1.0
      }
    }
  }

  // 获取币种的市场数据
  const coinData = getCoinData(symbol, marketData)
  if (!coinData) {
    return { result: StrategyResult.SKIP, reason: '无法获取币种数据', multiplier: 1.0 }
  }

  const marketCap = coinData.market_cap || 0
  const gainersRank = getGainersRank(symbol, marketData)

  // 检查不开空条件
  if (conditions.no_short_below_market_cap && marketCap < conditions.market_cap_limit_short * 10000) {
    return {
      result: StrategyResult.SKIP,
      reason: `市值${(marketCap/10000).toFixed(1)}万低于${conditions.market_cap_limit_short}万不开空`,
      multiplier: 1.0
    }
  }

  // 检查开空条件：涨幅前N且市值高于阈值
  if (conditions.short_on_gainers &&
      gainersRank <= conditions.gainers_rank_limit &&
      marketCap >= conditions.market_cap_limit_short * 10000) {
    return {
      result: StrategyResult.SELL,
      reason: `涨幅排名第${gainersRank}位，市值${(marketCap/10000).toFixed(1)}万，符合开空条件`,
      multiplier: conditions.short_multiplier
    }
  }

  // 检查开多条件：市值低于阈值且涨幅前N
  if (conditions.long_on_small_gainers &&
      marketCap < conditions.market_cap_limit_long * 10000 &&
      gainersRank <= conditions.gainers_rank_limit_long) {
    return {
      result: StrategyResult.BUY,
      reason: `市值${(marketCap/10000).toFixed(1)}万，涨幅排名第${gainersRank}位，符合开多条件`,
      multiplier: conditions.long_multiplier
    }
  }

  return { result: StrategyResult.NO_OP, reason: '不符合任何策略条件', multiplier: 1.0 }
}

/**
 * 检查是否有现货交易
 * @param {string} symbol - 交易对
 * @param {Object} marketData - 市场数据
 * @returns {boolean}
 */
function checkHasSpot(symbol, marketData) {
  // 简化实现，实际应该检查币安API或其他数据源
  // 这里假设大部分主流币种都有现货
  const baseSymbol = symbol.replace(/USDT$/, '')
  const majorCoins = ['BTC', 'ETH', 'BNB', 'ADA', 'SOL', 'DOT', 'DOGE', 'AVAX', 'LTC', 'MATIC']
  return majorCoins.includes(baseSymbol) || symbol.includes('USDT')
}

/**
 * 检查是否有合约交易
 * @param {string} symbol - 交易对
 * @param {Object} marketData - 市场数据
 * @returns {boolean}
 */
function checkHasFutures(symbol, marketData) {
  // 简化实现，实际应该检查币安期货API
  // 这里假设大部分主流币种都有合约
  const baseSymbol = symbol.replace(/USDT$/, '')
  const majorCoins = ['BTC', 'ETH', 'BNB', 'ADA', 'SOL', 'DOT', 'DOGE', 'AVAX', 'LTC', 'MATIC']
  return majorCoins.includes(baseSymbol)
}

/**
 * 获取币种数据
 * @param {string} symbol - 交易对
 * @param {Object} marketData - 市场数据
 * @returns {Object|null}
 */
function getCoinData(symbol, marketData) {
  // 简化实现，实际应该从市场数据中获取
  // 这里返回模拟数据
  const baseSymbol = symbol.replace(/USDT$/, '')

  // 模拟市值数据（实际应该从API获取）
  const mockMarketCaps = {
    'BTC': 1200000000000,   // 1.2万亿
    'ETH': 400000000000,    // 4000亿
    'BNB': 80000000000,     // 800亿
    'ADA': 20000000000,     // 200亿
    'SOL': 50000000000,     // 500亿
    'DOT': 15000000000,     // 150亿
    'DOGE': 25000000000,    // 250亿
    'AVAX': 10000000000,    // 100亿
    'LTC': 8000000000,      // 80亿
    'MATIC': 12000000000,   // 120亿
  }

  return {
    symbol: baseSymbol,
    market_cap: mockMarketCaps[baseSymbol] || 10000000000, // 默认100亿
    price_change_percent: Math.random() * 20 - 10 // -10% 到 +10%的随机涨幅
  }
}

/**
 * 获取涨幅排名
 * @param {string} symbol - 交易对
 * @param {Object} marketData - 市场数据
 * @returns {number} 排名（1开始）
 */
function getGainersRank(symbol, marketData) {
  // 从市场数据中查找排名
  if (marketData && Array.isArray(marketData)) {
    const item = marketData.find(item => item.symbol === symbol)
    if (item && item.rank) {
      return item.rank
    }
  }

  // 如果没有找到排名，返回默认值
  return 50 // 默认第50名
}

/**
 * 验证策略配置
 * @param {Object} strategy - 策略对象
 * @returns {Object} { valid: boolean, errors: string[] }
 */
export function validateStrategy(strategy) {
  const errors = []

  if (!strategy.name || strategy.name.trim().length === 0) {
    errors.push('策略名称不能为空')
  }

  const conditions = strategy.conditions

  if (conditions.market_cap_limit_short <= 0) {
    errors.push('不开空市值限制必须大于0')
  }

  if (conditions.gainers_rank_limit <= 0) {
    errors.push('开空涨幅排名限制必须大于0')
  }

  if (conditions.short_multiplier <= 0) {
    errors.push('开空倍数必须大于0')
  }

  if (conditions.market_cap_limit_long <= 0) {
    errors.push('开多市值限制必须大于0')
  }

  if (conditions.gainers_rank_limit_long <= 0) {
    errors.push('开多涨幅排名限制必须大于0')
  }

  if (conditions.long_multiplier <= 0) {
    errors.push('开多倍数必须大于0')
  }

  return {
    valid: errors.length === 0,
    errors
  }
}

/**
 * 获取默认策略配置
 * @returns {Object}
 */
export function getDefaultStrategy() {
  return {
    name: '',
    conditions: {
      spot_contract: true,
      no_short_below_market_cap: true,
      market_cap_limit_short: 5000,
      short_on_gainers: true,
      gainers_rank_limit: 12,
      short_multiplier: 1.0,
      long_on_small_gainers: true,
      market_cap_limit_long: 2500,
      gainers_rank_limit_long: 12,
      long_multiplier: 1.0
    }
  }
}
