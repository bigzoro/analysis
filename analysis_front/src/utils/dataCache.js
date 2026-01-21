// 前端智能缓存工具 - 支持多层缓存策略

// 缓存层级定义
const CACHE_LEVELS = {
  MEMORY: 'memory',           // 内存缓存 - 最快，页面刷新后丢失
  SESSION: 'session',         // sessionStorage - 标签页关闭后丢失
  LOCAL: 'local',            // localStorage - 长期保存
  INDEXEDDB: 'indexeddb'      // IndexedDB - 大容量，复杂数据
}

// 默认配置
const DEFAULT_CONFIG = {
  ttl: {
    [CACHE_LEVELS.MEMORY]: 5 * 60 * 1000,    // 5分钟
    [CACHE_LEVELS.SESSION]: 30 * 60 * 1000,  // 30分钟
    [CACHE_LEVELS.LOCAL]: 2 * 60 * 60 * 1000, // 2小时
    [CACHE_LEVELS.INDEXEDDB]: 24 * 60 * 60 * 1000 // 24小时
  },
  maxSize: {
    [CACHE_LEVELS.MEMORY]: 100,     // 内存缓存最大100条
    [CACHE_LEVELS.SESSION]: 500,    // sessionStorage最大500条
    [CACHE_LEVELS.LOCAL]: 1000,     // localStorage最大1000条
    [CACHE_LEVELS.INDEXEDDB]: 10000 // IndexedDB最大10000条
  },
  compression: true,  // 是否启用压缩
  encryption: false,  // 是否启用加密
  autoCleanup: true   // 是否自动清理过期数据
}

// 缓存实例
const cacheInstances = new Map()

// 缓存统计
const cacheStats = {
  hits: 0,
  misses: 0,
  sets: 0,
  deletes: 0,
  errors: 0,
  get hitRate() {
    const total = this.hits + this.misses
    return total > 0 ? (this.hits / total * 100).toFixed(1) : 0
  }
}

/**
 * 生成缓存键 - 支持参数排序确保一致性
 * @param {string} key - 基础键
 * @param {object} params - 参数对象
 * @returns {string} 缓存键
 */
function generateKey(key, params = {}) {
  if (!params || Object.keys(params).length === 0) {
    return key
  }

  // 对参数键进行排序，确保一致性
  const sortedKeys = Object.keys(params).sort()
  const sortedParams = {}

  for (const k of sortedKeys) {
    sortedParams[k] = params[k]
  }

  const paramStr = JSON.stringify(sortedParams)
  return `${key}:${paramStr}`
}

/**
 * 数据压缩
 * @param {any} data - 要压缩的数据
 * @returns {string} 压缩后的字符串
 */
function compress(data) {
  try {
    const jsonStr = JSON.stringify(data)
    // 简单压缩：移除多余空格
    return jsonStr.replace(/\s+/g, ' ').trim()
  } catch (e) {
    return JSON.stringify(data)
  }
}

/**
 * 数据解压
 * @param {string} compressedData - 压缩的数据
 * @returns {any} 解压后的数据
 */
function decompress(compressedData) {
  try {
    return JSON.parse(compressedData)
  } catch (e) {
    return null
  }
}

/**
 * 数据加密（基础实现）
 * @param {string} data - 要加密的数据
 * @returns {string} 加密后的数据
 */
function encrypt(data) {
  // 基础的Base64编码作为示例
  try {
    return btoa(encodeURIComponent(data))
  } catch (e) {
    return data
  }
}

/**
 * 数据解密
 * @param {string} encryptedData - 加密的数据
 * @returns {string} 解密后的数据
 */
function decrypt(encryptedData) {
  try {
    return decodeURIComponent(atob(encryptedData))
  } catch (e) {
    return encryptedData
  }
}

// ============================================================================
// 缓存层实现
// ============================================================================

/**
 * 内存缓存实现
 */
class MemoryCache {
  constructor(maxSize = 100) {
    this.cache = new Map()
    this.maxSize = maxSize
    this.accessOrder = [] // LRU访问顺序
  }

  get(key) {
    const item = this.cache.get(key)
    if (!item) return null

    if (item.isExpired()) {
      this.delete(key)
      return null
    }

    // 更新访问顺序
    this.updateAccessOrder(key)
    item.accessCount = (item.accessCount || 0) + 1

    return item.data
  }

  set(key, data, ttl = DEFAULT_CONFIG.ttl[CACHE_LEVELS.MEMORY]) {
    const item = {
      data,
      expiresAt: Date.now() + ttl,
      createdAt: Date.now(),
      accessCount: 0
    }

    // 如果键已存在，更新并调整顺序
    if (this.cache.has(key)) {
      this.cache.set(key, item)
      this.updateAccessOrder(key)
      return
    }

    // 检查容量限制
    if (this.cache.size >= this.maxSize) {
      this.evictLRU()
    }

    this.cache.set(key, item)
    this.accessOrder.push(key)
  }

  delete(key) {
    this.cache.delete(key)
    const index = this.accessOrder.indexOf(key)
    if (index > -1) {
      this.accessOrder.splice(index, 1)
    }
  }

  clear() {
    this.cache.clear()
    this.accessOrder = []
  }

  updateAccessOrder(key) {
    const index = this.accessOrder.indexOf(key)
    if (index > -1) {
      this.accessOrder.splice(index, 1)
    }
    this.accessOrder.push(key)
  }

  evictLRU() {
    if (this.accessOrder.length > 0) {
      const lruKey = this.accessOrder.shift()
      this.cache.delete(lruKey)
    }
  }

  getStats() {
    const valid = Array.from(this.cache.values()).filter(item => !item.isExpired()).length
    const expired = this.cache.size - valid

    return {
      total: this.cache.size,
      valid,
      expired,
      maxSize: this.maxSize,
      utilization: (this.cache.size / this.maxSize * 100).toFixed(1)
    }
  }
}

/**
 * Web存储缓存实现（sessionStorage/localStorage）
 */
class WebStorageCache {
  constructor(storage, maxSize = 1000) {
    this.storage = storage
    this.maxSize = maxSize
    this.prefix = storage === sessionStorage ? 'session_cache_' : 'local_cache_'
  }

  get(key) {
    try {
      const item = JSON.parse(this.storage.getItem(this.prefix + key))
      if (!item) return null

      if (Date.now() > item.expiresAt) {
        this.delete(key)
        return null
      }

      return item.data
    } catch (e) {
      this.delete(key) // 解析失败，删除损坏的数据
      return null
    }
  }

  set(key, data, ttl = DEFAULT_CONFIG.ttl[this.storage === sessionStorage ? CACHE_LEVELS.SESSION : CACHE_LEVELS.LOCAL]) {
    try {
      const item = {
        data,
        expiresAt: Date.now() + ttl,
        createdAt: Date.now()
      }

      // 检查容量限制
      if (this.getSize() >= this.maxSize) {
        this.evictExpired()
        if (this.getSize() >= this.maxSize) {
          this.evictLRU()
        }
      }

      this.storage.setItem(this.prefix + key, JSON.stringify(item))
    } catch (e) {
      console.warn('WebStorage cache set failed:', e)
    }
  }

  delete(key) {
    this.storage.removeItem(this.prefix + key)
  }

  clear() {
    const keys = Object.keys(this.storage)
    keys.forEach(key => {
      if (key.startsWith(this.prefix)) {
        this.storage.removeItem(key)
      }
    })
  }

  getSize() {
    let count = 0
    for (let i = 0; i < this.storage.length; i++) {
      const key = this.storage.key(i)
      if (key && key.startsWith(this.prefix)) {
        count++
      }
    }
    return count
  }

  evictExpired() {
    const keys = Object.keys(this.storage)
    keys.forEach(key => {
      if (key.startsWith(this.prefix)) {
        try {
          const item = JSON.parse(this.storage.getItem(key))
          if (item && Date.now() > item.expiresAt) {
            this.storage.removeItem(key)
          }
        } catch (e) {
          this.storage.removeItem(key)
        }
      }
    })
  }

  evictLRU() {
    // 简单的LRU实现：删除最旧的项
    let oldestKey = null
    let oldestTime = Date.now()

    for (let i = 0; i < this.storage.length; i++) {
      const key = this.storage.key(i)
      if (key && key.startsWith(this.prefix)) {
        try {
          const item = JSON.parse(this.storage.getItem(key))
          if (item.createdAt < oldestTime) {
            oldestTime = item.createdAt
            oldestKey = key
          }
        } catch (e) {
          // 删除损坏的数据
          this.storage.removeItem(key)
        }
      }
    }

    if (oldestKey) {
      this.storage.removeItem(oldestKey)
    }
  }

  getStats() {
    const size = this.getSize()
    return {
      total: size,
      maxSize: this.maxSize,
      utilization: (size / this.maxSize * 100).toFixed(1)
    }
  }
}

/**
 * IndexedDB缓存实现（用于大数据）
 */
class IndexedDBCache {
  constructor(dbName = 'AppCache', storeName = 'cache', maxSize = 10000) {
    this.dbName = dbName
    this.storeName = storeName
    this.maxSize = maxSize
    this.db = null
    this.initPromise = this.initDB()
  }

  async initDB() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open(this.dbName, 1)

      request.onerror = () => reject(request.error)
      request.onsuccess = () => {
        this.db = request.result
        resolve()
      }

      request.onupgradeneeded = (event) => {
        const db = event.target.result
        if (!db.objectStoreNames.contains(this.storeName)) {
          const store = db.createObjectStore(this.storeName, { keyPath: 'key' })
          store.createIndex('expiresAt', 'expiresAt', { unique: false })
        }
      }
    })
  }

  async get(key) {
    await this.initPromise
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction([this.storeName], 'readonly')
      const store = transaction.objectStore(this.storeName)
      const request = store.get(key)

      request.onsuccess = () => {
        const item = request.result
        if (!item) {
          resolve(null)
          return
        }

        if (Date.now() > item.expiresAt) {
          this.delete(key) // 异步删除过期项
          resolve(null)
          return
        }

        resolve(item.data)
      }

      request.onerror = () => reject(request.error)
    })
  }

  async set(key, data, ttl = DEFAULT_CONFIG.ttl[CACHE_LEVELS.INDEXEDDB]) {
    await this.initPromise

    const item = {
      key,
      data,
      expiresAt: Date.now() + ttl,
      createdAt: Date.now()
    }

    return new Promise(async (resolve, reject) => {
      // 检查容量限制
      if (await this.getSize() >= this.maxSize) {
        await this.evictExpired()
        if (await this.getSize() >= this.maxSize) {
          await this.evictLRU()
        }
      }

      const transaction = this.db.transaction([this.storeName], 'readwrite')
      const store = transaction.objectStore(this.storeName)
      const request = store.put(item)

      request.onsuccess = () => resolve()
      request.onerror = () => reject(request.error)
    })
  }

  async delete(key) {
    await this.initPromise
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction([this.storeName], 'readwrite')
      const store = transaction.objectStore(this.storeName)
      const request = store.delete(key)

      request.onsuccess = () => resolve()
      request.onerror = () => reject(request.error)
    })
  }

  async clear() {
    await this.initPromise
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction([this.storeName], 'readwrite')
      const store = transaction.objectStore(this.storeName)
      const request = store.clear()

      request.onsuccess = () => resolve()
      request.onerror = () => reject(request.error)
    })
  }

  async getSize() {
    await this.initPromise
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction([this.storeName], 'readonly')
      const store = transaction.objectStore(this.storeName)
      const request = store.count()

      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })
  }

  async evictExpired() {
    await this.initPromise
    return new Promise((resolve, reject) => {
      const transaction = this.db.transaction([this.storeName], 'readwrite')
      const store = transaction.objectStore(this.storeName)
      const index = store.index('expiresAt')
      const range = IDBKeyRange.upperBound(Date.now())
      const request = index.openCursor(range)

      request.onsuccess = (event) => {
        const cursor = event.target.result
        if (cursor) {
          cursor.delete()
          cursor.continue()
        } else {
          resolve()
        }
      }

      request.onerror = () => reject(request.error)
    })
  }

  async evictLRU() {
    // 简化实现：随机删除一些旧数据
    await this.initPromise
    // 这里可以实现更复杂的LRU逻辑
  }

  async getStats() {
    const size = await this.getSize()
    return {
      total: size,
      maxSize: this.maxSize,
      utilization: (size / this.maxSize * 100).toFixed(1)
    }
  }
}

/**
 * 多层缓存管理器
 */
class MultiLevelCache {
  constructor(config = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }
    this.layers = new Map()

    // 初始化各层缓存
    this.initLayers()
  }

  initLayers() {
    // 内存缓存
    this.layers.set(CACHE_LEVELS.MEMORY, new MemoryCache(this.config.maxSize[CACHE_LEVELS.MEMORY]))

    // Web存储缓存
    if (typeof sessionStorage !== 'undefined') {
      this.layers.set(CACHE_LEVELS.SESSION, new WebStorageCache(sessionStorage, this.config.maxSize[CACHE_LEVELS.SESSION]))
    }

    if (typeof localStorage !== 'undefined') {
      this.layers.set(CACHE_LEVELS.LOCAL, new WebStorageCache(localStorage, this.config.maxSize[CACHE_LEVELS.LOCAL]))
    }

    // IndexedDB缓存
    if (typeof indexedDB !== 'undefined') {
      this.layers.set(CACHE_LEVELS.INDEXEDDB, new IndexedDBCache('AppCache', 'cache', this.config.maxSize[CACHE_LEVELS.INDEXEDDB]))
    }
  }

  async get(key, params = {}, preferredLevel = null) {
    const cacheKey = generateKey(key, params)

    // 如果指定了首选层级，直接从该层获取
    if (preferredLevel && this.layers.has(preferredLevel)) {
      const layer = this.layers.get(preferredLevel)
      try {
        const data = await layer.get(cacheKey)
        if (data !== null) {
          cacheStats.hits++
          // 同步到更快的层级
          this.syncToFasterLayers(cacheKey, data, preferredLevel)
          return data
        }
      } catch (e) {
        cacheStats.errors++
        console.warn(`Cache get from ${preferredLevel} failed:`, e)
      }
    }

    // 多层查找策略
    const searchOrder = this.getSearchOrder(preferredLevel)

    for (const level of searchOrder) {
      if (!this.layers.has(level)) continue

      const layer = this.layers.get(level)
      try {
        const data = await layer.get(cacheKey)
        if (data !== null) {
          cacheStats.hits++
          // 同步到更快的层级
          this.syncToFasterLayers(cacheKey, data, level)
          return data
        }
      } catch (e) {
        cacheStats.errors++
        console.warn(`Cache get from ${level} failed:`, e)
      }
    }

    cacheStats.misses++
    return null
  }

  async set(key, data, params = {}, options = {}) {
    const cacheKey = generateKey(key, params)
    const {
      level = null, // 指定存储层级
      ttl = null,   // 指定TTL
      sync = true   // 是否同步到其他层级
    } = options

    let processedData = data

    // 数据处理
    if (this.config.compression) {
      processedData = compress(processedData)
    }

    if (this.config.encryption) {
      processedData = encrypt(typeof processedData === 'string' ? processedData : JSON.stringify(processedData))
    }

    // 确定存储层级
    const targetLevels = level ? [level] : this.getDefaultStorageLevels(data)

    for (const targetLevel of targetLevels) {
      if (!this.layers.has(targetLevel)) continue

      const layer = this.layers.get(targetLevel)
      const levelTTL = ttl || this.config.ttl[targetLevel]

      try {
        await layer.set(cacheKey, processedData, levelTTL)
        cacheStats.sets++
      } catch (e) {
        cacheStats.errors++
        console.warn(`Cache set to ${targetLevel} failed:`, e)
      }
    }
  }

  async delete(key, params = {}) {
    const cacheKey = generateKey(key, params)

    for (const [level, layer] of this.layers) {
      try {
        await layer.delete(cacheKey)
        cacheStats.deletes++
      } catch (e) {
        cacheStats.errors++
        console.warn(`Cache delete from ${level} failed:`, e)
      }
    }
  }

  async clear(level = null) {
    if (level && this.layers.has(level)) {
      const layer = this.layers.get(level)
      await layer.clear()
    } else {
      // 清理所有层级
      for (const [levelName, layer] of this.layers) {
        try {
          await layer.clear()
        } catch (e) {
          console.warn(`Cache clear ${levelName} failed:`, e)
        }
      }
    }
  }

  getSearchOrder(preferredLevel = null) {
    // 搜索优先级：内存 -> session -> local -> indexeddb
    const defaultOrder = [CACHE_LEVELS.MEMORY, CACHE_LEVELS.SESSION, CACHE_LEVELS.LOCAL, CACHE_LEVELS.INDEXEDDB]

    if (!preferredLevel || !this.layers.has(preferredLevel)) {
      return defaultOrder
    }

    // 将首选层级移到最前面
    const order = [preferredLevel, ...defaultOrder.filter(l => l !== preferredLevel)]
    return order
  }

  getDefaultStorageLevels(data) {
    // 根据数据大小和重要性选择存储层级
    const dataSize = JSON.stringify(data).length

    if (dataSize < 1024) { // 小数据：所有层级
      return [CACHE_LEVELS.MEMORY, CACHE_LEVELS.SESSION, CACHE_LEVELS.LOCAL]
    } else if (dataSize < 10240) { // 中等数据：内存+持久化
      return [CACHE_LEVELS.MEMORY, CACHE_LEVELS.LOCAL, CACHE_LEVELS.INDEXEDDB]
    } else { // 大数据：只持久化
      return [CACHE_LEVELS.INDEXEDDB, CACHE_LEVELS.LOCAL]
    }
  }

  async syncToFasterLayers(key, data, sourceLevel) {
    const fasterLevels = this.getFasterLevels(sourceLevel)

    for (const level of fasterLevels) {
      if (this.layers.has(level)) {
        const layer = this.layers.get(level)
        const ttl = this.config.ttl[level]

        try {
          await layer.set(key, data, ttl)
        } catch (e) {
          // 同步失败不影响主流程
          console.warn(`Cache sync to ${level} failed:`, e)
        }
      }
    }
  }

  getFasterLevels(currentLevel) {
    const levelSpeed = {
      [CACHE_LEVELS.MEMORY]: 4,
      [CACHE_LEVELS.SESSION]: 3,
      [CACHE_LEVELS.LOCAL]: 2,
      [CACHE_LEVELS.INDEXEDDB]: 1
    }

    const currentSpeed = levelSpeed[currentLevel] || 0
    return Object.keys(levelSpeed)
      .filter(level => levelSpeed[level] > currentSpeed)
      .sort((a, b) => levelSpeed[b] - levelSpeed[a]) // 按速度降序
  }

  getStats() {
    const stats = {
      overall: {
        hits: cacheStats.hits,
        misses: cacheStats.misses,
        sets: cacheStats.sets,
        deletes: cacheStats.deletes,
        errors: cacheStats.errors,
        hitRate: cacheStats.hitRate
      },
      layers: {}
    }

    // 各层统计
    for (const [level, layer] of this.layers) {
      try {
        if (layer.getStats) {
          stats.layers[level] = layer.getStats()
        }
      } catch (e) {
        stats.layers[level] = { error: e.message }
      }
    }

    return stats
  }

  // 自动清理
  startAutoCleanup(interval = 5 * 60 * 1000) { // 默认5分钟
    if (!this.config.autoCleanup) return

    setInterval(() => {
      this.cleanup()
    }, interval)
  }

  async cleanup() {
    // 清理各层级的过期数据
    for (const [level, layer] of this.layers) {
      try {
        if (layer.evictExpired) {
          await layer.evictExpired()
        }
      } catch (e) {
        console.warn(`Cache cleanup ${level} failed:`, e)
      }
    }
  }
}

// 创建全局缓存实例
const globalCache = new MultiLevelCache()

// 启动自动清理
if (typeof window !== 'undefined') {
  globalCache.startAutoCleanup()

  // 页面卸载时保存缓存状态
  window.addEventListener('beforeunload', () => {
    try {
      const stats = globalCache.getStats()
      sessionStorage.setItem('cache_stats_backup', JSON.stringify(stats))
    } catch (e) {
      // 忽略保存失败
    }
  })

  // 页面加载时恢复缓存状态（用于调试）
  try {
    const backupStats = sessionStorage.getItem('cache_stats_backup')
    if (backupStats) {
      console.log('[Cache] 恢复之前的缓存统计:', JSON.parse(backupStats))
    }
  } catch (e) {
    // 忽略恢复失败
  }
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

// ============================================================================
// 兼容性API - 保持向后兼容
// ============================================================================

/**
 * 获取缓存数据（兼容性API，默认使用内存缓存）
 * @param {string} key - 缓存键
 * @param {object} params - 参数对象
 * @param {string} level - 缓存层级
 * @returns {Promise<any|null>} 缓存的数据
 */
export async function getCache(key, params = {}, level = CACHE_LEVELS.MEMORY) {
  try {
    return await globalCache.get(key, params, level)
  } catch (e) {
    console.warn('Cache get failed:', e)
    return null
  }
}

/**
 * 设置缓存数据（兼容性API）
 * @param {string} key - 缓存键
 * @param {any} data - 要缓存的数据
 * @param {object} params - 参数对象
 * @param {object} options - 缓存选项
 */
export async function setCache(key, data, params = {}, options = {}) {
  try {
    await globalCache.set(key, data, params, options)
  } catch (e) {
    console.warn('Cache set failed:', e)
  }
}

/**
 * 删除缓存（兼容性API）
 * @param {string} key - 缓存键
 * @param {object} params - 参数对象
 */
export async function deleteCache(key, params = {}) {
  try {
    await globalCache.delete(key, params)
  } catch (e) {
    console.warn('Cache delete failed:', e)
  }
}

/**
 * 清空缓存（兼容性API）
 * @param {string} level - 指定层级，不传则清空所有
 */
export async function clearCache(level = null) {
  try {
    await globalCache.clear(level)
  } catch (e) {
    console.warn('Cache clear failed:', e)
  }
}

/**
 * 获取缓存统计信息（增强版）
 * @returns {object} 缓存统计
 */
export function getCacheStats() {
  return globalCache.getStats()
}

/**
 * 清理过期缓存（兼容性API）
 */
export async function cleanExpiredCache() {
  try {
    await globalCache.cleanup()
  } catch (e) {
    console.warn('Cache cleanup failed:', e)
  }
}

// ============================================================================
// 高级API - 利用多层缓存特性
// ============================================================================

/**
 * 智能缓存 - 根据数据特征自动选择最佳缓存策略
 * @param {string} key - 缓存键
 * @param {Function} dataFetcher - 数据获取函数
 * @param {object} params - 参数对象
 * @param {object} options - 缓存选项
 * @returns {Promise<any>} 缓存或获取的数据
 */
export async function smartCache(key, dataFetcher, params = {}, options = {}) {
  // 先尝试从缓存获取
  let data = await getCache(key, params)

  if (data !== null) {
    return data
  }

  // 缓存未命中，调用数据获取函数
  try {
    data = await dataFetcher()

    if (data !== null && data !== undefined) {
      // 智能设置缓存选项
      const smartOptions = {
        ...options,
        ...getSmartCacheOptions(data)
      }

      await setCache(key, data, params, smartOptions)
    }

    return data
  } catch (e) {
    console.warn('Data fetch failed:', e)
    throw e
  }
}

/**
 * 批量缓存操作
 * @param {Array} operations - 操作数组
 * @returns {Promise<Array>} 结果数组
 */
export async function batchCache(operations) {
  const results = []

  for (const op of operations) {
    try {
      let result
      switch (op.type) {
        case 'get':
          result = await getCache(op.key, op.params, op.level)
          break
        case 'set':
          await setCache(op.key, op.data, op.params, op.options)
          result = true
          break
        case 'delete':
          await deleteCache(op.key, op.params)
          result = true
          break
        default:
          throw new Error(`Unknown operation type: ${op.type}`)
      }
      results.push({ success: true, result })
    } catch (e) {
      results.push({ success: false, error: e.message })
    }
  }

  return results
}

/**
 * 缓存预热 - 预加载常用数据
 * @param {Array} warmupItems - 预热项目数组
 */
export async function warmupCache(warmupItems) {
  const operations = warmupItems.map(item => ({
    type: 'set',
    key: item.key,
    data: item.data,
    params: item.params || {},
    options: {
      level: item.level || CACHE_LEVELS.MEMORY,
      ttl: item.ttl
    }
  }))

  return await batchCache(operations)
}

/**
 * 获取智能缓存选项
 * @param {any} data - 数据对象
 * @returns {object} 缓存选项
 */
function getSmartCacheOptions(data) {
  const options = {}

  // 根据数据大小选择存储层级
  const dataSize = JSON.stringify(data).length

  if (dataSize < 1024) {
    // 小数据：多层存储
    options.level = null // 使用默认策略
  } else if (dataSize < 10240) {
    // 中等数据：内存+持久化
    options.level = CACHE_LEVELS.LOCAL
  } else {
    // 大数据：只用持久化存储
    options.level = CACHE_LEVELS.INDEXEDDB
  }

  // 根据数据类型设置TTL
  if (Array.isArray(data) && data.length > 0) {
    // 列表数据：较短TTL
    options.ttl = 10 * 60 * 1000 // 10分钟
  } else if (typeof data === 'object' && data.timestamp) {
    // 带时间戳的数据：中等TTL
    options.ttl = 30 * 60 * 1000 // 30分钟
  } else {
    // 其他数据：默认TTL
    options.ttl = DEFAULT_CONFIG.ttl[CACHE_LEVELS.MEMORY]
  }

  return options
}

// ============================================================================
// 工具函数
// ============================================================================

/**
 * 导出缓存配置常量
 */
export { CACHE_LEVELS, DEFAULT_CONFIG }

/**
 * 获取缓存实例（高级用法）
 * @returns {MultiLevelCache} 缓存实例
 */
export function getCacheInstance() {
  return globalCache
}

/**
 * 配置缓存系统
 * @param {object} config - 配置对象
 */
export function configureCache(config) {
  Object.assign(DEFAULT_CONFIG, config)
  // 重新初始化缓存实例
  globalCache.config = { ...DEFAULT_CONFIG, ...config }
}

// ============================================================================
// 缓存监控和调试工具
// ============================================================================

/**
 * 缓存调试器
 */
class CacheDebugger {
  constructor() {
    this.enabled = false
    this.logs = []
    this.maxLogs = 1000
    this.performanceMarks = new Map()
  }

  enable() {
    this.enabled = true
    console.log('[CacheDebugger] 缓存调试已启用')
  }

  disable() {
    this.enabled = false
    console.log('[CacheDebugger] 缓存调试已禁用')
  }

  log(operation, key, params = {}, details = {}) {
    if (!this.enabled) return

    const logEntry = {
      timestamp: new Date().toISOString(),
      operation,
      key,
      params,
      ...details
    }

    this.logs.push(logEntry)

    // 限制日志数量
    if (this.logs.length > this.maxLogs) {
      this.logs.shift()
    }

    // 控制台输出（开发模式）
    if (process.env.NODE_ENV === 'development') {
      console.log(`[Cache:${operation}]`, key, params, details)
    }
  }

  markStart(operation, key) {
    if (!this.enabled) return

    const markKey = `${operation}:${key}:${Date.now()}`
    this.performanceMarks.set(markKey, performance.now())
    return markKey
  }

  markEnd(markKey, additionalData = {}) {
    if (!this.enabled || !this.performanceMarks.has(markKey)) return

    const startTime = this.performanceMarks.get(markKey)
    const duration = performance.now() - startTime

    this.performanceMarks.delete(markKey)

    this.log('performance', markKey.split(':')[1], {}, {
      duration: Math.round(duration * 100) / 100, // 保留两位小数
      ...additionalData
    })
  }

  getLogs(filter = {}) {
    let filteredLogs = [...this.logs]

    // 按操作类型过滤
    if (filter.operation) {
      filteredLogs = filteredLogs.filter(log => log.operation === filter.operation)
    }

    // 按键过滤
    if (filter.key) {
      filteredLogs = filteredLogs.filter(log => log.key.includes(filter.key))
    }

    // 按时间范围过滤
    if (filter.startTime) {
      filteredLogs = filteredLogs.filter(log => log.timestamp >= filter.startTime)
    }

    if (filter.endTime) {
      filteredLogs = filteredLogs.filter(log => log.timestamp <= filter.endTime)
    }

    return filteredLogs
  }

  getPerformanceStats() {
    const performanceLogs = this.getLogs({ operation: 'performance' })

    if (performanceLogs.length === 0) {
      return { avgDuration: 0, minDuration: 0, maxDuration: 0, totalOperations: 0 }
    }

    const durations = performanceLogs.map(log => log.duration)
    const sum = durations.reduce((a, b) => a + b, 0)

    return {
      avgDuration: Math.round(sum / durations.length * 100) / 100,
      minDuration: Math.min(...durations),
      maxDuration: Math.max(...durations),
      totalOperations: performanceLogs.length
    }
  }

  clearLogs() {
    this.logs = []
    this.performanceMarks.clear()
  }

  exportLogs() {
    return {
      metadata: {
        exportedAt: new Date().toISOString(),
        totalLogs: this.logs.length,
        enabled: this.enabled
      },
      logs: this.logs,
      stats: this.getPerformanceStats(),
      cacheStats: getCacheStats()
    }
  }
}

// 创建全局调试器实例
const cacheDebugger = new CacheDebugger()

// 在开发模式下自动禁用调试
if (typeof window !== 'undefined' && process.env.NODE_ENV === 'development') {
  cacheDebugger.disable()
}

// ============================================================================
// 监控面板集成
// ============================================================================

/**
 * 缓存监控面板数据提供者
 */
class CacheMonitorProvider {
  constructor() {
    this.updateInterval = 5000 // 5秒更新一次
    this.listeners = new Set()
    this.intervalId = null
  }

  startMonitoring() {
    if (this.intervalId) return

    this.intervalId = setInterval(() => {
      this.notifyListeners()
    }, this.updateInterval)

    console.log('[CacheMonitor] 缓存监控已启动')
  }

  stopMonitoring() {
    if (this.intervalId) {
      clearInterval(this.intervalId)
      this.intervalId = null
    }

    console.log('[CacheMonitor] 缓存监控已停止')
  }

  addListener(callback) {
    this.listeners.add(callback)
  }

  removeListener(callback) {
    this.listeners.delete(callback)
  }

  notifyListeners() {
    const data = this.getMonitorData()
    this.listeners.forEach(callback => {
      try {
        callback(data)
      } catch (e) {
        console.warn('[CacheMonitor] 监听器回调失败:', e)
      }
    })
  }

  getMonitorData() {
    const cacheStats = getCacheStats()
    const debugStats = cacheDebugger.getPerformanceStats()
    const debugLogs = cacheDebugger.getLogs()

    // 计算健康度评分
    const healthScore = this.calculateHealthScore(cacheStats, debugStats)

    return {
      timestamp: new Date().toISOString(),
      cacheStats,
      performanceStats: debugStats,
      recentLogs: debugLogs.slice(-10), // 最近10条日志
      healthScore,
      recommendations: this.generateRecommendations(cacheStats, debugStats)
    }
  }

  calculateHealthScore(cacheStats, perfStats) {
    let score = 100

    // 命中率影响（权重40%）
    const hitRate = parseFloat(cacheStats.overall.hitRate)
    if (hitRate < 50) score -= 40
    else if (hitRate < 70) score -= 20
    else if (hitRate < 85) score -= 10

    // 错误率影响（权重30%）
    const errorRate = cacheStats.overall.errors / (cacheStats.overall.hits + cacheStats.overall.misses + 1) * 100
    if (errorRate > 10) score -= 30
    else if (errorRate > 5) score -= 15
    else if (errorRate > 2) score -= 5

    // 性能影响（权重20%）
    const avgDuration = perfStats.avgDuration
    if (avgDuration > 100) score -= 20
    else if (avgDuration > 50) score -= 10
    else if (avgDuration > 20) score -= 5

    // 内存使用影响（权重10%）
    const memoryUsage = cacheStats.layers?.memory?.utilization
    if (memoryUsage && parseFloat(memoryUsage) > 90) score -= 10
    else if (memoryUsage && parseFloat(memoryUsage) > 75) score -= 5

    return Math.max(0, Math.min(100, score))
  }

  generateRecommendations(cacheStats, perfStats) {
    const recommendations = []

    // 命中率建议
    const hitRate = parseFloat(cacheStats.overall.hitRate)
    if (hitRate < 60) {
      recommendations.push({
        type: 'warning',
        message: '缓存命中率较低，建议增加缓存容量或调整TTL策略',
        action: 'increase_cache_capacity'
      })
    }

    // 性能建议
    if (perfStats.avgDuration > 50) {
      recommendations.push({
        type: 'warning',
        message: '缓存响应时间较慢，建议优化存储层级或减少数据大小',
        action: 'optimize_performance'
      })
    }

    // 内存使用建议
    const memoryUsage = cacheStats.layers?.memory?.utilization
    if (memoryUsage && parseFloat(memoryUsage) > 80) {
      recommendations.push({
        type: 'info',
        message: '内存缓存使用率较高，建议启用LRU淘汰或增加容量',
        action: 'optimize_memory_usage'
      })
    }

    // 错误率建议
    const errorRate = cacheStats.overall.errors / (cacheStats.overall.hits + cacheStats.overall.misses + 1) * 100
    if (errorRate > 5) {
      recommendations.push({
        type: 'error',
        message: '缓存错误率较高，请检查存储后端连接和数据完整性',
        action: 'check_errors'
      })
    }

    return recommendations
  }
}

// 创建全局监控提供者
const cacheMonitor = new CacheMonitorProvider()

// ============================================================================
// 增强的缓存API（带调试和监控）
// ============================================================================

// 包装原始缓存操作，添加调试和监控
const originalGetCache = getCache
const originalSetCache = setCache
const originalDeleteCache = deleteCache

// 重写缓存操作函数
window.getCache = async function(key, params = {}, level = CACHE_LEVELS.MEMORY) {
  const markKey = cacheDebugger.markStart('get', key)

  try {
    const result = await originalGetCache(key, params, level)

    cacheDebugger.markEnd(markKey, {
      hit: result !== null,
      level,
      params
    })

    cacheDebugger.log('get', key, params, {
      hit: result !== null,
      level,
      resultSize: result ? JSON.stringify(result).length : 0
    })

    return result
  } catch (e) {
    cacheDebugger.markEnd(markKey, { error: e.message })
    cacheDebugger.log('get', key, params, { error: e.message, level })
    throw e
  }
}

window.setCache = async function(key, data, params = {}, options = {}) {
  const markKey = cacheDebugger.markStart('set', key)

  try {
    await originalSetCache(key, data, params, options)

    cacheDebugger.markEnd(markKey, {
      dataSize: JSON.stringify(data).length,
      options,
      params
    })

    cacheDebugger.log('set', key, params, {
      dataSize: JSON.stringify(data).length,
      options
    })
  } catch (e) {
    cacheDebugger.markEnd(markKey, { error: e.message })
    cacheDebugger.log('set', key, params, { error: e.message, options })
    throw e
  }
}

window.deleteCache = async function(key, params = {}) {
  const markKey = cacheDebugger.markStart('delete', key)

  try {
    await originalDeleteCache(key, params)

    cacheDebugger.markEnd(markKey, { params })
    cacheDebugger.log('delete', key, params)
  } catch (e) {
    cacheDebugger.markEnd(markKey, { error: e.message })
    cacheDebugger.log('delete', key, params, { error: e.message })
    throw e
  }
}

// 导出调试和监控工具
export { cacheDebugger, cacheMonitor }

// ============================================================================
// 缓存预热服务
// ============================================================================

/**
 * 缓存预热服务
 */
class CacheWarmupService {
  constructor() {
    this.warmupTasks = []
    this.isRunning = false
  }

  /**
   * 注册预热任务
   * @param {string} name - 任务名称
   * @param {Function} task - 预热函数
   * @param {object} options - 选项
   */
  registerTask(name, task, options = {}) {
    this.warmupTasks.push({
      name,
      task,
      priority: options.priority || 5,
      timeout: options.timeout || 10000,
      retryCount: options.retryCount || 3,
      dependencies: options.dependencies || []
    })
  }

  /**
   * 执行预热
   * @param {object} options - 执行选项
   */
  async warmup(options = {}) {
    if (this.isRunning) {
      console.warn('[CacheWarmup] 预热已在进行中')
      return
    }

    this.isRunning = true

    try {
      console.log('[CacheWarmup] 开始缓存预热...')

      // 按优先级排序任务
      const sortedTasks = [...this.warmupTasks].sort((a, b) => b.priority - a.priority)

      const results = []
      const completedTasks = new Set()

      for (const task of sortedTasks) {
        // 检查依赖
        if (!this.checkDependencies(task, completedTasks)) {
          console.log(`[CacheWarmup] 跳过任务 ${task.name}：依赖未满足`)
          continue
        }

        console.log(`[CacheWarmup] 执行预热任务: ${task.name}`)

        let success = false
        for (let attempt = 1; attempt <= task.retryCount; attempt++) {
          try {
            const timeoutPromise = new Promise((_, reject) => {
              setTimeout(() => reject(new Error('Timeout')), task.timeout)
            })

            await Promise.race([task.task(), timeoutPromise])
            success = true
            break
          } catch (e) {
            console.warn(`[CacheWarmup] 任务 ${task.name} 第 ${attempt} 次尝试失败:`, e)
            if (attempt === task.retryCount) {
              console.error(`[CacheWarmup] 任务 ${task.name} 最终失败`)
            }
          }
        }

        results.push({ name: task.name, success })
        if (success) {
          completedTasks.add(task.name)
        }
      }

      const successCount = results.filter(r => r.success).length
      console.log(`[CacheWarmup] 预热完成: ${successCount}/${results.length} 个任务成功`)

      return results
    } finally {
      this.isRunning = false
    }
  }

  /**
   * 检查任务依赖
   */
  checkDependencies(task, completedTasks) {
    return task.dependencies.every(dep => completedTasks.has(dep))
  }

  /**
   * 获取预热状态
   */
  getStatus() {
    return {
      isRunning: this.isRunning,
      taskCount: this.warmupTasks.length,
      tasks: this.warmupTasks.map(t => ({
        name: t.name,
        priority: t.priority,
        timeout: t.timeout
      }))
    }
  }
}

// 创建全局预热服务实例
const cacheWarmupService = new CacheWarmupService()

// 注册默认预热任务
cacheWarmupService.registerTask(
  '用户偏好缓存',
  async () => {
    const prefs = {
      theme: 'dark',
      language: 'zh-CN',
      notifications: true
    }
    await setCache('user_prefs', prefs, {}, {
      level: CACHE_LEVELS.LOCAL,
      ttl: 24 * 60 * 60 * 1000 // 24小时
    })
  },
  { priority: 8, timeout: 2000 }
)

cacheWarmupService.registerTask(
  '市场常量缓存',
  async () => {
    const constants = {
      symbols: ['BTCUSDT', 'ETHUSDT', 'BNBUSDT'],
      timeframes: ['1m', '5m', '1h', '1d'],
      exchanges: ['binance', 'okx', 'bybit']
    }
    await setCache('market_constants', constants, {}, {
      level: CACHE_LEVELS.LOCAL,
      ttl: 7 * 24 * 60 * 60 * 1000 // 7天
    })
  },
  { priority: 9, timeout: 1000 }
)

// 页面加载时自动预热（延迟执行，避免阻塞页面加载）
if (typeof window !== 'undefined') {
  setTimeout(() => {
    cacheWarmupService.warmup().catch(e => {
      console.warn('[CacheWarmup] 自动预热失败:', e)
    })
  }, 2000)
}

// ============================================================================
// 导出工具
// ============================================================================

export { cacheWarmupService }

// 默认导出智能缓存函数作为主要API
export default smartCache


