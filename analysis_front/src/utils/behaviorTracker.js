// 用户行为追踪工具
class BehaviorTracker {
  constructor() {
    this.sessionId = this.generateSessionId();
    this.userId = null;
    this.queue = [];
    this.batchSize = 10;
    this.flushInterval = 30000; // 30秒批量发送
    this.init();
  }

  // 初始化
  init() {
    // 从本地存储获取或创建会话ID
    const storedSessionId = localStorage.getItem('session_id');
    if (storedSessionId) {
      this.sessionId = storedSessionId;
    } else {
      localStorage.setItem('session_id', this.sessionId);
    }

    // 获取用户信息
    const authToken = localStorage.getItem('auth_token');
    if (authToken) {
      try {
        const payload = JSON.parse(atob(authToken.split('.')[1]));
        this.userId = payload.user_id;
      } catch (e) {
        console.warn('无法解析用户令牌:', e);
      }
    }

    // 启动定期批量发送
    this.startBatchFlush();

    // 页面卸载时发送剩余数据
    window.addEventListener('beforeunload', () => {
      this.flush();
    });

    // 页面可见性变化时发送数据
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'hidden') {
        this.flush();
      }
    });
  }

  // 生成会话ID
  generateSessionId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
  }

  // 追踪页面访问
  trackPageView(page, metadata = {}) {
    this.track('page_view', page, {
      ...metadata,
      url: window.location.href,
      referrer: document.referrer,
      title: document.title
    });
  }

  // 追踪推荐查看
  trackRecommendationView(recommendation, position) {
    this.track('recommendation_view', recommendation.symbol, {
      recommendation_id: recommendation.id,
      base_symbol: recommendation.base_symbol,
      rank: recommendation.rank,
      total_score: recommendation.total_score,
      position: position,
      page: window.location.pathname
    });
  }

  // 追踪推荐点击
  trackRecommendationClick(recommendation, position) {
    this.track('recommendation_click', recommendation.symbol, {
      recommendation_id: recommendation.id,
      base_symbol: recommendation.base_symbol,
      rank: recommendation.rank,
      position: position,
      page: window.location.pathname
    });
  }

  // 追踪推荐保存
  trackRecommendationSave(recommendation) {
    this.track('recommendation_save', recommendation.symbol, {
      recommendation_id: recommendation.id,
      base_symbol: recommendation.base_symbol,
      rank: recommendation.rank
    });
  }

  // 追踪回测运行
  trackBacktestRun(config) {
    this.track('backtest_run', config.symbol, {
      strategy: config.strategy,
      start_date: config.startDate,
      end_date: config.endDate,
      initial_cash: config.initialCash
    });
  }

  // 追踪搜索行为
  trackSearch(query, resultsCount) {
    this.track('search', query, {
      results_count: resultsCount,
      page: window.location.pathname
    });
  }

  // 追踪筛选行为
  trackFilter(filters) {
    this.track('filter', JSON.stringify(filters), {
      page: window.location.pathname
    });
  }

  // 通用追踪方法
  track(actionType, actionValue, metadata = {}) {
    const event = {
      session_id: this.sessionId,
      user_id: this.userId,
      action_type: actionType,
      action_value: actionValue,
      page: window.location.pathname,
      metadata: {
        ...metadata,
        timestamp: Date.now(),
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone
      },
      user_agent: navigator.userAgent,
      device_info: this.getDeviceInfo(),
      ip_address: null // 由后端获取
    };

    this.queue.push(event);

    // 如果队列达到批量大小，立即发送
    if (this.queue.length >= this.batchSize) {
      this.flush();
    }
  }

  // 获取设备信息
  getDeviceInfo() {
    return {
      platform: navigator.platform,
      language: navigator.language,
      cookie_enabled: navigator.cookieEnabled,
      screen_resolution: `${screen.width}x${screen.height}`,
      viewport_size: `${window.innerWidth}x${window.innerHeight}`,
      touch_support: 'ontouchstart' in window,
      connection_type: navigator.connection?.effectiveType || 'unknown'
    };
  }

  // 批量发送数据
  async flush() {
    if (this.queue.length === 0) return;

    const events = [...this.queue];
    this.queue = [];

    try {
      await fetch('/api/user/behavior/track', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('auth_token') || ''}`
        },
        body: JSON.stringify({ events }),
        keepalive: true // 页面卸载时仍发送
      });
    } catch (error) {
      console.warn('行为数据发送失败:', error);
      // 失败时重新放回队列
      this.queue.unshift(...events);
    }
  }

  // 启动定期批量发送
  startBatchFlush() {
    setInterval(() => {
      this.flush();
    }, this.flushInterval);
  }

  // 设置用户ID（登录后调用）
  setUserId(userId) {
    this.userId = userId;
  }

  // 清除会话（登出时调用）
  clearSession() {
    this.sessionId = this.generateSessionId();
    localStorage.setItem('session_id', this.sessionId);
    this.userId = null;
  }
}

// 创建全局实例
const behaviorTracker = new BehaviorTracker();

// 导出单例
export default behaviorTracker;

// 便捷方法
export const trackPageView = (page, metadata) => behaviorTracker.trackPageView(page, metadata);
export const trackRecommendationView = (recommendation, position) => behaviorTracker.trackRecommendationView(recommendation, position);
export const trackRecommendationClick = (recommendation, position) => behaviorTracker.trackRecommendationClick(recommendation, position);
export const trackRecommendationSave = (recommendation) => behaviorTracker.trackRecommendationSave(recommendation);
export const trackBacktestRun = (config) => behaviorTracker.trackBacktestRun(config);
export const trackSearch = (query, resultsCount) => behaviorTracker.trackSearch(query, resultsCount);
export const trackFilter = (filters) => behaviorTracker.trackFilter(filters);
