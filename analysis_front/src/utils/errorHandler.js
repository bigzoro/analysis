// 统一错误处理工具

let toastInstance = null

/**
 * 初始化 Toast 实例
 * @param {Object} instance - Toast 组件实例
 */
export function initToast(instance) {
  toastInstance = instance
}

/**
 * 提取错误消息
 * @param {Error|Object|string} error - 错误对象
 * @returns {string} 错误消息
 */
function extractErrorMessage(error) {
  if (!error) return '未知错误'
  
  if (typeof error === 'string') {
    return error
  }
  
  // 尝试从不同字段获取错误信息
  return error.message || 
         error.error || 
         error.msg || 
         (error.response?.data?.error) ||
         (error.response?.data?.message) ||
         '未知错误'
}

/**
 * 显示 Toast 通知
 * @param {string} message - 消息内容
 * @param {string} type - 类型: success, error, warning, info
 * @param {number} duration - 显示时长（毫秒），0 表示不自动关闭
 */
export function showToast(message, type = 'info', duration = 3000) {
  if (!toastInstance) {
    // 降级到 console 和 alert
    console[type === 'error' ? 'error' : 'log'](`[${type.toUpperCase()}]`, message)
    if (type === 'error') {
      alert(message)
    }
    return
  }
  
  toastInstance.add(message, type, duration)
}

/**
 * 处理错误
 * @param {Error|Object|string} error - 错误对象
 * @param {string} context - 上下文信息（可选）
 * @param {Object} options - 选项
 * @param {boolean} options.log - 是否记录到 console（默认 true）
 * @param {boolean} options.showToast - 是否显示 Toast（默认 true）
 * @param {string} options.customMessage - 自定义错误消息
 */
export function handleError(error, context = '', options = {}) {
  const {
    log = true,
    showToast: show = true,
    customMessage = null
  } = options
  
  const message = customMessage || extractErrorMessage(error)
  const fullMessage = context ? `${context}: ${message}` : message
  
  // 记录错误日志
  if (log) {
    console.error(`[Error${context ? ` - ${context}` : ''}]`, error)
  }
  
  // 显示 Toast 通知
  if (show) {
    showToast(fullMessage, 'error', 4000)
  }
  
  return message
}

/**
 * 处理成功消息
 * @param {string} message - 成功消息
 * @param {string} context - 上下文信息（可选）
 */
export function handleSuccess(message, context = '') {
  const fullMessage = context ? `${context}: ${message}` : message
  showToast(fullMessage, 'success', 2000)
}

/**
 * 处理警告消息
 * @param {string} message - 警告消息
 * @param {string} context - 上下文信息（可选）
 */
export function handleWarning(message, context = '') {
  const fullMessage = context ? `${context}: ${message}` : message
  showToast(fullMessage, 'warning', 3000)
}

/**
 * 处理信息消息
 * @param {string} message - 信息消息
 * @param {string} context - 上下文信息（可选）
 */
export function handleInfo(message, context = '') {
  const fullMessage = context ? `${context}: ${message}` : message
  showToast(fullMessage, 'info', 3000)
}

/**
 * 包装异步函数，自动处理错误
 * @param {Function} fn - 异步函数
 * @param {string} context - 上下文信息
 * @param {Object} options - 选项
 * @returns {Function} 包装后的函数
 */
export function withErrorHandling(fn, context = '', options = {}) {
  return async (...args) => {
    try {
      return await fn(...args)
    } catch (error) {
      handleError(error, context, options)
      throw error // 重新抛出，让调用者可以处理
    }
  }
}

