# 🎨 前端项目使用指南

## 📋 项目概述

基于 Vue.js 3 开发的币种推荐算法系统前端，提供完整的AI推荐和风险监控功能。

### 🎯 主要功能

- **🤖 AI智能推荐**: 基于机器学习的币种推荐系统
- **🛡️ 风险监控中心**: 实时风险评估与智能告警
- **📊 数据可视化**: ECharts图表展示分析结果
- **🔄 实时更新**: WebSocket实时数据推送
- **📱 响应式设计**: 支持桌面端和移动端
- **⏰ 智能定时订单**: 支持策略自动化执行的定时交易系统

## 🚀 快速开始

### 1. 环境准备

```bash
# 确保已安装 Node.js (推荐 16+ 版本)
node --version
npm --version

# 如果没有安装，下载并安装 Node.js
# https://nodejs.org/
```

### 2. 安装依赖

```bash
cd analysis_front

# 安装项目依赖
npm install
```

### 3. 开发模式启动

```bash
# 启动开发服务器
npm run dev

# 浏览器访问: http://localhost:5173
```

### 4. 生产构建

```bash
# 构建生产版本
npm run build

# 预览构建结果
npm run preview
```

## ⚙️ 配置说明

### 环境变量配置

创建 `.env` 文件配置后端API地址：

```env
# 开发环境
VITE_API_BASE=http://127.0.0.1:8010

# 生产环境
# VITE_API_BASE=https://your-api-domain.com

# 默认交易所
VITE_DEFAULT_ENTITY=binance
```

### 后端服务要求

确保后端服务正在运行，并提供以下API接口：

- `/api/v1/recommend` - AI推荐接口
- `/api/v1/risk/report` - 风险报告接口
- `/api/v1/risk/alerts` - 风险告警接口
- `ws://localhost:8080/ws/recommend` - 实时推荐WebSocket

## 📁 项目结构

```
analysis_front/
├── src/
│   ├── api/                 # API接口封装
│   │   └── api.js          # 统一API接口
│   ├── components/         # 通用组件
│   │   ├── Toast.vue       # 消息提示组件
│   │   ├── TopNav.vue      # 顶部导航
│   │   └── ...             # 其他组件
│   ├── views/              # 页面组件
│   │   ├── AIRecommendations.vue    # 🤖 AI推荐页面
│   │   ├── RiskMonitoring.vue       # 🛡️ 风险监控页面
│   │   ├── Dashboard.vue            # 仪表盘
│   │   └── ...                      # 其他页面
│   ├── router/             # 路由配置
│   │   └── router.js       # Vue Router配置
│   ├── stores/             # 状态管理
│   │   └── auth.js         # 认证状态
│   ├── utils/              # 工具函数
│   │   ├── apiClient.js    # API客户端
│   │   ├── behaviorTracker.js # 用户行为追踪
│   │   └── ...             # 其他工具
│   ├── App.vue             # 根组件
│   └── main.js             # 应用入口
├── public/                 # 静态资源
├── package.json           # 项目配置
└── vite.config.js         # Vite配置
```

## 🎨 核心功能详解

### 🤖 AI智能推荐系统

#### 功能特性
- **多币种推荐**: 支持BTC、ETH、ADA等主流币种
- **智能评分**: 基于技术指标、基本面、市场情绪等多维度评分
- **实时更新**: WebSocket实时推荐更新
- **风险控制**: 内置风险评估和仓位建议

#### 使用方法
```javascript
import { api } from '@/api/api.js'

// 获取AI推荐
const recommendations = await api.getAIRecommendations({
  symbols: ['BTC', 'ETH', 'ADA'],
  limit: 5,
  risk_level: 'moderate'
});

// 实时推荐流
const ws = new WebSocket(api.getRealtimeRecommendWS());
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // 处理实时推荐更新
};
```

### ⏰ 智能定时订单系统

#### 功能特性
- **策略自动化**: 支持涨幅前N & 市值阈值等智能策略条件
- **杠杆交易**: 自动设置合约交易杠杆倍数
- **定时执行**: 精确到秒的定时订单执行
- **Bracket订单**: 支持止盈止损的一键三连订单
- **策略预览**: 实时预览符合条件的币种和执行动作

#### 支持的策略类型
```javascript
// 做空策略示例
{
  short_on_gainers: true,        // 开启做空条件
  gainers_rank_limit: 7,         // 涨幅排名前7
  market_cap_limit_short: 5000,  // 市值大于5000万
  short_multiplier: 3.0          // 开空3倍杠杆
}

// 开多策略示例
{
  long_on_small_gainers: true,   // 开启开多条件
  market_cap_limit_long: 2500,   // 市值小于2500万
  gainers_rank_limit_long: 12,   // 涨幅排名前12
  long_multiplier: 2.0           // 开多2倍杠杆
}
```

#### 使用方法
```javascript
// 1. 创建策略
const strategy = await api.createTradingStrategy({
  name: "涨幅前7做空策略",
  conditions: {
    short_on_gainers: true,
    gainers_rank_limit: 7,
    market_cap_limit_short: 5000,
    short_multiplier: 3.0
  }
});

// 2. 预览符合条件的币种
const eligibleSymbols = await api.scanEligibleSymbols(strategy.id);
// 返回: [{ symbol: "BTCUSDT", action: "sell", multiplier: 3.0, ... }]

// 3. 创建定时订单（自动关联策略）
const order = await api.createScheduledOrder({
  symbol: "BTCUSDT",
  exchange: "binance_futures",
  side: "SELL",  // 策略会自动覆盖为正确的方向
  leverage: 1,   // 策略会自动设置为3倍
  strategy_id: strategy.id,
  trigger_time: "2025-01-01T10:00:00Z"
});

// 4. 系统会在指定时间自动执行:
// - 检查策略条件
// - 根据用户选择的"开多仓位"/"开空仓位"/"平多仓位"/"平空仓位"执行相应操作
// - 自动设置杠杆倍数
// - 执行合约交易订单
```

#### 策略执行流程
1. **策略配置**: 在前端设置策略条件和杠杆倍数
2. **条件扫描**: 系统扫描符合涨幅排名和市值条件的币种
3. **订单创建**: 创建定时订单并关联策略
4. **自动执行**: 到期时自动判断执行方向和杠杆倍数
5. **合约交易**: 通过币安期货API执行杠杆交易

### 🛡️ 风险监控中心

#### 功能特性
- **实时告警**: 风险阈值触发即时告警
- **风险评估**: 单个资产和组合风险分析
- **历史追踪**: 风险变化趋势分析
- **智能建议**: 基于风险水平的操作建议

#### 使用方法
```javascript
// 获取风险报告
const riskReport = await api.getRiskReport();

// 评估单个资产风险
const assessment = await api.assessRisk({
  symbol: 'BTC',
  include_history: true
});

// 投资组合风险分析
const portfolioAnalysis = await api.analyzePortfolio(positions, {
  totalValue: 100000,
  riskTolerance: 'moderate'
});
```

## 🛠️ 开发指南

### 添加新页面

1. **创建页面组件**
```javascript
// src/views/NewPage.vue
<template>
  <div class="new-page">
    <h1>新页面</h1>
    <!-- 页面内容 -->
  </div>
</template>

<script>
export default {
  name: 'NewPage',
  // 组件逻辑
}
</script>
```

2. **添加路由**
```javascript
// src/router/router.js
import NewPage from '../views/NewPage.vue'

// 添加路由
{ path: '/new-page', component: NewPage, meta: { title: '新页面' } }
```

3. **更新导航**
```javascript
// src/components/TopNav.vue
<RouterLink to="/new-page" class="tab">新页面</RouterLink>
```

### 添加新API接口

```javascript
// src/api/api.js
export const api = {
  // ... 现有接口

  // 新增接口
  newApiMethod(params) {
    return postJSON('/new/endpoint', params)
  }
}
```

### 使用状态管理

```javascript
// src/stores/newStore.js
import { createStore } from 'vuex'

export default createStore({
  state: {
    data: null
  },
  mutations: {
    setData(state, data) {
      state.data = data
    }
  },
  actions: {
    async fetchData({ commit }) {
      const data = await api.newApiMethod()
      commit('setData', data)
    }
  }
})
```

## 🎨 样式指南

### CSS变量
```css
:root {
  --primary: #667eea;
  --secondary: #764ba2;
  --success: #10b981;
  --warning: #f59e0b;
  --error: #ef4444;
  --text: #333;
  --muted: #666;
  --border: #e0e0e0;
}
```

### 响应式设计
```css
/* 移动端适配 */
@media (max-width: 768px) {
  .container {
    padding: 0 15px;
  }

  .grid {
    grid-template-columns: 1fr;
  }
}
```

## 🧪 测试与调试

### 开发工具
```bash
# 启动Vue DevTools
npm install -g @vue/devtools

# ESLint代码检查
npm run lint

# 格式化代码
npm run format
```

### 调试技巧
- 使用Vue DevTools检查组件状态
- 浏览器Network面板查看API请求
- Console面板查看错误日志
- 使用debugger语句设置断点

## 🚀 部署说明

### Nginx 配置
```nginx
server {
    listen 80;
    server_name your-domain.com;
    root /path/to/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://localhost:8010;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Docker 部署
```dockerfile
FROM node:16-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

## 📚 参考资料

- [Vue.js 官方文档](https://vuejs.org/)
- [Vite 构建工具](https://vitejs.dev/)
- [Pinia 状态管理](https://pinia.vuejs.org/)
- [ECharts 图表库](https://echarts.apache.org/)
- [Element Plus UI库](https://element-plus.org/)

## 🆘 常见问题

### 1. API请求失败
**问题**: 后端API无法访问
**解决**:
```bash
# 检查后端服务状态
curl http://localhost:8080/health

# 检查环境变量配置
echo $VITE_API_BASE

# 查看浏览器控制台错误信息
```

### 2. WebSocket连接失败
**问题**: 实时数据无法接收
**解决**:
```javascript
// 检查WebSocket URL
console.log(api.getRealtimeRecommendWS());

// 手动测试连接
const ws = new WebSocket('ws://localhost:8080/ws/recommend');
ws.onopen = () => console.log('连接成功');
ws.onerror = (error) => console.error('连接失败', error);
```

### 3. 样式不生效
**问题**: CSS样式无法正确应用
**解决**:
```bash
# 重新构建样式
npm run build

# 检查CSS变量定义
:root {
  --primary: #667eea;
  /* 其他变量 */
}
```

## 🎉 总结

恭喜您成功启动币种推荐算法系统的前端项目！

### ✅ 已完成功能
- 🤖 AI智能推荐系统
- 🛡️ 风险监控中心
- 📊 数据可视化
- 🔄 实时数据更新
- 📱 响应式设计
- ⏰ 智能定时订单系统（支持策略自动化执行）

### 🚀 下一步
1. 根据业务需求定制页面
2. 集成更多数据源
3. 添加用户认证功能
4. 优化性能和用户体验

如有任何问题，请参考本文档或提交GitHub Issue。

---

**最后更新时间**: 2025年12月19日
**版本**: v1.1.0
