// 前端数据缓存工具（简单内存缓存）

const cache = new Map()
const DEFAULT_TTL = 5 * 60 * 1000 // 5分钟

/**
 * 生成缓存键
 * @param {string} key - 基础键
 * @param {object} params - 参数对象
 * @returns {string} 缓存键
 */
function generateKey(key, params = {}) {
  if (!params || Object.keys(params).length === 0) {
    return key
  }
  const paramStr = JSON.stringify(params)
  return `${key}:${paramStr}`
}

/**
 * 缓存项
 */
class CacheItem {
  constructor(data, ttl = DEFAULT_TTL) {
    this.data = data
    this.expiresAt = Date.now() + ttl
  }

  isExpired() {
    return Date.now() > this.expiresAt
  }
}

/**
 * 获取缓存数据
 * @param {string} key - 缓存键
 * @param {object} params - 参数对象
 * @returns {any|null} 缓存的数据，如果不存在或已过期则返回 null
 */
export function getCache(key, params = {}) {
  const cacheKey = generateKey(key, params)
  const item = cache.get(cacheKey)
  
  if (!item) {
    return null
  }
  
  if (item.isExpired()) {
    cache.delete(cacheKey)
    return null
  }
  
  return item.data
}

/**
 * 设置缓存数据
 * @param {string} key - 缓存键
 * @param {any} data - 要缓存的数据
 * @param {object} params - 参数对象
 * @param {number} ttl - 过期时间（毫秒）
 */
export function setCache(key, data, params = {}, ttl = DEFAULT_TTL) {
  const cacheKey = generateKey(key, params)
  cache.set(cacheKey, new CacheItem(data, ttl))
}

/**
 * 删除缓存
 * @param {string} key - 缓存键（支持前缀匹配）
 * @param {object} params - 参数对象
 */
export function deleteCache(key, params = {}) {
  if (params && Object.keys(params).length > 0) {
    const cacheKey = generateKey(key, params)
    cache.delete(cacheKey)
  } else {
    // 删除所有以 key 开头的缓存
    for (const [cacheKey] of cache) {
      if (cacheKey.startsWith(key)) {
        cache.delete(cacheKey)
      }
    }
  }
}

/**
 * 清空所有缓存
 */
export function clearCache() {
  cache.clear()
}

/**
 * 获取缓存统计信息
 * @returns {object} 缓存统计
 */
export function getCacheStats() {
  let expired = 0
  let valid = 0
  
  for (const [, item] of cache) {
    if (item.isExpired()) {
      expired++
    } else {
      valid++
    }
  }
  
  return {
    total: cache.size,
    valid,
    expired,
  }
}

/**
 * 清理过期缓存
 */
export function cleanExpiredCache() {
  for (const [key, item] of cache) {
    if (item.isExpired()) {
      cache.delete(key)
    }
  }
}

// 定期清理过期缓存（每5分钟）
if (typeof window !== 'undefined') {
  setInterval(cleanExpiredCache, 5 * 60 * 1000)
}

