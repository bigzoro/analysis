-- 为推荐表添加技术指标字段
ALTER TABLE `coin_recommendations` 
  ADD COLUMN IF NOT EXISTS `technical_indicators` JSON COMMENT '技术指标（JSON格式，包含RSI、MACD等）';

