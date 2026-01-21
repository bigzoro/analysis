<template>
  <div class="recommendation-feedback">
    <div class="feedback-header">
      <h4>推荐反馈</h4>
      <button @click="closeFeedback" class="close-btn">×</button>
    </div>

    <div class="feedback-content">
      <div class="recommendation-info">
        <div class="coin-info">
          <span class="coin-symbol">{{ recommendation.symbol }}</span>
          <span class="coin-name">{{ recommendation.base_symbol }}</span>
        </div>
        <div class="score-info">
          <span class="score">{{ recommendation.total_score }}</span>
          <span class="score-label">综合评分</span>
        </div>
      </div>

      <div class="feedback-form">
        <div class="form-group">
          <label>您的评价</label>
          <div class="rating-stars">
            <span
              v-for="star in 5"
              :key="star"
              :class="['star', { active: rating >= star }]"
              @click="setRating(star)"
            >
              ★
            </span>
          </div>
        </div>

        <div class="form-group">
          <label>您的反馈</label>
          <textarea
            v-model="reason"
            placeholder="请分享您对这个推荐的看法..."
            rows="3"
          ></textarea>
        </div>

        <div class="form-actions">
          <button @click="closeFeedback" class="cancel-btn">取消</button>
          <button
            @click="submitFeedback"
            :disabled="submitting"
            class="submit-btn"
          >
            {{ submitting ? '提交中...' : '提交反馈' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue'
import { api } from '@/api/api.js'
import behaviorTracker from '@/utils/behaviorTracker.js'

const props = defineProps({
  recommendation: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'submitted'])

const rating = ref(0)
const reason = ref('')
const submitting = ref(false)

function setRating(value) {
  rating.value = value
}

function closeFeedback() {
  emit('close')
}

async function submitFeedback() {
  if (rating.value === 0) {
    alert('请先选择评分')
    return
  }

  submitting.value = true

  try {
    const feedback = {
      recommendation_id: props.recommendation.id,
      symbol: props.recommendation.symbol,
      base_symbol: props.recommendation.base_symbol,
      action: 'view', // 可以扩展为其他动作
      rating: rating.value,
      reason: reason.value
    }

    await api.submitRecommendationFeedback(feedback)

    // 行为追踪
    behaviorTracker.track('recommendation_feedback', props.recommendation.symbol, {
      recommendation_id: props.recommendation.id,
      rating: rating.value,
      has_reason: reason.value.length > 0
    })

    emit('submitted', feedback)
    closeFeedback()
  } catch (error) {
    console.error('提交反馈失败:', error)
    alert('提交反馈失败，请稍后重试')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.recommendation-feedback {
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  max-width: 400px;
  width: 100%;
}

.feedback-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e9ecef;
}

.feedback-header h4 {
  margin: 0;
  color: #333;
}

.close-btn {
  background: none;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: #666;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.feedback-content {
  padding: 20px;
}

.recommendation-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 16px;
  background: #f8f9fa;
  border-radius: 6px;
}

.coin-info {
  display: flex;
  flex-direction: column;
}

.coin-symbol {
  font-size: 18px;
  font-weight: bold;
  color: #333;
}

.coin-name {
  font-size: 14px;
  color: #666;
}

.score-info {
  text-align: right;
}

.score {
  font-size: 24px;
  font-weight: bold;
  color: #28a745;
  display: block;
}

.score-label {
  font-size: 12px;
  color: #666;
}

.feedback-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-weight: 500;
  color: #333;
}

.rating-stars {
  display: flex;
  gap: 4px;
}

.star {
  font-size: 24px;
  color: #ddd;
  cursor: pointer;
  transition: color 0.2s;
}

.star.active {
  color: #ffc107;
}

.star:hover {
  color: #ffc107;
}

textarea {
  width: 100%;
  padding: 12px;
  border: 1px solid #ced4da;
  border-radius: 4px;
  resize: vertical;
  font-family: inherit;
}

textarea:focus {
  outline: none;
  border-color: #007bff;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.cancel-btn, .submit-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.2s;
}

.cancel-btn {
  background: #6c757d;
  color: white;
}

.cancel-btn:hover {
  background: #5a6268;
}

.submit-btn {
  background: #007bff;
  color: white;
}

.submit-btn:hover:not(:disabled) {
  background: #0056b3;
}

.submit-btn:disabled {
  background: #6c757d;
  cursor: not-allowed;
}
</style>
