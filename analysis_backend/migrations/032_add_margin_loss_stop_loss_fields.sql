-- 添加保证金损失止损字段到trading_strategies表
-- 支持基于保证金亏损百分比的智能止损功能
-- +migrate Up

-- 为trading_strategies表添加保证金损失止损相关字段
ALTER TABLE trading_strategies
    ADD COLUMN enable_margin_loss_stop_loss TINYINT(1) DEFAULT 0 COMMENT '启用保证金损失止损',
    ADD COLUMN margin_loss_stop_loss_percent DECIMAL(5,2) DEFAULT 30.00 COMMENT '保证金损失止损百分比';

-- 更新现有记录的默认值
UPDATE trading_strategies
SET margin_loss_stop_loss_percent = 30.00
WHERE margin_loss_stop_loss_percent IS NULL;

-- 添加字段注释说明
-- enable_margin_loss_stop_loss: 是否启用基于保证金亏损的止损机制
-- margin_loss_stop_loss_percent: 当保证金亏损达到此百分比时触发止损（例如30.00表示30%）

-- +migrate Down

-- 移除保证金损失止损相关字段
ALTER TABLE trading_strategies
    DROP COLUMN enable_margin_loss_stop_loss,
    DROP COLUMN margin_loss_stop_loss_percent;