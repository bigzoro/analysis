<template>
  <header class="topnav">
    <div class="nav-inner container">
      <RouterLink to="/" class="brand">
<!--        <span class="brand-icon">📊</span>-->
        <span class="brand-text">区块链数据分析平台</span>
      </RouterLink>

      <!-- 桌面端导航 -->
      <nav class="nav-desktop">
        <!-- 核心导航标签 -->
        <RouterLink to="/market" class="tab">
<!--          <span class="tab-icon">📈</span>-->
          <span>涨幅榜</span>
        </RouterLink>
        <RouterLink to="/realtime-gainers" class="tab">
<!--          <span class="tab-icon">⚡</span>-->
          <span>实时涨幅榜</span>
        </RouterLink>
<!--        <RouterLink to="/dashboard" class="tab">-->
<!--          <span>仪表盘</span>-->
<!--        </RouterLink>-->

        <!-- 资金流向下拉菜单 -->
<!--        <div class="dropdown" @mouseenter="showDropdown('funds')" @mouseleave="hideDropdown('funds')">-->
<!--          <button class="dropdown-trigger tab">-->
<!--&lt;!&ndash;            <span class="tab-icon">💰</span>&ndash;&gt;-->
<!--            <span>资金流向</span>-->
<!--            <span class="dropdown-arrow">▼</span>-->
<!--          </button>-->
<!--          <div v-show="activeDropdown === 'funds'" class="dropdown-menu">-->
<!--            <RouterLink to="/chain-flows" class="dropdown-item">-->
<!--              <span>资金链</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/whales" class="dropdown-item">-->
<!--              <span>大户监控</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/transfers" class="dropdown-item">-->
<!--              <span>转账实时</span>-->
<!--            </RouterLink>-->
<!--          </div>-->
<!--        </div>-->

        <!-- 其他功能 -->
        <RouterLink to="/realtime-quotes" class="tab">
          <span>实时行情</span>
        </RouterLink>
        <RouterLink to="/announcements" class="tab">
          <span>公告</span>
        </RouterLink>
        <RouterLink to="/data-sync-monitor" class="tab">
          <span>同步监控</span>
        </RouterLink>

        <!-- 交易功能（需要登录） -->
        <RouterLink v-if="isAuthed" to="/scheduled-orders" class="tab">
          <span>交易中心</span>
        </RouterLink>

        <!-- AI功能下拉菜单 -->
<!--        <div class="dropdown" @mouseenter="showDropdown('ai')" @mouseleave="hideDropdown('ai')">-->
<!--          <button class="dropdown-trigger tab">-->
<!--            <span class="tab-icon">🤖</span>-->
<!--            <span>AI功能</span>-->
<!--            <span class="dropdown-arrow">▼</span>-->
<!--          </button>-->
<!--          <div v-show="activeDropdown === 'ai'" class="dropdown-menu">-->
<!--            <RouterLink to="/ai-dashboard" class="dropdown-item">-->
<!--              <span class="tab-icon">🚀</span>-->
<!--              <span>AI仪表盘</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/ai-recommendations" class="dropdown-item">-->
<!--              <span class="tab-icon">🤖</span>-->
<!--              <span>AI推荐</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/ai-lab" class="dropdown-item">-->
<!--              <span class="tab-icon">🔬</span>-->
<!--              <span>AI实验室</span>-->
<!--            </RouterLink>-->
<!--          </div>-->
<!--        </div>-->

        <!-- 风险管理下拉菜单 -->
<!--        <div class="dropdown" @mouseenter="showDropdown('risk')" @mouseleave="hideDropdown('risk')">-->
<!--          <button class="dropdown-trigger tab">-->
<!--            <span class="tab-icon">⚠️</span>-->
<!--            <span>风险管理</span>-->
<!--            <span class="dropdown-arrow">▼</span>-->
<!--          </button>-->
<!--          <div v-show="activeDropdown === 'risk'" class="dropdown-menu">-->
<!--            <RouterLink to="/advanced-risk" class="dropdown-item">-->
<!--              <span class="tab-icon">⚠️</span>-->
<!--              <span>高级风险</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/advanced-backtest" class="dropdown-item">-->
<!--              <span class="tab-icon">📈</span>-->
<!--              <span>高级回测</span>-->
<!--            </RouterLink>-->
<!--            <RouterLink to="/risk-monitoring" class="dropdown-item">-->
<!--              <span class="tab-icon">🛡️</span>-->
<!--              <span>风险监控</span>-->
<!--            </RouterLink>-->
<!--          </div>-->
<!--        </div>-->

      </nav>

      <!-- 移动端汉堡菜单 -->
      <button class="mobile-menu-btn" @click="toggleMobileMenu" :class="{ active: mobileMenuOpen }">
        <span></span>
        <span></span>
        <span></span>
      </button>

      <div class="spacer"></div>

      <!-- 用户认证状态 -->
      <div class="auth" v-if="isAuthed">
        <div class="user-info">
          <div class="user-avatar">
            {{ (username || 'U')[0].toUpperCase() }}
          </div>
          <span class="user-name">{{ username || '用户' }}</span>
        </div>
        <div class="user-actions">
        <button class="logout-btn" @click="onLogout">
<!--          <span>🚪</span>-->
          <span>退出</span>
        </button>
        </div>
      </div>

      <div v-else class="auth">
        <RouterLink to="/login" class="login-btn">
<!--          <span>🔑</span>-->
          <span>登录</span>
        </RouterLink>
<!--        <RouterLink to="/register" class="register-btn">-->
<!--          <span>✨</span>-->
<!--          <span>注册</span>-->
<!--        </RouterLink>-->
      </div>
    </div>

    <!-- 移动端菜单遮罩 -->
    <div v-show="mobileMenuOpen" class="mobile-menu-overlay" @click="closeMobileMenu">
      <nav class="mobile-menu">
        <div class="mobile-menu-header">
          <h3>导航菜单</h3>
          <button class="mobile-menu-close" @click="closeMobileMenu">✕</button>
        </div>

        <!-- 核心功能 -->
        <div class="mobile-menu-section">
          <RouterLink to="/market" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📈</span>
            <span>涨幅榜</span>
          </RouterLink>
          <RouterLink to="/realtime-gainers" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">⚡</span>
            <span>实时涨幅榜</span>
          </RouterLink>
          <RouterLink to="/dashboard" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📊</span>
            <span>仪表盘</span>
          </RouterLink>
        </div>

        <!-- 资金流向 -->
        <div class="mobile-menu-section">
          <h4 class="mobile-menu-title">资金流向</h4>
          <RouterLink to="/chain-flows" class="mobile-menu-item" @click="closeMobileMenu">
            <span>资金链</span>
          </RouterLink>
          <RouterLink to="/whales" class="mobile-menu-item" @click="closeMobileMenu">
            <span>大户监控</span>
          </RouterLink>
          <RouterLink to="/transfers" class="mobile-menu-item" @click="closeMobileMenu">
            <span>转账实时</span>
          </RouterLink>
        </div>

        <!-- AI功能 -->
        <div class="mobile-menu-section">
          <h4 class="mobile-menu-title">AI功能</h4>
          <RouterLink to="/ai-dashboard" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">🚀</span>
            <span>AI仪表盘</span>
          </RouterLink>
          <RouterLink to="/ai-recommendations" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">🤖</span>
            <span>AI推荐</span>
          </RouterLink>
          <RouterLink to="/ai-lab" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">🔬</span>
            <span>AI实验室</span>
          </RouterLink>
        </div>

        <!-- 风险管理 -->
        <div class="mobile-menu-section">
          <h4 class="mobile-menu-title">风险管理</h4>
          <RouterLink to="/advanced-risk" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">⚠️</span>
            <span>高级风险</span>
          </RouterLink>
          <RouterLink to="/advanced-backtest" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📈</span>
            <span>高级回测</span>
          </RouterLink>
          <RouterLink to="/risk-monitoring" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">🛡️</span>
            <span>风险监控</span>
          </RouterLink>
        </div>

        <!-- 其他功能 -->
        <div class="mobile-menu-section">
          <RouterLink to="/realtime-quotes" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📈</span>
            <span>实时行情</span>
          </RouterLink>
          <RouterLink to="/announcements" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📢</span>
            <span>公告</span>
          </RouterLink>
          <RouterLink v-if="isAuthed" to="/scheduled-orders" class="mobile-menu-item" @click="closeMobileMenu">
            <span class="tab-icon">📋</span>
            <span>交易中心</span>
          </RouterLink>
        </div>
      </nav>
    </div>
  </header>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../stores/auth.js'
import { api } from '../api/api.js'

const router = useRouter()
const { isAuthed, username, logout } = useAuth()
const envMode = import.meta.env.MODE

// 下拉菜单状态
const activeDropdown = ref('')

// 移动端菜单状态
const mobileMenuOpen = ref(false)



function onLogout() {
  logout()
  router.replace('/login')
}

// 下拉菜单控制
function showDropdown(menu) {
  activeDropdown.value = menu
}

function hideDropdown(menu) {
  if (activeDropdown.value === menu) {
    activeDropdown.value = ''
  }
}

// 移动端菜单控制
function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

</script>

<style scoped lang="scss">
.topnav {
  position: sticky;
  top: 0;
  z-index: 100;
  backdrop-filter: blur(12px);
  background: var(--bg-overlay);
  border-bottom: 1px solid var(--border-light);
  box-shadow: var(--shadow-sm);
  transition: all var(--transition-normal);
}

.nav-inner {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  padding: var(--space-3) var(--space-6);
  max-width: 1400px;
  margin: 0 auto;
}

.brand {
  font-weight: var(--font-bold);
  font-size: var(--text-lg);
  letter-spacing: 0.025em;
  color: var(--primary-600);
  white-space: nowrap;
  background: linear-gradient(135deg, var(--primary-600), var(--primary-800));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  user-select: none;
}

.nav-desktop {
  display: flex;
  gap: var(--space-1);
  white-space: nowrap;
}

.tab {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  text-decoration: none;
  color: var(--text-secondary);
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  transition: all var(--transition-fast);
  position: relative;
  white-space: nowrap;
}

.tab:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
  transform: translateY(-1px);
}

.tab.router-link-exact-active {
  background: var(--primary-600);
  color: var(--text-inverse);
  border-color: var(--primary-600);
  box-shadow: var(--shadow-md);
}

.tab.router-link-exact-active::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 50%;
  transform: translateX(-50%);
  width: 20px;
  height: 2px;
  background: var(--text-inverse);
  border-radius: 1px;
}

/* 下拉菜单样式 */
.dropdown {
  position: relative;
  margin-top: 2px;
  height: 30px;
}

.dropdown-trigger {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: none;
  border: none;
  cursor: pointer;
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-lg);
  border: 1px solid transparent;
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  transition: all var(--transition-fast);
  color: var(--text-secondary);
  white-space: nowrap;
}

.dropdown-trigger:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
  transform: translateY(-1px);
}

.dropdown-arrow {
  font-size: var(--text-xs);
  transition: transform var(--transition-fast);
}

.dropdown:hover .dropdown-arrow {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  left: 0;
  min-width: 180px;
  background: var(--bg-primary);
  border: 1px solid var(--border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  z-index: 1000;
  padding: var(--space-2) 0;
  margin-top: var(--space-1);
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  color: var(--text-secondary);
  text-decoration: none;
  font-size: var(--text-sm);
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.dropdown-item:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.dropdown-item.router-link-exact-active {
  background: var(--primary-50);
  color: var(--primary-600);
  font-weight: var(--font-medium);
}

/* 移动端汉堡菜单按钮 */
.mobile-menu-btn {
  display: none;
  flex-direction: column;
  justify-content: space-between;
  width: 24px;
  height: 18px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
  margin: 0 var(--space-3) 0 0;
}

.mobile-menu-btn span {
  width: 100%;
  height: 2px;
  background: var(--text-secondary);
  border-radius: 1px;
  transition: all var(--transition-fast);
  transform-origin: center;
}

.mobile-menu-btn.active span:nth-child(1) {
  transform: rotate(45deg) translate(5px, 5px);
}

.mobile-menu-btn.active span:nth-child(2) {
  opacity: 0;
}

.mobile-menu-btn.active span:nth-child(3) {
  transform: rotate(-45deg) translate(7px, -6px);
}

/* 移动端菜单遮罩 */
.mobile-menu-overlay {
  position: fixed;
  top: 60px;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  z-index: 999;
}

.mobile-menu {
  position: absolute;
  top: 0;
  left: 0;
  width: 280px;
  height: 100vh;
  background: var(--bg-primary);
  border-right: 1px solid var(--border-light);
  padding: var(--space-4);
  overflow-y: auto;
}

.mobile-menu-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border-light);
}

.mobile-menu-header h3 {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin: 0;
}

.mobile-menu-close {
  background: none;
  border: none;
  font-size: var(--text-xl);
  color: var(--text-secondary);
  cursor: pointer;
  padding: var(--space-1);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.mobile-menu-close:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.mobile-menu-section {
  margin-bottom: var(--space-6);
}

.mobile-menu-section:last-child {
  margin-bottom: 0;
}

.mobile-menu-title {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--text-muted);
  margin: 0 0 var(--space-2) 0;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.mobile-menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-2);
  color: var(--text-secondary);
  text-decoration: none;
  font-size: var(--text-base);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
  margin-bottom: var(--space-1);
}

.mobile-menu-item:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
  transform: translateX(4px);
}

.mobile-menu-item.router-link-exact-active {
  background: var(--primary-50);
  color: var(--primary-600);
  font-weight: var(--font-medium);
  border-left: 3px solid var(--primary-500);
}

.spacer {
  flex: 1;
}

.auth {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.user-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.user-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--primary-500);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: var(--font-semibold);
  font-size: var(--text-xs);
}

.logout-btn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-medium);
  background: var(--bg-primary);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.logout-btn:hover {
  background: var(--error-50);
  border-color: var(--error-200);
  color: var(--error-600);
}



.login-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  border: 1px solid var(--primary-600);
  background: var(--primary-600);
  color: var(--text-inverse);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.login-btn:hover {
  background: var(--primary-700);
  border-color: var(--primary-700);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.register-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-medium);
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}

.register-btn:hover {
  background: var(--bg-secondary);
  border-color: var(--border-dark);
}

/* 响应式适配 */
@media (max-width: 1024px) {
  .nav-inner {
    padding: var(--space-3) var(--space-4);
  }

  .nav-desktop {
    gap: var(--space-1);
  }

  .dropdown-menu {
    min-width: 160px;
  }

  .tab,
  .dropdown-trigger {
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-xs);
  }

  .brand {
    font-size: var(--text-base);
  }

  .dropdown-item {
    font-size: var(--text-xs);
    padding: var(--space-2) var(--space-3);
  }
}

@media (max-width: 768px) {
  .nav-inner {
    gap: var(--space-3);
    padding: var(--space-2) var(--space-3);
  }

  /* 显示汉堡菜单，隐藏桌面导航 */
  .nav-desktop {
    display: none;
  }

  .mobile-menu-btn {
    display: flex;
  }

  .brand {
    display: none;
  }

  .auth {
    gap: var(--space-2);
  }

  .user-info {
    display: none;
  }

  .login-btn,
  .register-btn {
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-xs);
  }
}

@media (max-width: 480px) {
  .mobile-menu {
    width: 260px;
    padding: var(--space-3);
  }

  .mobile-menu-item {
    font-size: var(--text-sm);
    padding: var(--space-2) var(--space-1);
  }
}

/* 深色模式支持 */
@media (prefers-color-scheme: dark) {
  .topnav {
    background: rgba(0, 0, 0, 0.8);
    border-bottom-color: rgba(255, 255, 255, 0.1);
  }

  .tab,
  .dropdown-trigger {
    color: rgba(255, 255, 255, 0.7);
  }

  .tab:hover,
  .dropdown-trigger:hover {
    background: rgba(255, 255, 255, 0.1);
    color: rgba(255, 255, 255, 0.9);
  }

  .dropdown-menu {
    background: var(--bg-primary);
    border-color: rgba(255, 255, 255, 0.1);
  }

  .dropdown-item {
    color: rgba(255, 255, 255, 0.7);
  }

  .dropdown-item:hover {
    background: rgba(255, 255, 255, 0.1);
    color: rgba(255, 255, 255, 0.9);
  }

  .dropdown-item.router-link-exact-active {
    background: rgba(59, 130, 246, 0.1);
    color: #60a5fa;
  }

  .mobile-menu {
    background: var(--bg-primary);
    border-right-color: rgba(255, 255, 255, 0.1);
  }

  .mobile-menu-header {
    border-bottom-color: rgba(255, 255, 255, 0.1);
  }

  .mobile-menu-item {
    color: rgba(255, 255, 255, 0.7);
  }

  .mobile-menu-item:hover {
    background: rgba(255, 255, 255, 0.1);
    color: rgba(255, 255, 255, 0.9);
  }

  .mobile-menu-item.router-link-exact-active {
    background: rgba(59, 130, 246, 0.1);
    color: #60a5fa;
  }
}
</style>
