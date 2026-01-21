<template>
  <div class="technical-indicators-card">
    <div class="card-header">
      <h3>📈 技术指标分析</h3>
      <div class="trend-badge" :class="'trend-' + (recommendationData?.technical_indicators?.trend || 'sideways')">
        {{ getTrendIcon(recommendationData?.technical_indicators?.trend) }}
        {{ getTrendText(recommendationData?.technical_indicators?.trend) }}
      </div>
    </div>
    <div class="card-body">
      <div class="indicators-grid">
        <!-- RSI指标 -->
        <div class="indicator-item">
          <div class="indicator-header">
            <span class="indicator-name">RSI</span>
            <span class="indicator-value">{{ recommendationData?.technical_indicators?.rsi?.toFixed(2) || 'N/A' }}</span>
          </div>
          <div class="indicator-bar">
            <div class="indicator-fill rsi" :style="{ width: Math.min(recommendationData?.technical_indicators?.rsi || 0, 100) + '%' }"></div>
            <div class="indicator-zones">
              <div class="zone oversold">超卖</div>
              <div class="zone neutral">正常</div>
              <div class="zone overbought">超买</div>
            </div>
          </div>
          <div class="indicator-signal" :class="getRSIClass(recommendationData?.technical_indicators?.rsi)">
            {{ getRSISignal(recommendationData?.technical_indicators?.rsi) }}
          </div>
        </div>

        <!-- MACD指标 -->
        <div class="indicator-item">
          <div class="indicator-header">
            <span class="indicator-name">MACD</span>
            <span class="indicator-value">{{ recommendationData?.technical_indicators?.macd?.toFixed(3) || 'N/A' }}</span>
          </div>
          <div class="macd-details">
            <div class="macd-line">
              <span class="label">MACD线:</span>
              <span class="value">{{ recommendationData?.technical_indicators?.macd?.toFixed(3) || 'N/A' }}</span>
            </div>
            <div class="macd-line">
              <span class="label">信号线:</span>
              <span class="value">{{ recommendationData?.technical_indicators?.macd_signal?.toFixed(3) || 'N/A' }}</span>
            </div>
            <div class="macd-line">
              <span class="label">柱状图:</span>
              <span class="value">{{ recommendationData?.technical_indicators?.macd_hist?.toFixed(3) || 'N/A' }}</span>
            </div>
          </div>
          <div class="indicator-signal" :class="getMACDSignal(recommendationData?.technical_indicators)">
            {{ getMACDSignal(recommendationData?.technical_indicators) }}
          </div>
        </div>

        <!-- 布林带指标 -->
        <div class="indicator-item">
          <div class="indicator-header">
            <span class="indicator-name">布林带</span>
            <span class="indicator-value">{{ (recommendationData?.technical_indicators?.bb_position * 100)?.toFixed(1) || 'N/A' }}%</span>
          </div>
          <div class="bollinger-details">
            <div class="bb-line">
              <span class="label">上轨:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.bb_upper?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="bb-line">
              <span class="label">中轨:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.bb_middle?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="bb-line">
              <span class="label">下轨:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.bb_lower?.toFixed(2) || 'N/A' }}</span>
            </div>
          </div>
          <div class="indicator-signal">
            {{ getBollingerSignal(recommendationData?.technical_indicators?.bb_position) }}
          </div>
        </div>

        <!-- 移动平均线 -->
        <div class="indicator-item">
          <div class="indicator-header">
            <span class="indicator-name">移动平均</span>
            <span class="indicator-value">多头排列</span>
          </div>
          <div class="ma-details">
            <div class="ma-line">
              <span class="label">MA5:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.ma5?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="ma-line">
              <span class="label">MA10:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.ma10?.toFixed(2) || 'N/A' }}</span>
            </div>
            <div class="ma-line">
              <span class="label">MA20:</span>
              <span class="value">${{ recommendationData?.technical_indicators?.ma20?.toFixed(2) || 'N/A' }}</span>
            </div>
          </div>
          <div class="indicator-signal positive">
            金叉信号
          </div>
        </div>
      </div>

      <!-- 支撑阻力位 -->
      <div class="support-resistance">
        <div class="sr-item">
          <span class="label">支撑位:</span>
          <span class="value">${{ recommendationData?.technical_indicators?.support_level?.toFixed(2) || 'N/A' }}</span>
        </div>
        <div class="sr-item">
          <span class="label">阻力位:</span>
          <span class="value">${{ recommendationData?.technical_indicators?.resistance_level?.toFixed(2) || 'N/A' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'

const props = defineProps({
  recommendationData: {
    type: Object,
    default: () => ({})
  }
})

// 技术指标相关方法
const getTrendText = (trend) => {
  const texts = {
    'up': '上涨趋势',
    'down': '下跌趋势',
    'sideways': '横盘震荡'
  }
  return texts[trend] || '未知趋势'
}

const getTrendIcon = (trend) => {
  const icons = {
    'up': '📈',
    'down': '📉',
    'sideways': '➡️'
  }
  return icons[trend] || '❓'
}

const getRSIClass = (rsi) => {
  if (!rsi) return ''
  if (rsi > 70) return 'overbought'
  if (rsi < 30) return 'oversold'
  return 'neutral'
}

const getRSISignal = (rsi) => {
  if (!rsi) return '无数据'
  if (rsi > 70) return '超买'
  if (rsi < 30) return '超卖'
  return '正常'
}

const getMACDSignal = (indicators) => {
  if (!indicators?.macd || !indicators?.macd_signal) return '无数据'
  if (indicators.macd > indicators.macd_signal) return '金叉'
  if (indicators.macd < indicators.macd_signal) return '死叉'
  return '持平'
}

const getBollingerSignal = (position) => {
  if (!position) return '无数据'
  if (position < 0.2) return '下轨附近'
  if (position > 0.8) return '上轨附近'
  return '中轨附近'
}
</script>

<style scoped lang="scss">
.technical-indicators-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  margin-bottom: 24px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  overflow: hidden;

  .card-header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 20px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;

    h3 {
      margin: 0;
      font-size: 20px;
      font-weight: 600;
    }

    .trend-badge {
      padding: 6px 12px;
      border-radius: 20px;
      font-size: 12px;
      font-weight: 500;
      text-transform: uppercase;

      &.trend-up {
        background: #dcfce7;
        color: #166534;
      }

      &.trend-down {
        background: #fee2e2;
        color: #991b1b;
      }

      &.trend-sideways {
        background: #f3f4f6;
        color: #374151;
      }
    }
  }

  .card-body {
    padding: 24px;
  }

  .indicators-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 20px;
    margin-bottom: 30px;

    .indicator-item {
      padding: 20px;
      background: #f8f9fa;
      border-radius: 12px;
      border: 1px solid #e5e7eb;

      .indicator-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;

        .indicator-name {
          font-size: 16px;
          font-weight: 600;
          color: #1a1a1a;
        }

        .indicator-value {
          font-size: 18px;
          font-weight: 700;
          color: #667eea;
        }
      }

      .indicator-bar {
        position: relative;
        height: 20px;
        background: #e5e7eb;
        border-radius: 10px;
        margin-bottom: 12px;
        overflow: hidden;

        .indicator-fill {
          height: 100%;
          background: linear-gradient(90deg, #667eea, #764ba2);
          border-radius: 10px;
          transition: width 0.3s ease;

          &.rsi {
            background: linear-gradient(90deg, #10b981, #059669);
          }
        }

        .indicator-zones {
          position: absolute;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          display: flex;

          .zone {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 10px;
            font-weight: 500;
            color: #666;

            &.oversold {
              background: rgba(16, 185, 129, 0.1);
              color: #059669;
            }

            &.neutral {
              background: rgba(156, 163, 175, 0.1);
              color: #6b7280;
            }

            &.overbought {
              background: rgba(239, 68, 68, 0.1);
              color: #dc2626;
            }
          }
        }
      }

      .macd-details, .bollinger-details, .ma-details {
        margin-bottom: 12px;

        .macd-line, .bb-line, .ma-line {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 6px;
          font-size: 13px;

          .label {
            color: #666;
            font-weight: 500;
          }

          .value {
            color: #1a1a1a;
            font-weight: 600;
          }
        }
      }

      .indicator-signal {
        font-size: 12px;
        font-weight: 500;
        padding: 4px 8px;
        border-radius: 12px;
        text-align: center;

        &.overbought, &.overbought {
          background: #fee2e2;
          color: #dc2626;
        }

        &.oversold {
          background: #dcfce7;
          color: #059669;
        }

        &.gold, &.bullish {
          background: #dcfce7;
          color: #059669;
        }

        &.death, &.bearish {
          background: #fee2e2;
          color: #dc2626;
        }

        &.positive {
          background: #dcfce7;
          color: #059669;
        }
      }
    }
  }

  .support-resistance {
    display: flex;
    gap: 30px;
    justify-content: center;

    .sr-item {
      text-align: center;
      padding: 16px 24px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2);
      color: white;
      border-radius: 12px;
      box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);

      .label {
        font-size: 12px;
        opacity: 0.9;
        margin-bottom: 4px;
        display: block;
      }

      .value {
        font-size: 20px;
        font-weight: 700;
      }
    }
  }
}
</style>
