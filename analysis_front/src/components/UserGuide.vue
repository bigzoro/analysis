<template>
  <Teleport to="body">
    <transition name="modal">
      <div v-if="show" class="guide-overlay" @click="closeGuide">
        <div class="guide-modal" @click.stop>
          <!-- 引导头部 -->
          <div class="guide-header">
            <div class="guide-title">
              <div class="title-icon">🎯</div>
              <div class="title-content">
                <h2>欢迎使用AI量化投资平台</h2>
                <p>让我们快速了解平台的核心功能</p>
              </div>
            </div>
            <button class="close-btn" @click="closeGuide">✕</button>
          </div>

          <!-- 引导内容 -->
          <div class="guide-content">
            <div class="steps-container">
              <div
                v-for="(step, index) in steps"
                :key="step.id"
                :class="['step-item', { active: currentStep === index }]"
                @click="goToStep(index)"
              >
                <div class="step-number">{{ index + 1 }}</div>
                <div class="step-content">
                  <div class="step-title">{{ step.title }}</div>
                  <div class="step-desc">{{ step.description }}</div>
                </div>
                <div class="step-icon">{{ step.icon }}</div>
              </div>
            </div>

            <!-- 详细内容区域 -->
            <div class="step-details">
              <div class="detail-header">
                <div class="detail-icon">{{ currentStepData.icon }}</div>
                <div class="detail-title">{{ currentStepData.title }}</div>
              </div>

              <div class="detail-content">
                <div class="detail-description">{{ currentStepData.description }}</div>

                <div v-if="currentStepData.features" class="feature-list">
                  <h4>核心功能</h4>
                  <div class="features">
                    <div
                      v-for="feature in currentStepData.features"
                      :key="feature.title"
                      class="feature-item"
                    >
                      <div class="feature-icon">{{ feature.icon }}</div>
                      <div class="feature-content">
                        <div class="feature-title">{{ feature.title }}</div>
                        <div class="feature-desc">{{ feature.description }}</div>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="currentStepData.demo" class="demo-section">
                  <h4>功能演示</h4>
                  <div class="demo-content">
                    <div class="demo-text">{{ currentStepData.demo }}</div>
                    <div v-if="currentStepData.demoImage" class="demo-image">
                      <img :src="currentStepData.demoImage" :alt="currentStepData.title" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 引导底部 -->
          <div class="guide-footer">
            <div class="progress-indicator">
              <div
                v-for="(step, index) in steps"
                :key="step.id"
                :class="['progress-dot', { active: currentStep === index }]"
                @click="goToStep(index)"
              ></div>
            </div>

            <div class="footer-actions">
              <button
                class="btn btn-secondary"
                @click="previousStep"
                :disabled="currentStep === 0"
              >
                上一步
              </button>

              <button
                v-if="currentStep < steps.length - 1"
                class="btn btn-primary"
                @click="nextStep"
              >
                下一步
              </button>

              <button
                v-else
                class="btn btn-success"
                @click="completeGuide"
              >
                开始使用 ✨
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'complete'])

const show = ref(false)
const currentStep = ref(0)

const steps = [
  {
    id: 'welcome',
    title: '平台概览',
    description: '了解AI量化投资平台的核心能力',
    icon: '🚀',
    description: 'AI量化投资平台是一款基于人工智能和量化分析的 cryptocurrency 投资决策工具，集成了市场数据分析、AI智能推荐、风险管理和策略回测等全方位功能。',
    features: [
      {
        icon: '📊',
        title: '实时数据监控',
        description: '7×24小时监控各大交易所价格、成交量、资金流向等关键指标'
      },
      {
        icon: '🤖',
        title: 'AI智能推荐',
        description: '基于机器学习算法，为您提供个性化的投资建议和交易信号'
      },
      {
        icon: '⚠️',
        title: '智能风控',
        description: '多维度风险评估，保障投资安全'
      }
    ]
  },
  {
    id: 'market',
    title: '市场数据',
    description: '实时掌握市场动态和趋势',
    icon: '📈',
    description: '平台提供全面的市场数据监控，包括价格走势、交易量分析、资金流向追踪等，帮助您及时把握市场机会。',
    features: [
      {
        icon: '📊',
        title: '涨幅榜单',
        description: '实时更新各大交易对的价格涨幅，快速发现热点'
      },
      {
        icon: '💰',
        title: '资金流向',
        description: '追踪大资金的交易动向，洞察市场情绪'
      },
      {
        icon: '🐋',
        title: '大户监控',
        description: '监控鲸鱼用户的交易行为，提前预警'
      }
    ],
    demo: '点击导航栏的"涨幅榜"即可查看实时市场数据，"资金链"和"大户监控"帮助您深入了解市场资金动向。'
  },
  {
    id: 'ai-features',
    title: 'AI功能',
    description: '体验智能化的投资决策支持',
    icon: '🤖',
    description: '平台集成了先进的AI算法，为投资决策提供科学依据，从数据分析到策略生成，全程AI驱动。',
    features: [
      {
        icon: '🎯',
        title: '智能推荐',
        description: '基于多维度分析，为您推荐优质投资标的'
      },
      {
        icon: '🔬',
        title: 'AI实验室',
        description: '实验最新的AI模型和算法'
      },
      {
        icon: '📊',
        title: 'AI仪表盘',
        description: '可视化展示AI分析结果和投资建议'
      }
    ],
    demo: '访问"AI推荐"页面查看智能投资建议，"AI实验室"体验前沿算法，"AI仪表盘"获取综合投资洞察。'
  },
  {
    id: 'risk-management',
    title: '风险管理',
    description: '专业级的风险控制体系',
    icon: '🛡️',
    description: '平台提供全面的风险管理系统，包括实时风险监控、压力测试、投资组合优化等专业功能。',
    features: [
      {
        icon: '⚠️',
        title: '高级风险分析',
        description: 'VaR、夏普比率等多维度风险指标'
      },
      {
        icon: '📈',
        title: '策略回测',
        description: '历史数据验证策略有效性'
      },
      {
        icon: '🎲',
        title: '蒙特卡洛模拟',
        description: '概率分布分析投资风险'
      }
    ],
    demo: '"高级风险"页面提供专业风险分析工具，"高级回测"验证策略表现，"风险监控"实时守护您的投资安全。'
  },
  {
    id: 'trading',
    title: '交易功能',
    description: '便捷的交易执行和订单管理',
    icon: '📋',
    description: '平台支持交易中心，帮助您制定和执行交易策略，监控订单状态。',
    features: [
      {
        icon: '📝',
        title: '策略订单',
        description: '创建基于条件的自动化交易订单'
      },
      {
        icon: '👀',
        title: '订单监控',
        description: '实时跟踪订单执行状态和结果'
      },
      {
        icon: '📊',
        title: '绩效分析',
        description: '分析交易策略的执行效果'
      }
    ],
    demo: '登录后访问"交易中心"页面，创建和管理您的交易策略。平台会自动执行符合条件的交易。'
  }
]

const currentStepData = computed(() => steps[currentStep.value])

const watch = () => {
  show.value = props.modelValue
}

const closeGuide = () => {
  show.value = false
  emit('update:modelValue', false)
}

const goToStep = (stepIndex) => {
  currentStep.value = stepIndex
}

const nextStep = () => {
  if (currentStep.value < steps.length - 1) {
    currentStep.value++
  }
}

const previousStep = () => {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

const completeGuide = () => {
  // 保存用户已完成引导的状态
  localStorage.setItem('userGuideCompleted', 'true')

  closeGuide()
  emit('complete')

  // 显示完成提示
  if (window.$toast) {
    window.$toast.success('🎉 欢迎使用AI量化投资平台！', '您可以开始探索各项功能了')
  }
}

// 检查是否需要显示引导
const shouldShowGuide = () => {
  const completed = localStorage.getItem('userGuideCompleted')
  const isNewUser = !completed

  if (isNewUser) {
    // 延迟显示，让页面先加载完成
    setTimeout(() => {
      show.value = true
      emit('update:modelValue', true)
    }, 1500)
  }
}

onMounted(() => {
  shouldShowGuide()
})
</script>

<style scoped>
.guide-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  padding: var(--space-4);
}

.guide-modal {
  background: var(--bg-primary);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  max-width: 900px;
  width: 100%;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.guide-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-6);
  border-bottom: 1px solid var(--border-light);
}

.guide-title {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.title-icon {
  font-size: 2rem;
}

.title-content h2 {
  margin: 0 0 var(--space-1) 0;
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
}

.title-content p {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.close-btn {
  background: none;
  border: none;
  font-size: var(--text-xl);
  color: var(--text-muted);
  cursor: pointer;
  padding: var(--space-2);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.close-btn:hover {
  background: var(--bg-secondary);
  color: var(--text-primary);
}

.guide-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.steps-container {
  width: 300px;
  border-right: 1px solid var(--border-light);
  padding: var(--space-4);
  overflow-y: auto;
}

.step-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-bottom: var(--space-2);
}

.step-item:hover {
  background: var(--bg-secondary);
}

.step-item.active {
  background: var(--primary-50);
  border: 1px solid var(--primary-200);
}

.step-number {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: var(--font-semibold);
  font-size: var(--text-sm);
  color: var(--text-secondary);
}

.step-item.active .step-number {
  background: var(--primary-500);
  color: var(--text-inverse);
}

.step-content {
  flex: 1;
}

.step-title {
  font-weight: var(--font-medium);
  color: var(--text-primary);
  font-size: var(--text-sm);
  margin-bottom: var(--space-1);
}

.step-desc {
  font-size: var(--text-xs);
  color: var(--text-muted);
  line-height: 1.4;
}

.step-icon {
  font-size: var(--text-lg);
  opacity: 0.7;
}

.step-details {
  flex: 1;
  padding: var(--space-6);
  overflow-y: auto;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
}

.detail-icon {
  font-size: 3rem;
}

.detail-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
}

.detail-content {
  color: var(--text-secondary);
  line-height: 1.6;
}

.detail-description {
  margin-bottom: var(--space-6);
  font-size: var(--text-base);
}

.feature-list h4 {
  margin: 0 0 var(--space-4) 0;
  color: var(--text-primary);
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

.features {
  display: grid;
  gap: var(--space-4);
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-light);
}

.feature-icon {
  font-size: var(--text-xl);
  margin-top: var(--space-1);
}

.feature-content {
  flex: 1;
}

.feature-title {
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-1);
}

.feature-desc {
  font-size: var(--text-sm);
  color: var(--text-muted);
  line-height: 1.5;
}

.demo-section h4 {
  margin: var(--space-6) 0 var(--space-4) 0;
  color: var(--text-primary);
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

.demo-content {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  border: 1px solid var(--border-light);
}

.demo-text {
  margin-bottom: var(--space-4);
  color: var(--text-secondary);
}

.demo-image {
  text-align: center;
}

.demo-image img {
  max-width: 100%;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
}

.guide-footer {
  border-top: 1px solid var(--border-light);
  padding: var(--space-6);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.progress-indicator {
  display: flex;
  gap: var(--space-2);
}

.progress-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--border-medium);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.progress-dot.active {
  background: var(--primary-500);
  transform: scale(1.2);
}

.footer-actions {
  display: flex;
  gap: var(--space-3);
}

/* 动画 */
.modal-enter-active,
.modal-leave-active {
  transition: all var(--transition-normal);
}

.modal-enter-from {
  opacity: 0;
  transform: scale(0.9);
}

.modal-leave-to {
  opacity: 0;
  transform: scale(0.9);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .guide-modal {
    max-width: 95vw;
    max-height: 95vh;
  }

  .guide-content {
    flex-direction: column;
  }

  .steps-container {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid var(--border-light);
    max-height: 200px;
  }

  .step-item {
    padding: var(--space-2);
  }

  .guide-footer {
    flex-direction: column;
    gap: var(--space-4);
  }

  .footer-actions {
    width: 100%;
    justify-content: space-between;
  }
}

@media (max-width: 480px) {
  .guide-header {
    padding: var(--space-4);
  }

  .guide-title {
    gap: var(--space-3);
  }

  .title-icon {
    font-size: 1.5rem;
  }

  .title-content h2 {
    font-size: var(--text-xl);
  }

  .step-details {
    padding: var(--space-4);
  }

  .detail-header {
    gap: var(--space-3);
  }

  .detail-icon {
    font-size: 2rem;
  }

  .detail-title {
    font-size: var(--text-xl);
  }
}
</style>
