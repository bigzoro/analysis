-- 为推荐表添加风险评级字段
ALTER TABLE `coin_recommendations` 
  ADD COLUMN IF NOT EXISTS `volatility_risk` DECIMAL(5,2) COMMENT '波动率风险 0-100',
  ADD COLUMN IF NOT EXISTS `liquidity_risk` DECIMAL(5,2) COMMENT '流动性风险 0-100',
  ADD COLUMN IF NOT EXISTS `market_risk` DECIMAL(5,2) COMMENT '市场风险 0-100',
  ADD COLUMN IF NOT EXISTS `technical_risk` DECIMAL(5,2) COMMENT '技术风险 0-100',
  ADD COLUMN IF NOT EXISTS `overall_risk` DECIMAL(5,2) COMMENT '综合风险 0-100',
  ADD COLUMN IF NOT EXISTS `risk_level` VARCHAR(16) COMMENT '风险等级 low/medium/high',
  ADD COLUMN IF NOT EXISTS `risk_warnings` JSON COMMENT '风险提示（JSON数组）';

